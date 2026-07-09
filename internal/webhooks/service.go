package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"go.uber.org/zap"
)

type Service struct {
	config Config

	projectsSvc *projects.Service
	tasksSvc    *tasks.Service
	commentsSvc *comments.Service
	usersSvc    *users.Service
	logger      *zap.Logger

	keywordParser *KeywordParser
	botUserID     int64
}

func NewService(
	cfg Config,
	projectsSvc *projects.Service,
	tasksSvc *tasks.Service,
	commentsSvc *comments.Service,
	usersSvc *users.Service,
	logger *zap.Logger,
) (*Service, error) {
	return &Service{
		config: cfg,

		projectsSvc: projectsSvc,
		tasksSvc:    tasksSvc,
		commentsSvc: commentsSvc,
		usersSvc:    usersSvc,
		logger:      logger,

		keywordParser: nil,
		botUserID:     -1,
	}, nil
}

func (s *Service) Init(ctx context.Context) error {
	parser, err := NewKeywordParser(s.config.ActionKeywords)
	if err != nil {
		return fmt.Errorf("failed to build keyword parser: %w", err)
	}
	s.keywordParser = parser

	botUser, err := s.usersSvc.GetByEmail(ctx, s.config.BotUserEmail)
	if err != nil {
		return fmt.Errorf("failed to resolve bot user %q: %w", s.config.BotUserEmail, err)
	}

	s.logger.Info("webhook service initialized",
		zap.String("bot_user_email", s.config.BotUserEmail),
		zap.Int64("bot_user_id", botUser.ID),
	)

	s.botUserID = botUser.ID
	return nil
}

func (s *Service) VerifyPushEvent(rawBody []byte, signatureHeader string) error {
	if s.config.Secret == "" {
		return fmt.Errorf("%w: secret not configured", ErrInvalidSignature)
	}

	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return fmt.Errorf("%w: invalid signature format", ErrInvalidSignature)
	}

	expectedSig, err := hex.DecodeString(signatureHeader[7:])
	if err != nil {
		return fmt.Errorf("%w: invalid signature hex: %w", ErrInvalidSignature, err)
	}

	mac := hmac.New(sha256.New, []byte(s.config.Secret))
	mac.Write(rawBody)
	computedSig := mac.Sum(nil)

	if !hmac.Equal(computedSig, expectedSig) {
		return fmt.Errorf("%w: signature mismatch", ErrInvalidSignature)
	}

	return nil
}

func (s *Service) ProcessPushEvent(
	ctx context.Context, repoFullName string, commits []PushCommit,
) (*ProcessResult, error) {
	result := NewProcessResult()

	if repoFullName == "" {
		s.logger.Warn("webhook push event with empty repository full_name")
		return result, nil
	}

	repoURL := "https://bitbucket.org/" + repoFullName + "/"

	project, err := s.projectsSvc.FindByRepoURL(ctx, repoURL)
	if err != nil {
		if errors.Is(err, projects.ErrNotFound) {
			s.logger.Info("no project found for repository",
				zap.String("repo", repoFullName),
				zap.Error(err),
			)
			return result, nil
		}
		s.logger.Error("failed to look up project for repository",
			zap.String("repo", repoFullName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("project lookup failed: %w", err)
	}

	s.logger.Info("matched push event to project",
		zap.String("project_slug", project.ID),
		zap.String("repo", repoFullName),
	)

	for _, commit := range commits {
		s.processCommit(ctx, project, commit, result)
	}

	s.logger.Info("webhook push processed",
		zap.String("project_slug", project.ID),
		zap.Int("matched", result.Matched),
		zap.Int("resolved", result.Resolved),
		zap.Int("mentioned", result.Mentioned),
	)

	return result, nil
}

func (s *Service) processCommit(
	ctx context.Context,
	project *projects.Project,
	commit PushCommit,
	result *ProcessResult,
) {
	refs := s.keywordParser.ParseCommitMessage(commit.Message)
	if len(refs) == 0 {
		return
	}

	s.logger.Debug("scanning commit",
		zap.String("hash", shortHash(commit.Hash)),
		zap.String("message", commit.Message),
		zap.Int("refs", len(refs)),
	)

	for _, ref := range refs {
		ref.CommitHash = commit.Hash
		ref.CommitMessage = commit.Message

		task, err := s.tasksSvc.GetByProjectAndNumber(ctx, project.ID, ref.TaskNumber)
		if err != nil {
			s.logger.Info("task not found for reference",
				zap.String("project_slug", project.ID),
				zap.Int("number", ref.TaskNumber),
				zap.Error(err),
			)
			continue
		}

		s.logger.Debug("found task for reference",
			zap.Int64("task_id", task.ID),
			zap.Int("number", ref.TaskNumber),
			zap.String("current_status", string(task.Status)),
		)

		if ref.Action != nil {
			s.applyStatusTransition(ctx, task, ref, result)
		} else {
			s.addMentionComment(ctx, task, ref, result)
		}
	}
}

func (s *Service) applyStatusTransition(
	ctx context.Context,
	task *tasks.Task,
	ref ParsedReference,
	result *ProcessResult,
) {
	if task.Status == ref.Action.Status {
		s.logger.Info("task already in target status, skipping",
			zap.Int64("task_id", task.ID),
			zap.Int("number", ref.TaskNumber),
			zap.String("status", string(ref.Action.Status)),
		)
		result.AddMatched()
		return
	}

	update := tasks.TaskUpdate{
		Title:       nil,
		Description: nil,
		Priority:    nil,
		Status:      &ref.Action.Status,
		Kind:        nil,
		AssigneeID:  nil,
		DueDate:     nil,
	}
	if _, err := s.tasksSvc.Update(ctx, task.ID, update); err != nil {
		s.logger.Error("failed to update task status",
			zap.Int64("task_id", task.ID),
			zap.Int("number", ref.TaskNumber),
			zap.String("target_status", string(ref.Action.Status)),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("task status updated",
		zap.Int64("task_id", task.ID),
		zap.Int("number", ref.TaskNumber),
		zap.String("new_status", string(ref.Action.Status)),
	)

	commentContent := fmt.Sprintf("%s by commit <<cset %s>>: %s",
		ref.Action.Verb, ref.CommitHash, firstLine(ref.CommitMessage),
	)
	if _, err := s.commentsSvc.Create(ctx, comments.CommentInput{
		TaskID:   task.ID,
		AuthorID: s.botUserID,
		Content:  commentContent,
	}); err != nil {
		s.logger.Error("failed to add status comment",
			zap.Int64("task_id", task.ID),
			zap.Int("number", ref.TaskNumber),
			zap.Error(err),
		)
		return
	}

	result.AddResolved()
}

func (s *Service) addMentionComment(ctx context.Context, task *tasks.Task, ref ParsedReference, result *ProcessResult) {
	commentContent := fmt.Sprintf("Mentioned in commit <<cset %s>>: %s",
		ref.CommitHash, firstLine(ref.CommitMessage),
	)
	if _, err := s.commentsSvc.Create(ctx, comments.CommentInput{
		TaskID:   task.ID,
		AuthorID: s.botUserID,
		Content:  commentContent,
	}); err != nil {
		s.logger.Error("failed to add mention comment",
			zap.Int64("task_id", task.ID),
			zap.Int("number", ref.TaskNumber),
			zap.Error(err),
		)
		return
	}

	result.AddMentioned()
}

const shortHashLen = 7

func shortHash(hash string) string {
	if len(hash) > shortHashLen {
		return hash[:shortHashLen]
	}
	return hash
}
