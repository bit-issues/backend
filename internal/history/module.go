package history

import (
	"github.com/go-core-fx/fxutil"
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the history package.
//
// The module provides:
//   - history.Service (public) - reads for HTTP handlers and the event consumer
//   - history.Consumer (private) - records activity rows from published operations
//   - history.Repository (private) - internal data access layer
//   - Named logger "history" for structured logging
func Module() fx.Option {
	return fx.Module(
		"history",
		logger.WithNamedLogger("history"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewConsumer, fx.Private),
		fx.Provide(NewService),
		fx.Invoke(fxutil.RegisterRunnable[*Consumer]()),
	)
}
