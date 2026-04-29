package commands

import (
	"context"
	"fmt"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/config"
	"github.com/bit-issues/backend/internal/db"
	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server"
	"github.com/bit-issues/backend/internal/storage"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/pkg/miniofx"
	"github.com/go-core-fx/bunfx"
	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/goosefx"
	"github.com/go-core-fx/healthfx"
	"github.com/go-core-fx/logger"
	"github.com/go-core-fx/sqlfx"
	"github.com/go-core-fx/validatorfx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Serve(ctx context.Context, version healthfx.Version) error {
	app := fx.New(
		// CORE MODULES
		logger.Module(),
		logger.WithFxDefaultLogger(),
		// badgerfx.Module(),
		bunfx.Module(),
		// cachefx.Module(),
		fiberfx.Module(),
		// gocqlfx.Module(),
		// gocqlxfx.Module(),
		sqlfx.Module(),
		goosefx.Module(),
		// gormfx.Module(),
		healthfx.Module(),
		// openrouterfx.Module(),
		// redisfx.Module(),
		// sqlxfx.Module(),
		// telegofx.Module(true),
		validatorfx.Module(),
		// watermillfx.Module(),
		miniofx.Module(),
		//
		// APP MODULES
		config.Module(),
		db.Module(),
		server.Module(),
		storage.Module(),
		//
		// BUSINESS MODULES
		fx.Supply(version),
		jwt.Module(),
		users.Module(),
		projects.Module(),
		tasks.Module(),
		attachments.Module(),
		comments.Module(),
		//
		fx.Invoke(func(lc fx.Lifecycle, logger *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					logger.Info("app started")
					return nil
				},
				OnStop: func(_ context.Context) error {
					logger.Info("app stopped")
					return nil
				},
			})
		}),
	)

	startCtx, cancelStart := context.WithTimeout(ctx, app.StartTimeout())
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	select {
	case <-ctx.Done():
	case <-app.Done():
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), app.StopTimeout())
	defer cancelStop()

	if err := app.Stop(stopCtx); err != nil {
		return fmt.Errorf("failed to stop app: %w", err)
	}

	return nil
}
