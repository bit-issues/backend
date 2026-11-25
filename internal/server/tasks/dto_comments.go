package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
)

// CommentCreateRequest represents the request body for creating a new comment.
//
//	@Description	Comment creation request with content.
type CommentCreateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

// CommentUpdateRequest represents the request body for updating a comment.
//
//	@Description	Comment update request with content.
type CommentUpdateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

// toCommentResponse converts a domain Comment to the shared wire response.
// It enriches the author field from the pre-fetched users map.
// If the user is not found, only the ID is populated (stub).
func toCommentResponse(comment *comments.Comment, usersMap map[int64]users.User) clientgo.CommentResponse {
	return clientgo.CommentResponse{
		ID:        comment.ID,
		Author:    *dto.ResolveUserBrief(&comment.AuthorID, usersMap),
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: comment.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// toCommentsList converts a list of comments to shared wire responses.
func toCommentsList(items []comments.Comment, usersMap map[int64]users.User) []clientgo.CommentResponse {
	comments := make([]clientgo.CommentResponse, 0, len(items))
	for _, item := range items {
		comments = append(comments, toCommentResponse(&item, usersMap))
	}

	return comments
}
