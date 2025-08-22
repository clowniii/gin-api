package service

import (
	"context"
	"errors"
	"fmt"
	"go-apiadmin/internal/config"
	"go-apiadmin/internal/metrics"
	"go-apiadmin/internal/repository/dao"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	"go-apiadmin/pkg/crypto"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AuthService 提供用户认证服务
type AuthService struct {
	Users     *dao.AdminUserDAO
	JWT       *jwt.Manager
	Redis     *redisrepo.Client
	JTIPrefix string
	Cfg       *config.Config // 新增: 读取 auth.rotate_refresh / session ttl 配置
}

// tracer
func (s *AuthService) tracer() trace.Tracer { return otel.Tracer("service.auth") }

// NewAuthService 创建一个新的 AuthService 实例
func NewAuthService(u *dao.AdminUserDAO, j *jwt.Manager, r *redisrepo.Client, cfg *config.Config) *AuthService {
	return &AuthService{Users: u, JWT: j, Redis: r, JTIPrefix: rPrefix(r), Cfg: cfg}
}

func rPrefix(r *redisrepo.Client) string { // 兼容 nil
	if r == nil {
		return "jwt:jti:"
	}
	return "jwt:jti:"
}

// Login 使用旧 MD5 方案校验，返回 accessToken 与 refreshToken
func (s *AuthService) Login(ctx context.Context, username, password string) (string, string, error) {
	ctx, span := s.tracer().Start(ctx, "AuthService.Login")
	defer span.End()
	user, err := s.Users.FindByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", fmt.Errorf("find user: %w", err)
	}
	if user == nil || !crypto.VerifyPassword(password, user.Password) {
		err = errors.New("invalid credentials")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}
	if user.Status != 1 {
		err = errors.New("user disabled")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}
	// 如果是 legacy md5，异步升级为 bcrypt
	if crypto.IsLegacyMD5(user.Password) {
		go func(uid int64, plain string) {
			// 忽略上下文取消
			h := crypto.HashPassword(plain)
			_ = s.Users.UpdatePassword(context.Background(), uid, h)
		}(user.ID, password)
	}
	jti := uuid.NewString()
	token, err := s.JWT.Generate(user.ID, []int64{}, jti)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	_ = s.Redis.SetTTL(ctx, s.redisJTIPrefix()+jti, 1, s.JWT.ExpireDuration())
	// 生成 refresh token
	refreshJTI := uuid.NewString()
	refreshKey := s.refreshKey(refreshJTI)
	refreshTTL := s.refreshTTL()
	_ = s.Redis.SetTTL(ctx, refreshKey, user.ID, refreshTTL)
	span.SetStatus(codes.Ok, "login success")
	return token, refreshJTI, nil
}

// Refresh 根据 refresh token 生成新的访问 token，返回 access 与新的/旧的 refresh 以及 userID
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, int64, error) {
	ctx, span := s.tracer().Start(ctx, "AuthService.Refresh")
	defer span.End()
	if refreshToken == "" {
		return "", "", 0, errors.New("empty refresh token")
	}
	// 优先读取 uid
	key := s.refreshKey(refreshToken)
	uidStr := s.Redis.Get(ctx, key)
	if uidStr == "" {
		err := errors.New("invalid refresh token")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", 0, err
	}
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil || uid <= 0 {
		err = errors.New("invalid uid in refresh token store")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", 0, err
	}
	rotate := s.Cfg == nil || s.Cfg.Auth.RotateRefresh
	newRefresh := refreshToken
	if rotate {
		// 使用 Lua 原子操作: 删除旧 key 写入新 key，避免并发刷新导致旧 token 仍可用
		var script = redis.NewScript(`local v=redis.call('GET', KEYS[1]); if not v then return {0,''}; end; redis.call('DEL', KEYS[1]); redis.call('SET', KEYS[2], v, 'PX', ARGV[1]); return {1, v};`)
		newRefresh = uuid.NewString()
		newKey := s.refreshKey(newRefresh)
		ttl := s.refreshTTL()
		res, err := script.Run(ctx, s.Redis.Client, []string{key, newKey}, ttl.Milliseconds()).Result()
		if err != nil {
			// 回退到非原子方案
			s.Redis.Del(ctx, key)
			_ = s.Redis.SetTTL(ctx, newKey, uid, ttl)
		} else {
			if arr, ok := res.([]interface{}); ok && len(arr) == 2 {
				if okFlag, _ := arr[0].(int64); okFlag == 1 {
					metrics.AuthRefreshRotateTotal.Inc()
				}
			}
		}
	}
	jti := uuid.NewString()
	token, err := s.JWT.Generate(uid, []int64{}, jti)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", 0, fmt.Errorf("generate token: %w", err)
	}
	_ = s.Redis.SetTTL(ctx, s.redisJTIPrefix()+jti, 1, s.JWT.ExpireDuration())
	span.SetStatus(codes.Ok, "refresh success")
	return token, newRefresh, uid, nil
}

// Logout 删除当前 JTI 使 token 立即失效（需在上层解析出 jti）
func (s *AuthService) Logout(ctx context.Context, jti string) error {
	if jti == "" || s.Redis == nil {
		return nil
	}
	return s.Redis.Client.Del(ctx, s.redisJTIPrefix()+jti).Err()
}

func (s *AuthService) redisJTIPrefix() string       { return s.JTIPrefix }
func (s *AuthService) refreshKey(jti string) string { return "jwt:refresh:" + jti }
func (s *AuthService) refreshTTL() time.Duration {
	// refresh token TTL 采用会话 TTL 的 2 倍（或默认 7 天）: 可配置需求可再扩展
	if s.Cfg != nil && s.Cfg.Auth.SessionTTLSeconds > 0 {
		st := time.Duration(s.Cfg.Auth.SessionTTLSeconds) * time.Second
		if st*2 > 24*time.Hour*14 { // 上限 14 天
			return 14 * 24 * time.Hour
		}
		if st*2 < time.Hour { // 下限 1 小时
			return time.Hour
		}
		return st * 2
	}
	return 7 * 24 * time.Hour
}
