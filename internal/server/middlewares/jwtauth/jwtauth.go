package jwtauth

import (
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/users"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

type localsKey string

const (
	userKey localsKey = "user"
)

// New returns a middleware that validates JWT token and sets user info in context.
func New(jwtSvc *jwt.Service, usersSvc *users.Service) fiber.Handler {
	return keyauth.New(keyauth.Config{
		Validator: func(c *fiber.Ctx, token string) (bool, error) {
			claims, err := jwtSvc.ValidateToken(token)
			if err != nil {
				return false, fmt.Errorf("failed to validate token: %w", err)
			}

			user, err := usersSvc.GetByID(c.Context(), claims.UserID)
			if err != nil {
				return false, fiber.NewError(fiber.StatusUnauthorized, "user not found")
			}

			if user.Status != users.StatusActive {
				return false, fiber.NewError(fiber.StatusForbidden, "user is not active")
			}

			// Set user info in context
			c.Locals(userKey, user)

			return true, nil
		},
	})
}

func ErrorsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err == nil {
			return nil
		}

		switch {
		case errors.Is(err, jwt.ErrInvalidToken):
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		case errors.Is(err, jwt.ErrExpiredToken):
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())

		default:
			return err //nolint:wrapcheck // err is already wrapped
		}
	}
}

func WithRole(required users.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := GetUser(c)
		if !ok {
			return fiber.ErrUnauthorized
		}

		if user.Role != required {
			return fiber.ErrForbidden
		}

		return c.Next()
	}
}

func GetUser(c *fiber.Ctx) (*users.User, bool) {
	user, ok := c.Locals(userKey).(*users.User)
	return user, ok
}
