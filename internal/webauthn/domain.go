package webauthn

import (
	"time"

	"github.com/bit-issues/backend/internal/users"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type Credential struct {
	ID              int64
	UserID          int64
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transport       []string
	AAGUID          []byte
	Flags           uint8
	SignCount       uint32
	Name            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type webauthnUser struct {
	user        *users.User
	credentials []webauthn.Credential
}

func newWebAuthnUser(user *users.User, creds []Credential) *webauthnUser {
	wc := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		transports := make([]protocol.AuthenticatorTransport, len(c.Transport))
		for j, t := range c.Transport {
			transports[j] = protocol.AuthenticatorTransport(t)
		}

		credID := c.CredentialID

		wc[i] = webauthn.Credential{
			ID:              credID,
			Flags:           webauthn.NewCredentialFlags(protocol.AuthenticatorFlags(c.Flags)),
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:       c.AAGUID,
				SignCount:    c.SignCount,
				CloneWarning: false,
				Attachment:   "",
			},
		}
	}

	return &webauthnUser{
		user:        user,
		credentials: wc,
	}
}

func (u *webauthnUser) WebAuthnID() []byte {
	return idToBytes(u.user.ID)
}

func (u *webauthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.user.Name
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webauthnUser) WebAuthnIcon() string {
	return ""
}
