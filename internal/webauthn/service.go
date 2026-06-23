package webauthn

import (
	"context"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/users"
	"github.com/go-core-fx/cachefx/cache"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"go.uber.org/zap"
)

type Service struct {
	webAuthn    *webauthn.WebAuthn
	credentials *Repository
	sessions    *sessions
	usersSvc    *users.Service
	logger      *zap.Logger
}

func NewService(
	cfg Config,
	credentials *Repository,
	usersSvc *users.Service,
	backend cache.Cache,
	logger *zap.Logger,
) (*Service, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn instance: %w", err)
	}

	return &Service{
		webAuthn:    w,
		credentials: credentials,
		sessions:    newSessions(backend),
		usersSvc:    usersSvc,
		logger:      logger,
	}, nil
}

func (s *Service) BeginRegistration(ctx context.Context, user *users.User) (*protocol.CredentialCreation, error) {
	creds, err := s.credentials.GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user credentials: %w", err)
	}

	wuser := newWebAuthnUser(user, creds)

	options, sessionData, err := s.webAuthn.BeginRegistration(
		wuser,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.AuthenticatorAttachment(""),
			RequireResidentKey:      protocol.ResidentKeyRequired(),
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			UserVerification:        protocol.VerificationPreferred,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	if storeErr := s.sessions.Store(ctx, sessionData); storeErr != nil {
		return nil, fmt.Errorf("failed to store registration session: %w", storeErr)
	}

	return options, nil
}

func (s *Service) FinishRegistration(ctx context.Context, user *users.User, body []byte) (*Credential, error) {
	parsedResponse, err := protocol.ParseCredentialCreationResponseBytes(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registration response: %w", errors.Join(ErrInvalidWebAuthnPayload, err))
	}

	challenge := parsedResponse.Response.CollectedClientData.Challenge

	sessionData, err := s.sessions.Consume(ctx, challenge)
	if err != nil {
		return nil, err
	}

	creds, err := s.credentials.GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user credentials: %w", err)
	}

	wuser := newWebAuthnUser(user, creds)

	credential, err := s.webAuthn.CreateCredential(wuser, *sessionData, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	transports := parsedResponse.Raw.AttestationResponse.Transports
	if transports == nil {
		transports = []string{}
	}

	cred := newCredentialModel(
		user.ID,
		credential,
		transports,
		AAGUID(credential.Authenticator.AAGUID).DeviceName(),
	)

	if createErr := s.credentials.Create(ctx, cred); createErr != nil {
		return nil, fmt.Errorf("failed to store credential: %w", createErr)
	}

	return cred.toDomain(), nil
}

func (s *Service) BeginLogin(ctx context.Context) (*protocol.CredentialAssertion, error) {
	options, sessionData, err := s.webAuthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationPreferred),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	if storeErr := s.sessions.Store(ctx, sessionData); storeErr != nil {
		return nil, fmt.Errorf("failed to store login session: %w", storeErr)
	}

	return options, nil
}

func (s *Service) FinishLogin(ctx context.Context, body []byte) (*users.User, error) {
	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", errors.Join(ErrInvalidWebAuthnPayload, err))
	}

	challenge := parsedResponse.Response.CollectedClientData.Challenge

	sessionData, err := s.sessions.Consume(ctx, challenge)
	if err != nil {
		return nil, err
	}

	var credModel *Credential
	handler := func(rawID, _ []byte) (webauthn.User, error) {
		cred, getErr := s.credentials.GetByCredentialID(ctx, rawID)
		if getErr != nil {
			return nil, fmt.Errorf("credential not found: %w", getErr)
		}

		user, getErr := s.usersSvc.GetByID(ctx, cred.UserID)
		if getErr != nil {
			return nil, fmt.Errorf("failed to get user: %w", getErr)
		}

		credModel = cred

		allCreds, getErr := s.credentials.GetByUserID(ctx, user.ID)
		if getErr != nil {
			return nil, fmt.Errorf("failed to get user credentials: %w", getErr)
		}

		return newWebAuthnUser(user, allCreds), nil
	}

	_, credential, err := s.webAuthn.ValidatePasskeyLogin(handler, *sessionData, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to validate login: %w", err)
	}

	if updErr := s.credentials.UpdateSignCount(ctx, credModel.ID, credential.Authenticator.SignCount); updErr != nil {
		s.logger.Warn("failed to update sign count", zap.Error(updErr))
		return nil, fmt.Errorf("failed to update sign count: %w", updErr)
	}

	user, err := s.usersSvc.GetByID(ctx, credModel.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *Service) GetCredentials(ctx context.Context, userID int64) ([]Credential, error) {
	return s.credentials.GetByUserID(ctx, userID)
}

func (s *Service) RenameCredential(ctx context.Context, id int64, userID int64, name string) error {
	return s.credentials.UpdateName(ctx, id, userID, name)
}

func (s *Service) DeleteCredential(ctx context.Context, id int64, userID int64) error {
	return s.credentials.Delete(ctx, id, userID)
}

func idToBytes(id int64) []byte {
	return fmt.Appendf(nil, "%d", id)
}
