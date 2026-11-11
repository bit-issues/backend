package attachments

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"attachments",
		logger.WithNamedLogger("attachments"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
