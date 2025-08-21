package middleware

import (
	"go-apiadmin/internal/config"

	"github.com/gin-gonic/gin"
)

// ConfigInjector 将全局配置对象注入到 gin.Context，供下游中间件/handler 使用
// 仅做一次简单 Set，不包含任何业务逻辑，保持单一职责。
func ConfigInjector(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg != nil { // 防御性判空
			c.Set("app_config", cfg)
		}
		c.Next()
	}
}
