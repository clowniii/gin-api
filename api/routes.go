package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	v1 "gin-app/internal/v1/routes"
)

func SetupRoutes(r *gin.Engine, db *sqlx.DB, rdb *redis.Client) {
	v1.AdminRoutes(r, db, rdb)
}
