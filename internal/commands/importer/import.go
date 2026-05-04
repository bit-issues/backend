package importer

import (
	"context"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/config"
	"github.com/bit-issues/backend/internal/db"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/storage"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/go-core-fx/bunfx"
	"github.com/go-core-fx/fxutil"
	"github.com/go-core-fx/healthfx"
	"github.com/go-core-fx/logger"
	"github.com/go-core-fx/sqlfx"
	"github.com/urfave/cli/v3"
	"go.uber.org/fx"
)

func Command(_ healthfx.Version) *cli.Command {
	return &cli.Command{
		Name:        "import",
		Usage:       "Import issues from a JSON file",
		Description: `Import issues from a BitBucket export JSON file into a specified project`,
		Flags:       (*Config)(nil).Flags(),
		Action:      run,
	}
}

// ImportResult holds the results of an import operation.
type ImportResult struct {
	IssuesImported   int
	IssuesSkipped    int
	CommentsImported int
	CommentsSkipped  int
}

// run imports issues from a BitBucket export JSON file.
func run(ctx context.Context, cmd *cli.Command) error {
	// Run the import within an FX app
	app := fx.New(
		logger.Module(),
		logger.WithFxDefaultLogger(),
		bunfx.Module(),
		sqlfx.Module(),

		config.Module(),
		db.Module(),
		storage.Module(),

		users.Module(),
		projects.Module(),
		tasks.Module(),
		attachments.Module(),
		comments.Module(),

		fx.Supply(parseConfig(cmd)),

		fx.Provide(newImporter),
		fx.Invoke(fxutil.RegisterRunnable[*importer]()),
	)

	startCtx, cancelStart := context.WithTimeout(ctx, app.StartTimeout())
	defer cancelStart()

	if startErr := app.Start(startCtx); startErr != nil {
		return fmt.Errorf("failed to start app: %w", startErr)
	}

	var runErr error
	select {
	case <-ctx.Done():
		runErr = ctx.Err()
	case sig := <-app.Wait():
		if sig.ExitCode != 0 {
			runErr = cli.Exit("app exited with non-zero status code", sig.ExitCode)
		}
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), app.StopTimeout())
	defer cancelStop()

	if stopErr := app.Stop(stopCtx); stopErr != nil {
		return fmt.Errorf("failed to stop app: %w", stopErr)
	}

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		return fmt.Errorf("import failed: %w", runErr)
	}

	return nil
}
