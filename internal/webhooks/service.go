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
	"github.com/bit-issues/backend/internal/commitref"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"go.uber.org/zap"
)

type Service struct {
	config    Config
	botUserID int64

	projectsSvc  *projects.Service
	tasksSvc     *tasks.Service
	commentsSvc  *comments.Service
	usersSvc     *users.Service
	commitrefSvc *commitref.Service

	logger *zap.Logger
}

func NewService(
	cfg Config,
	commitrefSvc *commitref.Service,
	projectsSvc *projects.Service,
	tasksSvc *tasks.Service,
	commentsSvc *comments.Service,
	usersSvc *users.Service,
	logger *zap.Logger,
) (*Service, error) {
	return &Service{
		config:    cfg,
		botUserID: -1,

		projectsSvc:  projectsSvc,
		tasksSvc:     tasksSvc,
		commentsSvc:  commentsSvc,
		usersSvc:     usersSvc,
		commitrefSvc: commitrefSvc,

		logger: logger,
	}, nil
}

func (s *Service) Init(ctx context.Context) error {
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
	refs := s.commitrefSvc.ParseCommitMessage(commit.Message)
	if len(refs) == 0 {
		return
	}

	s.logger.Debug("scanning commit",
		zap.String("hash", shortHash(commit.Hash)),
		zap.String("message", commit.Message),
		zap.Int("refs", len(refs)),
	)

	resolved := make([]resolvedRef, 0, len(refs))
	for _, ref := range refs {
		ref.CommitHash = commit.Hash
		ref.CommitMessage = commit.Message

		projectID := project.ID
		if ref.ProjectSlug != "" {
			target, err := s.projectsSvc.GetBySlug(ctx, normalizeSlug(ref.ProjectSlug))
			if err != nil {
				if errors.Is(err, projects.ErrNotFound) {
					s.logger.Info("project not found for reference",
						zap.String("project_slug", ref.ProjectSlug),
						zap.Int("number", ref.TaskNumber),
					)
				} else {
					s.logger.Error("failed to resolve project for reference",
						zap.String("project_slug", ref.ProjectSlug),
						zap.Int("number", ref.TaskNumber),
						zap.Error(err),
					)
				}
				continue
			}

			projectID = target.ID
		}

		resolved = append(resolved, resolvedRef{projectID: projectID, ref: ref})
	}

	for _, r := range dedupResolvedRefs(resolved) {
		task, err := s.tasksSvc.GetByProjectAndNumber(ctx, r.projectID, r.ref.TaskNumber)
		if err != nil {
			s.logger.Info("task not found for reference",
				zap.String("project_slug", r.projectID),
				zap.Int("number", r.ref.TaskNumber),
				zap.Error(err),
			)
			continue
		}

		s.logger.Debug("found task for reference",
			zap.Int64("task_id", task.ID),
			zap.Int("number", r.ref.TaskNumber),
			zap.String("current_status", string(task.Status)),
		)

		if r.ref.Action != nil {
			s.applyStatusTransition(ctx, task, r.ref, result)
		} else {
			s.addMentionComment(ctx, task, r.ref, result)
		}
	}
}

// normalizeSlug lowercases a project slug before lookup. projects.GetBySlug is
// an exact match on the slug primary key, so case-insensitive slug matching is
// the webhooks layer's responsibility.
func normalizeSlug(slug string) string {
	return strings.ToLower(slug)
}

// resolvedRef pairs a parsed reference with the project it resolves to. For a
// ref without a slug prefix the project is the repository-matched one; for a
// prefixed ref it is the project returned by the slug lookup.
type resolvedRef struct {
	ref       commitref.ParsedReference
	projectID string
}

// resolvedRefKey identifies the task a resolved reference points at.
type resolvedRefKey struct {
	projectID string
	number    int
}

// dedupResolvedRefs collapses references that resolve to the same
// (project, task number) pair, keeping the first reference in slice order.
// The parser already dedups raw (slug, number) pairs; this is the final pass
// over resolved targets, so it also collapses a prefixed ref with a bare one
// when both point at the same project (e.g. "myproj#13" and "#13" while
// processing myproj).
//
// Slice order is reference-kind order (keyword, then prefixed, then bare), not
// message position, so a keyword reference is retained over a plain one for
// the same resolved task even when it appears later in the commit message
// (e.g. "resto-web#13; fixes #13"). This mirrors the parser's documented
// keyword-wins precedence from the original pre-commitref keyword parser and
// is deliberate: an explicit action keyword outranks a bare mention regardless
// of position.
func dedupResolvedRefs(refs []resolvedRef) []resolvedRef {
	seen := make(map[resolvedRefKey]struct{}, len(refs))
	out := make([]resolvedRef, 0, len(refs))

	for _, r := range refs {
		key := resolvedRefKey{projectID: r.projectID, number: r.ref.TaskNumber}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}

	return out
}

func (s *Service) applyStatusTransition(
	ctx context.Context,
	task *tasks.Task,
	ref commitref.ParsedReference,
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

func (s *Service) addMentionComment(
	ctx context.Context, task *tasks.Task, ref commitref.ParsedReference, result *ProcessResult,
) {
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
