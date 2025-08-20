package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthGroupHandler struct{ d Dependencies }

func NewAuthGroupHandler(d Dependencies) *AuthGroupHandler { return &AuthGroupHandler{d: d} }

func (h *AuthGroupHandler) Index(c *gin.Context) {
	res, err := h.d.AuthGroup.List(c.Request.Context())
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *AuthGroupHandler) Add(c *gin.Context) {
	var req struct {
		Name, Description string
		Status            int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.AuthGroup.Add(c.Request.Context(), service.AddGroupParams{Name: req.Name, Description: req.Description, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AuthGroupHandler) Edit(c *gin.Context) {
	var req struct {
		ID                int64
		Name, Description *string
		Status            *int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.AuthGroup.Edit(c.Request.Context(), service.EditGroupParams{ID: req.ID, Name: req.Name, Description: req.Description, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AuthGroupHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.AuthGroup.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AuthGroupHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.AuthGroup.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
