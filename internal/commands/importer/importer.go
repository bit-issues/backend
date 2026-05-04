package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/pkg/bitbucket"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type importer struct {
	config      Config
	tasksSvc    *tasks.Service
	commentsSvc *comments.Service
	usersSvc    *users.Service
	logger      *zap.Logger

	sh fx.Shutdowner
}

func newImporter(
	config Config,
	tasksSvc *tasks.Service,
	commentsSvc *comments.Service,
	usersSvc *users.Service,
	logger *zap.Logger,
	sh fx.Shutdowner,
) *importer {
	return &importer{
		config:      config,
		tasksSvc:    tasksSvc,
		commentsSvc: commentsSvc,
		usersSvc:    usersSvc,
		logger:      logger,

		sh: sh,
	}
}

func (i *importer) Run(ctx context.Context) error {
	var result ImportResult

	logger := i.logger.With(zap.String("project", i.config.ProjectSlug), zap.Bool("dryRun", i.config.DryRun))

	// Parse the JSON file
	export, err := i.parseExportFile(i.config.Filename)
	if err != nil {
		return fmt.Errorf("failed to parse export file: %w", err)
	}

	logger.Info("Parsed export file", zap.Int("issues", len(export.Issues)), zap.Int("comments", len(export.Comments)))

	if i.config.DryRun {
		logger.Info("DRY RUN MODE - no changes will be made")
	}

	// Resolve default user
	defaultUser, err := i.resolveUser(ctx, i.usersSvc, i.config.DefaultUser)
	if err != nil {
		return fmt.Errorf("failed to resolve default user: %w", err)
	}

	logger.Info("Starting import")

	// Build a map of issue ID to imported task for comments
	issueToTask := make(map[int]int64) // BitBucket issue ID -> internal task ID

	// Import issues
	for _, issue := range export.Issues {
		if i.config.DryRun {
			logger.Info("Would import issue", zap.Int("issueID", issue.ID))
			result.IssuesImported++
			continue
		}

		task, importErr := i.importIssue(ctx, i.tasksSvc, i.config.ProjectSlug, defaultUser, &issue)
		if importErr != nil {
			result.IssuesSkipped++
			logger.Warn("Failed to import issue", zap.Int("issueID", issue.ID), zap.Error(importErr))
			continue
		}

		issueToTask[issue.ID] = task.ID
		result.IssuesImported++
		logger.Info("Imported issue", zap.Int("issueID", issue.ID), zap.Int64("taskID", task.ID))
	}

	// Import comments
	for _, comment := range export.Comments {
		if i.config.DryRun {
			logger.Info("Would import comment", zap.Int("commentID", comment.ID))
			result.CommentsImported++
			continue
		}

		taskID, ok := issueToTask[comment.Issue]
		if !ok {
			result.CommentsSkipped++
			logger.Warn(
				"Could not find task for comment",
				zap.Int("commentID", comment.ID),
				zap.Int("issueID", comment.Issue),
			)
			continue
		}

		if importErr := i.importComment(ctx, i.commentsSvc, taskID, defaultUser, &comment); importErr != nil {
			result.CommentsSkipped++
			logger.Warn("Failed to import comment", zap.Int("commentID", comment.ID), zap.Error(importErr))
			continue
		}

		result.CommentsImported++
		logger.Info("Imported comment", zap.Int("commentID", comment.ID), zap.Int64("taskID", taskID))
	}

	// Print results
	logger.Info("Import complete",
		zap.Int("issuesImported", result.IssuesImported),
		zap.Int("issuesSkipped", result.IssuesSkipped),
		zap.Int("commentsImported", result.CommentsImported),
		zap.Int("commentsSkipped", result.CommentsSkipped),
	)

	if shErr := i.sh.Shutdown(); shErr != nil {
		return fmt.Errorf("failed to shutdown importer: %w", shErr)
	}

	return nil
}

// importIssue imports a single issue.
func (i *importer) importIssue(
	ctx context.Context,
	tasksSvc *tasks.Service,
	projectSlug string,
	author *users.User,
	issue *bitbucket.Issue,
) (*tasks.Task, error) {
	// Normalize priority (BitBucket uses lowercase)
	priority := i.normalizePriority(issue.Priority)

	// Normalize status (BitBucket uses lowercase)
	status := i.normalizeStatus(issue.Status)

	// Normalize kind
	kind := i.normalizeKind(issue.Kind)

	// Create task domain object
	task := &tasks.Task{
		ID:          0,
		ProjectSlug: projectSlug,
		Number:      issue.ID, // BitBucket issue ID becomes the task number
		Title:       issue.Title,
		Description: issue.Content,
		Priority:    priority,
		Status:      status,
		Kind:        kind,
		AuthorID:    author.ID,
		AssigneeID:  nil, // Always clear assignee per requirements
		DueDate:     nil,
		CreatedAt:   issue.CreatedOn,
		UpdatedAt:   issue.UpdatedOn,
		DeletedAt:   nil,
	}

	// Import task
	var err error
	if task, err = tasksSvc.Import(ctx, *task); err != nil {
		return nil, fmt.Errorf("failed to import task: %w", err)
	}

	return task, nil
}

// importComment imports a single comment.
func (i *importer) importComment(
	ctx context.Context,
	commentsSvc *comments.Service,
	taskID int64,
	author *users.User,
	comment *bitbucket.Comment,
) error {
	updatedOn := time.Time{}
	if comment.UpdatedOn != nil {
		updatedOn = *comment.UpdatedOn
	}

	// Create comment domain object
	commentDomain := comments.Comment{
		ID:        0,
		TaskID:    taskID,
		AuthorID:  author.ID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedOn,
		UpdatedAt: updatedOn,
		DeletedAt: nil,
	}

	// Import comment
	if _, err := commentsSvc.Import(ctx, commentDomain); err != nil {
		return fmt.Errorf("failed to import comment: %w", err)
	}

	return nil
}

// parseExportFile reads and parses the BitBucket export JSON file.
func (i *importer) parseExportFile(filePath string) (*bitbucket.Export, error) {
	var export bitbucket.Export

	h, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer h.Close()

	if jsonErr := json.NewDecoder(h).Decode(&export); jsonErr != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", jsonErr)
	}

	return &export, nil
}

// resolveUser finds a user by numeric ID.
func (i *importer) resolveUser(ctx context.Context, usersSvc *users.Service, userStr string) (*users.User, error) {
	userID, err := strconv.Atoi(userStr)
	if err != nil {
		return nil, fmt.Errorf("default user must be a numeric user ID: %w", err)
	}

	user, err := usersSvc.GetByID(ctx, int64(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// normalizePriority converts BitBucket priority to internal format.
func (i *importer) normalizePriority(priority string) tasks.Priority {
	switch strings.ToLower(priority) {
	case "trivial":
		return tasks.PriorityTrivial
	case "minor":
		return tasks.PriorityMinor
	case "major":
		return tasks.PriorityMajor
	case "critical":
		return tasks.PriorityCritical
	case "blocker":
		return tasks.PriorityBlocker
	default:
		return tasks.PriorityMinor // default
	}
}

// normalizeStatus converts BitBucket status to internal format.
func (i *importer) normalizeStatus(status string) tasks.Status {
	switch strings.ToLower(status) {
	case "new":
		return tasks.StatusNew
	case "open":
		return tasks.StatusOpen
	case "in progress":
		return tasks.StatusInProgress
	case "resolved":
		return tasks.StatusResolved
	case "closed":
		return tasks.StatusClosed
	case "reopened":
		return tasks.StatusReopened
	default:
		return tasks.StatusNew // default
	}
}

func (i *importer) normalizeKind(kind string) tasks.Kind {
	switch strings.ToLower(kind) {
	case "bug":
		return tasks.KindBug
	case "enhancement":
		return tasks.KindEnhancement
	case "task":
		return tasks.KindTask
	case "proposal":
		return tasks.KindProposal
	default:
		return tasks.KindTask // default
	}
}
