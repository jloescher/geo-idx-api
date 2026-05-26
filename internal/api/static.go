package api

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/quantyralabs/idx-api/internal/web"
)

func mountStatic(app *fiber.App) error {
	root, err := web.StaticFS()
	if err != nil {
		return err
	}
	app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(root),
	}))
	return nil
}
