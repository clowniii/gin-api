package security

import (
	"context"
	"strings"

	"go-apiadmin/internal/logging"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	JWT    *jwt.Manager
	Logger *logging.Logger
	Redis  *redisrepo.Client
	Prefix string
}

// Auth 认证中间件（与旧 middleware.Auth 等价，迁移至 security 包）
func Auth(j *jwt.Manager, lg *logging.Logger) gin.HandlerFunc {
	m := &AuthMiddleware{JWT: j, Logger: lg}
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			response.Error(c, retcode.AUTH_ERROR, "missing token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(auth[7:])
		claims, err := m.JWT.Parse(token)
		if err != nil {
			response.Error(c, retcode.AUTH_ERROR, "invalid token")
			c.Abort()
			return
		}
		// 单点：校验 JTI 是否存在（若需退出即删除）
		if m.Redis != nil {
			val := m.Redis.Get(context.Background(), m.Prefix+claims.JTI)
			if val == "" { // 不存在
				response.Error(c, retcode.ACCESS_TOKEN_TIMEOUT, "token expired")
				c.Abort()
				return
			}
		}
		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}
