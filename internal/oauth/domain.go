package oauth

import (
	"fmt"
	"time"
)

// Token is the domain representation of the stored Bitbucket OAuth credential.
// The AccessToken and RefreshToken here are always the decrypted plaintext;
// they are encrypted at rest by the repository before persistence.
type Token struct {
	AccessToken  string
	RefreshToken string
	Scopes       string
	ExpiresAt    time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type EncryptedToken struct {
	Token

	Fingerprint string
}

func NewEncryptedToken(enc *Encryptor, token Token) (*EncryptedToken, error) {
	fp := fingerprint(token.RefreshToken)

	var err error
	if enc != nil {
		token.AccessToken, err = enc.Encrypt(token.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt access token: %w", err)
		}
		token.RefreshToken, err = enc.Encrypt(token.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
	}

	return &EncryptedToken{
		Token:       token,
		Fingerprint: fp,
	}, nil
}

func (f *EncryptedToken) toToken(enc *Encryptor) (*Token, error) {
	if f == nil {
		return nil, nil //nolint:nilnil //empty value
	}

	if enc == nil {
		return &f.Token, nil
	}

	var err error
	token := f.Token
	token.AccessToken, err = enc.Decrypt(f.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}
	token.RefreshToken, err = enc.Decrypt(f.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	return &token, nil
}
