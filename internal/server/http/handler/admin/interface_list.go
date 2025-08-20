package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type InterfaceListHandler struct{ d Dependencies }

func NewInterfaceListHandler(d Dependencies) *InterfaceListHandler {
	return &InterfaceListHandler{d: d}
}

func (h *InterfaceListHandler) Index(c *gin.Context) {
	page, limit := pageLimit(c)
	status := qInt8Ptr(c, "status")
	res, err := h.d.IfList.List(c.Request.Context(), service.ListInterfaceParams{Keywords: c.Query("keywords"), GroupHash: c.Query("group_hash"), Status: status, Page: page, Limit: limit})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *InterfaceListHandler) Add(c *gin.Context) {
	var req struct {
		APIClass, Info, ReturnStr, GroupHash string
		AccessToken, Status, Method, IsTest  int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	id, err := h.d.IfList.Add(c.Request.Context(), service.AddInterfaceParams{APIClass: req.APIClass, AccessToken: req.AccessToken, Status: req.Status, Method: req.Method, Info: req.Info, IsTest: req.IsTest, ReturnStr: req.ReturnStr, GroupHash: req.GroupHash})
	if err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
func (h *InterfaceListHandler) Edit(c *gin.Context) {
	var req struct {
		ID                                   int64
		APIClass, Info, ReturnStr, GroupHash *string
		AccessToken, Status, Method, IsTest  *int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.IfList.Edit(c.Request.Context(), service.EditInterfaceParams{ID: req.ID, APIClass: req.APIClass, AccessToken: req.AccessToken, Status: req.Status, Method: req.Method, Info: req.Info, IsTest: req.IsTest, ReturnStr: req.ReturnStr, GroupHash: req.GroupHash}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *InterfaceListHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.IfList.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *InterfaceListHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.IfList.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *InterfaceListHandler) GetHash(c *gin.Context) {
	response.Success(c, gin.H{"hash": generateUniqID()})
}
func (h *InterfaceListHandler) Refresh(c *gin.Context) {
	tpl := filepath.Clean("../install/apiRoute.tpl")
	out := filepath.Clean("../route/apiRoute.php")
	if err := h.d.IfList.RefreshRoutes(c.Request.Context(), tpl, out); err != nil {
		response.Error(c, retcode.FILE_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func generateUniqID() string { return time.Now().Format("20060102150405.000000000") }
