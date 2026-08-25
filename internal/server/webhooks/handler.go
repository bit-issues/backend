package webhooks

import (
	"errors"
	"strings"

	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	svc    *webhooks.Service
	logger *zap.Logger
}

func NewHandler(
	svc *webhooks.Service,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

func (h *Handler) Register(r fiber.Router) {
	r.Post("/webhooks/bitbucket/push", h.handlePush)
}

//	@Summary		BitBucket push webhook
//	@Description	Receives BitBucket push event webhooks. Scans commit messages
//	@Description	for task references (e.g. "fixes #42") and transitions task status
//	@Description	or adds mention comments accordingly.
//	@Tags			Webhooks
//	@Accept			json
//	@Produce		json
//	@Param			body			body		PushEvent	true	"BitBucket push event payload"
//	@Param			X-Hub-Signature	header		string		true	"BitBucket signature header"
//	@Success		202				{object}	webhooks.ProcessResult
//	@Failure		400				{object}	fiberfx.ErrorResponse
//	@Failure		401				{object}	fiberfx.ErrorResponse
//	@Router			/webhooks/bitbucket/push [post]
//
// handlePush processes a BitBucket push event webhook.
func (h *Handler) handlePush(c *fiber.Ctx) error {
	rawBody := c.BodyRaw()

	sigHeader := c.Get("X-Hub-Signature")
	if sigHeader == "" {
		sigHeader = c.Get("X-Hub-Signature-256")
	}

	if err := h.svc.VerifyPushEvent(rawBody, sigHeader); err != nil {
		if errors.Is(err, webhooks.ErrInvalidSignature) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid signature")
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var event PushEvent
	if err := c.BodyParser(&event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON payload")
	}

	commits := toPushCommits(event)
	repoFullName := strings.TrimSpace(event.Repository.FullName)

	h.logger.Info("received push webhook",
		zap.String("repo", repoFullName),
		zap.Int("commits", len(commits)),
	)

	result, err := h.svc.ProcessPushEvent(c.Context(), repoFullName, commits)
	if err != nil {
		h.logger.Error("failed to process push event",
			zap.String("repo", repoFullName),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to process webhook")
	}

	return c.Status(fiber.StatusAccepted).JSON(result)
}
