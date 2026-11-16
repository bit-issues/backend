package feed

import "errors"

// ErrWindowTooLarge is returned when a feed query's offset+limit exceeds the
// bounded window; the query is rejected before any history read.
var ErrWindowTooLarge = errors.New("offset is too large")
