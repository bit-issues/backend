package storage

import (
	"fmt"
	"net/url"

	"github.com/bit-issues/backend/pkg/miniofx"
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"storage",
		logger.WithNamedLogger("storage"),
		fx.Provide(func(c Config) (miniofx.Config, error) {
			u, err := url.Parse(c.URL)
			if err != nil {
				return miniofx.Config{}, fmt.Errorf("failed to parse storage URL: %w", err)
			}

			return miniofx.Config{
				Endpoint: u.Query().Get("endpoint"),
				Region:   u.Query().Get("region"),
			}, nil
		}),
		fx.Provide(NewService),
	)
}
