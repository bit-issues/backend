package webhooks_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/bit-issues/backend/internal/projects"
	serverwebhooks "github.com/bit-issues/backend/internal/server/webhooks"
	internalwebhooks "github.com/bit-issues/backend/internal/webhooks"
	"github.com/bit-issues/backend/pkg/bitbucket"
	"github.com/gofiber/fiber/v2"
)

func TestClassifyWebhookError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{"oauth disabled", internalwebhooks.ErrOAuthDisabled, fiber.StatusNotImplemented, ""},
		{"callback not configured", internalwebhooks.ErrCallbackURLNotConfigured, fiber.StatusNotImplemented, ""},
		{"callback invalid", internalwebhooks.ErrCallbackURLInvalid, fiber.StatusNotImplemented, ""},
		{"secret not configured", internalwebhooks.ErrSecretNotConfigured, fiber.StatusNotImplemented, ""},
		{
			"oauth not connected",
			internalwebhooks.ErrOAuthNotConnected,
			fiber.StatusConflict,
			"reconnect Bitbucket OAuth",
		},
		{"upstream 401", bitbucket.ErrUnauthorized, fiber.StatusConflict, "reconnect Bitbucket OAuth"},
		{
			"wrapped upstream 401",
			fmt.Errorf("list hooks: %w", bitbucket.ErrUnauthorized),
			fiber.StatusConflict,
			"reconnect Bitbucket OAuth",
		},
		{"upstream 403", bitbucket.ErrForbidden, fiber.StatusForbidden, "admin access"},
		{
			"wrapped upstream 403",
			fmt.Errorf("create hook: %w", bitbucket.ErrForbidden),
			fiber.StatusForbidden,
			"admin access",
		},
		{"upstream 404", bitbucket.ErrNotFound, fiber.StatusNotFound, "not found"},
		{
			"not bitbucket repo",
			internalwebhooks.ErrNotBitbucketRepo,
			fiber.StatusUnprocessableEntity,
			"not a bitbucket.org repository",
		},
		{"project not found", projects.ErrNotFound, fiber.StatusNotFound, "project not found"},
		{
			"transport failure",
			&bitbucket.APIError{StatusCode: 0, Message: "bitbucket api unavailable"},
			fiber.StatusBadGateway,
			"Bitbucket API unavailable",
		},
		{
			"upstream 5xx",
			&bitbucket.APIError{StatusCode: fiber.StatusBadGateway, Message: "bitbucket api error"},
			fiber.StatusBadGateway,
			"Bitbucket API unavailable",
		},
		{
			"upstream 429",
			&bitbucket.APIError{StatusCode: fiber.StatusTooManyRequests, Message: "bitbucket api error"},
			fiber.StatusServiceUnavailable,
			"rate limit",
		},
		{
			"upstream 400",
			&bitbucket.APIError{StatusCode: fiber.StatusBadRequest, Message: "bitbucket api error"},
			fiber.StatusBadGateway,
			"Bitbucket API unavailable",
		},
		{"unknown error", io.ErrUnexpectedEOF, fiber.StatusInternalServerError, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serverwebhooks.ClassifyWebhookErrorForTest(tt.err)
			var fe *fiber.Error
			if !errors.As(got, &fe) {
				if tt.wantStatus != fiber.StatusInternalServerError {
					t.Fatalf("expected fiber error with status %d, got %v", tt.wantStatus, got)
				}
				return
			}
			if fe.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", fe.Code, tt.wantStatus, tt.err)
			}
			if tt.wantMessage != "" && !strings.Contains(strings.ToLower(fe.Message), strings.ToLower(tt.wantMessage)) {
				t.Fatalf("message %q does not contain %q", fe.Message, tt.wantMessage)
			}
		})
	}
}

func TestClassifyWebhookErrorRedactsRepositoryURL(t *testing.T) {
	// The repository URL is user supplied and may carry credentials, so the
	// 422 response must not echo it back.
	repoURL := "https://user:token@github.com/org/repo"
	err := fmt.Errorf("%w: %s", internalwebhooks.ErrNotBitbucketRepo, repoURL)

	got := serverwebhooks.ClassifyWebhookErrorForTest(err)
	var fe *fiber.Error
	if !errors.As(got, &fe) {
		t.Fatalf("expected fiber error, got %v", got)
	}
	if fe.Code != fiber.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", fe.Code, fiber.StatusUnprocessableEntity)
	}
	if strings.Contains(fe.Message, repoURL) || strings.Contains(fe.Message, "user:token") {
		t.Fatalf("message %q must not contain the repository URL", fe.Message)
	}
}

func TestWebhookStatusResponseJSON(t *testing.T) {
	// Registered state: all contract keys present with values.
	uuid := "{abc-123}"
	updated := "2026-09-29T12:00:00Z"
	st := &internalwebhooks.WebhookStatus{
		Status:      internalwebhooks.StatusRegistered,
		HookUUID:    &uuid,
		CallbackURL: testCallbackURL,
		UpdatedAt:   &updated,
	}
	data, err := json.Marshal(serverwebhooks.NewWebhookStatusResponse(st))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	body := string(data)
	for _, key := range []string{`"status"`, `"hook_uuid"`, `"callback_url"`, `"failure_reason"`, `"updated_at"`} {
		if !strings.Contains(body, key) {
			t.Fatalf("missing key %s in %s", key, body)
		}
	}
	if !strings.Contains(body, `"status":"registered"`) || !strings.Contains(body, `"hook_uuid":"{abc-123}"`) {
		t.Fatalf("unexpected registered body: %s", body)
	}

	// Unavailable state: callback_url empty string, nullable fields null.
	reason := "Bitbucket OAuth is not connected"
	st = &internalwebhooks.WebhookStatus{
		Status:        internalwebhooks.StatusUnavailable,
		FailureReason: &reason,
	}
	data, err = json.Marshal(serverwebhooks.NewWebhookStatusResponse(st))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	body = string(data)
	if !strings.Contains(body, `"callback_url":""`) {
		t.Fatalf("callback_url must be an empty string when unavailable: %s", body)
	}
	if !strings.Contains(body, `"hook_uuid":null`) {
		t.Fatalf("hook_uuid must be null: %s", body)
	}
	if !strings.Contains(body, `"updated_at":null`) {
		t.Fatalf("updated_at must be null: %s", body)
	}
	if !strings.Contains(body, `"failure_reason":"Bitbucket OAuth is not connected"`) {
		t.Fatalf("failure_reason missing: %s", body)
	}
}

const testCallbackURL = "https://bitissues.example.com/api/v1/webhooks/bitbucket/push"
