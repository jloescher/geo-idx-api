// Package openapi serves the embedded OpenAPI document and Swagger UI.
// Source of truth: docs/yaak-api-collection.json (run `make openapi-sync` after editing).
package openapi

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed spec/openapi.json
var specJSON []byte

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Quantyra GeoIDX API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js" crossorigin></script>
<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js" crossorigin></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset
      ],
      layout: 'StandaloneLayout',
    });
  };
</script>
</body>
</html>`

// Register mounts public OpenAPI and Swagger UI routes (no auth).
func Register(app *fiber.App) {
	app.Get("/openapi.json", serveSpec)
	app.Get("/swagger", serveUI)
	app.Get("/swagger/", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger", fiber.StatusMovedPermanently)
	})
}

func serveSpec(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=300")
	return c.Send(specJSON)
}

func serveUI(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(swaggerUIHTML)
}
