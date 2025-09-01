package main

import (
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	api "bookmark-chat/api/generated"
	"bookmark-chat/internal/handlers"
	"bookmark-chat/internal/services"
	"bookmark-chat/internal/storage"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize storage
	store, err := storage.New("file:bookmarks.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create handler instance with storage
	handler := handlers.NewHandler(store)

	// Start background processing for pending bookmarks (if OpenAI key is available)
	if os.Getenv("OPENAI_API_KEY") != "" {
		log.Println("🤖 OpenAI API key found - starting background embedding processor...")
		startBackgroundProcessor(store)
	} else {
		log.Println("⚠️  No OpenAI API key found - background embedding processing disabled")
		log.Println("   Set OPENAI_API_KEY environment variable to enable embeddings")
	}

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
	log.Println("  POST   /api/bookmarks/{id}/categorize")
	log.Println("  POST   /api/bookmarks/categorize/bulk")
	log.Println("  POST   /api/scraping/start")
	log.Println("  POST   /api/scraping/pause")
	log.Println("  POST   /api/scraping/resume")
	log.Println("  POST   /api/scraping/stop")
	log.Println("  GET    /api/scraping/status")
	log.Println("  POST   /api/search")
	log.Println("  GET    /api/categories")
	log.Println("  POST   /api/chat")
	log.Println("  GET    /api/chat/conversations")
	log.Println("  GET    /api/chat/conversations/{id}")
	log.Println("  GET    /api/health")
	log.Println("  GET    /api/stats")

	log.Fatal(e.Start(":8080"))
}

// startBackgroundProcessor starts a background goroutine to process pending bookmarks
func startBackgroundProcessor(store *storage.Storage) {
	go func() {
		// Create content processor
		processor, err := services.NewContentProcessor(store)
		if err != nil {
			log.Printf("❌ Failed to create background ContentProcessor: %v", err)
			return
		}

		log.Println("✅ Background embedding processor started")
		log.Println("   - Checking for pending bookmarks every 30 seconds")
		log.Println("   - Processing up to 5 bookmarks per batch")

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Process pending bookmarks in batches
				bookmarks, err := store.ListBookmarks()
				if err != nil {
					log.Printf("❌ Failed to list bookmarks for background processing: %v", err)
					continue
				}

				pendingCount := 0
				processedCount := 0
				maxBatch := 5 // Process max 5 per cycle to avoid overwhelming

				for _, bookmark := range bookmarks {
					if bookmark.Status == "pending" {
						pendingCount++
						if processedCount >= maxBatch {
							continue // Skip processing but count total pending
						}

						log.Printf("🔄 Background processing bookmark: %s", bookmark.URL)

						err := processor.ProcessBookmarkContent(bookmark.ID)
						if err != nil {
							log.Printf("❌ Background processing failed for %s: %v", bookmark.URL, err)
						} else {
							log.Printf("✅ Background processing completed for %s", bookmark.URL)
							processedCount++
						}
					}
				}

				if pendingCount > 0 {
					log.Printf("📊 Background processor: %d pending bookmarks, %d processed this cycle", pendingCount, processedCount)
				}
			}
		}
	}()
}
