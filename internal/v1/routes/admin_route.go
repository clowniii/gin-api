package routes

import (
	controller2 "gin-app/internal/v1/admin/controller"
	"gin-app/internal/v1/controllers"
	"gin-app/internal/v1/middlewares"
	"gin-app/internal/v1/services"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func AdminRoutes(r *gin.Engine, db *sqlx.DB, rdb *redis.Client) {
	service := services.NewService(db, rdb)
	testuserController := controllers.NewUserController(service)
	loginController := controller2.NewLoginController(service)
	menuController := controller2.NewMenuController(service)
	userController := controller2.NewUserController(service)

	adminRouter := r.Group("/admin")
	{
		test := adminRouter.Group("/users")
		{
			test.Use(middlewares.AuthMiddleware(rdb))
			test.GET("", testuserController.GetAll)
			test.GET("/:id", testuserController.GetByID)
			test.POST("", testuserController.Create)
			test.PUT("/:id", testuserController.Update)
			test.DELETE("/:id", testuserController.Delete)
		}
		login := adminRouter.Group("/Login")
		{
			login.POST("/index", loginController.Login)
		}
		login2 := adminRouter.Group("/Login")
		{
			login2.Use(middlewares.AuthMiddleware(rdb))
			login2.GET("/getUserInfo", loginController.GetUserInfo)
			login2.GET("/logout", loginController.Logout)
		}
		menu := adminRouter.Group("/Menu")
		{
			menu.Use(middlewares.AuthMiddleware(rdb))
			menu.GET("/index", menuController.GetMenus)
			menu.POST("/add", menuController.Add)
			menu.POST("/edit", menuController.Edit)
			menu.POST("/changeStatus", menuController.ChangeStatus)
			menu.POST("/del", menuController.Del)
		}
		user := adminRouter.Group("/User")
		{
			//user.Use(middlewares.AuthMiddleware(rdb))
			user.GET("/index", userController.Index)
			user.GET("/getUsers", userController.GetUsers)
			user.POST("/add", userController.Add)
			user.POST("/edit", userController.Edit)
			user.GET("/changeStatus", userController.ChangeStatus)
			user.POST("/del", userController.Del)
		}
	}
}
