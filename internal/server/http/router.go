package http

import (
	"context"
	"time"

	"go-apiadmin/internal/config"
	"go-apiadmin/internal/discovery/etcd"
	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/mq/kafka"
	redisrepo "go-apiadmin/internal/repository/redis"
	"go-apiadmin/internal/security/jwt"
	handlerset "go-apiadmin/internal/server/http/handler"
	adm "go-apiadmin/internal/server/http/handler/admin"
	wikih "go-apiadmin/internal/server/http/handler/wiki"
	"go-apiadmin/internal/server/http/middleware" // keep for ResponseWrapper, CORS
	obs "go-apiadmin/internal/server/http/middleware/observability"
	sec "go-apiadmin/internal/server/http/middleware/security"
	"go-apiadmin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

// NewRouter 仅负责分组与中间件装配，具体业务放在 handler 层
func NewRouter(jwtm *jwt.Manager, logger *logging.Logger, producer *kafka.Producer, db *gorm.DB, redis *redisrepo.Client, authSvc *service.AuthService, userSvc *service.UserService, permSvc *service.PermissionService, menuSvc *service.MenuService, authGroupSvc *service.AuthGroupService, authRuleSvc *service.AuthRuleService, appSvc *service.AppService, appGroupSvc *service.AppGroupService, ifgSvc *service.InterfaceGroupService, iflSvc *service.InterfaceListService, fieldsSvc *service.FieldsService, logSvc *service.LogService, etcdCli *etcd.Client, cfg *config.Config, wikiSvc *service.WikiService) *gin.Engine {
	r := gin.New()
	// 新增 ConfigInjector 放最前确保后续中间件可读取 app_config
	r.Use(middleware.ConfigInjector(cfg), gin.Recovery(), middleware.CORS(), obs.TraceMiddleware(), obs.LoggerContextMiddleware(logger), middleware.ResponseWrapper(), obs.Metrics())

	// 健康检查
	hc := NewHealthChecker(db, redis, producer, etcdCli)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, hc.Liveness()) })
	r.GET("/readyz", func(c *gin.Context) {
		if c.Query("refresh") == "1" {
			hc.cacheMu.Lock()
			hc.cacheExpiry = time.Time{}
			hc.cacheMu.Unlock()
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
		defer cancel()
		res, code := hc.Readiness(ctx)
		c.JSON(code, res)
	})
	// Prometheus
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 依赖注入给 handler 构造器 (拆分 admin / wiki 子包依赖)
	ad := adm.Dependencies{
		Auth: authSvc, User: userSvc, Perm: permSvc, Menu: menuSvc, AuthGroup: authGroupSvc, AuthRule: authRuleSvc,
		App: appSvc, AppGroup: appGroupSvc, IfGroup: ifgSvc, IfList: iflSvc, Fields: fieldsSvc, Log: logSvc,
		JWT: jwtm, Logger: logger, Producer: producer, Config: cfg, Cache: menuSvc.Cache,
	}
	wd := wikih.Dependencies{Wiki: wikiSvc, Config: cfg, Logger: logger, Cache: menuSvc.Cache}
	h := handlerset.NewHandlerSet(ad, wd)

	// 登录/公共接口
	v1 := r.Group("/admin") // 沿用原路径结构 (轻量公共 + 认证接口分组)，不含 OperationLog
	{
		v1.POST("/Login/index", h.Auth.Login)
		// 新增刷新令牌接口：POST /admin/Login/refresh
		v1.POST("/Login/refresh", h.Auth.Refresh)
		v1.GET("/Login/getUserInfo", sec.Auth(jwtm, logger), sec.Permission(permSvc), sec.Require(), h.Auth.GetUserInfo)
		v1.GET("/Login/getAccessMenu", sec.Auth(jwtm, logger), sec.Permission(permSvc), h.Auth.GetAccessMenu)
		v1.POST("/Login/logout", h.Auth.Logout)
		// 兼容新增：GET /admin/Login/logout (原 PHP 为 GET 且需要认证+日志，无权限校验)
		v1.GET("/Login/logout", sec.Auth(jwtm, logger), obs.OperationLog(producer), h.Auth.Logout)
		// NOTE: 以下四个 Auth 相关接口已迁入 admin 组以补齐操作日志 (2025-08 重构)
		// v1.GET("/Auth/delMember", ...)
		// v1.GET("/Auth/getGroups", ...)
		// v1.GET("/Auth/getRuleList", ...)
		// v1.POST("/Auth/editRule", ...)
	}

	// 需认证+操作日志+权限预加载 (admin 主业务分组)
	adminGrp := r.Group("/admin", sec.Auth(jwtm, logger), obs.OperationLog(producer), sec.Permission(permSvc))
	{
		// 用户
		userGroup := adminGrp.Group("/User")
		{
			userGroup.GET("/index", sec.Require(), h.User.List)
			userGroup.GET("/getUsers", sec.Require(), h.User.List)
			userGroup.POST("/add", sec.Require(), h.User.Add)
			userGroup.POST("/edit", sec.Require(), h.User.Edit)
			userGroup.GET("/changeStatus", sec.Require(), h.User.ChangeStatus)
			userGroup.GET("/del", sec.Require(), h.User.Delete)
			userGroup.POST("/own", sec.Require(), h.User.Own)
		}
		// 菜单
		menuGroup := adminGrp.Group("/Menu")
		{
			menuGroup.GET("/index", sec.Require(), h.Menu.Index)
			menuGroup.GET("/changeStatus", sec.Require(), h.Menu.ChangeStatus)
			menuGroup.POST("/add", sec.Require(), h.Menu.Add)
			menuGroup.POST("/edit", sec.Require(), h.Menu.Edit)
			menuGroup.GET("/del", sec.Require(), h.Menu.Delete)
		}
		// 权限组
		agGroup := adminGrp.Group("/AuthGroup")
		{
			agGroup.GET("/index", sec.Require(), h.AuthGroup.Index)
			agGroup.POST("/add", sec.Require(), h.AuthGroup.Add)
			agGroup.POST("/edit", sec.Require(), h.AuthGroup.Edit)
			agGroup.GET("/changeStatus", sec.Require(), h.AuthGroup.ChangeStatus)
			agGroup.GET("/del", sec.Require(), h.AuthGroup.Delete)
		}
		// 兼容路由 /admin/Auth/* 映射到 AuthGroupHandler + (扩展) AuthHandler 成员/规则接口
		compatAuth := adminGrp.Group("/Auth")
		{
			compatAuth.GET("/index", sec.Require(), h.AuthGroup.Index)
			compatAuth.POST("/add", sec.Require(), h.AuthGroup.Add)
			compatAuth.POST("/edit", sec.Require(), h.AuthGroup.Edit)
			compatAuth.GET("/changeStatus", sec.Require(), h.AuthGroup.ChangeStatus)
			compatAuth.GET("/del", sec.Require(), h.AuthGroup.Delete)
			// 新增迁入: 以下四个此前位于 v1 组，现统一在 admin 组以具备 OperationLog
			compatAuth.GET("/delMember", sec.Require(), h.Auth.DelMember)
			compatAuth.GET("/getGroups", sec.Require(), h.Auth.GetGroups)
			compatAuth.GET("/getRuleList", sec.Require(), h.Auth.GetRuleList)
			compatAuth.POST("/editRule", sec.Require(), h.Auth.EditRule)
		}
		// 权限规则
		arGroup := adminGrp.Group("/AuthRule")
		{
			arGroup.GET("/index", sec.Require(), h.AuthRule.Index)
			arGroup.POST("/add", sec.Require(), h.AuthRule.Add)
			arGroup.POST("/edit", sec.Require(), h.AuthRule.Edit)
			arGroup.GET("/changeStatus", sec.Require(), h.AuthRule.ChangeStatus)
			arGroup.GET("/del", sec.Require(), h.AuthRule.Delete)
		}
		// App
		appGroup := adminGrp.Group("/App")
		{
			appGroup.GET("/index", sec.Require(), h.App.Index)
			appGroup.GET("/changeStatus", sec.Require(), h.App.ChangeStatus)
			appGroup.GET("/getAppInfo", sec.Require(), h.App.GetInfo)
			appGroup.POST("/add", sec.Require(), h.App.Add)
			appGroup.POST("/edit", sec.Require(), h.App.Edit)
			appGroup.GET("/del", sec.Require(), h.App.Delete)
			appGroup.GET("/refreshAppSecret", sec.Require(), h.App.RefreshSecret)
		}
		// AppGroup
		appgGroup := adminGrp.Group("/AppGroup")
		{
			appgGroup.GET("/index", sec.Require(), h.AppGroup.Index)
			appgGroup.GET("/getAll", sec.Require(), h.AppGroup.Index)
			appgGroup.POST("/add", sec.Require(), h.AppGroup.Add)
			appgGroup.POST("/edit", sec.Require(), h.AppGroup.Edit)
			appgGroup.GET("/changeStatus", sec.Require(), h.AppGroup.ChangeStatus)
			appgGroup.GET("/del", sec.Require(), h.AppGroup.Delete)
		}
		// InterfaceGroup
		ifgGroup := adminGrp.Group("/InterfaceGroup")
		{
			ifgGroup.GET("/index", sec.Require(), h.InterfaceGroup.Index)
			ifgGroup.GET("/getAll", sec.Require(), h.InterfaceGroup.GetAll)
			ifgGroup.POST("/add", sec.Require(), h.InterfaceGroup.Add)
			ifgGroup.POST("/edit", sec.Require(), h.InterfaceGroup.Edit)
			ifgGroup.GET("/changeStatus", sec.Require(), h.InterfaceGroup.ChangeStatus)
			ifgGroup.GET("/del", sec.Require(), h.InterfaceGroup.Delete)
		}
		// InterfaceList
		iflGroup := adminGrp.Group("/InterfaceList")
		{
			iflGroup.GET("/index", sec.Require(), h.InterfaceList.Index)
			iflGroup.GET("/getHash", sec.Require(), h.InterfaceList.GetHash)
			iflGroup.GET("/refresh", sec.Require(), h.InterfaceList.Refresh)
			iflGroup.POST("/add", sec.Require(), h.InterfaceList.Add)
			iflGroup.POST("/edit", sec.Require(), h.InterfaceList.Edit)
			iflGroup.GET("/changeStatus", sec.Require(), h.InterfaceList.ChangeStatus)
			iflGroup.GET("/del", sec.Require(), h.InterfaceList.Delete)
		}
		// Fields
		fieldsGroup := adminGrp.Group("/Fields")
		{
			fieldsGroup.GET("/index", sec.Require(), h.Fields.Index)
			fieldsGroup.GET("/request", sec.Require(), h.Fields.Request)
			fieldsGroup.GET("/response", sec.Require(), h.Fields.Response)
			fieldsGroup.POST("/add", sec.Require(), h.Fields.Add)
			fieldsGroup.POST("/edit", sec.Require(), h.Fields.Edit)
			fieldsGroup.GET("/del", sec.Require(), h.Fields.Delete)
			fieldsGroup.POST("/upload", sec.Require(), h.Fields.Upload)
		}
		// Log
		logGroup := adminGrp.Group("/Log")
		{
			logGroup.GET("/index", sec.Require(), h.Log.List)
			logGroup.GET("/del", sec.Require(), h.Log.Delete)
		}
		// Cache metrics
		cacheGroup := adminGrp.Group("/Cache")
		{
			cacheGroup.GET("/metrics", h.Cache.Metrics)
			cacheGroup.GET("/reset", h.Cache.Reset)
		}
		// Index (upload)
		idxGroup := adminGrp.Group("/Index")
		{
			idxGroup.POST("/upload", sec.Require(), h.Index.Upload)
		}
	}

	// Wiki (兼容两套前缀)
	wikiGrp := r.Group("/wiki")
	{
		wikiGrp.GET("/errorCode", h.Wiki.ErrorCode)
		wikiGrp.POST("/login", h.Wiki.Login)
		wikiGrp.GET("/groupList", sec.NewWikiAuth(redis), h.Wiki.GroupList)
		wikiGrp.GET("/detail", sec.NewWikiAuth(redis), h.Wiki.Detail)
		wikiGrp.POST("/logout", sec.NewWikiAuth(redis), h.Wiki.Logout)
		// 新增接口
		wikiGrp.GET("/search", sec.NewWikiAuth(redis), h.Wiki.Search)
		wikiGrp.GET("/groupHot", sec.NewWikiAuth(redis), h.Wiki.GroupHot)
		wikiGrp.GET("/fields", sec.NewWikiAuth(redis), h.Wiki.Fields)
		wikiGrp.GET("/appInfo", sec.NewWikiAuth(redis), h.Wiki.AppInfo)
		wikiGrp.GET("/dataType", h.Wiki.DataType)

		api := wikiGrp.Group("/Api")
		{
			api.GET("/errorCode", h.Wiki.ErrorCode)
			api.POST("/login", h.Wiki.Login)
			api.GET("/groupList", sec.NewWikiAuth(redis), h.Wiki.GroupList)
			api.GET("/detail", sec.NewWikiAuth(redis), h.Wiki.Detail)
			api.POST("/logout", sec.NewWikiAuth(redis), h.Wiki.Logout)
			// 新增接口（兼容路径）
			api.GET("/search", sec.NewWikiAuth(redis), h.Wiki.Search)
			api.GET("/groupHot", sec.NewWikiAuth(redis), h.Wiki.GroupHot)
			api.GET("/fields", sec.NewWikiAuth(redis), h.Wiki.Fields)
			api.GET("/appInfo", sec.NewWikiAuth(redis), h.Wiki.AppInfo)
			api.GET("/dataType", h.Wiki.DataType)
		}
	}
	// 统一 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(200, gin.H{"code": -8, "msg": "不存在", "data": gin.H{}})
	})
	return r
}
