package events

import (
	"context"

	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

// Module creates and returns an FX module for the events package.
//
// The module provides:
//   - events.Hub (public) - in-memory fan-out bus for domain services and the stream handler
//   - Named logger "events" for structured logging
//
// The hub is drained on application stop.
func Module() fx.Option {
	return fx.Module(
		"events",
		logger.WithNamedLogger("events"),
		fx.Provide(NewHub),
		fx.Invoke(func(lc fx.Lifecycle, hub *Hub) {
			lc.Append(fx.StopHook(func(context.Context) error {
				hub.Close()
				return nil
			}))
		}),
	)
}
