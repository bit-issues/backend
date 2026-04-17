package comments

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the comments package.
//
// The module provides:
//   - comments.Service (public) - for use by HTTP handlers and other modules
//   - comments.Repository (private) - internal data access layer
//   - Named logger "comments" for structured logging
func Module() fx.Option {
	return fx.Module(
		"comments",
		logger.WithNamedLogger("comments"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
