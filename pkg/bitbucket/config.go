package bitbucket

// Config holds Bitbucket API client settings.
type Config struct {
	// AccessToken is the Bitbucket token sent as a Bearer authorization header.
	AccessToken string
	// CallbackURL is the public base URL used to build webhook callback URLs.
	CallbackURL string
	// BaseURL overrides the Bitbucket API base URL when set (used in tests).
	BaseURL string
}
