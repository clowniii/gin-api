package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type AppHandler struct{ d Dependencies }

func NewAppHandler(d Dependencies) *AppHandler { return &AppHandler{d: d} }

func (h *AppHandler) Index(c *gin.Context) {
	page, limit := pageLimit(c)
	status := qInt8Ptr(c, "status")
	res, err := h.d.App.List(c.Request.Context(), service.ListAppParams{Keywords: c.Query("keywords"), Status: status, Page: page, Limit: limit})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *AppHandler) Add(c *gin.Context) {
	var req struct {
		AppName, AppInfo, AppGroup string
		Status                     int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	id, err := h.d.App.Add(c.Request.Context(), service.AddAppParams{AppName: req.AppName, AppInfo: req.AppInfo, AppGroup: req.AppGroup, Status: req.Status})
	if err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
func (h *AppHandler) Edit(c *gin.Context) {
	var req struct {
		ID                         int64
		AppName, AppInfo, AppGroup *string
		Status                     *int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.App.Edit(c.Request.Context(), service.EditAppParams{ID: req.ID, AppName: req.AppName, AppInfo: req.AppInfo, AppGroup: req.AppGroup, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AppHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.App.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AppHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.App.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AppHandler) GetInfo(c *gin.Context) {
	id := qInt64(c, "id")
	res, err := h.d.App.GetInfo(c.Request.Context(), id)
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *AppHandler) RefreshSecret(c *gin.Context) {
	id := qInt64(c, "id")
	sec, err := h.d.App.RefreshSecret(c.Request.Context(), id)
	if err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"secret": sec})
}
