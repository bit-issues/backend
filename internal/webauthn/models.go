package webauthn

import (
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/uptrace/bun"
)

type credentialModel struct {
	bun.BaseModel `bun:"table:webauthn_credentials,alias:wc"`

	ID              int64        `bun:"id,pk,autoincrement"`
	UserID          int64        `bun:"user_id,notnull"`
	CredentialID    CredentialID `bun:"credential_id,notnull,unique"`
	PublicKey       []byte       `bun:"public_key,notnull"`
	AttestationType string       `bun:"attestation_type,notnull"`
	Transport       Transports   `bun:"transport"`
	AAGUID          AAGUID       `bun:"aaguid,notnull"`
	Flags           uint8        `bun:"flags,notnull"`
	SignCount       uint32       `bun:"sign_count,notnull"`
	Name            string       `bun:"name,notnull"`
	CreatedAt       time.Time    `bun:"created_at,notnull"`
	UpdatedAt       time.Time    `bun:"updated_at,notnull"`
}

func (m *credentialModel) toDomain() *Credential {
	if m == nil {
		return nil
	}

	return &Credential{
		ID:              m.ID,
		UserID:          m.UserID,
		CredentialID:    m.CredentialID,
		PublicKey:       m.PublicKey,
		AttestationType: m.AttestationType,
		Transport:       append([]string(nil), m.Transport...),
		AAGUID:          m.AAGUID,
		Flags:           m.Flags,
		SignCount:       m.SignCount,
		Name:            m.Name,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func newCredentialModel(userID int64, cred *webauthn.Credential, transports []string, name string) *credentialModel {
	now := time.Now()

	return &credentialModel{
		BaseModel: bun.BaseModel{},

		ID:              0,
		UserID:          userID,
		CredentialID:    CredentialID(cred.ID),
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       Transports(transports),
		AAGUID:          AAGUID(cred.Authenticator.AAGUID),
		Flags:           byte(cred.Flags.ProtocolValue()),
		SignCount:       cred.Authenticator.SignCount,
		Name:            name,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

type Transports []string

// Value implements [driver.Valuer].
func (t *Transports) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil //nolint:nilnil //empty value
	}

	if len(*t) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(*t)
	if err != nil {
		return nil, fmt.Errorf("marshal transports: %w", err)
	}
	return string(data), nil
}

// Scan implements [sql.Scanner].
func (t *Transports) Scan(src any) error {
	if src == nil {
		return nil
	}

	var err error
	switch src := src.(type) {
	case []byte:
		err = json.Unmarshal(src, t)
	case string:
		err = json.Unmarshal([]byte(src), t)
	default:
		err = fmt.Errorf("%w: %T", ErrUnexpectedType, src)
	}

	if err != nil {
		return fmt.Errorf("scan transports: %w", err)
	}
	return nil
}

var _ sql.Scanner = (*Transports)(nil)
var _ driver.Valuer = (*Transports)(nil)

type CredentialID []byte //nolint:recvcheck // String on value

// String implements [fmt.Stringer].
func (c CredentialID) String() string {
	if c == nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(c)
}

// Value implements [driver.Valuer].
func (c CredentialID) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil //nolint:nilnil //empty value
	}

	return c.String(), nil
}

// Scan implements [sql.Scanner].
func (c *CredentialID) Scan(src any) error {
	if src == nil {
		*c = nil
		return nil
	}

	var s string
	switch src := src.(type) {
	case []byte:
		s = string(src)
	case string:
		s = src
	default:
		return fmt.Errorf("%w: %T", ErrUnexpectedType, src)
	}

	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	*c = b

	return nil
}

var _ sql.Scanner = (*CredentialID)(nil)
var _ driver.Valuer = (*CredentialID)(nil)
var _ fmt.Stringer = (CredentialID)(nil)

type AAGUID []byte //nolint:recvcheck // String on value

// String implements [fmt.Stringer].
func (a AAGUID) String() string {
	const aaguidLen = 16
	if len(a) != aaguidLen {
		return hex.EncodeToString(a)
	}
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		[]byte(a[0:4]),
		[]byte(a[4:6]),
		[]byte(a[6:8]),
		[]byte(a[8:10]),
		[]byte(a[10:16]),
	)
}

// Value implements [driver.Valuer].
func (a *AAGUID) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil //nolint:nilnil //empty value
	}

	return a.String(), nil
}

// Scan implements [sql.Scanner].
func (a *AAGUID) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}

	var s string
	switch src := src.(type) {
	case []byte:
		s = string(src)
	case string:
		s = src
	default:
		return fmt.Errorf("%w: %T", ErrUnexpectedType, src)
	}

	s = strings.ReplaceAll(s, "-", "")
	b, err := hex.DecodeString(s)
	if err != nil {
		return fmt.Errorf("failed to decode hex: %w", err)
	}

	*a = b

	return nil
}

func (a AAGUID) DeviceName() string {
	names := map[string]string{
		"00000000-0000-0000-0000-000000000000": "Unknown Device",
		"adce0002-35bc-c60a-648b-0b25f1f05503": "Chrome on Mac",
		"089b7b64-0f30-4f8c-8838-666944e5c09e": "Touch ID",
		"6028b017-b1d4-4c02-b4b3-afcd7c96e1c1": "Windows Hello",
		"dd3ec08a-88f2-4e0b-b3f2-0ab636182cf5": "iCloud Keychain",
		"fdb141b2-5d98-4b6c-8d4e-45c0a3e1c7a8": "Google Password Manager",
		"ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4": "Android Passkey",
		"50757fe4-208c-4cbb-a72b-05b48def77b7": "1Password",
	}
	if name, ok := names[a.String()]; ok {
		return name
	}
	return "Passkey"
}

var _ sql.Scanner = (*AAGUID)(nil)
var _ driver.Valuer = (*AAGUID)(nil)
var _ fmt.Stringer = (AAGUID)(nil)
