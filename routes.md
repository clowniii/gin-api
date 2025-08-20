# 路由对照 (PHP vs Go)

本文件用于对照原 PHP `route/app.php` 与 Go `router.go` 中的管理端路由（仅 admin 部分，wiki 单独保持兼容）。

列说明：
- Legacy(PHP): 原路径+方法。
- Go 新路径: 当前 Go 版本实际注册的路径+方法。
- Handler: Go 对应处理器。
- Status: DONE(已实现), COMPAT(新增兼容别名), TODO(未实现), DIFF(行为或校验差异)。

## 登录 & 用户信息
| Legacy (PHP) | Go | Handler | Status | 备注 |
|-------------|----|---------|--------|------|
| POST /admin/Login/index | POST /admin/Login/index | AuthHandler.Login | DONE | 登录 |
| GET /admin/Login/logout | GET /admin/Login/logout | AuthHandler.Logout | COMPAT | PHP 为 GET，Go 早期仅 POST，现已补 GET |
| GET /admin/Login/getUserInfo | GET /admin/Login/getUserInfo | AuthHandler.GetUserInfo | DONE | |
| GET /admin/Login/getAccessMenu | GET /admin/Login/getAccessMenu | AuthHandler.GetAccessMenu | DONE | |

## 权限组 (AuthGroup) & 兼容 /admin/Auth/*
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/Auth/index | GET /admin/Auth/index | AuthGroupHandler.Index | COMPAT | 原路由别名，主路径 /AuthGroup/index |
| POST /admin/Auth/add | POST /admin/Auth/add | AuthGroupHandler.Add | COMPAT | 同上 |
| POST /admin/Auth/edit | POST /admin/Auth/edit | AuthGroupHandler.Edit | COMPAT | 同上 |
| GET /admin/Auth/changeStatus | GET /admin/Auth/changeStatus | AuthGroupHandler.ChangeStatus | COMPAT | 同上 |
| GET /admin/Auth/del | GET /admin/Auth/del | AuthGroupHandler.Delete | COMPAT | 同上 |
| GET /admin/Auth/delMember | GET /admin/Auth/delMember | AuthHandler.DelMember | DONE | 移除用户与组关系 |
| GET /admin/Auth/getGroups | GET /admin/Auth/getGroups | AuthHandler.GetGroups | DONE | 仅 status=1 |
| GET /admin/Auth/getRuleList | GET /admin/Auth/getRuleList | AuthHandler.GetRuleList | DONE | 菜单树+checked 标记 |
| POST /admin/Auth/editRule | POST /admin/Auth/editRule | AuthHandler.EditRule | DONE | 批量更新规则 |

## 菜单 (Menu)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/Menu/index | GET /admin/Menu/index | MenuHandler.Index | DONE | |
| GET /admin/Menu/changeStatus | GET /admin/Menu/changeStatus | MenuHandler.ChangeStatus | DONE | |
| POST /admin/Menu/add | POST /admin/Menu/add | MenuHandler.Add | DONE | |
| POST /admin/Menu/edit | POST /admin/Menu/edit | MenuHandler.Edit | DONE | |
| GET /admin/Menu/del | GET /admin/Menu/del | MenuHandler.Delete | DONE | |

## 用户 (User)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/User/index | GET /admin/User/index | UserHandler.List | DONE | 别名 getUsers 也兼容 |
| GET /admin/User/getUsers | GET /admin/User/getUsers | UserHandler.List | DONE | |
| GET /admin/User/changeStatus | GET /admin/User/changeStatus | UserHandler.ChangeStatus | DONE | |
| POST /admin/User/add | POST /admin/User/add | UserHandler.Add | DONE | |
| POST /admin/User/edit | POST /admin/User/edit | UserHandler.Edit | DONE | |
| GET /admin/User/del | GET /admin/User/del | UserHandler.Delete | DONE | |
| POST /admin/User/own | POST /admin/User/own | UserHandler.Own | DONE | 设置归属 |

## 应用 (App)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/App/index | GET /admin/App/index | AppHandler.Index | DONE | |
| GET /admin/App/changeStatus | GET /admin/App/changeStatus | AppHandler.ChangeStatus | DONE | |
| GET /admin/App/getAppInfo | GET /admin/App/getAppInfo | AppHandler.GetInfo | DONE | |
| POST /admin/App/add | POST /admin/App/add | AppHandler.Add | DONE | |
| POST /admin/App/edit | POST /admin/App/edit | AppHandler.Edit | DONE | |
| GET /admin/App/del | GET /admin/App/del | AppHandler.Delete | DONE | |
| GET /admin/App/refreshAppSecret | GET /admin/App/refreshAppSecret | AppHandler.RefreshSecret | DONE | |

## 应用分组 (AppGroup)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/AppGroup/index | GET /admin/AppGroup/index | AppGroupHandler.Index | DONE | getAll 复用 index |
| GET /admin/AppGroup/getAll | GET /admin/AppGroup/getAll | AppGroupHandler.Index | DONE | |
| POST /admin/AppGroup/add | POST /admin/AppGroup/add | AppGroupHandler.Add | DONE | |
| POST /admin/AppGroup/edit | POST /admin/AppGroup/edit | AppGroupHandler.Edit | DONE | |
| GET /admin/AppGroup/changeStatus | GET /admin/AppGroup/changeStatus | AppGroupHandler.ChangeStatus | DONE | |
| GET /admin/AppGroup/del | GET /admin/AppGroup/del | AppGroupHandler.Delete | DONE | |

## 接口分组 (InterfaceGroup)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/InterfaceGroup/index | GET /admin/InterfaceGroup/index | InterfaceGroupHandler.Index | DONE | |
| GET /admin/InterfaceGroup/getAll | GET /admin/InterfaceGroup/getAll | InterfaceGroupHandler.GetAll | DONE | |
| POST /admin/InterfaceGroup/add | POST /admin/InterfaceGroup/add | InterfaceGroupHandler.Add | DONE | |
| POST /admin/InterfaceGroup/edit | POST /admin/InterfaceGroup/edit | InterfaceGroupHandler.Edit | DONE | |
| GET /admin/InterfaceGroup/changeStatus | GET /admin/InterfaceGroup/changeStatus | InterfaceGroupHandler.ChangeStatus | DONE | |
| GET /admin/InterfaceGroup/del | GET /admin/InterfaceGroup/del | InterfaceGroupHandler.Delete | DONE | |

## 接口列表 (InterfaceList)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/InterfaceList/index | GET /admin/InterfaceList/index | InterfaceListHandler.Index | DONE | |
| GET /admin/InterfaceList/changeStatus | GET /admin/InterfaceList/changeStatus | InterfaceListHandler.ChangeStatus | DONE | |
| GET /admin/InterfaceList/getHash | GET /admin/InterfaceList/getHash | InterfaceListHandler.GetHash | DONE | |
| GET /admin/InterfaceList/refresh | GET /admin/InterfaceList/refresh | InterfaceListHandler.Refresh | DONE | |
| POST /admin/InterfaceList/add | POST /admin/InterfaceList/add | InterfaceListHandler.Add | DONE | |
| POST /admin/InterfaceList/edit | POST /admin/InterfaceList/edit | InterfaceListHandler.Edit | DONE | |
| GET /admin/InterfaceList/del | GET /admin/InterfaceList/del | InterfaceListHandler.Delete | DONE | |

## 字段 (Fields)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/Fields/index | GET /admin/Fields/index | FieldsHandler.Index | DONE | |
| GET /admin/Fields/request | GET /admin/Fields/request | FieldsHandler.Request | DONE | |
| GET /admin/Fields/response | GET /admin/Fields/response | FieldsHandler.Response | DONE | |
| POST /admin/Fields/add | POST /admin/Fields/add | FieldsHandler.Add | DONE | |
| POST /admin/Fields/edit | POST /admin/Fields/edit | FieldsHandler.Edit | DONE | |
| GET /admin/Fields/del | GET /admin/Fields/del | FieldsHandler.Delete | DONE | |
| POST /admin/Fields/upload | POST /admin/Fields/upload | FieldsHandler.Upload | DONE | 结构上传 |

## 日志 (Log)
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| GET /admin/Log/index | GET /admin/Log/index | LogHandler.List | DONE | |
| GET /admin/Log/del | GET /admin/Log/del | LogHandler.Delete | DONE | |

## 其它
| Legacy | Go | Handler | Status | 备注 |
|--------|----|---------|--------|------|
| POST /admin/Index/upload | POST /admin/Index/upload | IndexHandler.Upload | DONE | 新增兼容文件上传 |
| (N/A) | GET /admin/Cache/metrics | CacheHandler.Metrics | NEW | 缓存指标 |
| (N/A) | GET /admin/Cache/reset | CacheHandler.Reset | NEW | 重置指标 |

## Wiki / 文档
保持 /wiki 与 /wiki/Api 双前缀，已在 Go 中补充新增接口：search, groupHot, fields, appInfo, dataType。

## TODO / 差异汇总
目前已完成列出的全部管理端路由兼容；若后续发现遗漏可在此处追加。

新增差异（2025-08 调整）：
- /admin/Auth/delMember, /admin/Auth/getGroups, /admin/Auth/getRuleList, /admin/Auth/editRule 四条原先在轻量 v1 组（无操作日志）现统一迁入 admin 分组并继承 OperationLog 中间件，行为增加了操作日志记录；权限校验保持不变。
- /admin/Login/getAccessMenu 在 PHP 中仅做认证+响应包装，Go 侧保留 Permission 预加载校验用于构建访问菜单的权限过滤，属安全增强；未按旧行为移除。
- 其余差异：新增 /admin/Cache/metrics, /admin/Cache/reset, /wiki/* 扩展接口均为功能增强，不影响原有路由兼容。

## 错误码统一说明
为保持与旧版 PHP ReturnCode 兼容，所有业务返回使用 `code` 字段，HTTP 状态固定 200。近期重构统一了 handler 中的错误码：

- 解析/绑定失败: 统一使用 `JSON_PARSE_FAIL` (-9) 而非 400。
- 参数缺失: 使用 `EMPTY_PARAMS` (-12)。
- 数据新增/更新/删除失败: `DB_SAVE_ERROR` (-2)。
- 数据读取失败或查询异常: `DB_READ_ERROR` (-3)。
- 文件/路由刷新写入失败: `FILE_SAVE_ERROR` (-6)。
- 登录失败: `LOGIN_ERROR` (-7)。
- 权限/认证失败: `AUTH_ERROR` (-14)。
- 访问令牌过期: `ACCESS_TOKEN_TIMEOUT` (-996)。
- 未知内部错误（框架/依赖空指针等兜底）建议使用 `UNKNOWN` (-998) 或 `EXCEPTION` (-999)；当前 middleware.permission 中缺依赖使用 `UNKNOWN`。

规范约定：
1. 不再直接传入 400/500/http.StatusBadRequest 等 HTTP 码到 `response.Error`；若误传 >=0 将被自动转为 `INVALID` (-1)。
2. 所有错误响应结构：`{"code": <负值>, "msg": <字符串>, "data": null}`。
3. 成功统一 `SUCCESS` (1) 且 `msg` 为 `success`，`data` 为实际载荷。
4. 前端判断逻辑：`code === 1` 表示成功；其余全部视为错误并根据 `msg` 提示。
5. WikiAuth 中间件已切换为统一 `response.Error`，移除直接 `c.JSON`。

验证脚本建议（可选）：通过 grep/CI 检查 `response.Error(c, 4`、`response.Error(c, 5`、`response.Error(c, http.Status`、`response.Error(c, -[0-9]+)` 非 retcode 常量的遗留。当前代码库已无此类直接数字调用。

后续若新增业务码，请在 `internal/util/retcode/retcode.go` 中追加并在此处同步说明。

## 新增：服务注册与优雅下线 (2025-08)
- 启动时通过 etcd.Register 写入 key: `/services/apiadmin/{version}/{addr}`，并保持租约 KeepAlive。
- Register 现在返回 leaseID，`App` 保存 `serviceKey` 与 `leaseID`。
- 关闭时 `App.Close()` 调用 `etcd.Deregister`：主动 Delete key + Revoke 租约，避免等待 TTL 过期。
- 若注册失败仅记录日志，不影响主进程启动。
- 监控/排障：下线后 key 立即消失，可用于探测实例真实存活。

## 新增：Kafka 消费端链路追踪 (2025-08)
- 新增 `internal/mq/kafka/consumer.go`，`Consumer.Start` 在读取消息后解析 `trace_id` header 写入 context。
- 业务 handler 可使用全局 logger.WithContext(ctx) 输出统一 trace_id，实现生产->消费链路串联。
- 若消息无 trace_id，可在上层 handler 中补生成，以便后续 OpenTelemetry 接入时统一。

## 新增：OpenTelemetry 集成 (2025-08)
- 配置新增 `otel` 节点：`enable`, `endpoint`, `insecure`, `sampler_ratio`。
- 启动时若启用：初始化 OTLP gRPC 导出，注册 TracerProvider（批处理 + 采样）。
- HTTP TraceMiddleware 现在桥接 OTel：生成/提取自定义 trace_id，同时创建 span，写入自定义属性 `custom.trace_id`。
- 未来可扩展：GORM、Redis hook 及 Kafka producer/consumer 注入 W3C 上下文，当前保留 trace_id header 兼容前端。

