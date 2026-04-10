package users

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"users",
		logger.WithNamedLogger("users"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
