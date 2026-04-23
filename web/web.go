package web

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func Register(r fiber.Router) {
	const maxAge = 3600

	r.Get("/", func(c *fiber.Ctx) error {
		f, err := Files.Open("templates/index.html")
		if err != nil {
			return fmt.Errorf("failed to open index.html: %w", err)
		}
		defer f.Close()

		c.Type("html", "utf-8")

		return c.Status(http.StatusOK).SendStream(f)
	})

	r.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(Files),
		PathPrefix: "static",
		Browse:     false,
		MaxAge:     maxAge,
	}))
}
