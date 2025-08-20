package wiki

import (
	"go-apiadmin/internal/config"
	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/service"
)

// Dependencies wiki 子包最小依赖集合
type Dependencies struct {
	Wiki   *service.WikiService
	Config *config.Config
	Logger *logging.Logger
	Cache  cache.Cache
}
