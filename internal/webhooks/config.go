package webhooks

import "github.com/bit-issues/backend/internal/commitref"

type Config struct {
	Secret         string
	BotUserEmail   string
	ActionKeywords map[string]commitref.KeywordEntry
}
