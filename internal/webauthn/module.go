package webauthn

import (
	"github.com/go-core-fx/cachefx"
	"github.com/go-core-fx/cachefx/cache"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"webauthn",
		fx.Provide(
			NewRepository,
		),
		fx.Provide(
			func(factory cachefx.Factory) (cache.Cache, error) {
				return factory.New("webauthn")
			},
			fx.Private,
		),
		fx.Provide(
			NewService,
		),
	)
}
