package bitbucket

import "context"

// TokenResolver dynamically resolves the Bearer token used for a Bitbucket
// API call. Resolvers must never log or serialize raw tokens.
type TokenResolver func(ctx context.Context) (string, error)

// Config holds Bitbucket API client settings.
type Config struct {
	// AccessToken is the static Bitbucket token sent as a Bearer
	// authorization header when TokenResolver is nil.
	AccessToken string
	// CallbackURL is the public base URL used to build webhook callback URLs.
	CallbackURL string
	// BaseURL overrides the Bitbucket API base URL when set (used in tests).
	BaseURL string
	// TokenResolver resolves the Bearer token per request. When nil the
	// static AccessToken is used.
	TokenResolver TokenResolver
}
