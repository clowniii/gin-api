package response

import (
	"go-apiadmin/internal/util/retcode"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func JSON(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(200, Body{Code: code, Msg: msg, Data: data})
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, retcode.SUCCESS, "success", data)
}

// Error 约定：code 传入 legacy 业务码(负值)。若传入 >=0 且非 SUCCESS，将自动转为 retcode.INVALID。
func Error(c *gin.Context, code int, msg string) {
	if code >= 0 { // 避免误传 HTTP 状态码
		code = retcode.INVALID
	}
	JSON(c, code, msg, nil)
}
