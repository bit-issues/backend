package webhooks

import (
	"net/http"

	"github.com/bit-issues/backend/pkg/bitbucket"
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"webhooks",
		logger.WithNamedLogger("webhooks"),
		fx.Provide(NewService),
		fx.Provide(
			func(httpClient *http.Client) *bitbucket.Client {
				return bitbucket.NewClient(bitbucket.DefaultBaseURL, httpClient)
			},
			fx.Private,
		),
		fx.Provide(NewManagementService),
		fx.Invoke(func(lc fx.Lifecycle, s *Service) {
			lc.Append(fx.StartHook(s.Init))
		}),
	)
}
