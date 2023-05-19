package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type returnData struct {
	code int
	msg  interface{}
	data interface{}
}

func Success(ctx *gin.Context, arg ...any) {
	var s returnData
	s.code = 1
	s.msg = "操作成功"
	if arg != nil {
		s.data = arg[0]
	}
	ctx.JSON(http.StatusOK, gin.H{"code": s.code, "msg": s.msg, "data": s.data})
}

func Failed(ctx *gin.Context, arg ...any) {
	var s returnData
	s.code = -1
	s.msg = "操作失败"
	switch len(arg) {
	case 1:
		s.msg = arg[0]
		break
	case 2:
		s.msg = arg[0]
		s.data = arg[1]
		break
	}
	ctx.JSON(http.StatusOK, gin.H{"code": s.code, "msg": s.msg, "data": s.data})
}
