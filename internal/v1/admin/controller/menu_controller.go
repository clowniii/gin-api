package controller

import (
	"bytes"
	"fmt"
	"gin-app/internal/v1/services"
	adminS "gin-app/internal/v1/services/admin"
	"gin-app/models/admin"
	"github.com/gin-gonic/gin"
	"io"

	"gin-app/utils/response"
)

type menuController struct {
	service *adminS.MenuService
}
type search struct {
	Keywords string `json:"keywords,omitempty"`
}

var MenuController = &menuController{}

func NewMenuController(s *services.Service) *menuController {
	MenuController.service = (*adminS.MenuService)(s)
	return MenuController
}
func (c *menuController) GetMenus(ctx *gin.Context) {
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

	data, err := c.service.GetMenuList(keywords.Keywords)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx, data)
}

func (c *menuController) Add(ctx *gin.Context) {
	menu := admin.AdminMenu{}
	if err := ctx.BindJSON(&menu); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	err := c.service.Add(menu)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}

func (c *menuController) Edit(ctx *gin.Context) {
	menu := admin.AdminMenu{}
	if err := ctx.BindJSON(&menu); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	err := c.service.Edit(menu)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
func (c *menuController) ChangeStatus(ctx *gin.Context) {
	menu := admin.AdminMenu{}
	if err := ctx.BindJSON(&menu); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	changeStatusMenu := admin.AdminMenu{ID: menu.ID, Show: menu.Show}
	err := c.service.Edit(changeStatusMenu)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
func (c *menuController) Del(ctx *gin.Context) {
	menu := admin.AdminMenu{}
	if err := ctx.BindJSON(&menu); err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	err := c.service.Del(menu)
	if err != nil {
		response.Failed(ctx, err.Error())
		return
	}
	response.Success(ctx)
}
