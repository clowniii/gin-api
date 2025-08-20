package security

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

// NewWikiAuth 迁移自 middleware 包
func NewWikiAuth(r *redisrepo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiAuth := c.GetHeader("Api-Auth")
		if apiAuth == "" {
			apiAuth = c.GetHeader("ApiAuth")
		}
		apiAuth = strings.TrimSpace(apiAuth)
		if apiAuth == "" {
			response.Error(c, retcode.AUTH_ERROR, "缺少ApiAuth")
			c.Abort()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
		defer cancel()
		if bs := r.Get(ctx, "Login:"+apiAuth); bs != "" {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(bs), &m)
			if m == nil {
				m = map[string]interface{}{}
			}
			m["app_id"] = "-1"
			c.Set("wiki_user", m)
			c.Next()
			return
		}
		if bs := r.Get(ctx, "WikiLogin:"+apiAuth); bs != "" {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(bs), &m)
			if m == nil {
				m = map[string]interface{}{}
			}
			c.Set("wiki_user", m)
			c.Next()
			return
		}
		response.Error(c, retcode.AUTH_ERROR, "ApiAuth不匹配")
		c.Abort()
	}
}
