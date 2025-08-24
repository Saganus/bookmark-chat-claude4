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
	importService *services.ImportService
	storage       *storage.Storage
	scraper       services.Scraper
	bulkScraper   *services.BulkScraper
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

	return &Handler{
		importService: services.NewImportService(storage),
		storage:       storage,
		scraper:       scraper,
		bulkScraper:   services.NewBulkScraper(scraper, storage),
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

	// Log the parsed structure for debugging
	ctx.Logger().Infof("Import completed: %s format, %d bookmarks, %d folders",
		parseResult.Source, parseResult.TotalCount, len(parseResult.Folders))

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

	return ctx.JSON(http.StatusOK, api.SearchResponse{
		Results: []api.SearchResult{
			{
				Bookmark: api.Bookmark{
					Id:          uuid.New(),
					Url:         "https://example.com",
					Title:       strPtr("Example Website"),
					Description: strPtr("Search result matching query: " + req.Query),
					CreatedAt:   time.Now().Add(-24 * time.Hour),
					UpdatedAt:   time.Now().Add(-24 * time.Hour),
				},
				RelevanceScore: 0.95,
				Snippet:        strPtr("...highlighted snippet containing search terms..."),
			},
		},
		TotalResults: 1,
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
	return ctx.JSON(http.StatusOK, api.StatsResponse{
		BookmarkCount:     42,
		ConversationCount: 7,
		IndexStatus: struct {
			EmbeddingsGenerated *int       `json:"embeddings_generated,omitempty"`
			EmbeddingsPending   *int       `json:"embeddings_pending,omitempty"`
			LastIndexed         *time.Time `json:"last_indexed,omitempty"`
		}{
			EmbeddingsGenerated: intPtr(40),
			EmbeddingsPending:   intPtr(2),
			LastIndexed:         timePtr(time.Now().Add(-30 * time.Minute)),
		},
		StorageSizeMb: float32Ptr(125.5),
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
