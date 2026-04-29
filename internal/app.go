package internal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bit-issues/backend/internal/commands"
	"github.com/go-core-fx/healthfx"
	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
)

func Run(version healthfx.Version) {
	app := &cli.Command{
		Name:           "backend",
		Usage:          "BitIssues Backend",
		Description:    `BitIssues Backend`,
		Version:        version.Version,
		DefaultCommand: "serve",
		Flags:          []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name:        "serve",
				Usage:       "Start the HTTP server",
				Description: `Start the HTTP server`,
				Action: func(ctx context.Context, _ *cli.Command) error {
					if err := commands.Serve(ctx, version); err != nil {
						return fmt.Errorf("serve: %w", err)
					}
					return nil
				},
			},
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	if err := app.Run(ctx, os.Args); err != nil {
		exitCode := 1
		if exitErr, ok := lo.ErrorsAs[cli.ExitCoder](err); ok {
			exitCode = exitErr.ExitCode()
		}
		stop()
		os.Exit(exitCode)
	}
	stop()
}
