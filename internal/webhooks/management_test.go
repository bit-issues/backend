package webhooks_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bit-issues/backend/internal/projects"
	webhookssvc "github.com/bit-issues/backend/internal/webhooks"
	"github.com/bit-issues/backend/pkg/bitbucket"
	restkit "github.com/capcom6/go-restkit"
)

const (
	mgmtTestAccessToken   = "test-access-token"
	mgmtTestWebhookSecret = "test-webhook-secret"
	mgmtTestCallbackURL   = "https://issues.example.com"
	mgmtTestCallbackPath  = "/api/v1/webhooks/bitbucket/push"
	mgmtTestRepoURL       = "https://bitbucket.org/workspace/repo-slug"
	mgmtTestHookUUID      = "{abc-123}"
	mgmtTestHookCreatedAt = "2026-08-20T10:00:00+00:00"
	mgmtTestHookPath      = "/2.0/repositories/workspace/repo-slug/hooks"
)

// newManagementService builds a management service wired to the real
// Bitbucket client pointed at the given mock server.
func newManagementService(t *testing.T, handler http.Handler, secret string) *webhookssvc.ManagementService {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL:     server.URL,
		AccessToken: mgmtTestAccessToken,
		CallbackURL: mgmtTestCallbackURL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	return webhookssvc.NewManagementService(
		webhookssvc.Config{Secret: secret},
		bitbucket.Config{AccessToken: mgmtTestAccessToken, CallbackURL: mgmtTestCallbackURL},
		client,
	)
}

// mgmtHookJSON mirrors the Bitbucket webhook object used by the mock server.
func mgmtHookJSON(description string) map[string]any {
	return map[string]any{
		"uuid":        mgmtTestHookUUID,
		"url":         mgmtTestCallbackURL + mgmtTestCallbackPath,
		"description": description,
		"active":      true,
		"events":      []string{"repo:push"},
		"created_at":  mgmtTestHookCreatedAt,
		"secret_set":  true,
	}
}

// mgmtTestProject returns a project pointing at the test repository.
func mgmtTestProject() *projects.Project {
	return &projects.Project{
		ID:      "my-project",
		RepoURL: mgmtTestRepoURL,
	}
}

// assertPayload asserts the webhook request body fields for create/update.
func assertPayload(t *testing.T, payload map[string]any, description string) {
	t.Helper()

	for field, want := range map[string]any{
		"description": description,
		"url":         mgmtTestCallbackURL + mgmtTestCallbackPath,
		"active":      true,
		"secret":      mgmtTestWebhookSecret,
	} {
		if got := payload[field]; got != want {
			t.Errorf("payload[%q] = %v, want %v", field, got, want)
		}
	}

	events, ok := payload["events"].([]any)
	if !ok || len(events) != 1 || events[0] != "repo:push" {
		t.Errorf("payload[events] = %#v, want [repo:push]", payload["events"])
	}
}

func TestManagementRegisterWebhookCreatesWhenAbsent(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuth    string
		gotPayload map[string]any
	)

	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		case http.MethodPost:
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			if unmarshalErr := json.Unmarshal(body, &gotPayload); unmarshalErr != nil {
				t.Errorf("unmarshal body: %v", unmarshalErr)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(mgmtHookJSON("BitIssues webhook"))
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}), mgmtTestWebhookSecret)

	status, err := svc.RegisterWebhook(context.Background(), mgmtTestProject())
	if err != nil {
		t.Fatalf("RegisterWebhook() error = %v", err)
	}

	if status.Status != webhookssvc.WebhookStatusRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhookssvc.WebhookStatusRegistered)
	}
	if status.HookUUID != mgmtTestHookUUID {
		t.Errorf("HookUUID = %q, want %q", status.HookUUID, mgmtTestHookUUID)
	}
	if status.CallbackURL != mgmtTestCallbackURL+mgmtTestCallbackPath {
		t.Errorf("CallbackURL = %q, want %q", status.CallbackURL, mgmtTestCallbackURL+mgmtTestCallbackPath)
	}
	if status.UpdatedAt != mgmtTestHookCreatedAt {
		t.Errorf("UpdatedAt = %q, want %q", status.UpdatedAt, mgmtTestHookCreatedAt)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != mgmtTestHookPath {
		t.Errorf("path = %q, want %q", gotPath, mgmtTestHookPath)
	}
	if gotAuth != "Bearer "+mgmtTestAccessToken {
		t.Errorf("Authorization = %q, want Bearer %q", gotAuth, mgmtTestAccessToken)
	}

	assertPayload(t, gotPayload, "BitIssues webhook")
}

func TestManagementRegisterWebhookUpdatesExistingHook(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotPayload map[string]any
	)

	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{mgmtHookJSON("existing hook")}})
		case http.MethodPut:
			gotMethod = r.Method
			gotPath = r.URL.Path

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			if unmarshalErr := json.Unmarshal(body, &gotPayload); unmarshalErr != nil {
				t.Errorf("unmarshal body: %v", unmarshalErr)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mgmtHookJSON("existing hook"))
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}), mgmtTestWebhookSecret)

	status, err := svc.RegisterWebhook(context.Background(), mgmtTestProject())
	if err != nil {
		t.Fatalf("RegisterWebhook() error = %v", err)
	}

	if status.Status != webhookssvc.WebhookStatusRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhookssvc.WebhookStatusRegistered)
	}
	if status.HookUUID != mgmtTestHookUUID {
		t.Errorf("HookUUID = %q, want %q", status.HookUUID, mgmtTestHookUUID)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != mgmtTestHookPath+"/"+mgmtTestHookUUID {
		t.Errorf("path = %q, want %q", gotPath, mgmtTestHookPath+"/"+mgmtTestHookUUID)
	}

	assertPayload(t, gotPayload, "existing hook")
}

func TestManagementUnregisterWebhookDeletesWhenPresent(t *testing.T) {
	var gotDeletePath string

	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{mgmtHookJSON("BitIssues webhook")}})
		case http.MethodDelete:
			gotDeletePath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}), mgmtTestWebhookSecret)

	status, err := svc.UnregisterWebhook(context.Background(), mgmtTestProject())
	if err != nil {
		t.Fatalf("UnregisterWebhook() error = %v", err)
	}

	if status.Status != webhookssvc.WebhookStatusNotRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhookssvc.WebhookStatusNotRegistered)
	}
	if status.HookUUID != "" {
		t.Errorf("HookUUID = %q, want empty", status.HookUUID)
	}
	if status.CallbackURL != mgmtTestCallbackURL+mgmtTestCallbackPath {
		t.Errorf("CallbackURL = %q, want %q", status.CallbackURL, mgmtTestCallbackURL+mgmtTestCallbackPath)
	}
	if gotDeletePath != mgmtTestHookPath+"/"+mgmtTestHookUUID {
		t.Errorf("delete path = %q, want %q", gotDeletePath, mgmtTestHookPath+"/"+mgmtTestHookUUID)
	}
}

func TestManagementUnregisterWebhookNoOpWhenAbsent(t *testing.T) {
	requests := 0

	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
	}), mgmtTestWebhookSecret)

	status, err := svc.UnregisterWebhook(context.Background(), mgmtTestProject())
	if err != nil {
		t.Fatalf("UnregisterWebhook() error = %v", err)
	}

	if status.Status != webhookssvc.WebhookStatusNotRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhookssvc.WebhookStatusNotRegistered)
	}
	if requests != 1 {
		t.Errorf("bitbucket requests = %d, want 1 (list only)", requests)
	}
}

func TestManagementGetWebhookStatusRegistered(t *testing.T) {
	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{mgmtHookJSON("BitIssues webhook")}})
	}), mgmtTestWebhookSecret)

	status, err := svc.GetWebhookStatus(context.Background(), mgmtTestProject())
	if err != nil {
		t.Fatalf("GetWebhookStatus() error = %v", err)
	}

	if status.Status != webhookssvc.WebhookStatusRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhookssvc.WebhookStatusRegistered)
	}
	if status.HookUUID != mgmtTestHookUUID {
		t.Errorf("HookUUID = %q, want %q", status.HookUUID, mgmtTestHookUUID)
	}
	if status.UpdatedAt != mgmtTestHookCreatedAt {
		t.Errorf("UpdatedAt = %q, want %q", status.UpdatedAt, mgmtTestHookCreatedAt)
	}
}

func TestManagementGetWebhookStatusNotRegistered(t *testing.T) {
	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
	}), mgmtTestWebhookSecret)

	status, err := svc.GetWebhookStatus(context.Background(), mgmtTestProject())
	if err != nil {
		t.Fatalf("GetWebhookStatus() error = %v", err)
	}

	if status.Status != webhookssvc.WebhookStatusNotRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhookssvc.WebhookStatusNotRegistered)
	}
	if status.HookUUID != "" {
		t.Errorf("HookUUID = %q, want empty", status.HookUUID)
	}
	if status.CallbackURL != mgmtTestCallbackURL+mgmtTestCallbackPath {
		t.Errorf("CallbackURL = %q, want %q", status.CallbackURL, mgmtTestCallbackURL+mgmtTestCallbackPath)
	}
}

func TestManagementRegisterWebhookFailsWhenSecretEmpty(t *testing.T) {
	requests := 0

	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}), "")

	_, err := svc.RegisterWebhook(context.Background(), mgmtTestProject())
	if !errors.Is(err, webhookssvc.ErrWebhookSecretNotConfigured) {
		t.Fatalf("RegisterWebhook() error = %v, want ErrWebhookSecretNotConfigured", err)
	}
	if requests != 0 {
		t.Errorf("bitbucket requests = %d, want 0", requests)
	}
}

func TestManagementRegisterWebhookRejectsInvalidRepoURL(t *testing.T) {
	requests := 0

	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}), mgmtTestWebhookSecret)

	project := &projects.Project{ID: "bad", RepoURL: "not-a-repository-url"}

	_, err := svc.RegisterWebhook(context.Background(), project)
	if !errors.Is(err, projects.ErrInvalidURL) {
		t.Fatalf("RegisterWebhook() error = %v, want ErrInvalidURL", err)
	}
	if requests != 0 {
		t.Errorf("bitbucket requests = %d, want 0", requests)
	}
}

func TestManagementGetWebhookStatusMapsBitbucket403ToClientError(t *testing.T) {
	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"access denied"}}`))
	}), mgmtTestWebhookSecret)

	_, err := svc.GetWebhookStatus(context.Background(), mgmtTestProject())
	if err == nil {
		t.Fatal("GetWebhookStatus() error = nil, want error")
	}
	if !restkit.IsClientError(err) {
		t.Errorf("IsClientError(err) = false, want true")
	}
	apiErr, ok := restkit.AsAPIError(err)
	if !ok || apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("AsAPIError(err).StatusCode = %v, want %d", apiErr, http.StatusForbidden)
	}
}

func TestManagementGetWebhookStatusMapsBitbucket400ToClientError(t *testing.T) {
	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"callback url is not reachable"}}`))
	}), mgmtTestWebhookSecret)

	_, err := svc.GetWebhookStatus(context.Background(), mgmtTestProject())
	if err == nil {
		t.Fatal("GetWebhookStatus() error = nil, want error")
	}
	if !restkit.IsClientError(err) {
		t.Errorf("IsClientError(err) = false, want true")
	}
	apiErr, ok := restkit.AsAPIError(err)
	if !ok || apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("AsAPIError(err).StatusCode = %v, want %d", apiErr, http.StatusBadRequest)
	}
}

func TestManagementGetWebhookStatusMapsBitbucketServerError(t *testing.T) {
	svc := newManagementService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}), mgmtTestWebhookSecret)

	_, err := svc.GetWebhookStatus(context.Background(), mgmtTestProject())
	if err == nil {
		t.Fatal("GetWebhookStatus() error = nil, want error")
	}
	if !restkit.IsServerError(err) {
		t.Errorf("IsServerError(err) = false, want true")
	}
	if restkit.IsClientError(err) {
		t.Errorf("IsClientError(err) = true, want false")
	}
}

func TestManagementGetWebhookStatusMapsInfrastructureError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()

	client, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL:     baseURL,
		AccessToken: mgmtTestAccessToken,
		CallbackURL: mgmtTestCallbackURL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	svc := webhookssvc.NewManagementService(
		webhookssvc.Config{Secret: mgmtTestWebhookSecret},
		bitbucket.Config{AccessToken: mgmtTestAccessToken, CallbackURL: mgmtTestCallbackURL},
		client,
	)

	_, err = svc.GetWebhookStatus(context.Background(), mgmtTestProject())
	if err == nil {
		t.Fatal("GetWebhookStatus() error = nil, want error")
	}
	if !restkit.IsInfrastructureError(err) {
		t.Errorf("IsInfrastructureError(err) = false, want true")
	}
}
