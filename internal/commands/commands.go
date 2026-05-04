package commands

import (
	"github.com/bit-issues/backend/internal/commands/importer"
	"github.com/bit-issues/backend/internal/commands/serve"
	"github.com/go-core-fx/healthfx"
	"github.com/urfave/cli/v3"
)

func Commands(version healthfx.Version) []*cli.Command {
	return []*cli.Command{
		serve.Command(version),
		importer.Command(version),
	}
}
