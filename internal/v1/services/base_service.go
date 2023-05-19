package services

import (
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	DB  *sqlx.DB
	RDB *redis.Client
}

func NewService(db *sqlx.DB, rdb *redis.Client) *Service {
	return &Service{DB: db, RDB: rdb}
}
