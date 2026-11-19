package dto_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/server/projects"
	"github.com/bit-issues/backend/internal/server/tasks"
	"github.com/go-playground/validator/v10"
)

func newValidator(t *testing.T) *validator.Validate {
	t.Helper()

	v := validator.New()
	if err := dto.RegisterValidations(v); err != nil {
		t.Fatalf("RegisterValidations: %v", err)
	}

	return v
}

func assertInvalid(t *testing.T, err error, field, tag string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected validation error on %s", field)
	}

	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		t.Fatalf("expected validator errors, got %T: %v", err, err)
	}

	for _, e := range errs {
		if e.Field() == field && e.Tag() == tag {
			return
		}
	}

	t.Fatalf("expected %s to fail on %s, got %v", field, tag, errs)
}

func TestMaxBytes(t *testing.T) {
	t.Parallel()

	type payload struct {
		Text string `validate:"maxbytes=65535"`
	}

	tests := []struct {
		name  string
		text  string
		valid bool
	}{
		{name: "ascii at limit", text: strings.Repeat("a", dto.MaxTextBytes), valid: true},
		{name: "ascii over limit", text: strings.Repeat("a", dto.MaxTextBytes+1), valid: false},
		{name: "multibyte at limit", text: strings.Repeat("🙂", dto.MaxTextBytes/4), valid: true},
		{name: "multibyte over limit", text: strings.Repeat("🙂", dto.MaxTextBytes/4+1), valid: false},
		{name: "empty", text: "", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := newValidator(t).Struct(payload{Text: tt.text})
			if tt.valid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestMaxBytesPointer(t *testing.T) {
	t.Parallel()

	type payload struct {
		Text *string `validate:"omitempty,maxbytes=65535"`
	}

	v := newValidator(t)
	long := strings.Repeat("a", dto.MaxTextBytes+1)

	if err := v.Struct(payload{}); err != nil {
		t.Fatalf("nil pointer: %v", err)
	}

	assertInvalid(t, v.Struct(payload{Text: &long}), "Text", "maxbytes")
}

func TestDTOTags(t *testing.T) {
	t.Parallel()

	v := newValidator(t)
	atLimit := strings.Repeat("a", dto.MaxTextBytes)
	overLimit := strings.Repeat("a", dto.MaxTextBytes+1)

	if err := v.Struct(tasks.TaskCreateRequest{ProjectSlug: "p", Title: "t", Description: atLimit}); err != nil {
		t.Fatalf("create description at limit: %v", err)
	}
	assertInvalid(t, v.Struct(tasks.TaskCreateRequest{
		ProjectSlug: "p", Title: "t", Description: overLimit,
	}), "Description", "maxbytes")

	assertInvalid(t, v.Struct(tasks.TaskUpdateRequest{Description: &overLimit}), "Description", "maxbytes")
	assertInvalid(t, v.Struct(tasks.TaskUpdateRequest{Comment: &overLimit}), "Comment", "maxbytes")
	assertInvalid(t, v.Struct(tasks.CommentCreateRequest{Content: overLimit}), "Content", "maxbytes")
	assertInvalid(t, v.Struct(tasks.CommentUpdateRequest{Content: overLimit}), "Content", "maxbytes")

	repoURL := "https://example.com/" + strings.Repeat("a", 493)
	if err := v.Struct(projects.ProjectRequest{
		Name: strings.Repeat("a", 255), RepoURL: repoURL[:512],
	}); err != nil {
		t.Fatalf("project fields at limit: %v", err)
	}
	assertInvalid(t, v.Struct(projects.ProjectRequest{
		Name: strings.Repeat("a", 256), RepoURL: "https://example.com",
	}), "Name", "max")
	assertInvalid(t, v.Struct(projects.ProjectRequest{
		Name: "Backend Service", RepoURL: repoURL,
	}), "RepoURL", "max")
}
