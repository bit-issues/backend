package oauth

import "time"

// Token is the domain representation of the stored Bitbucket OAuth
// credential. Access and refresh tokens are stored plaintext for the MVP.
type Token struct {
	AccessToken  string
	RefreshToken string
	Scopes       string
	ExpiresAt    time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
