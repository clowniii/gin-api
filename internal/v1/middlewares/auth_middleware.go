package middlewares

import (
	"encoding/json"
	admin2 "gin-app/internal/v1/services/admin"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gin-app/config"
	"gin-app/utils/response"
)

func AuthMiddleware(rdb *redis.Client) gin.HandlerFunc {

	return func(ctx *gin.Context) {
		ApiAuth := ctx.GetHeader("Api-Auth")
		if ApiAuth == "" {
			response.Failed(ctx, "missing Authorization header")
			ctx.Abort()
			return
		}
		AuthData := rdb.HGet(ctx, "user_info", ApiAuth)
		if _, err := AuthData.Result(); err != nil {
			response.Failed(ctx, "Authorization is error")
			ctx.Abort()
			return
		}
		AuthExpireTime := rdb.HGet(ctx, "user_expire_time", ApiAuth)
		expireTime := AuthExpireTime.Val()
		now := time.Now().Format("2006-01-02 15:04:05")
		t, _ := time.Parse("2006-01-02 15:04:05", now)
		t2, _ := time.Parse("2006-01-02 15:04:05", expireTime)
		duration := t.Sub(t2)
		if int(duration.Seconds()) > config.Conf.RedisCache.ExpireTime {
			response.Failed(ctx, "Authorization expire")
			ctx.Abort()
			return
		}

		var userInfo admin2.UserInfo

		err := json.Unmarshal([]byte(AuthData.Val()), &userInfo)
		if err != nil {
			response.Failed(ctx, "解码失败，请重试")
			ctx.Abort()
			return
		}

		rdb.HSet(ctx, "user_expire_time", ApiAuth, time.Now().Format("2006-01-02 15:04:05"))
		ctx.Next()
	}
}
