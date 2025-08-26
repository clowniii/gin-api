package boot

import (
	"context"
	"encoding/json"
	"fmt"
	"go-apiadmin/internal/config"
	"go-apiadmin/internal/discovery/etcd"
	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/mq/kafka"
	"go-apiadmin/internal/repository/postgres"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	"net"
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

	"github.com/google/uuid"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"gorm.io/plugin/opentelemetry/tracing"

	"go-apiadmin/internal/metrics"
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

	AsyncAccessSender *kafka.AccessAsyncSender

	serviceKey string
	leaseID    clientv3.LeaseID
	serviceVal string // 新增: 缓存首次注册的原始metadata value，供重注册恢复
	tracerProv *trace.TracerProvider
	stopCh     chan struct{} // 新增: 心跳协程关闭
}

// Provider constructors for wire
func NewPostgres(c *config.Config) (*gorm.DB, error) {
	return postgres.New(postgres.Config{DSN: c.Postgres.DSN, MaxOpen: c.Postgres.MaxOpen, MaxIdle: c.Postgres.MaxIdle, AutoMigrate: c.Postgres.AutoMigrate})
}

func NewRedis(c *config.Config) *redisrepo.Client {
	return redisrepo.New(redisrepo.Config{Addr: c.Redis.Addr, Password: c.Redis.Password, DB: c.Redis.DB,
		DialTimeout:  time.Duration(c.Redis.DialTimeoutMS) * time.Millisecond,
		ReadTimeout:  time.Duration(c.Redis.ReadTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(c.Redis.WriteTimeoutMS) * time.Millisecond,
		PingTimeout:  time.Duration(c.Redis.PingTimeoutMS) * time.Millisecond,
	})
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
			&model.AdminGroup{}, // 新增：admin_group 表
			&model.AdminInterfaceGroup{},
			&model.AdminInterfaceList{},
			&model.AdminField{},
			&model.AdminUserData{},
		); err != nil {
			l.Error("auto_migrate_failed", zap.Error(err))
		}
	}
	app := &App{Config: c, Logger: l, DB: db, Redis: r, Kafka: k, Etcd: e, JWT: j, HTTP: engine, stopCh: make(chan struct{})}
	// Redis 启动健康检查（避免登录慢才暴露问题）
	if r != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Redis.PingTimeoutMS)*time.Millisecond)
		defer cancel()
		if err := r.Ping(ctx); err != nil {
			l.Error("redis_ping_failed", zap.Error(err), zap.String("addr", c.Redis.Addr))
		} else {
			l.Info("redis_ping_ok", zap.String("addr", c.Redis.Addr))
		}
		// 启动 Redis 心跳
		go func() {
			interval := time.Duration(c.Redis.HeartbeatSec) * time.Second
			if interval < 2*time.Second { // 下限保护
				interval = 2 * time.Second
			}
			var lastUp bool
			for {
				select {
				case <-app.stopCh:
					return
				case <-time.After(interval):
					ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(c.Redis.PingTimeoutMS)*time.Millisecond)
					err := r.Ping(ctx2)
					cancel2()
					if err != nil {
						metrics.RedisUp.Set(0)
						if lastUp { // 状态切换
							l.Warn("redis_down", zap.Error(err))
						}
						lastUp = false
					} else {
						metrics.RedisUp.Set(1)
						if !lastUp {
							l.Info("redis_recovered")
						}
						lastUp = true
					}
				}
			}
		}()
	}
	if e != nil && len(c.Etcd.Endpoints) > 0 {
		go func() {
			ctx := context.Background()
			// 生成 instance_id
			instanceID := uuid.New().String()
			// 解析监听地址端口，尝试获取本机首个非 loopback IPv4
			addrPort := c.HTTP.Addr
			if addrPort == "" {
				addrPort = ":8080"
			}
			// 分离端口
			port := ""
			if addrPort[0] == ':' { // 形如 :8080
				port = addrPort[1:]
			} else {
				if host, p, err := net.SplitHostPort(addrPort); err == nil {
					port = p
					_ = host // host 不强制使用
				}
			}
			if port == "" {
				port = "0"
			}
			ip := firstNonLoopbackIPv4()
			if ip == "" {
				ip = "127.0.0.1"
			}
			// 将 key 的最后一段改为 ip:port，避免每次重启生成新的 instance_id key，便于稳定发现
			serviceKey := fmt.Sprintf("/services/apiadmin/%s/%s/%s:%s", c.AppMeta.Env, c.AppMeta.Version, ip, port)
			meta := map[string]interface{}{
				"instance_id":  instanceID, // 仍然保留 instance_id 作为内部唯一标识
				"env":          c.AppMeta.Env,
				"version":      c.AppMeta.Version,
				"ip":           ip,
				"port":         port,
				"addr":         c.HTTP.Addr,
				"startup_unix": time.Now().Unix(),
			}
			valBytes, _ := json.Marshal(meta)
			val := string(valBytes)
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
				app.serviceVal = val // 缓存原始 value
				app.leaseID = leaseID
				metrics.EtcdUp.Set(1) // 注册成功标记UP
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
		metrics.EtcdUp.Set(0) // 下线标记
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
	// 异步访问日志发送器关闭（若存在）
	if a.AsyncAccessSender != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = a.AsyncAccessSender.Close(ctx)
	}
	// 关闭心跳
	if a.stopCh != nil {
		close(a.stopCh)
	}
}

// 获取首个非 loopback IPv4
func firstNonLoopbackIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}
