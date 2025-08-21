package security

import (
	"context"
	"strings"

	"go-apiadmin/internal/config"
	"go-apiadmin/internal/logging"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		// 若配置中有 jti_prefix, 动态设置 (通过全局 config 放在 gin context? 简化: 读取已注入的 redis prefix via header later)
		// 这里如果中间件实例有 Prefix 则使用
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
			prefix := m.Prefix
			if prefix == "" {
				// 尝试从全局配置（已通过依赖注入）放置的值
				if cfgAny, ok := c.MustGet("app_config").(*config.Config); ok {
					prefix = cfgAny.Redis.JTIPrefix
				}
			}
			val := m.Redis.Get(context.Background(), prefix+claims.JTI)
			if val == "" { // 不存在
				response.Error(c, retcode.ACCESS_TOKEN_TIMEOUT, "token expired")
				c.Abort()
				return
			}
		}
		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)
		// 注入 logger 上下文字段
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		// trace_id 已由 TraceMiddleware 设置
		lg.WithContext(ctx).Info("auth_ok", zap.Int64("user_id", claims.UserID))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
