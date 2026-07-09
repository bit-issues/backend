package webhooks

type KeywordEntry struct {
	Status string `json:"status"`
	Verb   string `json:"verb"`
}

type Config struct {
	Secret         string
	BotUserEmail   string
	ActionKeywords map[string]KeywordEntry
}
