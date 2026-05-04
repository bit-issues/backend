package bitbucket

import "time"

// Issue represents a single issue from BitBucket export.
type Issue struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Reporter  User      `json:"reporter"`
	Assignee  *User     `json:"assignee"`
	Content   string    `json:"content"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	Kind      string    `json:"kind"`
	Milestone any       `json:"milestone"`
	Component any       `json:"component"`
	Version   any       `json:"version"`
	Watchers  []User    `json:"watchers"`
	Voters    []User    `json:"voters"`
}

// User represents a user in BitBucket export.
type User struct {
	DisplayName string `json:"display_name"`
	AccountID   string `json:"account_id"`
}

// Comment represents a comment from BitBucket export.
type Comment struct {
	ID        int        `json:"id"`
	Issue     int        `json:"issue"`
	User      User       `json:"user"`
	Content   string     `json:"content"`
	CreatedOn time.Time  `json:"created_on"`
	UpdatedOn *time.Time `json:"updated_on"`
}

// Export represents the full JSON export structure.
type Export struct {
	Meta        map[string]any `json:"meta"`
	Issues      []Issue        `json:"issues"`
	Attachments []any          `json:"attachments"`
	Comments    []Comment      `json:"comments"`
	Logs        []any          `json:"logs"`
}
