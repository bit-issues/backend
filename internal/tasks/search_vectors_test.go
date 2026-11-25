package tasks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bit-issues/backend/internal/tasks"
)

type searchVector struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Accepted bool   `json:"accepted"`
}

type searchVectors struct {
	Schema             int            `json:"schema"`
	Description        string         `json:"description"`
	StrippedCharacters []string       `json:"stripped_characters"`
	Cases              []searchVector `json:"cases"`
}

func loadSearchVectors(t *testing.T) searchVectors {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "search_strip_vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}

	var vectors searchVectors
	err = json.Unmarshal(data, &vectors)
	if err != nil {
		t.Fatalf("decode vectors: %v", err)
	}
	if vectors.Schema != 1 {
		t.Fatalf("unexpected schema: %d", vectors.Schema)
	}
	if len(vectors.StrippedCharacters) == 0 {
		t.Fatal("no stripped characters in fixture")
	}
	if len(vectors.Cases) == 0 {
		t.Fatal("no cases in fixture")
	}

	return vectors
}

func TestSearchStripVectors(t *testing.T) {
	t.Parallel()

	vectors := loadSearchVectors(t)

	stripped := map[rune]bool{}
	for _, char := range vectors.StrippedCharacters {
		for _, r := range char {
			stripped[r] = true
		}
	}
	for _, r := range `+-<>()~*"@` {
		if !stripped[r] {
			t.Fatalf("fixture missing stripped rune %q", r)
		}
	}

	for _, tc := range vectors.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			got := tasks.SanitizeSearchForTest(tc.Input)
			if got != tc.Expected {
				t.Fatalf("sanitize %q: got %q want %q", tc.Input, got, tc.Expected)
			}
			if accepted := got != ""; accepted != tc.Accepted {
				t.Fatalf("accept %q: got %t want %t", tc.Input, accepted, tc.Accepted)
			}
		})
	}
}
