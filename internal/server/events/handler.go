package events

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bit-issues/backend/internal/events"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	heartbeatComment = ": ping\n\n"

	sseNoBufferHeader = "X-Accel-Buffering"
)

// Handler serves the authenticated Server-Sent Events stream.
type Handler struct {
	hub       *events.Hub
	heartbeat time.Duration

	logger *zap.Logger
}

// NewHandler creates a new Handler. A missing or non-positive configured
// heartbeat falls back to [events.DefaultHeartbeatInterval] because
// [time.NewTicker] panics on non-positive durations.
func NewHandler(
	hub *events.Hub,
	cfg events.Config,
	logger *zap.Logger,
) handler.Handler {
	heartbeat := cfg.HeartbeatInterval
	if heartbeat <= 0 {
		heartbeat = events.DefaultHeartbeatInterval
	}

	return &Handler{
		hub:       hub,
		heartbeat: heartbeat,

		logger: logger,
	}
}

// Register sets up the events routes on the given router.
func (h *Handler) Register(r fiber.Router) {
	r.Get("/events/stream", h.stream)
}

//	@Summary		Open the live events stream
//	@Description	Server-Sent Events stream of data changes. Each event carries
//	@Description	`event: <type>` and a JSON `data` payload with resource
//	@Description	identifiers; clients refetch the affected resource instead of
//	@Description	applying a partial payload. A `: ping` comment is sent every
//	@Description	heartbeat interval (EVENTS__HEARTBEAT_INTERVAL, default 15s)
//	@Description	to detect dead connections.
//	@Tags			Events
//	@Produce		text/event-stream
//	@Security		BearerAuth
//	@Success		200	{string}	string	"event stream (text/event-stream)"
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Router			/events/stream [get]
//
// stream subscribes the caller to the hub and streams events until the
// connection closes.
func (h *Handler) stream(c *fiber.Ctx) error {
	sub, cancel := h.hub.Subscribe()

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(sseNoBufferHeader, "no")

	done := c.Context().Done()

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer cancel()
		h.runEventStream(w, sub, done)
	})

	return nil
}

func (h *Handler) runEventStream(w *bufio.Writer, sub <-chan events.Event, done <-chan struct{}) {
	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			if err := writeEvent(w, ev); err != nil {
				h.logger.Debug("event stream write failed, closing", zap.Error(err))
				return
			}
		case <-ticker.C:
			if _, err := w.WriteString(heartbeatComment); err != nil {
				h.logger.Debug("event stream heartbeat write failed, closing", zap.Error(err))
				return
			}
			if err := w.Flush(); err != nil {
				h.logger.Debug("event stream heartbeat flush failed, closing", zap.Error(err))
				return
			}
		}
	}
}

func writeEvent(w *bufio.Writer, ev events.Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if _, err = w.WriteString("event: " + ev.Type + "\n"); err != nil {
		return fmt.Errorf("write event type: %w", err)
	}
	if _, err = w.Write(payload); err != nil {
		return fmt.Errorf("write event data: %w", err)
	}
	if _, err = w.WriteString("\n\n"); err != nil {
		return fmt.Errorf("write event terminator: %w", err)
	}
	if err = w.Flush(); err != nil {
		return fmt.Errorf("flush event: %w", err)
	}

	return nil
}
