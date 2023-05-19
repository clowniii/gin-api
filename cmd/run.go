package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	v1 "gin-app/api"
	"gin-app/config"
	"gin-app/utils/logs"
)

func Run() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	var conf = config.Conf
	modelLinkString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", conf.Database.Username, conf.Database.Password, conf.Database.Host, conf.Database.Port, conf.Database.Database)
	db, err := config.NewDB(conf.ModelType, modelLinkString)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(50)
	db.SetConnMaxLifetime(1 * time.Hour)

	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		logs.Error(err.Error())
	}
	defer func(db *sqlx.DB) {
		err = db.Close()
		if err != nil {
			logs.Error(err.Error())
			log.Fatal(err)
		}
	}(db)
	rdb := config.NewCache(conf.RedisCache)

	logs.InitLogger()

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "AccessToken", "X-CSRF-Token", "Authorization", "Token", "Api-Auth"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		AllowCredentials: true,
	}))
	router.Use(gin.Recovery())
	v1.SetupRoutes(router, db, rdb)
	gin.SetMode(gin.DebugMode) //开启dug
	gin.ForceConsoleColor()    //日志颜色打印
	_ = router.Run(fmt.Sprintf("%s:%s", conf.Server.Address, conf.Server.Port))
}
