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
	"time"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	go_otel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"gorm.io/plugin/opentelemetry/tracing"
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
	// 自动迁移（只在配置开启时）: 补充更多模型
	if c.Postgres.AutoMigrate {
		if err := postgres.AutoMigrateModels(db,
			&model.AdminUser{},
			&model.AdminAuthGroup{},
			&model.AdminAuthRule{},
			&model.AdminAuthGroupAccess{},
			&model.AdminUserAction{},
			&model.AdminMenu{},
			&model.AdminApp{},
			&model.AdminAppGroup{},
			&model.AdminInterfaceGroup{},
			&model.AdminInterfaceList{},
			&model.AdminField{},
		); err != nil {
			l.Error("auto_migrate_failed", zap.Error(err))
		}
	}
	app := &App{Config: c, Logger: l, DB: db, Redis: r, Kafka: k, Etcd: e, JWT: j, HTTP: engine}
	if e != nil && len(c.Etcd.Endpoints) > 0 {
		go func() {
			ctx := context.Background()
			serviceKey := fmt.Sprintf("/services/apiadmin/%s/%s", c.AppMeta.Version, c.HTTP.Addr)
			val := fmt.Sprintf("{\"addr\":\"%s\",\"version\":\"%s\"}", c.HTTP.Addr, c.AppMeta.Version)
			// 指数退避重试注册
			var (
				attempt     = 0
				maxAttempts = 5
			)
			for {
				leaseID, err := e.Register(ctx, serviceKey, val, int64(c.Etcd.TTL))
				if err != nil {
					attempt++
					if attempt >= maxAttempts {
						l.Error("etcd_register_failed", zap.Error(err), zap.Int("attempt", attempt))
						return
					}
					backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
					l.Error("etcd_register_retry", zap.Error(err), zap.Int("attempt", attempt), zap.Duration("backoff", backoff))
					time.Sleep(backoff)
					continue
				}
				app.serviceKey = serviceKey
				app.leaseID = leaseID
				l.Info("etcd_registered", zap.String("key", serviceKey))
				return
			}
		}()
	}
	// OpenTelemetry 初始化（可选）
	if c.OTel.Enable {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(c.OTel.Endpoint)}
		if c.OTel.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		} else {
			opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			l.Error("otel_exporter_init_failed", zap.Error(err))
		} else {
			res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(c.AppMeta.Name),
				semconv.ServiceVersionKey.String(c.AppMeta.Version),
			))
			var sampler trace.Sampler = trace.ParentBased(trace.TraceIDRatioBased(c.OTel.SamplerRatio))
			app.tracerProv = trace.NewTracerProvider(trace.WithBatcher(exp), trace.WithResource(res), trace.WithSampler(sampler))
			go_otel.SetTracerProvider(app.tracerProv)
			l.Info("otel_tracer_provider_initialized")
			// ===== GORM instrumentation =====
			if db != nil {
				if err := db.Use(tracing.NewPlugin()); err != nil {
					l.Error("gorm_tracing_plugin_failed", zap.Error(err))
				} else {
					l.Info("gorm_tracing_plugin_enabled")
				}
			}
			// ===== Redis instrumentation =====
			if r != nil {
				// 为 go-redis 注册 tracing hook
				if err := redisotel.InstrumentTracing(r.Client); err != nil {
					l.Error("redis_tracing_hook_failed", zap.Error(err))
				} else {
					l.Info("redis_otel_tracing_enabled")
				}
			}
			// Kafka 生产者无需额外初始化，此处仅记录
			l.Info("kafka_producer_tracing_enabled")
		}
	}
	return app
}

func (a *App) Close() {
	// 优雅下线 etcd
	if a.Etcd != nil && a.serviceKey != "" && a.leaseID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.Etcd.Deregister(ctx, a.serviceKey, a.leaseID); err != nil {
			a.Logger.Error("etcd_deregister_failed", zap.Error(err))
		}
	}
	if a.DB != nil {
		if sqlDB, err := a.DB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				a.Logger.Error("db_close_error", zap.Error(err))
			}
		}
	}
	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			a.Logger.Error("redis_close_error", zap.Error(err))
		}
	}
	if a.Kafka != nil {
		if err := a.Kafka.Close(); err != nil {
			a.Logger.Error("kafka_close_error", zap.Error(err))
		}
	}
	if a.Etcd != nil {
		if err := a.Etcd.Close(); err != nil {
			a.Logger.Error("etcd_close_error", zap.Error(err))
		}
	}
	if a.tracerProv != nil {
		if err := a.tracerProv.Shutdown(context.Background()); err != nil {
			a.Logger.Error("otel_tracer_shutdown_error", zap.Error(err))
		}
	}
}
