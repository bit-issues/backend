package serve_test

import (
	"testing"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/config"
	"github.com/bit-issues/backend/internal/db"
	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/oauth"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server"
	"github.com/bit-issues/backend/internal/storage"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/internal/webauthn"
	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/bit-issues/backend/pkg/miniofx"
	"github.com/go-core-fx/bunfx"
	"github.com/go-core-fx/cachefx"
	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/goosefx"
	"github.com/go-core-fx/healthfx"
	"github.com/go-core-fx/logger"
	"github.com/go-core-fx/sqlfx"
	"github.com/go-core-fx/validatorfx"
	"go.uber.org/fx"
)

func TestFxGraphValidates(t *testing.T) {
	options := []fx.Option{
		logger.Module(),
		logger.WithFxDefaultLogger(),
		bunfx.Module(),
		cachefx.Module(),
		fiberfx.Module(),
		sqlfx.Module(),
		goosefx.Module(),
		healthfx.Module(),
		validatorfx.Module(),
		miniofx.Module(),
		config.Module(),
		db.Module(),
		server.Module(),
		storage.Module(),
		fx.Supply(healthfx.Version{}),
		jwt.Module(),
		users.Module(),
		projects.Module(),
		tasks.Module(),
		attachments.Module(),
		comments.Module(),
		webauthn.Module(),
		webhooks.Module(),
		oauth.Module(),
	}

	if err := fx.ValidateApp(options...); err != nil {
		t.Fatalf("fx graph validation failed: %v", err)
	}
}
