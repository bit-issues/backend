package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/server/dto"
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

// CommentResponse represents the API response for a single comment.
//
//	@Description	Full comment details with author information.
type CommentResponse struct {
	ID        int64         `json:"id"`
	Author    dto.UserBrief `json:"author"`
	Content   string        `json:"content"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// toCommentResponse converts a domain Comment to an API response.
func toCommentResponse(comment *comments.Comment) CommentResponse {
	return CommentResponse{
		ID: comment.ID,
		Author: dto.UserBrief{
			ID:        comment.AuthorID,
			Email:     "",
			Role:      "",
			CreatedAt: "",
		},
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: comment.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// toCommentsList converts a list of comments to an API response.
func toCommentsList(items []comments.Comment) []CommentResponse {
	comments := make([]CommentResponse, 0, len(items))
	for _, item := range items {
		comments = append(comments, toCommentResponse(&item))
	}

	return comments
}
