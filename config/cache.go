package config

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	Address    string `yaml:"address"`
	Password   string `yaml:"password"`
	Port       int    `yaml:"port"`
	DB         int    `yaml:"db"`
	ExpireTime int    `yaml:"expire_time"`
}

func NewCache(cache RedisCache) (rdb *redis.Client) {
	str := fmt.Sprintf("%s:%d", cache.Address, cache.Port)
	rdb = redis.NewClient(&redis.Options{
		Addr:     str,
		Password: cache.Password,
		DB:       cache.DB,
	})
	return
}
