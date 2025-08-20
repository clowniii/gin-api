package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthRuleHandler struct{ d Dependencies }

func NewAuthRuleHandler(d Dependencies) *AuthRuleHandler { return &AuthRuleHandler{d: d} }

func (h *AuthRuleHandler) Index(c *gin.Context) {
	var gid *int64
	if v := qInt64(c, "group_id"); v > 0 {
		gid = &v
	}
	res, err := h.d.AuthRule.List(c.Request.Context(), service.ListRuleParams{GroupID: gid})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *AuthRuleHandler) Add(c *gin.Context) {
	var req struct {
		URL     string
		GroupID int64
		Auth    int64
		Status  int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.AuthRule.Add(c.Request.Context(), service.AddRuleParams{URL: req.URL, GroupID: req.GroupID, Auth: req.Auth, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AuthRuleHandler) Edit(c *gin.Context) {
	var req struct {
		ID      int64
		URL     *string
		GroupID *int64
		Auth    *int64
		Status  *int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.AuthRule.Edit(c.Request.Context(), service.EditRuleParams{ID: req.ID, URL: req.URL, GroupID: req.GroupID, Auth: req.Auth, Status: req.Status}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AuthRuleHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.AuthRule.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AuthRuleHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.AuthRule.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
