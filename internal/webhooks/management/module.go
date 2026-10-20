package management

import (
	"net/http"

	"github.com/bit-issues/backend/pkg/bitbucket"
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"webhooks-management",
		logger.WithNamedLogger("webhooks-management"),
		fx.Provide(NewService),
		fx.Provide(
			func(httpClient *http.Client) *bitbucket.Client {
				return bitbucket.NewClient(bitbucket.DefaultBaseURL, httpClient)
			},
			fx.Private,
		),
	)
}
