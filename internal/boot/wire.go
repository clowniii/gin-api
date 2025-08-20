package boot

import (
	"time"

	"go-apiadmin/internal/config"
	"go-apiadmin/internal/discovery/etcd"
	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/mq/kafka"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
	redisrepo "go-apiadmin/internal/repository/redis"
	jwtsec "go-apiadmin/internal/security/jwt"
	httpSrv "go-apiadmin/internal/server/http"
	"go-apiadmin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// ProvideConfig wraps config.Load for wire with external path param
func ProvideConfig(path string) (*config.Config, error) { return config.Load(path) }

// ProvideRouter 装配路由；这里为注入后的 service 提供。
func ProvideRouter(j *jwtsec.Manager, l *logging.Logger, p *kafka.Producer, db *gorm.DB, r *redisrepo.Client, a *service.AuthService, u *service.UserService, perm *service.PermissionService, menu *service.MenuService, ag *service.AuthGroupService, ar *service.AuthRuleService, app *service.AppService, appg *service.AppGroupService, ifg *service.InterfaceGroupService, ifl *service.InterfaceListService, fields *service.FieldsService, logSvc *service.LogService, e *etcd.Client, c *config.Config, wiki *service.WikiService) *gin.Engine {
	return httpSrv.NewRouter(j, l, p, db, r, a, u, perm, menu, ag, ar, app, appg, ifg, ifl, fields, logSvc, e, c, wiki)
}

func ProvideApp(c *config.Config, l *logging.Logger, db *gorm.DB, r *redisrepo.Client, k *kafka.Producer, e *etcd.Client, j *jwtsec.Manager, engine *gin.Engine) *App {
	return NewApp(c, l, db, r, k, e, j, engine)
}

// ProvideLayeredCache 构建一个通用 LayeredCache（L1 本地 60s, L2 Redis）
func ProvideLayeredCache(r *redisrepo.Client) cache.Cache {
	l1 := cache.NewSimpleAdapter(cache.New(60 * time.Second))
	l2 := cache.NewRedisAdapter(r)
	return cache.NewLayered(l1, l2)
}

var ProviderSet = wire.NewSet(
	ProvideConfig,
	NewLogger,
	NewPostgres,
	NewRedis,
	NewKafkaProducer,
	NewEtcd,
	NewJWTManager,
	ProvideLayeredCache,
	// DAO
	dao.NewAdminUserDAO,
	dao.NewAdminAuthGroupDAO,
	dao.NewAdminAuthGroupAccessDAO,
	dao.NewAdminAuthRuleDAO,
	dao.NewAdminMenuDAO,
	dao.NewAdminAppDAO,
	dao.NewAdminAppGroupDAO,
	dao.NewAdminGroupDAO, // 新增: WikiService 需要
	dao.NewAdminInterfaceGroupDAO,
	dao.NewAdminInterfaceListDAO,
	dao.NewAdminFieldsDAO,
	dao.NewAdminUserActionDAO, // 新增
	// Service (基础)
	service.NewAuthService,
	// 使用带缓存版本
	NewPermissionServiceWithLayered,
	NewAuthGroupServiceWithLayered,
	NewAuthRuleServiceWithLayered,
	NewAppServiceWithLayered,
	NewAppGroupServiceWithLayered,
	NewInterfaceGroupServiceWithLayered,
	NewInterfaceListServiceWithLayered,
	// 需要自定义构造的 Service 使用 provider 函数
	NewMenuServiceWithLayered,
	NewUserServiceWithLayered,
	NewFieldsServiceDefault,
	NewLogServiceDefault,
	NewWikiServiceWithLayered,
	ProvideRouter,
	ProvideApp,
)

// ===== Custom providers to inject layered cache =====
func NewMenuServiceWithLayered(d *dao.AdminMenuDAO, lc cache.Cache) *service.MenuService {
	return service.NewMenuServiceWithCache(d, lc)
}
func NewUserServiceWithLayered(u *dao.AdminUserDAO, g *dao.AdminAuthGroupDAO, gr *dao.AdminAuthGroupAccessDAO, db *gorm.DB, lc cache.Cache) *service.UserService {
	return service.NewUserServiceWithCache(u, g, gr, db, lc)
}
func NewFieldsServiceDefault(d *dao.AdminFieldsDAO, ifl *dao.AdminInterfaceListDAO) *service.FieldsService {
	return service.NewFieldsService(d, ifl)
}
func NewLogServiceDefault(d *dao.AdminUserActionDAO) *service.LogService {
	return service.NewLogService(d)
}
func NewWikiServiceWithLayered(app *dao.AdminAppDAO, grp *dao.AdminGroupDAO, list *dao.AdminInterfaceListDAO, fields *dao.AdminFieldsDAO, lc cache.Cache) *service.WikiService {
	return service.NewWikiService(app, grp, list, fields, lc)
}
func NewAppServiceWithLayered(d *dao.AdminAppDAO, g *dao.AdminAppGroupDAO, c cache.Cache) *service.AppService {
	return service.NewAppServiceWithCache(d, g, c)
}
func NewAuthGroupServiceWithLayered(g *dao.AdminAuthGroupDAO, rel *dao.AdminAuthGroupAccessDAO, perm *service.PermissionService, c cache.Cache) *service.AuthGroupService {
	return service.NewAuthGroupServiceWithCache(g, rel, perm, c)
}
func NewAuthRuleServiceWithLayered(r *dao.AdminAuthRuleDAO, perm *service.PermissionService, c cache.Cache) *service.AuthRuleService {
	return service.NewAuthRuleServiceWithCache(r, perm, c)
}
func NewAppGroupServiceWithLayered(d *dao.AdminAppGroupDAO, c cache.Cache) *service.AppGroupService {
	return service.NewAppGroupServiceWithCache(d, c)
}
func NewInterfaceGroupServiceWithLayered(d *dao.AdminInterfaceGroupDAO, c cache.Cache) *service.InterfaceGroupService {
	return service.NewInterfaceGroupServiceWithCache(d, c)
}
func NewInterfaceListServiceWithLayered(d *dao.AdminInterfaceListDAO, c cache.Cache) *service.InterfaceListService {
	return service.NewInterfaceListServiceWithCache(d, c)
}
func NewPermissionServiceWithLayered(gr *dao.AdminAuthGroupAccessDAO, rule *dao.AdminAuthRuleDAO, u *dao.AdminUserDAO, r *redisrepo.Client, c cache.Cache) *service.PermissionService {
	return service.NewPermissionServiceWithCache(gr, rule, u, r, c)
}
