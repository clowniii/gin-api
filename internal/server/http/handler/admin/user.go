package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{ d Dependencies }

func NewUserHandler(d Dependencies) *UserHandler { return &UserHandler{d: d} }

func (h *UserHandler) List(c *gin.Context) {
	page, limit := pageLimit(c)
	status := qInt8Ptr(c, "status")
	res, err := h.d.User.ListUsers(c.Request.Context(), service.ListUsersParams{Username: c.Query("username"), Status: status, Page: page, Limit: limit})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}

func (h *UserHandler) Add(c *gin.Context) {
	var req struct {
		Username, Password, Nickname string
		GroupIDs                     []int64
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	id, err := h.d.User.CreateUser(c.Request.Context(), service.CreateUserParams{Username: strings.TrimSpace(req.Username), Password: req.Password, Nickname: req.Nickname, GroupIDs: req.GroupIDs})
	if err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *UserHandler) Edit(c *gin.Context) {
	var req struct {
		ID       int64 `form:"id" json:"id"`
		Nickname string
		Password string
		Status   *int8
		GroupIDs []int64
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	var pwdPtr *string
	if strings.TrimSpace(req.Password) != "" {
		pwd := req.Password
		pwdPtr = &pwd
	}
	if err := h.d.User.EditUser(c.Request.Context(), service.EditUserParams{ID: req.ID, Nickname: req.Nickname, Password: pwdPtr, Status: req.Status, GroupIDs: req.GroupIDs}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *UserHandler) ChangeStatus(c *gin.Context) {
	id := qInt64(c, "id")
	st := qInt(c, "status", 0)
	if err := h.d.User.ChangeStatus(c.Request.Context(), id, int8(st)); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.User.DeleteUser(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *UserHandler) Own(c *gin.Context) {
	uid := c.GetInt64("user_id")
	var req struct {
		Nickname string
		Password string
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	var pwdPtr *string
	if strings.TrimSpace(req.Password) != "" {
		pwd := req.Password
		pwdPtr = &pwd
	}
	if err := h.d.User.UpdateOwnProfile(c.Request.Context(), service.UpdateOwnParams{UID: uid, Nickname: req.Nickname, Password: pwdPtr}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
