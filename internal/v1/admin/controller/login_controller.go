package controller

import (
	"gin-app/internal/v1/services"
	admin2 "gin-app/internal/v1/services/admin"
	"github.com/gin-gonic/gin"

	"gin-app/models/admin"
	"gin-app/utils"
	"gin-app/utils/response"
)

type loginController struct {
	service *admin2.LoginService
}

var LoginController = &loginController{}

func NewLoginController(s *services.Service) *loginController {
	LoginController.service = (*admin2.LoginService)(s)
	return LoginController
}
func (c *loginController) Login(ctx *gin.Context) {
	var user admin.ClientUser
	if err := ctx.BindJSON(&user); err != nil {
		response.Failed(ctx, err.Error())
		return
	}

	user.IP = utils.IP2Long(ctx.ClientIP())
	data, err := c.service.Login(user, ctx)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx, data)
}
func (c *loginController) GetUserInfo(ctx *gin.Context) {

	data, err := c.service.GetUserInfo(ctx)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx, data)
}
func (c *loginController) Logout(ctx *gin.Context) {
	err := c.service.Logout(ctx)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
func (c *loginController) GetAccessMenu(ctx *gin.Context) {
	var userID struct {
		ID int `json:"id"`
	}
	if err := ctx.BindJSON(&userID); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	data, err := c.service.GetAccessMenu(ctx, userID.ID)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx, data)
}
