package comments_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bit-issues/backend/internal/comments"
)

func TestCommentInputValidateContentLength(t *testing.T) {
	t.Parallel()

	atLimit := strings.Repeat("a", comments.MaxContentLength)
	overLimit := strings.Repeat("a", comments.MaxContentLength+1)

	if err := (comments.CommentInput{TaskID: 1, AuthorID: 1, Content: atLimit}).Validate(); err != nil {
		t.Fatalf("content at limit: %v", err)
	}
	if err := (comments.CommentInput{TaskID: 1, AuthorID: 1, Content: overLimit}).Validate(); !errors.Is(
		err,
		comments.ErrValidationFailed,
	) {
		t.Fatalf("expected validation error for over-limit content, got %v", err)
	}
}

func TestCommentUpdateValidateContentLength(t *testing.T) {
	t.Parallel()

	atLimit := strings.Repeat("a", comments.MaxContentLength)
	overLimit := strings.Repeat("a", comments.MaxContentLength+1)

	if err := (comments.CommentUpdate{Content: atLimit}).Validate(); err != nil {
		t.Fatalf("content at limit: %v", err)
	}
	if err := (comments.CommentUpdate{Content: overLimit}).Validate(); !errors.Is(err, comments.ErrValidationFailed) {
		t.Fatalf("expected validation error for over-limit content, got %v", err)
	}
}
