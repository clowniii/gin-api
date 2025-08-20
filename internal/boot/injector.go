//go:build wireinject
// +build wireinject

package boot

import (
	"github.com/google/wire"
)

func InitApp(configPath string) (*App, error) {
	wire.Build(ProviderSet)
	return nil, nil
}
