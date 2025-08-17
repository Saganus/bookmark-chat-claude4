package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	api "bookmark-chat/api/generated"
	"bookmark-chat/internal/handlers"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Create handler instance
	handler := handlers.NewHandler()

	// Register all generated handlers
	api.RegisterHandlers(e, handler)

	// Serve static frontend files
	e.Static("/", "frontend")

	// Serve index.html for the root path and any unmatched paths (SPA routing)
	e.GET("/", func(c echo.Context) error {
		return c.File("frontend/index.html")
	})

	log.Println("Server starting on :8080")
	log.Println("Frontend available at: http://localhost:8080")
	log.Println("Available endpoints:")
	log.Println("  GET    /api/bookmarks")
	log.Println("  POST   /api/bookmarks/import")
	log.Println("  GET    /api/bookmarks/{id}")
	log.Println("  PUT    /api/bookmarks/{id}")
	log.Println("  DELETE /api/bookmarks/{id}")
	log.Println("  POST   /api/bookmarks/{id}/rescrape")
	log.Println("  POST   /api/search")
	log.Println("  POST   /api/chat")
	log.Println("  GET    /api/chat/conversations")
	log.Println("  GET    /api/chat/conversations/{id}")
	log.Println("  GET    /api/health")
	log.Println("  GET    /api/stats")

	log.Fatal(e.Start(":8080"))
}
