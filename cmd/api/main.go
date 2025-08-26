package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go-apiadmin/internal/boot"

	"go.uber.org/zap"
)

func main() {
	// 支持通过环境变量 CONFIG_PATH 指定配置文件，默认使用 dev；不存在则回退 example
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.dev.yaml"
	}
	if _, err := os.Stat(cfgPath); err != nil {
		fallback := "configs/config.example.yaml"
		if _, err2 := os.Stat(fallback); err2 == nil {
			log.Printf("config %s not found, fallback to %s", cfgPath, fallback)
			cfgPath = fallback
		} else {
			log.Fatalf("config file not found: %s (fallback %s also missing)", cfgPath, fallback)
		}
	}
	// 归一化路径，方便日志可读
	if abs, err := filepath.Abs(cfgPath); err == nil {
		cfgPath = abs
	}

	app, err := boot.InitApp(cfgPath)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	srv := &http.Server{Addr: app.Config.HTTP.Addr, Handler: app.HTTP}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		app.Logger.Info("http_server_start", zap.String("addr", app.Config.HTTP.Addr), zap.String("config", cfgPath))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Error("http_server_error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	app.Logger.Info("shutting_down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	app.Close()
	app.Logger.Info("cleanup_done")
}
