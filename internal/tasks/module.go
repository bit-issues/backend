package tasks

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the tasks package.
//
// The module provides:
//   - tasks.Service (public) - for use by HTTP handlers and other modules
//   - tasks.Repository (private) - internal data access layer
//   - Named logger "tasks" for structured logging
func Module() fx.Option {
	return fx.Module(
		"tasks",
		logger.WithNamedLogger("tasks"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
