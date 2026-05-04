package importer

import "github.com/urfave/cli/v3"

type Config struct {
	Filename string

	ProjectSlug string
	DefaultUser string
	DryRun      bool
}

func (c *Config) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "project",
			Usage:    "Target project slug (required)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "file",
			Usage:    "Path to the JSON file (required)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "default-user",
			Usage:    "Default user ID or slug for unmapped authors (required)",
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Simulate import without writing to database",
			Value: false,
		},
	}
}

func parseConfig(cmd *cli.Command) Config {
	return Config{
		Filename:    cmd.String("file"),
		ProjectSlug: cmd.String("project"),
		DefaultUser: cmd.String("default-user"),
		DryRun:      cmd.Bool("dry-run"),
	}
}
