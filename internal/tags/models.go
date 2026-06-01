package tags

import (
	"github.com/go-core-fx/bunfx"
	"github.com/uptrace/bun"
)

// tagModel is the database representation of a tag.
type tagModel struct {
	bun.BaseModel `bun:"table:tags,alias:t"`
	bunfx.TimedModel

	Name string `bun:"name,pk"`
}
