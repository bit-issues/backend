package oauth_test

import (
	"testing"

	"github.com/bit-issues/backend/internal/oauth"
)

func TestGenerateStateUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		s, err := oauth.GenerateState()
		if err != nil {
			t.Fatalf("GenerateState: %v", err)
		}
		if len(s) != 64 { // 32 bytes -> 64 hex chars
			t.Fatalf("unexpected state length: %d", len(s))
		}
		if _, ok := seen[s]; ok {
			t.Fatal("duplicate state generated")
		}
		seen[s] = struct{}{}
	}
}
