package oauth

import (
	"github.com/go-core-fx/cachefx"
	"github.com/go-core-fx/cachefx/cache"
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"oauth",
		logger.WithNamedLogger("oauth"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(
			func(factory cachefx.Factory) (cache.Cache, error) {
				return factory.New("oauth")
			},
			fx.Private,
		),
		fx.Provide(NewService),
	)
}
