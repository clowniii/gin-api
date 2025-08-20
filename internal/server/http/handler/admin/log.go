package admin

import (
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LogHandler struct{ d Dependencies }

func NewLogHandler(d Dependencies) *LogHandler { return &LogHandler{d: d} }

func (h *LogHandler) List(c *gin.Context) {
	page := qInt(c, "page", 1)
	limit := qInt(c, "size", qInt(c, "limit", 20))
	if limit <= 0 {
		limit = 20
	}
	typ := qInt(c, "type", 0)
	keywords := c.Query("keywords")
	res, err := h.d.Log.List(c.Request.Context(), typ, keywords, page, limit)
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"list": res.List, "count": res.Count})
	c.Status(http.StatusOK)
}
func (h *LogHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if id == 0 {
		response.Error(c, retcode.EMPTY_PARAMS, "缺少必要参数")
		return
	}
	if err := h.d.Log.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, "删除失败")
		return
	}
	response.Success(c, gin.H{"ok": true})
	c.Status(http.StatusOK)
}
