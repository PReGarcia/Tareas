package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"taskmanager/internal/api"
	"taskmanager/internal/db"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed all:web/dist
var distFS embed.FS

func main() {
	if err := db.Init(); err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             2 * 1024 * 1024, // 2 MB es más que suficiente.
	})

	api.Register(app)

	// Evita que el navegador cachee index.html/app.js entre actualizaciones.
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.Next()
	})

	// Sirve el frontend embebido (HTMX + Alpine + Tailwind build).
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatalf("embed sub: %v", err)
	}
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(sub),
		Index:        "index.html",
		NotFoundFile: "index.html", // SPA fallback.
	}))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("task-manager escuchando en %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
