package postgres

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN         string
	MaxOpen     int
	MaxIdle     int
	AutoMigrate bool
}

func New(cfg Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		return nil, err
	}
	// otelgorm 插件在 provider 中统一开启 (boot/provider.go) 确保只注册一次
	// 若未来需要在此处直接启用可： db.Use(tracing.NewPlugin())
	// 保持最小职责，这里不重复启用。
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpen > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	}
	if cfg.MaxIdle > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	}
	sqlDB.SetConnMaxLifetime(2 * time.Hour)
	return db, nil
}

// AutoMigrateModels 供外部在初始化后调用
func AutoMigrateModels(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}

func Close(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	} else {
		log.Printf("postgres close err: %v", err)
	}
}
