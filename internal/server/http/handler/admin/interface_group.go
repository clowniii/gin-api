package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type InterfaceGroupHandler struct{ d Dependencies }

func NewInterfaceGroupHandler(d Dependencies) *InterfaceGroupHandler {
	return &InterfaceGroupHandler{d: d}
}

func (h *InterfaceGroupHandler) Index(c *gin.Context) {
	page, limit := pageLimit(c)
	status := qInt8Ptr(c, "status")
	res, err := h.d.IfGroup.List(c.Request.Context(), service.ListInterfaceGroupParams{Keywords: c.Query("keywords"), AppID: c.Query("app_id"), Status: status, Page: page, Limit: limit})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *InterfaceGroupHandler) Add(c *gin.Context) {
	var req struct {
		Name, AppID, Remark, Hash string
		Status                    int8
		Sort                      int
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	id, err := h.d.IfGroup.Add(c.Request.Context(), service.AddInterfaceGroupParams{Name: req.Name, AppID: req.AppID, Status: req.Status, Sort: req.Sort, Remark: req.Remark, Hash: req.Hash})
	if err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
func (h *InterfaceGroupHandler) Edit(c *gin.Context) {
	var req struct {
		ID                  int64
		Name, AppID, Remark *string
		Status              *int8
		Sort                *int
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.IfGroup.Edit(c.Request.Context(), service.EditInterfaceGroupParams{ID: req.ID, Name: req.Name, AppID: req.AppID, Status: req.Status, Sort: req.Sort, Remark: req.Remark}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *InterfaceGroupHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.IfGroup.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *InterfaceGroupHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.IfGroup.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *InterfaceGroupHandler) GetAll(c *gin.Context) {
	res, err := h.d.IfGroup.List(c.Request.Context(), service.ListInterfaceGroupParams{Status: int8Ptr(1), Page: 1, Limit: 10000})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"list": res.List})
}
