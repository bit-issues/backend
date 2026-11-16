package history

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the history package.
//
// The module provides:
//   - history.Service (public) - reads for HTTP handlers; record methods for
//     writing modules, which pass their own transaction into them
//   - history.Repository (private) - internal data access layer
//   - Named logger "history" for structured logging
func Module() fx.Option {
	return fx.Module(
		"history",
		logger.WithNamedLogger("history"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
