package history

import (
	"context"

	"github.com/uptrace/bun"
)

// AuditFunc records the history event for a repository operation inside the
// same transaction as that operation. The repository invokes it with the row
// the operation produced, so event and trigger commit or roll back together.
type AuditFunc[T any] func(ctx context.Context, tx bun.IDB, entity T) error
