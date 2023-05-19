package controllers

import (
	"net/http"
	"strconv"

	services2 "gin-app/internal/v1/services"

	"github.com/gin-gonic/gin"

	"gin-app/models"
	"gin-app/utils/response"
)

type userController struct {
	service *services2.UserService
}

var UserController = &userController{}

func NewUserController(s *services2.Service) *userController {
	UserController.service = (*services2.UserService)(s)
	return UserController
}

func (c *userController) GetAll(ctx *gin.Context) {
	users, err := c.service.GetAll()
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, users)
}

func (c *userController) GetByID(ctx *gin.Context) {
	id, ok := ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	user, err := c.service.GetByID(id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user == nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (c *userController) Create(ctx *gin.Context) {
	var user models.User
	if err := ctx.BindJSON(&user); err != nil {
		response.Failed(ctx, err.Error())
		//ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.service.Create(&user); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(ctx)
	//ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "msg": "添加成功", "data": []interface{}{}})
}

func (c *userController) Update(ctx *gin.Context) {
	id, ok := ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	var user models.User
	if err := ctx.BindJSON(&user); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.service.Update(id, &user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (c *userController) Delete(ctx *gin.Context) {
	id, ok := ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	if err := c.service.Delete(id); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

func ParseUintParam(ctx *gin.Context, name string) (uint, bool) {
	strVal := ctx.Param(name)
	if strVal == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing parameter: " + name})
		return 0, false
	}
	val, err := strconv.ParseUint(strVal, 10, 32)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid parameter: " + name})
		return 0, false
	}
	return uint(val), true
}
