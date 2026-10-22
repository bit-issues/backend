package webhooks_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/bit-issues/backend/pkg/bitbucket"
	"go.uber.org/zap"
)

const testWebhookURL = "https://bitissues.example.com/api/v1/webhooks/bitbucket/push"

func newManagementService(cfg webhooks.Config) *webhooks.ManagementService {
	return webhooks.NewManagementService(cfg, bitbucket.NewClient(""), nil, nil, zap.NewNop())
}

func TestValidateCallbackURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		err  error
	}{
		{"empty", "", "", webhooks.ErrCallbackURLNotConfigured},
		{"https base", "https://bitissues.example.com", testWebhookURL, nil},
		{"trailing slash", "https://bitissues.example.com/", testWebhookURL, nil},
		{
			"base path",
			"https://bitissues.example.com/base",
			"https://bitissues.example.com/base/api/v1/webhooks/bitbucket/push",
			nil,
		},
		{
			"base path trailing slash",
			"https://bitissues.example.com/base/",
			"https://bitissues.example.com/base/api/v1/webhooks/bitbucket/push",
			nil,
		},
		{
			"port",
			"https://bitissues.example.com:8443",
			"https://bitissues.example.com:8443/api/v1/webhooks/bitbucket/push",
			nil,
		},
		{"http scheme", "http://bitissues.example.com", "", webhooks.ErrCallbackURLInvalid},
		{"query string", "https://bitissues.example.com?x=1", "", webhooks.ErrCallbackURLInvalid},
		{"fragment", "https://bitissues.example.com#frag", "", webhooks.ErrCallbackURLInvalid},
		{"bare query", "https://bitissues.example.com/?", "", webhooks.ErrCallbackURLInvalid},
		{"no host", "https://", "", webhooks.ErrCallbackURLInvalid},
		{"garbage", "not-a-url", "", webhooks.ErrCallbackURLInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := webhooks.ValidateCallbackURLForTest(tt.raw)
			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Fatalf("expected %v, got %v", tt.err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRegisterFailClosed(t *testing.T) {
	tests := []struct {
		name string
		cfg  webhooks.Config
		want error
	}{
		{
			"missing callback and secret",
			webhooks.Config{CallbackURL: "", Secret: ""},
			webhooks.ErrCallbackURLNotConfigured,
		},
		{
			"invalid callback",
			webhooks.Config{CallbackURL: "http://example.com", Secret: "s"},
			webhooks.ErrCallbackURLInvalid,
		},
		{
			"missing secret",
			webhooks.Config{CallbackURL: "https://example.com", Secret: ""},
			webhooks.ErrSecretNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newManagementService(tt.cfg)
			_, err := svc.Register(t.Context(), 1, "proj")
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestUnregisterFailClosed(t *testing.T) {
	tests := []struct {
		name string
		cfg  webhooks.Config
		want error
	}{
		{"missing callback", webhooks.Config{}, webhooks.ErrCallbackURLNotConfigured},
		{"invalid callback", webhooks.Config{CallbackURL: "https://example.com?x=1"}, webhooks.ErrCallbackURLInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newManagementService(tt.cfg)
			if err := svc.Unregister(t.Context(), 1, "proj"); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestStatusProceedsWithoutSecret verifies the fail-closed ordering: status is
// a read-only live query that requires only a valid callback URL and works
// with an empty secret. Local prerequisites surface as a 200 unavailable
// status, not errors.
func TestStatusProceedsWithoutSecret(t *testing.T) {
	tests := []struct {
		name       string
		cfg        webhooks.Config
		wantReason string
	}{
		{"empty callback, empty secret", webhooks.Config{}, "Bitbucket callback URL is not configured"},
		{
			"invalid callback, empty secret",
			webhooks.Config{CallbackURL: "http://example.com"},
			"Bitbucket callback URL is invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newManagementService(tt.cfg)
			st, err := svc.Status(t.Context(), 1, "proj")
			if err != nil {
				t.Fatalf("status must not error on local prerequisites: %v", err)
			}
			if st.Status != webhooks.StatusUnavailable {
				t.Fatalf("status = %q, want %q", st.Status, webhooks.StatusUnavailable)
			}
			if st.CallbackURL != "" {
				t.Fatalf("callback_url = %q, want empty when unavailable", st.CallbackURL)
			}
			if st.FailureReason == nil || *st.FailureReason != tt.wantReason {
				t.Fatalf("failure_reason = %v, want %q", st.FailureReason, tt.wantReason)
			}
		})
	}
}

func TestFindMatchingHook(t *testing.T) {
	hooks := []bitbucket.Hook{
		testHook("{a}", "https://other.example/push", true),
		testHook("{b}", testWebhookURL, true),
	}

	got := webhooks.FindMatchingHookForTest(hooks, testWebhookURL)
	if got == nil || got.UUID != "{b}" {
		t.Fatalf("got %+v, want hook {b}", got)
	}
	if webhooks.FindMatchingHookForTest(hooks, "https://nope.example/push") != nil {
		t.Fatalf("expected no match for foreign URL")
	}
	if webhooks.FindMatchingHookForTest(nil, testWebhookURL) != nil {
		t.Fatalf("expected no match for empty hooks")
	}
}

func TestFindMatchingHookRequiresPushEvent(t *testing.T) {
	// A hook on the same callback URL that does not subscribe to the managed
	// push event is not the managed hook and must not be reported, updated or
	// deleted as such.
	other := testHook("{a}", testWebhookURL, true)
	other.Events = []string{"pullrequest:created"}
	managed := testHook("{b}", testWebhookURL, true)

	got := webhooks.FindMatchingHookForTest([]bitbucket.Hook{other, managed}, testWebhookURL)
	if got == nil || got.UUID != "{b}" {
		t.Fatalf("got %+v, want hook {b}", got)
	}

	if webhooks.FindMatchingHookForTest([]bitbucket.Hook{other}, testWebhookURL) != nil {
		t.Fatalf("expected no match for a hook without the push event")
	}
}

func TestWebhookStatusFromHook(t *testing.T) {
	// No hook -> not_registered with null uuid and updated_at.
	st := webhooks.WebhookStatusFromHookForTest(nil, testWebhookURL)
	if st.Status != webhooks.StatusNotRegistered {
		t.Fatalf("status = %q, want not_registered", st.Status)
	}
	if st.HookUUID != nil || st.UpdatedAt != nil {
		t.Fatalf("uuid/updated_at must be null without a hook: %+v", st)
	}
	if st.CallbackURL != testWebhookURL {
		t.Fatalf("callback_url = %q, want %q", st.CallbackURL, testWebhookURL)
	}

	// Active hook -> registered with uuid and RFC3339 updated_at.
	active := testHook("{a}", testWebhookURL, true)
	active.UpdatedOn = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	st = webhooks.WebhookStatusFromHookForTest(&active, testWebhookURL)
	if st.Status != webhooks.StatusRegistered {
		t.Fatalf("status = %q, want registered", st.Status)
	}
	if st.HookUUID == nil || *st.HookUUID != "{a}" {
		t.Fatalf("hook_uuid = %v, want {a}", st.HookUUID)
	}
	if st.UpdatedAt == nil || *st.UpdatedAt != "2026-09-29T12:00:00Z" {
		t.Fatalf("updated_at = %v, want RFC3339", st.UpdatedAt)
	}

	// Inactive hook -> failed; zero updated_on -> null.
	inactive := testHook("{b}", testWebhookURL, false)
	st = webhooks.WebhookStatusFromHookForTest(&inactive, testWebhookURL)
	if st.Status != webhooks.StatusFailed {
		t.Fatalf("status = %q, want failed", st.Status)
	}
	if st.UpdatedAt != nil {
		t.Fatalf("updated_at = %v, want null for zero value", st.UpdatedAt)
	}
}

func testHook(uuid, url string, active bool) bitbucket.Hook {
	return bitbucket.Hook{
		UUID:        uuid,
		Description: "BitIssues push webhook",
		URL:         url,
		Active:      active,
		Secret:      "",
		Events:      []string{"repo:push"},
		CreatedOn:   time.Time{},
		UpdatedOn:   time.Time{},
	}
}
