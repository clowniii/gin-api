package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go-apiadmin/internal/boot"
)

func main() {
	app, err := boot.InitApp("configs/config.example.yaml")
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	srv := &http.Server{Addr: app.Config.HTTP.Addr, Handler: app.HTTP}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		app.Logger.Info("http server start", zapField("addr", app.Config.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Error("server error", zapField("err", err.Error()))
		}
	}()

	<-ctx.Done()
	app.Logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	app.Close()
	app.Logger.Info("cleanup done")
}

// simple wrapper keep
func zapField(k string, v interface{}) loggingField { return loggingField{k, v} }

type loggingField struct {
	k string
	v interface{}
}

func (l loggingField) Key() string        { return l.k }
func (l loggingField) Value() interface{} { return l.v }
