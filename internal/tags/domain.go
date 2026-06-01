package tags

import "time"

// Tag represents a label that can be attached to entities for categorization.
type Tag struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TagInput contains the data required to create a tag.
type TagInput struct {
	Name string
}
