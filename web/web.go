package web

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func Register(r fiber.Router) {
	const maxAge = 3600

	r.Get("/", func(c *fiber.Ctx) error {
		data, err := Files.ReadFile("static/dist/index.html")
		if err != nil {
			return fmt.Errorf("failed to open index.html: %w", err)
		}

		c.Type("html", "utf-8")
		return c.Status(http.StatusOK).Send(data)
	})

	r.Use(
		"/static",
		etag.New(etag.Config{Weak: true}),
		filesystem.New(filesystem.Config{
			Root:       http.FS(Files),
			PathPrefix: "static",
			Browse:     false,
			MaxAge:     maxAge,
		}),
	)
}
