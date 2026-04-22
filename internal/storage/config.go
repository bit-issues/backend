package storage

import "time"

type Config struct {
	URL      string
	LinksTTL time.Duration
}
