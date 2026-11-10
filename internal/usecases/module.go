package usecases

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the usecases package.
//
// The module provides:
//   - usecases.Service (public) - atomic cross-module flows for HTTP handlers,
//     webhooks, and the importer
//
// The service joins the transactions of the tasks, history, and attachments
// repositories, so those repositories are provided by their modules without
// fx.Private.
func Module() fx.Option {
	return fx.Module(
		"usecases",
		logger.WithNamedLogger("usecases"),
		fx.Provide(NewService),
	)
}
