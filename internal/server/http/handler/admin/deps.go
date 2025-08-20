package admin

import (
	"go-apiadmin/internal/config"
	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/mq/kafka"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/security/jwt"
	"go-apiadmin/internal/service"
)

// Dependencies admin 子包最小依赖集合
// 仅包含 admin 相关业务与公共组件（JWT、Config、Cache、Producer、Logger 等）
type Dependencies struct {
	Auth      *service.AuthService
	User      *service.UserService
	Perm      *service.PermissionService
	Menu      *service.MenuService
	AuthGroup *service.AuthGroupService
	AuthRule  *service.AuthRuleService
	App       *service.AppService
	AppGroup  *service.AppGroupService
	IfGroup   *service.InterfaceGroupService
	IfList    *service.InterfaceListService
	Fields    *service.FieldsService
	Log       *service.LogService
	JWT       *jwt.Manager
	Config    *config.Config
	Cache     cache.Cache
	Producer  *kafka.Producer
	Logger    *logging.Logger
}
