package projects

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the projects package.
// This module wires up all dependencies for the projects functionality:
//   - Repository (private): Data access layer, only used within this module
//   - Service (public): Business logic layer, can be injected into other modules
//
// The module also registers a named logger for structured logging.
func Module() fx.Option {
	return fx.Module(
		"projects",
		// Add a named logger for this module
		logger.WithNamedLogger("projects"),

		// Provide the repository as a private dependency
		// This means it can only be used within this module
		fx.Provide(NewRepository, fx.Private),

		// Provide the service as a public dependency
		// This means it can be injected into other modules (e.g., tasks module)
		fx.Provide(NewService),
	)
}
