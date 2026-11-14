package feed

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the feed package.
//
// The module provides:
//   - feed.Service (public) - the task activity feed read model, for HTTP
//     handlers
func Module() fx.Option {
	return fx.Module(
		"feed",
		logger.WithNamedLogger("feed"),
		fx.Provide(NewService),
	)
}
