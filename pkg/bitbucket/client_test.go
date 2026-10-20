package bitbucket_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bit-issues/backend/pkg/bitbucket"
)

// hooksPage mirrors the paginated Bitbucket hooks list response.
type hooksPage struct {
	Values []bitbucket.Hook `json:"values"`
	Next   string           `json:"next"`
}

func testHook(uuid string, active bool) bitbucket.Hook {
	return bitbucket.Hook{
		UUID:        uuid,
		Description: "BitIssues push webhook",
		URL:         hookURL,
		Active:      active,
		Secret:      "",
		Events:      []string{pushEvent},
		CreatedOn:   time.Time{},
		UpdatedOn:   time.Time{},
	}
}

func TestClientListHooks(t *testing.T) {
	var (
		authHeaders []string
		pages       int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repositories/acme/widgets/hooks" {
			t.Errorf("path = %s, want /repositories/acme/widgets/hooks", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "100" {
			t.Errorf("pagelen = %q, want 100", r.URL.Query().Get("pagelen"))
		}
		if pages == 1 {
			if r.URL.Query().Get("page") != "" {
				t.Errorf("page 1 must not carry a page param, got %q", r.URL.Query().Get("page"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(hooksPage{
				Values: []bitbucket.Hook{testHook("{1}", true), testHook("{2}", false)},
				Next:   "https://api.bitbucket.org/2.0/repositories/acme/widgets/hooks?page=2",
			})
			return
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("page param = %q, want 2", r.URL.Query().Get("page"))
		}
		_ = json.NewEncoder(w).Encode(hooksPage{
			Values: []bitbucket.Hook{testHook("{3}", true)},
		})
	}))
	defer srv.Close()

	// Trailing slash on the base URL must be normalized away.
	client := bitbucket.NewClient(srv.URL + "/")
	hooks, err := client.ListHooks(t.Context(), "acme", "widgets", "tok-123")
	if err != nil {
		t.Fatalf("ListHooks failed: %v", err)
	}
	if len(hooks) != 3 {
		t.Fatalf("got %d hooks, want 3", len(hooks))
	}
	if hooks[0].UUID != "{1}" || hooks[2].UUID != "{3}" {
		t.Fatalf("unexpected hooks: %+v", hooks)
	}
	if pages != 2 {
		t.Fatalf("server saw %d pages, want 2", pages)
	}
	if len(authHeaders) != 2 || authHeaders[0] != "Bearer tok-123" || authHeaders[1] != "Bearer tok-123" {
		t.Fatalf("unexpected authorization headers: %v", authHeaders)
	}
}

func TestClientListHooksPageCap(t *testing.T) {
	const wantPages = 10

	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(hooksPage{
			Values: []bitbucket.Hook{testHook("{x}", true)},
			Next:   "https://api.bitbucket.org/2.0/repositories/acme/widgets/hooks?page=" + r.URL.Query().Get("page"),
		})
	}))
	defer srv.Close()

	client := bitbucket.NewClient(srv.URL)
	hooks, err := client.ListHooks(t.Context(), "acme", "widgets", "tok")
	if err != nil {
		t.Fatalf("ListHooks failed: %v", err)
	}
	if requests != wantPages {
		t.Fatalf("server saw %d requests, want %d (page cap)", requests, wantPages)
	}
	if len(hooks) != wantPages {
		t.Fatalf("got %d hooks, want %d", len(hooks), wantPages)
	}
}

func TestClientCreateHook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repositories/acme/widgets/hooks" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content-type = %q", got)
		}
		var input bitbucket.HookInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Errorf("decode input failed: %v", err)
			return
		}
		if input.Description != "BitIssues push webhook" {
			t.Errorf("description = %q", input.Description)
		}
		if input.URL != hookURL {
			t.Errorf("url = %q", input.URL)
		}
		if !input.Active {
			t.Errorf("active = false, want true")
		}
		if input.Secret != "s3cret-value" {
			t.Errorf("secret = %q", input.Secret)
		}
		if len(input.Events) != 1 || input.Events[0] != pushEvent {
			t.Errorf("events = %v", input.Events)
		}

		created := testHook("{new-1}", true)
		created.UpdatedOn = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
		_ = json.NewEncoder(w).Encode(created)
	}))
	defer srv.Close()

	client := bitbucket.NewClient(srv.URL)
	hook, err := client.CreateHook(t.Context(), "acme", "widgets", "tok", bitbucket.HookInput{
		Description: "BitIssues push webhook",
		URL:         hookURL,
		Active:      true,
		Secret:      "s3cret-value",
		Events:      []string{pushEvent},
	})
	if err != nil {
		t.Fatalf("CreateHook failed: %v", err)
	}
	if hook.UUID != "{new-1}" || !hook.Active || hook.UpdatedOn.IsZero() {
		t.Fatalf("unexpected hook: %+v", hook)
	}
}

func TestClientUpdateHook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.RequestURI, "/hooks/%7Babc-123%7D") {
			t.Errorf("path = %s, want escaped uuid braces", r.RequestURI)
		}
		if strings.Contains(r.RequestURI, "{abc-123}") {
			t.Errorf("path = %s, raw braces must be escaped", r.RequestURI)
		}
		var input bitbucket.HookInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Errorf("decode input failed: %v", err)
			return
		}
		if !input.Active || input.Secret != "s3cret-value" {
			t.Errorf("unexpected input: %+v", input)
		}
		_ = json.NewEncoder(w).Encode(testHook("{abc-123}", true))
	}))
	defer srv.Close()

	client := bitbucket.NewClient(srv.URL)
	hook, err := client.UpdateHook(t.Context(), "acme", "widgets", "{abc-123}", "tok", bitbucket.HookInput{
		Description: "BitIssues push webhook",
		URL:         hookURL,
		Active:      true,
		Secret:      "s3cret-value",
		Events:      []string{pushEvent},
	})
	if err != nil {
		t.Fatalf("UpdateHook failed: %v", err)
	}
	if hook.UUID != "{abc-123}" {
		t.Fatalf("unexpected hook: %+v", hook)
	}
}

func TestClientDeleteHook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if !strings.Contains(r.RequestURI, "/hooks/%7Babc%7D") {
			t.Errorf("path = %s, want escaped uuid braces", r.RequestURI)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := bitbucket.NewClient(srv.URL)
	if err := client.DeleteHook(t.Context(), "acme", "widgets", "{abc}", "tok"); err != nil {
		t.Fatalf("DeleteHook failed: %v", err)
	}
}

func TestClientErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"unauthorized", http.StatusUnauthorized, bitbucket.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, bitbucket.ErrForbidden},
		{"not found", http.StatusNotFound, bitbucket.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"type":"error","error":{"message":"top-secret-upstream-body"}}`))
			}))
			defer srv.Close()

			client := bitbucket.NewClient(srv.URL)
			_, err := client.ListHooks(t.Context(), "acme", "widgets", "tok-secret")
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if strings.Contains(err.Error(), "top-secret-upstream-body") {
				t.Fatalf("upstream body leaked into error text: %v", err)
			}
			if strings.Contains(err.Error(), "tok-secret") {
				t.Fatalf("token leaked into error text: %v", err)
			}
		})
	}
}

func TestClientAPIErrorStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"too many requests", http.StatusTooManyRequests},
		{"upstream 5xx", http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
			}))
			defer srv.Close()

			client := bitbucket.NewClient(srv.URL)
			_, err := client.ListHooks(t.Context(), "acme", "widgets", "tok")
			var apiErr *bitbucket.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %v", err)
			}
			if apiErr.StatusCode != tt.status {
				t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
		})
	}
}

func TestClientTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	client := bitbucket.NewClient(url)
	_, err := client.ListHooks(t.Context(), "acme", "widgets", "tok-secret")
	var apiErr *bitbucket.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError for transport failure, got %v", err)
	}
	if apiErr.StatusCode != 0 {
		t.Fatalf("StatusCode = %d, want 0 for transport failure", apiErr.StatusCode)
	}
	if strings.Contains(err.Error(), "tok-secret") {
		t.Fatalf("token leaked into transport error text: %v", err)
	}
}

func TestClientDefaultBaseURL(t *testing.T) {
	if bitbucket.DefaultBaseURL != "https://api.bitbucket.org/2.0" {
		t.Fatalf("DefaultBaseURL = %q", bitbucket.DefaultBaseURL)
	}
}
