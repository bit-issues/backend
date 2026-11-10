package history

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the history package.
//
// The module provides:
//   - history.Service (public) - for use by HTTP handlers and other modules
//   - history.Repository (public) - consumed by the use-case layer, which
//     records entries inside its own cross-module transactions
//   - Named logger "history" for structured logging
func Module() fx.Option {
	return fx.Module(
		"history",
		logger.WithNamedLogger("history"),
		fx.Provide(NewRepository),
		fx.Provide(NewService),
	)
}
