package events

import "time"

// DefaultHeartbeatInterval is used when the configured interval is missing or
// non-positive; the heartbeat doubles as the dead-connection detector for
// event streams.
const DefaultHeartbeatInterval = 15 * time.Second

// Config holds settings for the events stream endpoint.
type Config struct {
	HeartbeatInterval time.Duration `koanf:"heartbeat_interval"`
}
