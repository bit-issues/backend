package miniofx

import (
	"github.com/go-core-fx/logger"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"minio",
		logger.WithNamedLogger("minio"),
		fx.Provide(NewClient),
	)
}
