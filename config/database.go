package config

import (
	"github.com/jmoiron/sqlx"
)

type Database struct {
	ModelType string `yaml:"model_type"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Database  string `yaml:"database"`
}

func NewDB(dialect string, args string) (db *sqlx.DB, err error) {
	db, err = sqlx.Connect(dialect, args)
	return
}
