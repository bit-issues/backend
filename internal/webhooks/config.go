package webhooks

type KeywordEntry struct {
	Status string `json:"status"`
	Verb   string `json:"verb"`
}

type Config struct {
	Secret         string
	BotUserEmail   string
	ActionKeywords map[string]KeywordEntry

	// CallbackURL is the public base URL (https scheme://host[/path], no
	// query string or fragment) where Bitbucket webhooks are delivered. The
	// full delivery path is appended by the management service.
	CallbackURL string
}
