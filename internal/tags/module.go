package tags

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the tags package.
func Module() fx.Option {
	return fx.Module(
		"tags",
		logger.WithNamedLogger("tags"),

		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
