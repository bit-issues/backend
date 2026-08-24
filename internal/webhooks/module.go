package webhooks

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"webhooks",
		logger.WithNamedLogger("webhooks"),
		fx.Provide(NewService),
		fx.Provide(NewManagementService),
		fx.Invoke(func(lc fx.Lifecycle, s *Service) {
			lc.Append(fx.StartHook(s.Init))
		}),
	)
}
