package oauth

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates the Fx module for the OAuth domain.
//
// The module provides:
//   - oauth.Repository (private) - internal data access layer
//   - oauth.BitbucketClient (public) - Bitbucket OAuth exchange/refresh client
//   - oauth.TokenRefresher (public) - wired to BitbucketClient.Refresh
//   - oauth.Service (public) - token storage, refresh, and CSRF states
//
// oauth.Config is provided by the config module from environment settings.
// No encryption key exists in the MVP: tokens are stored plaintext.
func Module() fx.Option {
	return fx.Module(
		"oauth",
		logger.WithNamedLogger("oauth"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(
			NewBitbucketClient,
			func(client *BitbucketClient) TokenRefresher {
				return client.Refresh
			},
			NewService,
		),
	)
}
