package webhooks_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bit-issues/backend/internal/server/webhooks"
	webhookssvc "github.com/bit-issues/backend/internal/webhooks"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	testSecret = "test-secret"
	// Empty repository full_name keeps processing on the early-return path,
	// so the handler can be exercised end-to-end without backing services.
	testBody = `{"repository":{"full_name":""},"push":{"changes":[]}}`
)

// sign returns a WebSub-style signature as sent by Bitbucket Cloud:
// "sha256=" + lowercase hex HMAC-SHA256 of the raw body.
func sign(t *testing.T, secret, body string) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func newService(t *testing.T, secret string) *webhookssvc.Service {
	t.Helper()

	svc, err := webhookssvc.NewService(
		webhookssvc.Config{Secret: secret},
		nil, nil, nil, nil,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func newWebhookApp(t *testing.T, secret string) *fiber.App {
	t.Helper()

	app := fiber.New()
	webhooks.NewHandler(newService(t, secret), zap.NewNop()).Register(app)
	return app
}

// postPush sends the test body to the webhook endpoint with the given headers
// and returns the response status code.
func postPush(t *testing.T, app *fiber.App, headers map[string]string) int {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/webhooks/bitbucket/push", strings.NewReader(testBody))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func TestVerifyPushEventAcceptsValidSignatureFromHubSignatureHeader(t *testing.T) {
	svc := newService(t, testSecret)

	if err := svc.VerifyPushEvent([]byte(testBody), sign(t, testSecret, testBody)); err != nil {
		t.Fatalf("VerifyPushEvent() error = %v, want nil", err)
	}
}

func TestVerifyPushEventAcceptsValidSignatureFromHubSignature256Header(t *testing.T) {
	svc := newService(t, testSecret)

	// Same WebSub format regardless of which header carried it.
	if err := svc.VerifyPushEvent([]byte(testBody), sign(t, testSecret, testBody)); err != nil {
		t.Fatalf("VerifyPushEvent() error = %v, want nil", err)
	}
}

func TestVerifyPushEventRejectsInvalidPrefix(t *testing.T) {
	validSig := strings.TrimPrefix(sign(t, testSecret, testBody), "sha256=")

	tests := []struct {
		name string
		sig  string
	}{
		{name: "md5 prefix", sig: "md5=" + validSig},
		{name: "sha1 prefix", sig: "sha1=" + validSig},
		{name: "missing prefix", sig: validSig},
		{name: "prefix only", sig: "sha256="},
		{name: "wrong separator", sig: "sha256:" + validSig},
	}

	svc := newService(t, testSecret)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.VerifyPushEvent([]byte(testBody), tt.sig)
			if !errors.Is(err, webhookssvc.ErrInvalidSignature) {
				t.Fatalf("VerifyPushEvent(%q) error = %v, want ErrInvalidSignature", tt.sig, err)
			}
		})
	}
}

func TestVerifyPushEventRejectsMismatchedSignature(t *testing.T) {
	tests := []struct {
		name string
		sig  string
	}{
		{name: "signed with wrong secret", sig: sign(t, "other-secret", testBody)},
		{name: "signed different body", sig: sign(t, testSecret, testBody+"x")},
		{name: "garbage hex", sig: "sha256=zzzz-not-hex"},
		{name: "truncated hex", sig: "sha256=abcd"},
		{name: "all zeros", sig: "sha256=" + strings.Repeat("0", 64)},
	}

	svc := newService(t, testSecret)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.VerifyPushEvent([]byte(testBody), tt.sig)
			if !errors.Is(err, webhookssvc.ErrInvalidSignature) {
				t.Fatalf("VerifyPushEvent(%q) error = %v, want ErrInvalidSignature", tt.sig, err)
			}
		})
	}
}

func TestVerifyPushEventRejectsMissingHeaders(t *testing.T) {
	svc := newService(t, testSecret)

	err := svc.VerifyPushEvent([]byte(testBody), "")
	if !errors.Is(err, webhookssvc.ErrInvalidSignature) {
		t.Fatalf("VerifyPushEvent(\"\") error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyPushEventRejectsEmptySecret(t *testing.T) {
	svc := newService(t, "")

	err := svc.VerifyPushEvent([]byte(testBody), sign(t, "", testBody))
	if !errors.Is(err, webhookssvc.ErrInvalidSignature) {
		t.Fatalf("VerifyPushEvent() error = %v, want ErrInvalidSignature", err)
	}
}

func TestHandlePushAcceptsXHubSignatureHeader(t *testing.T) {
	app := newWebhookApp(t, testSecret)

	got := postPush(t, app, map[string]string{
		"X-Hub-Signature": sign(t, testSecret, testBody),
	})
	if got != fiber.StatusAccepted {
		t.Fatalf("status = %d, want %d (valid X-Hub-Signature must be accepted)", got, fiber.StatusAccepted)
	}
}

func TestHandlePushBackwardCompatXHubSignature256Header(t *testing.T) {
	app := newWebhookApp(t, testSecret)

	got := postPush(t, app, map[string]string{
		"X-Hub-Signature-256": sign(t, testSecret, testBody),
	})
	if got != fiber.StatusAccepted {
		t.Fatalf("status = %d, want %d (valid X-Hub-Signature-256 must stay accepted)", got, fiber.StatusAccepted)
	}
}

func TestHandlePushPrefersXHubSignatureOverXHubSignature256(t *testing.T) {
	app := newWebhookApp(t, testSecret)

	// First header wins: valid X-Hub-Signature + invalid -256 -> accepted.
	got := postPush(t, app, map[string]string{
		"X-Hub-Signature":     sign(t, testSecret, testBody),
		"X-Hub-Signature-256": "sha256=" + strings.Repeat("0", 64),
	})
	if got != fiber.StatusAccepted {
		t.Fatalf("status = %d, want %d (X-Hub-Signature must take precedence)", got, fiber.StatusAccepted)
	}

	// And not an OR: invalid X-Hub-Signature + valid -256 -> rejected.
	got = postPush(t, app, map[string]string{
		"X-Hub-Signature":     "sha256=" + strings.Repeat("0", 64),
		"X-Hub-Signature-256": sign(t, testSecret, testBody),
	})
	if got != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (invalid X-Hub-Signature must not fall through)", got, fiber.StatusUnauthorized)
	}
}

func TestHandlePushRejectsMissingSignature(t *testing.T) {
	app := newWebhookApp(t, testSecret)

	got := postPush(t, app, nil)
	if got != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", got, fiber.StatusUnauthorized)
	}
}

func TestHandlePushRejectsInvalidSignature(t *testing.T) {
	app := newWebhookApp(t, testSecret)

	for _, sig := range []string{
		sign(t, "other-secret", testBody),
		"md5=d41d8cd98f00b204e9800998ecf8427e",
	} {
		got := postPush(t, app, map[string]string{"X-Hub-Signature": sig})
		if got != fiber.StatusUnauthorized {
			t.Fatalf("status = %d, want %d for sig %q", got, fiber.StatusUnauthorized, sig)
		}
	}
}

func TestHandlePushRejectsEmptySecret(t *testing.T) {
	app := newWebhookApp(t, "")

	got := postPush(t, app, map[string]string{
		"X-Hub-Signature": sign(t, "", testBody),
	})
	if got != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (empty secret must stay rejected)", got, fiber.StatusUnauthorized)
	}
}
