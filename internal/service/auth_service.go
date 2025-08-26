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
	Cfg       *config.Config // 新增: 读取 auth.rotate_refresh / session ttl / login_mode 配置
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
	// === 登录策略处理 ===
	loginMode := "multi"
	maxSessions := 0
	if s.Cfg != nil {
		loginMode = s.Cfg.Auth.LoginMode
		maxSessions = s.Cfg.Auth.MaxMultiSessions
	}

	jti := uuid.NewString()
	roles := []int64{}
	token, err := s.JWT.Generate(user.ID, roles, jti)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	if s.Redis != nil { // 记录当前 jti
		_ = s.Redis.SetTTL(ctx, s.redisJTIPrefix()+jti, 1, s.JWT.ExpireDuration())
		if loginMode == "single" {
			// 单端: 删除该用户之前所有 JTI
			_ = s.clearUserSessions(ctx, user.ID, jti)
		} else if loginMode == "multi" {
			// 多端: 维护一个集合或列表，限制数量
			_ = s.appendUserSession(ctx, user.ID, jti, maxSessions)
		}
	}
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
	// 刷新也需要遵循登录策略（单端/多端），复用逻辑
	if s.Redis != nil {
		if s.Cfg != nil && s.Cfg.Auth.LoginMode == "single" {
			_ = s.clearUserSessions(ctx, uid, jti)
		} else {
			_ = s.appendUserSession(ctx, uid, jti, s.Cfg.Auth.MaxMultiSessions)
		}
	}
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

// ==== 多端/单端登录辅助 Redis Key 设计 ====
// user:sessions:<uid>  -> Redis List 按时间追加(左 push)，元素为 jti
// user:session:set:<uid> -> 可选：维护一个 Set, 用于快速判断是否存在 (目前可不需要, 直接 TTL key 查)

func (s *AuthService) sessionsListKey(uid int64) string {
	return "user:sessions:" + strconv.FormatInt(uid, 10)
}

// clearUserSessions 在单端模式下：删除该用户其他所有 jti，只保留 currentJTI
func (s *AuthService) clearUserSessions(ctx context.Context, uid int64, currentJTI string) error {
	if s.Redis == nil || uid <= 0 {
		return nil
	}
	listKey := s.sessionsListKey(uid)
	// 取出所有旧 jti (LRANGE 全量, 用户会话数通常较小)
	oldJTIs, err := s.Redis.Client.LRange(ctx, listKey, 0, -1).Result()
	if err == nil {
		for _, oj := range oldJTIs {
			if oj == currentJTI || oj == "" {
				continue
			}
			_ = s.Redis.Client.Del(ctx, s.redisJTIPrefix()+oj).Err()
		}
	}
	// 重置列表，只保留当前
	pipe := s.Redis.Client.TxPipeline()
	pipe.Del(ctx, listKey)
	pipe.LPush(ctx, listKey, currentJTI)
	pipe.Expire(ctx, listKey, s.JWT.ExpireDuration())
	_, _ = pipe.Exec(ctx)
	return nil
}

// appendUserSession 在多端模式下: 追加 jti 到列表, 并裁剪到最大会话数
func (s *AuthService) appendUserSession(ctx context.Context, uid int64, jti string, max int) error {
	if s.Redis == nil || uid <= 0 || jti == "" {
		return nil
	}
	listKey := s.sessionsListKey(uid)
	pipe := s.Redis.Client.TxPipeline()
	pipe.LPush(ctx, listKey, jti)
	// 列表 TTL 与 access token 过期保持一致
	pipe.Expire(ctx, listKey, s.JWT.ExpireDuration())
	// 如果有限制, 裁剪并同时删除尾部被挤掉的 jti token key
	var trimNeeded bool
	if max > 0 {
		// 获取长度判断是否需要裁剪
		pipe2 := s.Redis.Client
		length, _ := pipe2.LLen(ctx, listKey).Result()
		if int(length) >= max+5 { // 预判: 如果长度比 max 高很多先修剪 (避免频繁 LTRIM)
			trimNeeded = true
		}
	}
	_, _ = pipe.Exec(ctx)
	if max > 0 { // 精确裁剪
		for {
			length, _ := s.Redis.Client.LLen(ctx, listKey).Result()
			if int(length) <= max { // OK
				break
			}
			// 弹出最后一个 (最旧) jti，并删除其 token key
			oldJTI, err := s.Redis.Client.RPop(ctx, listKey).Result()
			if err != nil || oldJTI == "" {
				break
			}
			_ = s.Redis.Client.Del(ctx, s.redisJTIPrefix()+oldJTI).Err()
		}
	}
	if trimNeeded { // 再确保只保留前 max 个
		if max > 0 {
			_ = s.Redis.Client.LTrim(ctx, listKey, 0, int64(max-1)).Err()
		}
	}
	return nil
}
