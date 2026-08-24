package bitbucket_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bit-issues/backend/pkg/bitbucket"
	restkit "github.com/capcom6/go-restkit"
)

const testToken = "test-access-token"

// webhooksPage mirrors the unexported paginated listing shape the client
// decodes into; the external test package defines its own copy.
type webhooksPage struct {
	Values []bitbucket.Webhook `json:"values"`
	Next   string              `json:"next"`
}

func newTestClient(t *testing.T, handler http.Handler) *bitbucket.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL:     server.URL,
		AccessToken: testToken,
		CallbackURL: "https://issues.example.com",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	return client
}

// TestBearerAuthOnEveryRequest asserts Authorization: Bearer is sent on every
// outbound request regardless of method.
func TestBearerAuthOnEveryRequest(t *testing.T) {
	var gotAuth []string

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(bitbucket.Webhook{UUID: "{u1}"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(bitbucket.Webhook{UUID: "{u1}"})
		default:
			_ = json.NewEncoder(w).Encode(webhooksPage{})
		}
	}))

	ctx := context.Background()
	if _, err := client.CreateWebhook(ctx, "ws", "repo", bitbucket.CreateWebhookRequest{Description: "d"}); err != nil {
		t.Fatalf("CreateWebhook() error = %v", err)
	}
	if _, err := client.ListWebhooks(ctx, "ws", "repo"); err != nil {
		t.Fatalf("ListWebhooks() error = %v", err)
	}
	if _, err := client.UpdateWebhook(ctx, "ws", "repo", "{u1}", bitbucket.UpdateWebhookRequest{}); err != nil {
		t.Fatalf("UpdateWebhook() error = %v", err)
	}
	if err := client.DeleteWebhook(ctx, "ws", "repo", "{u1}"); err != nil {
		t.Fatalf("DeleteWebhook() error = %v", err)
	}

	if len(gotAuth) != 4 {
		t.Fatalf("captured %d requests, want 4", len(gotAuth))
	}
	for i, got := range gotAuth {
		if want := "Bearer " + testToken; got != want {
			t.Errorf("request %d Authorization = %q, want %q", i, got, want)
		}
	}
}

func TestCreateWebhook(t *testing.T) {
	var (
		gotMethod     string
		gotPath       string
		gotPayload    map[string]any
		gotAuthHeader string
	)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHeader = r.Header.Get("Authorization")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if unmarshalErr := json.Unmarshal(body, &gotPayload); unmarshalErr != nil {
			t.Errorf("unmarshal payload: %v", unmarshalErr)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":        "{abc-123}",
			"url":         "https://api.bitbucket.org/2.0/repositories/team/repo/hooks/{abc-123}",
			"description": "ci-hook",
			"active":      true,
			"events":      []string{"repo:push"},
			"created_at":  "2026-08-20T10:00:00+00:00",
			"secret_set":  true,
		})
	}))

	webhook, err := client.CreateWebhook(context.Background(), "team", "repo", bitbucket.CreateWebhookRequest{
		Description: "ci-hook",
		URL:         "https://issues.example.com/api/webhooks/bitbucket",
		Active:      true,
		Events:      []string{"repo:push", "issue:created"},
		Secret:      "shared-secret",
	})
	if err != nil {
		t.Fatalf("CreateWebhook() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/2.0/repositories/team/repo/hooks" {
		t.Errorf("path = %q, want /2.0/repositories/team/repo/hooks", gotPath)
	}
	if gotAuthHeader != "Bearer "+testToken {
		t.Errorf("Authorization = %q, want Bearer %q", gotAuthHeader, testToken)
	}

	for field, want := range map[string]any{
		"description": "ci-hook",
		"url":         "https://issues.example.com/api/webhooks/bitbucket",
		"active":      true,
		"secret":      "shared-secret",
	} {
		if got := gotPayload[field]; got != want {
			t.Errorf("payload[%q] = %v, want %v", field, got, want)
		}
	}
	events, ok := gotPayload["events"].([]any)
	if !ok || len(events) != 2 || events[0] != "repo:push" || events[1] != "issue:created" {
		t.Errorf("payload[events] = %#v, want [repo:push issue:created]", gotPayload["events"])
	}

	if webhook.UUID != "{abc-123}" {
		t.Errorf("UUID = %q, want {abc-123}", webhook.UUID)
	}
	if webhook.Description != "ci-hook" || !webhook.Active || webhook.SecretSet != true {
		t.Errorf("unexpected webhook: %+v", webhook)
	}
	if len(webhook.Events) != 1 || webhook.Events[0] != "repo:push" {
		t.Errorf("Events = %#v, want [repo:push]", webhook.Events)
	}
	if webhook.CreatedAt != "2026-08-20T10:00:00+00:00" {
		t.Errorf("CreatedAt = %q", webhook.CreatedAt)
	}
}

func TestListWebhooksPagination(t *testing.T) {
	requestCount := 0

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "", "1":
			_ = json.NewEncoder(w).Encode(webhooksPage{
				Values: []bitbucket.Webhook{{UUID: "{w1}", Description: "first"}},
				Next:   fmt.Sprintf("http://%s/2.0/repositories/team/repo/hooks?page=2", r.Host),
			})
		case "2":
			_ = json.NewEncoder(w).Encode(webhooksPage{
				Values: []bitbucket.Webhook{{UUID: "{w2}", Description: "second"}},
				Next:   fmt.Sprintf("http://%s/2.0/repositories/team/repo/hooks?page=3", r.Host),
			})
		default:
			_ = json.NewEncoder(w).Encode(webhooksPage{
				Values: []bitbucket.Webhook{{UUID: "{w3}", Description: "third"}},
			})
		}
	}))

	webhooks, err := client.ListWebhooks(context.Background(), "team", "repo")
	if err != nil {
		t.Fatalf("ListWebhooks() error = %v", err)
	}

	if requestCount != 3 {
		t.Errorf("requests = %d, want 3", requestCount)
	}
	if len(webhooks) != 3 {
		t.Fatalf("len(webhooks) = %d, want 3", len(webhooks))
	}
	for i, wantUUID := range []string{"{w1}", "{w2}", "{w3}"} {
		if webhooks[i].UUID != wantUUID {
			t.Errorf("webhooks[%d].UUID = %q, want %q", i, webhooks[i].UUID, wantUUID)
		}
	}
}

func TestListWebhooksSinglePage(t *testing.T) {
	requestCount := 0

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		_ = json.NewEncoder(w).Encode(webhooksPage{
			Values: []bitbucket.Webhook{{UUID: "{only}", Description: "only"}},
		})
	}))

	webhooks, err := client.ListWebhooks(context.Background(), "workspace-x", "repo.slug")
	if err != nil {
		t.Fatalf("ListWebhooks() error = %v", err)
	}

	if requestCount != 1 {
		t.Errorf("requests = %d, want 1", requestCount)
	}
	if len(webhooks) != 1 || webhooks[0].UUID != "{only}" {
		t.Errorf("webhooks = %+v, want single {only}", webhooks)
	}
}

func TestListWebhooksEmptyResult(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(webhooksPage{})
	}))

	webhooks, err := client.ListWebhooks(context.Background(), "team", "empty-repo")
	if err != nil {
		t.Fatalf("ListWebhooks() error = %v", err)
	}
	if len(webhooks) != 0 {
		t.Errorf("webhooks = %+v, want empty", webhooks)
	}
}

func TestUpdateWebhook(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotPayload map[string]any
	)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":        "{abc-123}",
			"description": "updated",
			"active":      false,
		})
	}))

	webhook, err := client.UpdateWebhook(
		context.Background(),
		"team",
		"repo",
		"{abc-123}",
		bitbucket.UpdateWebhookRequest{
			Description: "updated",
			URL:         "https://issues.example.com/api/webhooks/bitbucket",
			Active:      false,
			Events:      []string{"repo:push"},
		},
	)
	if err != nil {
		t.Fatalf("UpdateWebhook() error = %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/2.0/repositories/team/repo/hooks/{abc-123}" {
		t.Errorf("path = %q, want /2.0/repositories/team/repo/hooks/{abc-123}", gotPath)
	}
	if gotPayload["description"] != "updated" || gotPayload["active"] != false {
		t.Errorf("payload = %#v", gotPayload)
	}
	if webhook.Description != "updated" || webhook.Active {
		t.Errorf("webhook = %+v", webhook)
	}
}

func TestDeleteWebhook(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	err := client.DeleteWebhook(context.Background(), "team", "repo", "{del-uuid}")
	if err != nil {
		t.Fatalf("DeleteWebhook() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/2.0/repositories/team/repo/hooks/{del-uuid}" {
		t.Errorf("path = %q, want /2.0/repositories/team/repo/hooks/{del-uuid}", gotPath)
	}
}

func TestErrorMapping(t *testing.T) {
	t.Run("403 surfaces as client error with status code", func(t *testing.T) {
		client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error": {"message": "access denied"}}`))
		}))

		_, err := client.ListWebhooks(context.Background(), "team", "repo")
		if err == nil {
			t.Fatal("ListWebhooks() error = nil, want error")
		}
		if !restkit.IsClientError(err) {
			t.Errorf("IsClientError(err) = false, want true")
		}
		if restkit.IsServerError(err) {
			t.Errorf("IsServerError(err) = true, want false")
		}
		apiErr, ok := restkit.AsAPIError(err)
		if !ok {
			t.Fatalf("AsAPIError(err) = ok=false, want true")
		}
		if apiErr.StatusCode != http.StatusForbidden {
			t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
		}
	})

	t.Run("400 surfaces as client error with status code", func(t *testing.T) {
		client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": {"message": "invalid url"}}`))
		}))

		_, err := client.CreateWebhook(context.Background(), "team", "repo", bitbucket.CreateWebhookRequest{})
		if err == nil {
			t.Fatal("CreateWebhook() error = nil, want error")
		}
		if !restkit.IsClientError(err) {
			t.Errorf("IsClientError(err) = false, want true")
		}
		apiErr, ok := restkit.AsAPIError(err)
		if !ok || apiErr.StatusCode != http.StatusBadRequest {
			t.Fatalf("AsAPIError StatusCode = %d, ok = %v, want 400, true", apiErr.StatusCode, ok)
		}
	})

	t.Run("500 surfaces as server error", func(t *testing.T) {
		client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))

		err := client.DeleteWebhook(context.Background(), "team", "repo", "{u}")
		if err == nil {
			t.Fatal("DeleteWebhook() error = nil, want error")
		}
		if !restkit.IsServerError(err) {
			t.Errorf("IsServerError(err) = false, want true")
		}
		if restkit.IsClientError(err) {
			t.Errorf("IsClientError(err) = true, want false")
		}
	})

	t.Run("network failure maps to infrastructure error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		server.Close()

		client, err := bitbucket.NewClient(bitbucket.Config{BaseURL: server.URL, AccessToken: testToken})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		err = client.DeleteWebhook(context.Background(), "team", "repo", "{u}")
		if err == nil {
			t.Fatal("DeleteWebhook() error = nil, want error")
		}
		if !restkit.IsInfrastructureError(err) {
			t.Errorf("IsInfrastructureError(err) = false, want true")
		}
	})
}

func TestNewClientDefaultsAndValidation(t *testing.T) {
	t.Run("empty base URL falls back to Bitbucket API", func(t *testing.T) {
		client, err := bitbucket.NewClient(bitbucket.Config{AccessToken: testToken})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("client = nil, want non-nil")
		}
	})

	t.Run("invalid base URL returns error", func(t *testing.T) {
		_, err := bitbucket.NewClient(bitbucket.Config{BaseURL: "http://[::1]:named-port", AccessToken: testToken})
		if err == nil {
			t.Fatal("NewClient() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "base URL") && !strings.Contains(err.Error(), "parse") {
			t.Errorf("error = %v, want base URL parse failure", err)
		}
	})
}
