package middleware

import (
	"net/http"

	"go-apiadmin/internal/util/retcode"

	"github.com/gin-gonic/gin"
)

// Global response wrapper
func ResponseWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// 如果已经写出（状态码已设置并且有响应），且没有 resp，则跳过
		if c.IsAborted() {
			return
		}
		if v, exists := c.Get("resp"); exists {
			if body, ok := v.(gin.H); ok {
				// 兼容：若未设置 code/msg 填充默认
				if _, ok2 := body["code"]; !ok2 {
					body["code"] = retcode.SUCCESS
				}
				if _, ok2 := body["msg"]; !ok2 {
					body["msg"] = "success"
				}
				if _, ok2 := body["data"]; !ok2 {
					body["data"] = gin.H{}
				}
				c.JSON(http.StatusOK, body)
			}
		}
	}
}
