package bitbucket_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bit-issues/backend/pkg/bitbucket"
)

const (
	pushEvent = bitbucket.EventRepoPush
	hookURL   = "https://bitissues.example.com/api/v1/webhooks/bitbucket/push"
)

func TestHookJSON(t *testing.T) {
	t.Parallel()

	hook := bitbucket.Hook{
		UUID:        "{abc-123}",
		Description: "BitIssues push webhook",
		URL:         hookURL,
		Active:      true,
		Secret:      "",
		Events:      []bitbucket.Event{pushEvent},
		CreatedOn:   time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
		UpdatedOn:   time.Date(2026, 9, 29, 13, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(hook)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	body := string(data)
	for _, key := range []string{"uuid", "description", "url", "active", "events", "created_on", "updated_on"} {
		if !strings.Contains(body, `"`+key+`"`) {
			t.Fatalf("missing key %q in %s", key, body)
		}
	}
	if strings.Contains(body, "secret") {
		t.Fatalf("empty secret must be omitted from JSON: %s", body)
	}
}

func TestHookUnmarshal(t *testing.T) {
	t.Parallel()

	body := `{"uuid":"{abc}","description":"d","url":"https://x/push","active":false,"events":["repo:push"],"created_on":"2026-09-29T12:00:00+00:00","updated_on":"2026-09-29T13:00:00+00:00"}`
	var hook bitbucket.Hook
	if err := json.Unmarshal([]byte(body), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if hook.UUID != "{abc}" || hook.Active || hook.URL != "https://x/push" {
		t.Fatalf("unexpected hook: %+v", hook)
	}
	if hook.UpdatedOn.IsZero() {
		t.Fatalf("updated_on not parsed: %+v", hook)
	}
}

func TestHookInputJSON(t *testing.T) {
	t.Parallel()

	input := bitbucket.HookInput{
		Description: "BitIssues push webhook",
		URL:         hookURL,
		Active:      true,
		Secret:      "s3cret-value",
		Events:      []bitbucket.Event{pushEvent},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	body := string(data)
	for _, want := range []string{`"description":"BitIssues push webhook"`, `"active":true`, `"secret":"s3cret-value"`, `"events":["repo:push"]`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
}
