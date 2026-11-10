package attachments

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"attachments",
		logger.WithNamedLogger("attachments"),
		// The repository is exported: the use-case layer runs attachment
		// mutations inside its own cross-module transactions.
		fx.Provide(NewRepository),
		fx.Provide(NewService),
	)
}
