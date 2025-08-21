package service

import (
	"context"
	"errors"
	"fmt"
	"go-apiadmin/internal/repository/dao"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	"go-apiadmin/pkg/crypto"
	"strconv"
	"time"

	"github.com/google/uuid"
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
}

// tracer
func (s *AuthService) tracer() trace.Tracer { return otel.Tracer("service.auth") }

// NewAuthService 创建一个新的 AuthService 实例
func NewAuthService(u *dao.AdminUserDAO, j *jwt.Manager, r *redisrepo.Client) *AuthService {
	return &AuthService{Users: u, JWT: j, Redis: r, JTIPrefix: "jwt:jti:"}
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
	refreshTTL := 7 * 24 * time.Hour
	_ = s.Redis.SetTTL(ctx, refreshKey, user.ID, refreshTTL)
	span.SetStatus(codes.Ok, "login success")
	return token, refreshJTI, nil
}

// Refresh 根据 refresh token 生成新的访问 token，返回 access 与新的 refresh（可旋转）
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	ctx, span := s.tracer().Start(ctx, "AuthService.Refresh")
	defer span.End()
	if refreshToken == "" {
		return "", "", errors.New("empty refresh token")
	}
	uidStr := s.Redis.Get(ctx, s.refreshKey(refreshToken))
	if uidStr == "" {
		err := errors.New("invalid refresh token")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil || uid <= 0 {
		err = errors.New("invalid uid in refresh token store")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}
	// 旋转 refresh token: 删除旧的，创建新的
	s.Redis.Del(ctx, s.refreshKey(refreshToken))
	newRefreshJTI := uuid.NewString()
	_ = s.Redis.SetTTL(ctx, s.refreshKey(newRefreshJTI), uid, 7*24*time.Hour)
	jti := uuid.NewString()
	token, err := s.JWT.Generate(uid, []int64{}, jti)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	_ = s.Redis.SetTTL(ctx, s.redisJTIPrefix()+jti, 1, s.JWT.ExpireDuration())
	span.SetStatus(codes.Ok, "refresh success")
	return token, newRefreshJTI, nil
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
