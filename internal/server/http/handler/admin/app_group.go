package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type AppGroupHandler struct{ d Dependencies }

func NewAppGroupHandler(d Dependencies) *AppGroupHandler { return &AppGroupHandler{d: d} }

func (h *AppGroupHandler) Index(c *gin.Context) {
	res, err := h.d.AppGroup.List(c.Request.Context())
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *AppGroupHandler) Add(c *gin.Context) {
	var req struct {
		Name, Description, Hash string
		Status                  int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.AppGroup.Add(c.Request.Context(), service.AddAppGroupParams{Name: req.Name, Description: req.Description, Hash: req.Hash, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AppGroupHandler) Edit(c *gin.Context) {
	var req struct {
		ID                      int64
		Name, Description, Hash *string
		Status                  *int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.AppGroup.Edit(c.Request.Context(), service.EditAppGroupParams{ID: req.ID, Name: req.Name, Description: req.Description, Hash: req.Hash, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AppGroupHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.AppGroup.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AppGroupHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.AppGroup.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
