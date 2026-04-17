package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, input UserInput) (*User, error) {
	input.Email = strings.ToLower(input.Email)

	passwordHash, err := hashPasswordArgon2id(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.repo.Create(ctx, input, passwordHash)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*User, error) {
	email = strings.ToLower(email)

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCredential
		}

		return nil, err
	}

	if verifyErr := verifyPasswordArgon2id(password, user.PasswordHash); verifyErr != nil {
		return nil, fmt.Errorf("failed to verify password: %w", verifyErr)
	}

	if user.Status != StatusActive {
		return nil, ErrNotActive
	}

	return &user.User, nil
}

func (s *Service) List(ctx context.Context, status *Status, role *Role, pagination *Pagination) ([]User, int, error) {
	return s.repo.List(ctx, status, role, pagination)
}

func (s *Service) Update(ctx context.Context, id int64, update UserUpdate) error {
	return s.repo.Update(ctx, id, update)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &user.User, nil
}

func (s *Service) IsAdmin(ctx context.Context, id int64) (bool, error) {
	isAdmin, err := s.repo.IsAdminByID(ctx, id)
	if err != nil {
		return false, err
	}

	return isAdmin, nil
}

func (s *Service) ChangePassword(ctx context.Context, id int64, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if verifyErr := verifyPasswordArgon2id(oldPassword, user.PasswordHash); verifyErr != nil {
		return fmt.Errorf("failed to verify password: %w", verifyErr)
	}

	hash, err := hashPasswordArgon2id(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.repo.UpdatePasswordHash(ctx, id, hash)
}
