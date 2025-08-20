package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type MenuHandler struct{ d Dependencies }

func NewMenuHandler(d Dependencies) *MenuHandler { return &MenuHandler{d: d} }

func (h *MenuHandler) Index(c *gin.Context) {
	res, err := h.d.Menu.List(c.Request.Context(), c.Query("title"))
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *MenuHandler) Add(c *gin.Context) {
	var req service.AddMenuParams
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.Menu.Add(c.Request.Context(), req); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *MenuHandler) Edit(c *gin.Context) {
	var req service.EditMenuParams
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.Menu.Edit(c.Request.Context(), req); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *MenuHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.Menu.ChangeStatus(c.Request.Context(), id, st); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *MenuHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.Menu.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
