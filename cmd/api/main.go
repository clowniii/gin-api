package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go-apiadmin/internal/boot"

	"go.uber.org/zap"
)

func main() {
	app, err := boot.InitApp("../../configs/config.dev.yaml")
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	srv := &http.Server{Addr: app.Config.HTTP.Addr, Handler: app.HTTP}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		app.Logger.Info("http_server_start", zap.String("addr", app.Config.HTTP.Addr))
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
