package boot

import (
	"context"
	"fmt"
	"go-apiadmin/internal/config"
	"go-apiadmin/internal/discovery/etcd"
	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/mq/kafka"
	"go-apiadmin/internal/repository/postgres"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"

	clientv3 "go.etcd.io/etcd/client/v3"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	go_otel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Dependencies struct {
	Config *config.Config
	Logger *logging.Logger
	DB     *gorm.DB
	Redis  *redisrepo.Client
	Kafka  *kafka.Producer
	Etcd   *etcd.Client
	JWT    *jwt.Manager
}

type App struct {
	Config *config.Config
	Logger *logging.Logger
	DB     *gorm.DB
	Redis  *redisrepo.Client
	Kafka  *kafka.Producer
	Etcd   *etcd.Client
	JWT    *jwt.Manager
	HTTP   *gin.Engine

	serviceKey string
	leaseID    clientv3.LeaseID
	tracerProv *trace.TracerProvider
}

// Provider constructors for wire
func NewPostgres(c *config.Config) (*gorm.DB, error) {
	return postgres.New(postgres.Config{DSN: c.Postgres.DSN, MaxOpen: c.Postgres.MaxOpen, MaxIdle: c.Postgres.MaxIdle, AutoMigrate: c.Postgres.AutoMigrate})
}

func NewRedis(c *config.Config) *redisrepo.Client {
	return redisrepo.New(redisrepo.Config{Addr: c.Redis.Addr, Password: c.Redis.Password, DB: c.Redis.DB})
}

func NewKafkaProducer(c *config.Config) *kafka.Producer {
	return kafka.NewProducer(kafka.Config{Brokers: c.Kafka.Brokers, Topic: c.Kafka.OpLogTopic})
}

func NewEtcd(c *config.Config) (*etcd.Client, error) {
	return etcd.New(etcd.Config{Endpoints: c.Etcd.Endpoints, TTL: c.Etcd.TTL})
}

func NewJWTManager(c *config.Config) *jwt.Manager {
	return jwt.NewManager(c.JWT.Secret, c.JWT.ExpireSeconds, c.JWT.Issuer)
}

func NewLogger(c *config.Config) (*logging.Logger, error) {
	return logging.New(c.Log.Level, c.Log.Format)
}

func NewApp(c *config.Config, l *logging.Logger, db *gorm.DB, r *redisrepo.Client, k *kafka.Producer, e *etcd.Client, j *jwt.Manager, engine *gin.Engine) *App {
	// 自动迁移（只在配置开启时）
	if c.Postgres.AutoMigrate {
		_ = postgres.AutoMigrateModels(db,
			&model.AdminUser{},
			&model.AdminAuthGroup{},
			&model.AdminAuthRule{},
			&model.AdminAuthGroupAccess{},
			&model.AdminUserAction{},
		)
	}
	app := &App{Config: c, Logger: l, DB: db, Redis: r, Kafka: k, Etcd: e, JWT: j, HTTP: engine}
	if e != nil && len(c.Etcd.Endpoints) > 0 {
		go func() {
			ctx := context.Background()
			serviceKey := fmt.Sprintf("/services/apiadmin/%s/%s", c.AppMeta.Version, c.HTTP.Addr)
			val := fmt.Sprintf("{\"addr\":\"%s\",\"version\":\"%s\"}", c.HTTP.Addr, c.AppMeta.Version)
			leaseID, err := e.Register(ctx, serviceKey, val, int64(c.Etcd.TTL))
			if err != nil {
				l.Error("etcd register failed", zap.Error(err))
				return
			}
			app.serviceKey = serviceKey
			app.leaseID = leaseID
		}()
	}
	// OpenTelemetry 初始化（可选）
	if c.OTel.Enable {
		ctx, cancel := context.WithTimeout(context.Background(), 5e9) // 5s
		defer cancel()
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(c.OTel.Endpoint)}
		if c.OTel.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		} else {
			opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			l.Error("otel exporter init failed", zap.Error(err))
		} else {
			res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(c.AppMeta.Name),
				semconv.ServiceVersionKey.String(c.AppMeta.Version),
			))
			var sampler trace.Sampler = trace.ParentBased(trace.TraceIDRatioBased(c.OTel.SamplerRatio))
			app.tracerProv = trace.NewTracerProvider(trace.WithBatcher(exp), trace.WithResource(res), trace.WithSampler(sampler))
			go_otel.SetTracerProvider(app.tracerProv)
		}
	}
	return app
}

func (a *App) Close() {
	// 优雅下线 etcd
	if a.Etcd != nil && a.serviceKey != "" && a.leaseID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*1e9)
		defer cancel()
		_ = a.Etcd.Deregister(ctx, a.serviceKey, a.leaseID)
	}
	if a.DB != nil {
		if sqlDB, err := a.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	if a.Kafka != nil {
		_ = a.Kafka.Close()
	}
	if a.Etcd != nil {
		_ = a.Etcd.Close()
	}
	if a.tracerProv != nil {
		_ = a.tracerProv.Shutdown(context.Background())
	}
}
