package jwt

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the jwt package.
//
// The module provides:
//   - JWTService for token generation, validation, and refresh token management
func Module() fx.Option {
	return fx.Module(
		"jwt",
		logger.WithNamedLogger("jwt"),
		fx.Provide(NewRepository, fx.Private),
		fx.Provide(NewService),
	)
}
