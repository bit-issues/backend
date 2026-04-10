package jwt

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the jwt package.
//
// The module provides:
//   - JWTService for token generation and validation
func Module() fx.Option {
	return fx.Module(
		"jwt",
		logger.WithNamedLogger("jwt"),
		fx.Provide(NewService),
	)
}
