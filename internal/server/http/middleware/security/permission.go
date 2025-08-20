package security

import (
	"strings"

	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

// Permission 中间件：加载用户权限集合并保存于上下文，供 Require 使用
func Permission(permSvc *service.PermissionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetInt64("user_id")
		if uid <= 0 {
			response.Error(c, retcode.AUTH_ERROR, "unauthorized")
			c.Abort()
			return
		}
		perms, _ := permSvc.GetUserPermissions(c.Request.Context(), uid)
		c.Set("perm_svc", permSvc)
		c.Set("perm_set", perms)
		c.Next()
	}
}

// Require 指定当前路由所需权限（基于 URL 匹配）
func Require() gin.HandlerFunc { return RequirePerm() }

// RequirePerm 支持传入一个或多个权限标识；为空时退化为使用 c.FullPath()
func RequirePerm(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetInt64("user_id") == 1 { // 超级管理员
			c.Next()
			return
		}
		svcAny, ok := c.Get("perm_svc")
		if !ok {
			response.Error(c, retcode.UNKNOWN, "permission svc missing")
			c.Abort()
			return
		}
		permSvc := svcAny.(*service.PermissionService)
		uid := c.GetInt64("user_id")
		var targetPerms []string
		if len(perms) == 0 {
			targetPerms = []string{c.FullPath()}
		} else {
			targetPerms = perms
		}
		allowed := false
		if setAny, ok := c.Get("perm_set"); ok {
			if set, ok2 := setAny.(map[string]struct{}); ok2 {
				for _, p := range targetPerms {
					p = strings.ToLower(p)
					if !strings.HasPrefix(p, "/") {
						p = "/" + p
					}
					if _, ok := set[p]; ok {
						allowed = true
						break
					}
				}
			}
		}
		if !allowed {
			for _, p := range targetPerms {
				if permSvc.HasPermission(c.Request.Context(), uid, p) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			response.Error(c, retcode.AUTH_ERROR, "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}
