package commitref

import "go.uber.org/fx"

// Module creates and returns an FX module for the commitref package.
//
// The module provides:
//   - commitref.Service (public) - for use by other modules that scan commit messages
func Module() fx.Option {
	return fx.Module(
		"commitref",
		fx.Provide(NewService),
	)
}
