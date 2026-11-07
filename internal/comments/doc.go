// Package comments provides functionality for managing comments on tasks.
//
// The comments module supports:
//   - Creating comments with Markdown content
//   - Editing comments (author or admin only)
//   - Soft deleting comments (author or admin only)
//   - Retrieving comments by task
//
// # Usage
//
// Create a new comment:
//
//	svc := comments.NewService(repo, usersSvc, hub, logger)
//	input := comments.CommentInput{
//	    TaskID:   123,
//	    AuthorID: 456,
//	    Content:  "This is a comment",
//	}
//	comment, err := svc.Create(ctx, input)
//
// List comments for a task:
//
//	comments, total, err := svc.List(
//	    ctx,
//	    comments.CommentFilter{TaskID: &taskID, AuthorID: nil},
//	    comments.SortOrderCreatedAtAsc,
//	    comments.NewPagination(20, 0),
//	)
//
// Update a comment:
//
//	update := comments.CommentUpdate{Content: "Updated content"}
//	err := svc.Update(ctx, userID, commentID, update)
//
// Delete a comment (soft delete):
//
//	err := svc.Delete(ctx, userID, commentID)
package comments
