package server

import (
	"github.com/bit-issues/backend/internal/server/auth"
	"github.com/bit-issues/backend/internal/server/docs"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/server/projects"
	"github.com/bit-issues/backend/internal/server/tasks"
	"github.com/bit-issues/backend/internal/server/users"
	"github.com/bit-issues/backend/internal/server/webhooks"
	"github.com/bit-issues/backend/web"
	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-core-fx/fiberfx/health"
	"github.com/go-core-fx/fiberfx/openapi"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-core-fx/logger"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Module() fx.Option {
	return fx.Module(
		"server",
		logger.WithNamedLogger("server"),

		fx.Provide(func(log *zap.Logger) fiberfx.Options {
			opts := fiberfx.Options{}
			opts.WithErrorHandler(fiberfx.NewJSONErrorHandler(log))
			opts.WithMetrics()
			return opts
		}),
		fx.Supply(docs.SwaggerInfo),

		fx.Provide(
			fx.Annotate(jwtauth.New, fx.ResultTags(`name:"jwtauth"`)),
			fx.Private,
		),

		fx.Provide(
			fx.Annotate(users.NewHandler, fx.ResultTags(`group:"handlers"`)),
			fx.Annotate(auth.NewHandler, fx.ResultTags(`group:"handlers"`)),
			fx.Annotate(projects.NewHandler, fx.ResultTags(`group:"handlers"`)),
			fx.Annotate(tasks.NewHandler, fx.ResultTags(`group:"handlers"`)),
			fx.Private,
		),

		fx.Provide(
			health.NewHandler,
			openapi.NewHandler,
			fx.Private,
		),

		fx.Provide(
			webhooks.NewHandler,
			fx.Private,
		),

		fx.Invoke(
			fx.Annotate(
				func(handlers []handler.Handler, jwtAuth fiber.Handler, webhookHandler *webhooks.Handler, healthHandler *health.Handler, openapiHandler *openapi.Handler, app *fiber.App) {
					// Health endpoint
					healthHandler.Register(app)

					// Frontend
					web.Register(app)

					// Version 1 API group
					v1 := app.Group("/api/v1")
					openapiHandler.Register(v1.Group("/docs"))

					v1.Use(validation.Middleware)
					webhookHandler.Register(v1)

					v1.Use(
						jwtAuth,
						jwtauth.ErrorsHandler(),
					)

					for _, h := range handlers {
						h.Register(v1)
					}
				},
				fx.ParamTags(`group:"handlers"`, `name:"jwtauth"`),
			),
		),
	)
}
