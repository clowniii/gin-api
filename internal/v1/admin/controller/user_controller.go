package controller

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"gin-app/internal/v1/services"
	adminS "gin-app/internal/v1/services/admin"
	"gin-app/models/admin"

	"gin-app/utils/response"
)

type userController struct {
	service *adminS.UserService
}

var UserController = &userController{}

func NewUserController(s *services.Service) *userController {
	UserController.service = (*adminS.UserService)(s)
	return UserController
}
func (c *userController) Index(ctx *gin.Context) {
	var keywords search
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) != 0 {
		if err = ctx.BindJSON(&keywords); err != nil {
			fmt.Printf("%+v", keywords)
			response.Failed(ctx, err.Error())
			return
		}
	}

	data, err := c.service.GetList(keywords.Keywords)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx, data)
}
func (c *userController) GetUsers(ctx *gin.Context) {
	var searchUsers struct {
		Size int `json:"size,omitempty"`
		Page int `json:"page,omitempty"`
		Gid  int `json:"gid" binding:"required"`
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) != 0 {
		if err = ctx.BindJSON(&searchUsers); err != nil {
			response.Failed(ctx, err.Error())
			return
		}
	}

	data, err := c.service.GetUsersByGid(searchUsers.Size, searchUsers.Page, searchUsers.Gid)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx, data)
}

func (c *userController) Add(ctx *gin.Context) {
	obj := admin.AdminUser{}
	if err := ctx.BindJSON(&obj); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	err := c.service.Add(obj)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}

func (c *userController) Edit(ctx *gin.Context) {
	obj := admin.AdminUser{}
	if err := ctx.BindJSON(&obj); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	err := c.service.Edit(obj)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
func (c *userController) ChangeStatus(ctx *gin.Context) {
	id := ctx.Query("id")
	status := ctx.Query("status")
	id2, _ := strconv.Atoi(id)
	status2, _ := strconv.Atoi(status)

	type AdminUserRes struct {
		ID     int64
		Status string
	}

	//changeStatus := AdminUserRes{ID: int64(id2), Status: status}
	err := c.service.Edit(admin.AdminUser{ID: int64(id2), Status: int64(status2)})
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
func (c *userController) Del(ctx *gin.Context) {
	obj := admin.AdminUser{}
	if err := ctx.BindJSON(&obj); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	err := c.service.Del(obj)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
