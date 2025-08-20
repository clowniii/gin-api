package service

import (
	"context"
	"errors"
	"go-apiadmin/internal/repository/dao"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	"go-apiadmin/pkg/crypto"

	"github.com/google/uuid"
)

// AuthService 提供用户认证服务
type AuthService struct {
	Users *dao.AdminUserDAO
	JWT   *jwt.Manager
	Redis *redisrepo.Client
}

// NewAuthService 创建一个新的 AuthService 实例
func NewAuthService(u *dao.AdminUserDAO, j *jwt.Manager, r *redisrepo.Client) *AuthService {
	return &AuthService{Users: u, JWT: j, Redis: r}
}

// Login 使用旧 MD5 方案校验
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.Users.FindByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if user == nil || !crypto.VerifyPassword(password, user.Password) {
		return "", errors.New("invalid credentials")
	}
	if user.Status != 1 {
		return "", errors.New("user disabled")
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
		return "", err
	}
	_ = s.Redis.SetTTL(ctx, s.redisJTIPrefix()+jti, 1, s.JWT.ExpireDuration())
	return token, nil
}

// Logout 删除当前 JTI 使 token 立即失效（需在上层解析出 jti）
func (s *AuthService) Logout(ctx context.Context, jti string) error {
	if jti == "" || s.Redis == nil {
		return nil
	}
	return s.Redis.Client.Del(ctx, s.redisJTIPrefix()+jti).Err()
}

func (s *AuthService) redisJTIPrefix() string { return "jwt:jti:" }
