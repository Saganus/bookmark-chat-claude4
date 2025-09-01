package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	api "bookmark-chat/api/generated"
	"bookmark-chat/internal/services"
	"bookmark-chat/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	importService         *services.ImportService
	contentProcessor      *services.ContentProcessor
	categorizationService *services.CategorizationService
	storage               *storage.Storage
	scraper               services.Scraper
	bulkScraper           *services.BulkScraper
}

func NewHandler(storage *storage.Storage) *Handler {
	// Initialize scraper with default config
	scraperConfig := services.DefaultScraperConfig()
	scraper, err := services.NewScraper(scraperConfig)
	if err != nil {
		// Log error but continue with nil scraper
		// The scraper will be created on-demand in handlers if needed
		scraper = nil
	}

	// Initialize content processor for embedding generation
	contentProcessor, err := services.NewContentProcessor(storage)
	if err != nil {
		fmt.Printf("⚠️  Failed to create ContentProcessor (embeddings disabled): %v\n", err)
		contentProcessor = nil
	} else {
		fmt.Printf("✅ ContentProcessor initialized successfully (embeddings enabled)\n")
	}

	// Initialize categorization service
	categorizationService, err := services.NewCategorizationService(storage)
	if err != nil {
		fmt.Printf("⚠️  Failed to create CategorizationService (categorization disabled): %v\n", err)
		categorizationService = nil
	} else {
		fmt.Printf("✅ CategorizationService initialized successfully\n")
	}

	return &Handler{
		importService:         services.NewImportService(storage),
		contentProcessor:      contentProcessor,
		categorizationService: categorizationService,
		storage:               storage,
		scraper:               scraper,
		bulkScraper:           services.NewBulkScraper(scraper, storage),
	}
}

// Import bookmarks from file
// (POST /api/bookmarks/import)
func (h *Handler) ImportBookmarks(ctx echo.Context) error {
	// Get the uploaded file
	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "No file provided or invalid form data",
		})
	}

	// Validate the file
	if err := h.importService.ValidateFile(file); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: err.Error(),
		})
	}

	// Import the bookmarks
	importResult, parseResult, err := h.importService.ImportBookmarksFromFile(file)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "import_failed",
			Message: err.Error(),
		})
	}

	// Convert to API response format
	response := api.ImportResponse{
		Status: api.ImportResponseStatus(importResult.Status),
		Statistics: struct {
			Duplicates           *int `json:"duplicates,omitempty"`
			Failed               *int `json:"failed,omitempty"`
			SuccessfullyImported *int `json:"successfully_imported,omitempty"`
			TotalFound           *int `json:"total_found,omitempty"`
		}{
			TotalFound:           &importResult.Statistics.TotalFound,
			SuccessfullyImported: &importResult.Statistics.SuccessfullyImported,
			Failed:               &importResult.Statistics.Failed,
			Duplicates:           &importResult.Statistics.Duplicates,
		},
	}

	// Convert errors to API format
	if len(importResult.Errors) > 0 {
		errors := make([]struct {
			Error *string `json:"error,omitempty"`
			Url   *string `json:"url,omitempty"`
		}, len(importResult.Errors))

		for i, importErr := range importResult.Errors {
			errors[i] = struct {
				Error *string `json:"error,omitempty"`
				Url   *string `json:"url,omitempty"`
			}{
				Url:   &importErr.URL,
				Error: &importErr.Error,
			}
		}
		response.Errors = &errors
	}

	// Log the import results
	ctx.Logger().Infof("📁 Import completed: %s format", parseResult.Source)
	ctx.Logger().Infof("   📊 Statistics: %d total, %d imported, %d failed, %d duplicates",
		importResult.Statistics.TotalFound, importResult.Statistics.SuccessfullyImported,
		importResult.Statistics.Failed, importResult.Statistics.Duplicates)
	ctx.Logger().Infof("   📂 Folders: %d", len(parseResult.Folders))

	if importResult.Statistics.SuccessfullyImported > 0 {
		ctx.Logger().Infof("⚠️  Note: Imported bookmarks are in 'pending' status - use scraping API to generate embeddings")
	}

	return ctx.JSON(http.StatusOK, response)
}

// List all bookmarks
// (GET /api/bookmarks)
func (h *Handler) ListBookmarks(ctx echo.Context, params api.ListBookmarksParams) error {
	// Get bookmarks from database
	bookmarks, err := h.storage.ListBookmarks()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "database_error",
			Message: "Failed to retrieve bookmarks from database",
		})
	}

	// Convert storage bookmarks to API format
	apiBookmarks := make([]api.Bookmark, len(bookmarks))
	for i, bookmark := range bookmarks {
		// Convert string ID to UUID
		bookmarkUUID, err := uuid.Parse(bookmark.ID)
		if err != nil {
			ctx.Logger().Errorf("Invalid bookmark UUID: %s", bookmark.ID)
			continue
		}

		apiBookmarks[i] = api.Bookmark{
			Id:          bookmarkUUID,
			Url:         bookmark.URL,
			Title:       &bookmark.Title,
			Description: &bookmark.Description,
			FolderPath:  &bookmark.FolderPath,
			FaviconUrl:  &bookmark.FaviconURL,
			Tags:        &bookmark.Tags,
			CreatedAt:   bookmark.CreatedAt,
			UpdatedAt:   bookmark.UpdatedAt,
			ScrapedAt:   bookmark.ScrapedAt,
		}
	}

	totalItems := len(apiBookmarks)
	page := 1
	limit := 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.Limit != nil {
		limit = *params.Limit
	}

	totalPages := (totalItems + limit - 1) / limit

	return ctx.JSON(http.StatusOK, api.BookmarkListResponse{
		Bookmarks: apiBookmarks,
		Pagination: api.Pagination{
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
			TotalItems: totalItems,
		},
	})
}

// Get bookmark details
// (GET /api/bookmarks/{id})
func (h *Handler) GetBookmark(ctx echo.Context, id api.BookmarkId) error {
	// Get bookmark from database
	bookmark, err := h.storage.GetBookmark(id.String())
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.Error{
			Error:   "bookmark_not_found",
			Message: "Bookmark not found",
		})
	}

	// Get content if available
	var content *string
	if dbContent, err := h.storage.GetContent(bookmark.ID); err == nil {
		content = &dbContent.CleanText
	}

	return ctx.JSON(http.StatusOK, api.BookmarkDetail{
		Id:          id,
		Url:         bookmark.URL,
		Title:       &bookmark.Title,
		Description: &bookmark.Description,
		Content:     content,
		CreatedAt:   bookmark.CreatedAt,
		UpdatedAt:   bookmark.UpdatedAt,
		ScrapedAt:   bookmark.ScrapedAt,
		FolderPath:  &bookmark.FolderPath,
		FaviconUrl:  &bookmark.FaviconURL,
		Tags:        &bookmark.Tags,
	})
}

// Update bookmark
// (PUT /api/bookmarks/{id})
func (h *Handler) UpdateBookmark(ctx echo.Context, id api.BookmarkId) error {
	var req api.BookmarkUpdate
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	return ctx.JSON(http.StatusOK, api.BookmarkDetail{
		Id:          id,
		Url:         "https://example.com",
		Title:       req.Title,
		Description: req.Description,
		Content:     strPtr("Updated content. Implementation pending."),
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now(),
		Tags:        req.Tags,
	})
}

// Delete bookmark
// (DELETE /api/bookmarks/{id})
func (h *Handler) DeleteBookmark(ctx echo.Context, id api.BookmarkId) error {
	return ctx.NoContent(http.StatusNoContent)
}

// Re-scrape bookmark content
// (POST /api/bookmarks/{id}/rescrape)
func (h *Handler) RescrapeBookmark(ctx echo.Context, id api.BookmarkId) error {
	// Get bookmark from database
	bookmark, err := h.storage.GetBookmark(id.String())
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.Error{
			Error:   "bookmark_not_found",
			Message: "Bookmark not found",
		})
	}

	// Use pre-initialized scraper or create one on-demand
	scraper := h.scraper
	if scraper == nil {
		scraperConfig := services.DefaultScraperConfig()
		var err error
		scraper, err = services.NewScraper(scraperConfig)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, api.Error{
				Error:   "scraper_error",
				Message: "Failed to create scraper: " + err.Error(),
			})
		}
	}

	// Scrape the content
	scrapedContent, err := scraper.Scrape(ctx.Request().Context(), bookmark.URL, services.DefaultScrapeOptions())
	if err != nil || !scrapedContent.Success {
		errorMsg := "Failed to scrape content"
		if scrapedContent != nil && scrapedContent.Error != "" {
			errorMsg = scrapedContent.Error
		} else if err != nil {
			errorMsg = err.Error()
		}

		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "scraping_failed",
			Message: errorMsg,
		})
	}

	// Update bookmark with scraped data
	bookmark.Title = scrapedContent.Title
	bookmark.Description = scrapedContent.Description
	bookmark.FaviconURL = scrapedContent.FaviconURL
	bookmark.UpdatedAt = time.Now()
	now := time.Now()
	bookmark.ScrapedAt = &now

	err = h.storage.UpdateBookmark(bookmark)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "database_error",
			Message: "Failed to update bookmark: " + err.Error(),
		})
	}

	// Store the scraped content
	err = h.storage.StoreContent(bookmark.ID, scrapedContent.Content, scrapedContent.CleanText)
	if err != nil {
		ctx.Logger().Errorf("Failed to store content for bookmark %s: %v", bookmark.ID, err)
		// Don't fail the request, just log the error
	} else {
		ctx.Logger().Infof("✅ Stored content for bookmark %s: %s", bookmark.ID, bookmark.URL)

		// Generate embeddings if ContentProcessor is available
		if h.contentProcessor != nil {
			ctx.Logger().Infof("🔄 Generating embeddings for bookmark %s...", bookmark.ID)

			// Get the stored content to get the content ID
			content, err := h.storage.GetContent(bookmark.ID)
			if err != nil {
				ctx.Logger().Errorf("❌ Failed to get content for embedding: %v", err)
			} else {
				// Generate embedding for the clean text
				embedding, err := h.contentProcessor.GenerateQueryEmbedding(content.CleanText)
				if err != nil {
					ctx.Logger().Errorf("❌ Failed to generate embedding: %v", err)
				} else {
					// Store the embedding
					err = h.storage.StoreEmbedding(content.ID, embedding)
					if err != nil {
						ctx.Logger().Errorf("❌ Failed to store embedding: %v", err)
					} else {
						ctx.Logger().Infof("✅ Generated and stored embedding for bookmark %s", bookmark.ID)
					}
				}
			}
		} else {
			ctx.Logger().Warnf("⚠️  ContentProcessor not available - embeddings not generated for %s", bookmark.ID)
		}
	}

	// Return updated bookmark
	bookmarkUUID, _ := uuid.Parse(bookmark.ID)
	return ctx.JSON(http.StatusOK, api.BookmarkDetail{
		Id:          bookmarkUUID,
		Url:         bookmark.URL,
		Title:       &bookmark.Title,
		Description: &bookmark.Description,
		Content:     &scrapedContent.CleanText,
		CreatedAt:   bookmark.CreatedAt,
		UpdatedAt:   bookmark.UpdatedAt,
		ScrapedAt:   bookmark.ScrapedAt,
		FolderPath:  &bookmark.FolderPath,
		FaviconUrl:  &bookmark.FaviconURL,
		Tags:        &bookmark.Tags,
	})
}

// Hybrid search
// (POST /api/search)
func (h *Handler) SearchBookmarks(ctx echo.Context) error {
	var req api.SearchRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	ctx.Logger().Infof("🔍 Search request for query: '%s'", req.Query)

	var results []*storage.SearchResult
	var err error

	// Try hybrid search if ContentProcessor is available
	if h.contentProcessor != nil {
		ctx.Logger().Infof("🔄 Using hybrid search (semantic + keyword) for: '%s'", req.Query)
		results, err = h.contentProcessor.HybridSearch(req.Query)
		if err != nil {
			ctx.Logger().Errorf("❌ Hybrid search failed, falling back to keyword search: %v", err)
			// Fall back to keyword search
			results, err = h.storage.KeywordSearch(req.Query, 20)
			if err != nil {
				ctx.Logger().Errorf("❌ Keyword search also failed: %v", err)
				return ctx.JSON(http.StatusInternalServerError, api.Error{
					Error:   "search_failed",
					Message: "Both hybrid and keyword search failed: " + err.Error(),
				})
			}
			ctx.Logger().Infof("✅ Fallback keyword search found %d results", len(results))
		} else {
			ctx.Logger().Infof("✅ Hybrid search found %d results", len(results))
		}
	} else {
		// ContentProcessor not available, use keyword search only
		ctx.Logger().Infof("🔄 Using keyword-only search for: '%s'", req.Query)
		results, err = h.storage.KeywordSearch(req.Query, 20)
		if err != nil {
			ctx.Logger().Errorf("❌ Keyword search failed: %v", err)
			return ctx.JSON(http.StatusInternalServerError, api.Error{
				Error:   "search_failed",
				Message: "Keyword search failed: " + err.Error(),
			})
		}
		ctx.Logger().Infof("✅ Keyword search found %d results", len(results))
	}

	// Convert storage results to API format
	apiResults := make([]api.SearchResult, len(results))
	for i, result := range results {
		// Convert string ID to UUID
		bookmarkUUID, err := uuid.Parse(result.Bookmark.ID)
		if err != nil {
			ctx.Logger().Errorf("Invalid bookmark UUID in search result: %s", result.Bookmark.ID)
			continue
		}

		// Use Content's ScrapedAt if available, otherwise use Bookmark's ScrapedAt
		var scrapedAt *time.Time
		if result.Content != nil && !result.Content.ScrapedAt.IsZero() {
			scrapedAt = &result.Content.ScrapedAt
		} else {
			scrapedAt = result.Bookmark.ScrapedAt
		}

		apiResult := api.SearchResult{
			Bookmark: api.Bookmark{
				Id:          bookmarkUUID,
				Url:         result.Bookmark.URL,
				Title:       &result.Bookmark.Title,
				Description: &result.Bookmark.Description,
				FolderPath:  &result.Bookmark.FolderPath,
				FaviconUrl:  &result.Bookmark.FaviconURL,
				Tags:        &result.Bookmark.Tags,
				CreatedAt:   result.Bookmark.CreatedAt,
				UpdatedAt:   result.Bookmark.UpdatedAt,
				ScrapedAt:   scrapedAt,
			},
			RelevanceScore: float32(result.RelevanceScore),
		}

		// Add snippet if available
		if result.MatchedSnippet != "" {
			apiResult.Snippet = &result.MatchedSnippet
		}

		apiResults[i] = apiResult
	}

	ctx.Logger().Infof("✅ Returning %d search results for query: '%s'", len(apiResults), req.Query)

	return ctx.JSON(http.StatusOK, api.SearchResponse{
		Results:      apiResults,
		TotalResults: len(apiResults),
	})
}

// Send chat message
// (POST /api/chat)
func (h *Handler) SendChatMessage(ctx echo.Context) error {
	var req api.ChatRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	conversationId := uuid.New()
	if req.ConversationId != nil {
		conversationId = *req.ConversationId
	}

	return ctx.JSON(http.StatusOK, api.ChatResponse{
		Reply: "Implementation pending. Here's an example of how a response would look like: Based on your bookmarks about " + req.Message + ", I found several relevant resources...",
		Sources: &[]api.Bookmark{
			{
				Id:        uuid.New(),
				Url:       "https://example.com",
				Title:     strPtr("Relevant Bookmark"),
				CreatedAt: time.Now().Add(-24 * time.Hour),
				UpdatedAt: time.Now().Add(-24 * time.Hour),
			},
		},
		ConversationId: conversationId,
	})
}

// List conversations
// (GET /api/chat/conversations)
func (h *Handler) ListConversations(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, api.ConversationListResponse{
		Conversations: []api.ConversationSummary{
			{
				Id:           uuid.New(),
				Title:        "Example Conversation",
				MessageCount: intPtr(5),
				CreatedAt:    time.Now().Add(-2 * time.Hour),
				UpdatedAt:    time.Now().Add(-1 * time.Hour),
			},
			{
				Id:           uuid.New(),
				Title:        "Another Chat Session",
				MessageCount: intPtr(3),
				CreatedAt:    time.Now().Add(-24 * time.Hour),
				UpdatedAt:    time.Now().Add(-20 * time.Hour),
			},
		},
	})
}

// Get conversation history
// (GET /api/chat/conversations/{id})
func (h *Handler) GetConversation(ctx echo.Context, id api.ConversationId) error {
	return ctx.JSON(http.StatusOK, api.ConversationDetail{
		Id:    id,
		Title: "Example Conversation",
		Messages: []api.Message{
			{
				Id:        uuid.New(),
				Role:      api.User,
				Content:   "Tell me about my golang bookmarks",
				CreatedAt: time.Now().Add(-2 * time.Hour),
			},
			{
				Id:           uuid.New(),
				Role:         api.Assistant,
				Content:      "Based on your bookmarks, you have several Go-related resources saved...",
				BookmarkRefs: &[]uuid.UUID{uuid.New()},
				CreatedAt:    time.Now().Add(-1*time.Hour - 50*time.Minute),
			},
		},
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	})
}

// Health check
// (GET /api/health)
func (h *Handler) HealthCheck(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, api.HealthResponse{
		Status:    api.Healthy,
		Timestamp: time.Now(),
		Services: &struct {
			Database   *api.HealthResponseServicesDatabase   `json:"database,omitempty"`
			Embeddings *api.HealthResponseServicesEmbeddings `json:"embeddings,omitempty"`
			Scraper    *api.HealthResponseServicesScraper    `json:"scraper,omitempty"`
		}{
			Database:   (*api.HealthResponseServicesDatabase)(strPtr("up")),
			Embeddings: (*api.HealthResponseServicesEmbeddings)(strPtr("up")),
			Scraper:    (*api.HealthResponseServicesScraper)(strPtr("up")),
		},
	})
}

// System statistics
// (GET /api/stats)
func (h *Handler) GetSystemStats(ctx echo.Context) error {
	ctx.Logger().Infof("📊 Retrieving system statistics...")

	// Get actual bookmark count
	bookmarks, err := h.storage.ListBookmarks()
	if err != nil {
		ctx.Logger().Errorf("❌ Failed to get bookmarks for stats: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "database_error",
			Message: "Failed to retrieve system statistics",
		})
	}

	bookmarkCount := len(bookmarks)

	// Count pending and completed bookmarks
	pendingCount := 0
	completedCount := 0
	for _, bookmark := range bookmarks {
		if bookmark.Status == "pending" {
			pendingCount++
		} else if bookmark.Status == "completed" {
			completedCount++
		}
	}

	// Get embedding count from database
	var embeddingCount int
	err = h.storage.GetDB().QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&embeddingCount)
	if err != nil {
		ctx.Logger().Errorf("❌ Failed to count embeddings: %v", err)
		embeddingCount = 0
	}

	// Get content count
	var contentCount int
	err = h.storage.GetDB().QueryRow("SELECT COUNT(*) FROM content").Scan(&contentCount)
	if err != nil {
		ctx.Logger().Errorf("❌ Failed to count content: %v", err)
		contentCount = 0
	}

	ctx.Logger().Infof("📊 Stats: %d bookmarks (%d pending, %d completed), %d content, %d embeddings",
		bookmarkCount, pendingCount, completedCount, contentCount, embeddingCount)

	return ctx.JSON(http.StatusOK, api.StatsResponse{
		BookmarkCount:     bookmarkCount,
		ConversationCount: 0, // Not implemented yet
		IndexStatus: struct {
			EmbeddingsGenerated *int       `json:"embeddings_generated,omitempty"`
			EmbeddingsPending   *int       `json:"embeddings_pending,omitempty"`
			LastIndexed         *time.Time `json:"last_indexed,omitempty"`
		}{
			EmbeddingsGenerated: &embeddingCount,
			EmbeddingsPending:   &pendingCount,
			LastIndexed:         timePtr(time.Now()),
		},
		StorageSizeMb: float32Ptr(0.0), // Could calculate actual DB size if needed
	})
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

func float32Ptr(f float32) *float32 {
	return &f
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Bulk Scraping Handlers

// Start bulk scraping process
// (POST /api/scraping/start)
func (h *Handler) StartScraping(ctx echo.Context) error {
	var req struct {
		BookmarkIds []string `json:"bookmark_ids"`
	}

	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if len(req.BookmarkIds) == 0 {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "No bookmark IDs provided",
		})
	}

	err := h.bulkScraper.Start(context.Background(), req.BookmarkIds)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "scraping_failed",
			Message: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":          "started",
		"message":         fmt.Sprintf("Started scraping %d bookmarks", len(req.BookmarkIds)),
		"total_bookmarks": len(req.BookmarkIds),
	})
}

// Pause scraping process
// (POST /api/scraping/pause)
func (h *Handler) PauseScraping(ctx echo.Context) error {
	err := h.bulkScraper.Pause()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "pause_failed",
			Message: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":  "paused",
		"message": "Scraping paused",
	})
}

// Resume scraping process
// (POST /api/scraping/resume)
func (h *Handler) ResumeScraping(ctx echo.Context) error {
	err := h.bulkScraper.Resume()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "resume_failed",
			Message: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":  "running",
		"message": "Scraping resumed",
	})
}

// Stop scraping process
// (POST /api/scraping/stop)
func (h *Handler) StopScraping(ctx echo.Context) error {
	err := h.bulkScraper.Stop()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "stop_failed",
			Message: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":  "stopped",
		"message": "Scraping stopped",
	})
}

// Get scraping status
// (GET /api/scraping/status)
func (h *Handler) GetScrapingStatus(ctx echo.Context) error {
	status := h.bulkScraper.GetStatus()
	return ctx.JSON(http.StatusOK, status)
}

// Categorization Handlers

// Categorize a single bookmark using AI
// (POST /api/bookmarks/{id}/categorize)
func (h *Handler) CategorizeBookmark(ctx echo.Context, id api.BookmarkId) error {
	if h.categorizationService == nil {
		return ctx.JSON(http.StatusServiceUnavailable, api.Error{
			Error:   "service_unavailable",
			Message: "Categorization service is not available (OPENAI_API_KEY not configured)",
		})
	}

	ctx.Logger().Infof("🤖 Starting categorization for bookmark: %s", id.String())

	result, err := h.categorizationService.CategorizeBookmark(ctx.Request().Context(), id.String())
	if err != nil {
		ctx.Logger().Errorf("❌ Categorization failed for bookmark %s: %v", id.String(), err)
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "categorization_failed",
			Message: err.Error(),
		})
	}

	ctx.Logger().Infof("✅ Categorized bookmark %s: primary=%s, confidence=%.2f", 
		id.String(), result.PrimaryCategory, result.ConfidenceScore)

	// Convert to API format
	response := api.CategorizationResult{
		PrimaryCategory:     result.PrimaryCategory,
		SecondaryCategories: &result.SecondaryCategories,
		Tags:                &result.Tags,
		ConfidenceScore:     float32(result.ConfidenceScore),
		Reasoning:           &result.Reasoning,
	}

	return ctx.JSON(http.StatusOK, response)
}

// Bulk categorize bookmarks
// (POST /api/bookmarks/categorize/bulk)
func (h *Handler) CategorizeBulk(ctx echo.Context) error {
	if h.categorizationService == nil {
		return ctx.JSON(http.StatusServiceUnavailable, api.Error{
			Error:   "service_unavailable",
			Message: "Categorization service is not available (OPENAI_API_KEY not configured)",
		})
	}

	var req struct {
		BookmarkIds         []uuid.UUID `json:"bookmark_ids"`
		AutoApply          bool        `json:"auto_apply"`
		ConfidenceThreshold float64     `json:"confidence_threshold"`
	}

	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if len(req.BookmarkIds) == 0 {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Error:   "bad_request",
			Message: "No bookmark IDs provided",
		})
	}

	// Default confidence threshold
	if req.ConfidenceThreshold == 0 {
		req.ConfidenceThreshold = 0.8
	}

	// Convert UUIDs to strings
	bookmarkIDs := make([]string, len(req.BookmarkIds))
	for i, id := range req.BookmarkIds {
		bookmarkIDs[i] = id.String()
	}

	ctx.Logger().Infof("🚀 Starting bulk categorization for %d bookmarks (auto_apply=%v, threshold=%.2f)", 
		len(bookmarkIDs), req.AutoApply, req.ConfidenceThreshold)

	results, err := h.categorizationService.BulkCategorize(
		ctx.Request().Context(),
		bookmarkIDs,
		req.AutoApply,
		req.ConfidenceThreshold,
	)
	if err != nil {
		ctx.Logger().Errorf("❌ Bulk categorization failed: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "categorization_failed",
			Message: err.Error(),
		})
	}

	// Count auto-applied results
	appliedCount := 0
	for _, result := range results {
		if req.AutoApply && result.ConfidenceScore >= req.ConfidenceThreshold {
			appliedCount++
		}
	}

	ctx.Logger().Infof("✅ Bulk categorization complete: %d processed, %d applied", len(results), appliedCount)

	// Build response
	responseResults := make([]struct {
		BookmarkId     uuid.UUID                 `json:"bookmark_id"`
		Categorization api.CategorizationResult `json:"categorization"`
		Applied        bool                      `json:"applied"`
	}, len(results))

	for i, result := range results {
		bookmarkUUID, _ := uuid.Parse(bookmarkIDs[i])
		applied := req.AutoApply && result.ConfidenceScore >= req.ConfidenceThreshold

		responseResults[i] = struct {
			BookmarkId     uuid.UUID                 `json:"bookmark_id"`
			Categorization api.CategorizationResult `json:"categorization"`
			Applied        bool                      `json:"applied"`
		}{
			BookmarkId: bookmarkUUID,
			Categorization: api.CategorizationResult{
				PrimaryCategory:     result.PrimaryCategory,
				SecondaryCategories: &result.SecondaryCategories,
				Tags:                &result.Tags,
				ConfidenceScore:     float32(result.ConfidenceScore),
				Reasoning:           &result.Reasoning,
			},
			Applied: applied,
		}
	}

	response := struct {
		Results         interface{} `json:"results"`
		TotalProcessed  int         `json:"total_processed"`
		TotalApplied    int         `json:"total_applied"`
	}{
		Results:        responseResults,
		TotalProcessed: len(results),
		TotalApplied:   appliedCount,
	}

	return ctx.JSON(http.StatusOK, response)
}

// Get all user categories
// (GET /api/categories)
func (h *Handler) GetCategories(ctx echo.Context) error {
	ctx.Logger().Infof("📂 Retrieving categories...")

	categories, err := h.storage.GetCategories(ctx.Request().Context())
	if err != nil {
		ctx.Logger().Errorf("❌ Failed to get categories: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Error:   "database_error",
			Message: "Failed to retrieve categories",
		})
	}

	// Convert to API format
	apiCategories := make([]api.Category, len(categories))
	for i, cat := range categories {
		var color *string
		if cat.Color != "" {
			color = &cat.Color
		}
		
		apiCategories[i] = api.Category{
			Id:             cat.ID,
			Name:           cat.Name,
			ParentCategory: cat.ParentCategory,
			Color:          color,
			UsageCount:     cat.UsageCount,
			CreatedAt:      cat.CreatedAt,
			UpdatedAt:      cat.UpdatedAt,
		}
	}

	ctx.Logger().Infof("✅ Retrieved %d categories", len(apiCategories))
	return ctx.JSON(http.StatusOK, apiCategories)
}
