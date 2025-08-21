# ApiAdmin (Go 重构版)

统一的 Go 重构版 **ApiAdmin** 后台与 Wiki 服务，针对原 PHP 版本做结构化升级，提供认证、权限、菜单/接口管理、接口文档（Wiki）、操作日志、缓存监控、可观测性(Tracing + Metrics)、分布式注册、消息异步化等能力。

---
## 目录
- [架构概览](#架构概览)
- [核心特性](#核心特性)
- [目录结构](#目录结构)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [依赖注入与扩展](#依赖注入与扩展)
- [缓存体系 LayeredCache](#缓存体系-layeredcache)
- [可观测性](#可观测性)
- [安全与权限模型](#安全与权限模型)
- [路由说明与兼容策略](#路由说明与兼容策略)
- [错误码体系](#错误码体系)
- [操作日志与审计](#操作日志与审计)
- [服务注册与发现](#服务注册与发现)
- [Kafka 消息与 Trace 透传](#kafka-消息与-trace-透传)
- [开发工作流建议](#开发工作流建议)
- [测试建议](#测试建议)
- [旧版本兼容与迁移注意](#旧版本兼容与迁移注意)
- [常见问题 FAQ](#常见问题-faq)
- [后续规划 (Roadmap)](#后续规划-roadmap)
- [维护指引](#维护指引)

---
## 架构概览

采用分层 + 依赖注入模式：
```
┌───────────────┐
│    Router     │ 仅负责路由分组与中间件组合
└───────┬───────┘
        │ HandlerSet 聚合
┌───────▼────────┐
│   Handlers     │ admin / wiki 子包（无业务逻辑，只编排）
└───────┬────────┘
┌───────▼────────┐
│    Service     │ 领域逻辑（纯业务、无 gin 依赖）
└───────┬────────┘
┌───────▼────────┐
│      DAO       │ 数据访问 (GORM)
└───────┬────────┘
┌───────▼────────┐
│  Infrastructure│ Redis / Kafka / Etcd / Logger / Cache
└────────────────┘
```

---
## 核心特性
- 模块化 Handler：`handler/admin` 与 `handler/wiki` 分离，聚合于 `handler/deps.go` 的 `HandlerSet`。
- 统一缓存接口：`cache.Cache` + LayeredCache (L1 本地 + L2 Redis) 自动注入各服务。
- 可观测性：Trace (OpenTelemetry)、Metrics (Prometheus)、操作日志 (Kafka) 三合一。
- 安全：JWT 登录、权限预加载与校验、Wiki 独立鉴权 Header（ApiAuth / Api-Auth）。
- 兼容：保留原有 PHP 风格路由及响应格式，新增扩展接口不破坏旧前端。
- 配置化：`config.Config` 覆盖 App 元信息、数据源、凭证、Wiki 超时时间等。
- 插件式服务装配：通过 Google Wire 生成 `wire_gen.go`，统一在 `InitApp` 入口装载。
- 错误码统一枚举：`retcode` 包；`/wiki/errorCode` 提供可视化枚举列表。
- 健康与就绪：`/healthz`（存活）、`/readyz`（依赖就绪，支持 refresh）。
- Kafka trace 透传：请求 trace_id 注入 Kafka header，消费端可继续链路追踪。

---
## 目录结构
（仅列核心层次）
```
internal/
  boot/                # Wire 装配 + App 启动封装
  config/              # 配置加载 (YAML/ENV)
  discovery/etcd/      # 服务注册 / 反注册
  logging/             # Zap Logger 封装
  mq/kafka/            # Kafka Producer
  pkg/cache/           # Cache 抽象 + SimpleAdapter/RedisAdapter/LayeredCache
  repository/
    dao/               # GORM DAO 层
    redis/             # Redis 客户端包装
  security/jwt/        # JWT 管理
  server/http/
    handler/
      admin/           # 后台业务 Handler（依赖注入的数据服务）
      wiki/            # Wiki 相关 Handler
      deps.go          # HandlerSet 聚合 (NewHandlerSet)
    middleware/
      observability/   # Trace / Metrics / OperationLog
      security/        # Auth / Permission / WikiAuth
    router.go          # 路由与组装
  service/             # 领域服务（无框架依赖）
  util/retcode/        # 错误码
```

---
## 快速开始
```bash
# 1. 拉取依赖
go mod tidy

# 2. 生成 Wire 注入代码
go generate ./...

# 3. 构建
go build -o bin/apiadmin ./...

# 4. 运行 (示例 main)
CONFIG_PATH=./config/config.yaml go run ./cmd/main.go

# 5. 访问健康检查
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```
> 端口、连接串、凭证等以配置文件为准。

---
## 配置说明
`config.Config` 典型字段（示例）：
```yaml
appMeta:
  name: ApiAdmin
  version: 1.0.0
http:
  addr: :8080
postgres:
  dsn: "postgres://user:pass@127.0.0.1:5432/apiadmin?sslmode=disable"
redis:
  addr: 127.0.0.1:6379
  db: 0
kafka:
  brokers: ["127.0.0.1:9092"]
  topicLog: admin_oplog
jwt:
  secret: "xxxx"
  ttlMinutes: 1440
etcd:
  endpoints: ["127.0.0.1:2379"]
  serviceKey: /services/apiadmin
wiki:
  onlineTimeSeconds: 86400
cache:
  # 可扩展预留: 本地 TTL / 命名空间等
```
加载：`ProvideConfig(path)` → Wire 注入。

---
## 依赖注入与扩展
- 入口函数：`boot.InitApp(configPath)`
- Wire 描述：`boot/wire.go` + 生成文件 `boot/wire_gen.go`
- 添加新 Service 步骤：
  1. 在 `service/` 编写业务逻辑（构造函数 `NewXxxService`）。
  2. 若需缓存，添加 `NewXxxServiceWithLayered` provider 放入 `ProviderSet`。
  3. 运行 `go generate ./...` 重新生成。
  4. 在 `handler/admin` 或 `handler/wiki` 新增对应 Handler 并在 `handler/deps.go` 内聚合。

---
## 缓存体系 LayeredCache
- 接口：`cache.Cache`
- 实现：
  - `SimpleAdapter` (L1，基于内存 + TTL)
  - `RedisAdapter` (L2)
  - `LayeredCache` (组合：读优先 L1 → miss 回源 L2 → miss 回源 DAO/Service 写入链)
- 特性：
  - 指标统计：命中 / 未命中 / set / delete / fallback
  - 统一重置：`ResetMetrics()`
- 监控接口：
  - `GET /admin/Cache/metrics?type=layered|perm|all`
  - `GET /admin/Cache/reset?type=layered|perm|all`
- 服务级权限缓存独立指标：`type=perm`

---
## 可观测性
| 能力 | 说明 |
|------|------|
| Trace | `TraceMiddleware` 提取/生成 trace_id，注入上下文并传播到 Kafka header |
| Metrics | Prometheus `/metrics` 暴露；HTTP 计数 / 延迟；缓存与权限缓存自定义指标接口 |
| OperationLog | 中间件 `OperationLog` 捕获请求上下文（用户ID、路径、耗时），序列化异步写入 Kafka |

健康检查：
- `/healthz`：快速返回存活（不会阻塞长检查）
- `/readyz`：并行检查 DB / Redis / Kafka / Etcd，可用 `?refresh=1` 触发重新探测缓存结果

---
## 安全与权限模型
1. 登录：`POST /admin/Login/index` → 返回 JWT（Authorization: Bearer <token>）
2. 认证中间件：`Auth(jwtManager, logger)` → 解析 token，注入 `user_id`。
3. 权限预加载：`Permission(permissionService)` → 将用户拥有的 URL 权限集合放入上下文。
4. 强制要求权限：`Require()` → 在 handler 前判定当前 URL 是否在集合内。
5. Wiki 登录：`/wiki/login` 使用 appId/appSecret（返回 ApiAuth token → Redis 存有效期在线信息）。
6. Wiki 鉴权：`NewWikiAuth(redis)` → 校验 ApiAuth 并注入 `wiki_user`。

权限数据来源：`PermissionService` 聚合用户 → 组 → 规则（URL）映射，使用缓存提升查询效率。

---
## 路由说明与兼容策略
- Admin 主前缀：`/admin`
- Wiki 主前缀：`/wiki` （内部保留 `/wiki/Api` 子前缀兼容旧客户端）
- 兼容的 AuthGroup & Auth 辅助路由合并：原部分 `Auth/*` 移至带操作日志分组。

示例（节选）：
```
/admin/Login/index            POST  登录
/admin/Login/getUserInfo      GET   用户信息
/admin/User/index             GET   用户列表
/admin/Menu/add               POST  新增菜单
/admin/Cache/metrics          GET   缓存指标
/wiki/login                   POST  Wiki 登录
/wiki/groupList               GET   分组列表
/wiki/errorCode               GET   错误码枚举
```
> 完整映射可通过生成文档或查看 `router.go`。

---
## 错误码体系
- 定义：`internal/util/retcode`（集中常量 + 结构）
- Wiki 展示接口：`GET /wiki/errorCode`
- Handler 统一：`response.Error(c, retcode.XXX, "msg")` / `response.Success(c, data)`

---
## 操作日志与审计
- 中间件：`OperationLog(producer)` 放置在需要审计的路由组（排除登录等轻量接口）。
- 记录字段：用户ID、路径、方法、耗时、状态、追踪ID等。
- 输出：Kafka 指定 topic（配置 `kafka.topicLog`）。
- 可扩展：消费端异步入 ES / ClickHouse / OLAP 做行为分析。

---
## 服务注册与发现
- Etcd Client：启动时写入 `serviceKey`（含地址 / 元数据），优雅退出时删除。
- 失败重试：可在 `boot.NewEtcd` 或注册协程中扩展指数退避策略（TODO）。

---
## Kafka 消息与 Trace 透传
- 生产：操作日志 Producer `SendWithHeaders` 在 header 中加入 `trace_id`。
- 消费：下游服务可读取 header 继续 `context` 链接 Trace。
- 优点：实现跨进程链路追踪，统一观测平台可以串联 HTTP → Kafka → 消费者。

---
## 开发工作流建议
1. 修改或新增 Service → 添加 provider → `go generate ./...`
2. 编译：`go build ./...`
3. 本地运行并用 curl / Postman 验证登录 & 权限路径。
4. 查看 `/admin/Cache/metrics` 观察缓存命中。
5. 若改动权限/菜单结构，验证 `/admin/Login/getAccessMenu` 返回。
6. 刷新接口路由文件（需要时）：`GET /admin/InterfaceList/refresh`（基于模板 `install/apiRoute.tpl`）。
7. 监控：Prometheus 抓取 `/metrics`，Grafana 展示；Trace 上报（根据 OTel Exporter 配置）。

---
## 测试建议
| 场景 | 断言 |
|------|------|
| 登录成功 | 返回 token，TTL 合理 |
| 登录失败 | 错误码 LOGIN_ERROR |
| 权限校验 | 无权限返回统一错误码（如 AUTH_ERROR / NO_PERMISSION）|
| 缓存命中 | 重复访问列表接口命中率提升 |
| 退出登录 | token 加入黑名单（若实现）|
| Wiki 登录/鉴权 | 过期后访问返回登录错误 |
| 接口刷新 | 生成的 `route/apiRoute.php` 文件内容符合模板 |
| Ready 探针 | 依赖下线时 `/readyz` 返回非 200 或 degraded 状态 |
| Kafka 发送 | 操作日志写入、包含 trace_id header |

---
## 旧版本兼容与迁移注意
- 根目录旧 handler 文件已加 build tag `oldhandlers`（仅回滚需要时启用）。建议完成迁移后直接删除。
- 路由参数与字段命名保持与原前端一致，新增字段以向后兼容方式添加。
- 若前端依赖 GET + query 的旧行为（如删除操作），已保留同名 GET 路由。

---
## 常见问题 FAQ
**Q: Wire 生成报错 / 找不到 provider?**  
A: 确认已在 `ProviderSet` 中追加，并执行 `go generate ./...`。

**Q: 缓存指标为何为空?**  
A: 首次启动尚无访问，或 LayeredCache 未命中；进行几次请求后再查看。

**Q: 权限变更不生效?**  
A: 触发对应 Service 的缓存清理（编辑/删除内部已调用），或使用 `/admin/Cache/reset?type=perm`。

**Q: Wiki 鉴权失败?**  
A: 确认使用登录返回的 `ApiAuth` header，并在在线时长内。

---
## 后续规划 (Roadmap)
- [ ] GORM / Redis / Kafka 全量 OTel instrumentation (自动 span)
- [ ] 缓存键粒度标签化 + 统一失效广播
- [ ] OpenAPI / Swagger 文档自动导出
- [ ] 接口级别 RBAC 规则热更新推送
- [ ] 更丰富的权限策略 (资源 + 动作分离)
- [ ] CLI 工具：批量生成 CRUD Handler/Service 模板
- [ ] e2e 测试脚本 (Makefile + docker-compose 依赖环境)
- [ ] 多租户隔离（命名空间级缓存 + 权限域）

---
## 维护指引
| 操作 | 文件/位置 | 说明 |
|------|-----------|------|
| 新增 Service | `internal/service` | 保持纯业务逻辑，可选注入 cache |
| 新增 Handler | `internal/server/http/handler/{admin|wiki}` | 仅调用 Service，不写业务判断逻辑 |
| 聚合 Handler | `handler/deps.go` | `HandlerSet` 添加字段与构造 |
| 新增中间件 | `middleware/{observability|security}` | 确保幂等与 panic 安全 |
| 调整路由 | `server/http/router.go` | 不在 handler 内注册路由 |
| 添加配置 | `config/` & `config.Config` | 注意向下兼容，提供默认值 |
| 缓存策略 | `pkg/cache` | 避免直接在 Service 外层绕过 cache 接口 |

---
## 许可证
根据仓库根目录 `LICENSE` 文件（如为 MIT / Apache-2.0 等）执行。

---
## 贡献
欢迎通过 PR / Issue 提交增强建议：
- 性能优化
- 监控完善
- 更多缓存策略
- DevOps 流水线 / Helm Chart 部署

---

