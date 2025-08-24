package services

import (
	"context"
	"fmt"
	"log"

	"bookmark-chat/internal/storage"
)

// ContentProcessor handles the complete flow of content processing including embeddings
type ContentProcessor struct {
	storage          *storage.Storage
	embeddingService *EmbeddingService
	scraperService   Scraper
}

// NewContentProcessor creates a new content processor
func NewContentProcessor(store *storage.Storage) (*ContentProcessor, error) {
	embeddingService, err := NewEmbeddingService()
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding service: %w", err)
	}

	scraperService, err := NewScraper(DefaultScraperConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper service: %w", err)
	}

	return &ContentProcessor{
		storage:          store,
		embeddingService: embeddingService,
		scraperService:   scraperService,
	}, nil
}

// ProcessBookmarkContent scrapes content for a bookmark and generates embeddings
func (cp *ContentProcessor) ProcessBookmarkContent(bookmarkID string) error {
	// Get the bookmark
	bookmark, err := cp.storage.GetBookmark(bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to get bookmark: %w", err)
	}

	// Scrape the content
	scraped, err := cp.scraperService.Scrape(context.Background(), bookmark.URL, DefaultScrapeOptions())
	if err != nil {
		log.Printf("Failed to scrape %s: %v", bookmark.URL, err)
		// Update bookmark status to failed
		cp.storage.UpdateBookmarkStatus(bookmarkID, "failed")
		return fmt.Errorf("failed to scrape content: %w", err)
	}

	// Store the content
	err = cp.storage.StoreContent(bookmarkID, scraped.Content, scraped.CleanText)
	if err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}

	// Get the content to get the content ID
	content, err := cp.storage.GetContent(bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to get stored content: %w", err)
	}

	// Generate embeddings with chunking for the clean text
	embeddings, chunks, err := cp.embeddingService.GenerateEmbeddingWithChunking(content.CleanText)
	if err != nil {
		log.Printf("Failed to generate embeddings for %s: %v", bookmark.URL, err)
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	log.Printf("Generated %d chunks for %s", len(chunks), bookmark.URL)

	// Store the embeddings for all chunks
	err = cp.storage.StoreMultipleChunkEmbeddings(content.ID, embeddings, chunks)
	if err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	// Update bookmark status to completed
	err = cp.storage.UpdateBookmarkStatus(bookmarkID, "completed")
	if err != nil {
		return fmt.Errorf("failed to update bookmark status: %w", err)
	}

	log.Printf("Successfully processed content for bookmark %s: %s", bookmarkID, bookmark.URL)
	return nil
}

// ProcessAllPendingBookmarks processes all bookmarks with pending status
func (cp *ContentProcessor) ProcessAllPendingBookmarks() error {
	bookmarks, err := cp.storage.ListBookmarks()
	if err != nil {
		return fmt.Errorf("failed to list bookmarks: %w", err)
	}

	processed := 0
	failed := 0

	for _, bookmark := range bookmarks {
		if bookmark.Status == "pending" {
			log.Printf("Processing bookmark: %s", bookmark.URL)

			err := cp.ProcessBookmarkContent(bookmark.ID)
			if err != nil {
				log.Printf("Failed to process bookmark %s: %v", bookmark.URL, err)
				failed++
			} else {
				processed++
			}
		}
	}

	log.Printf("Finished processing bookmarks. Processed: %d, Failed: %d", processed, failed)
	return nil
}

// GenerateQueryEmbedding generates an embedding for a search query
func (cp *ContentProcessor) GenerateQueryEmbedding(query string) ([]float32, error) {
	return cp.embeddingService.GenerateEmbedding(query)
}

// HybridSearch performs semantic + keyword search
func (cp *ContentProcessor) HybridSearch(query string) ([]*storage.SearchResult, error) {
	// Generate embedding for the query
	queryEmbedding, err := cp.embeddingService.GenerateEmbedding(query)
	if err != nil {
		// If embedding generation fails, fall back to keyword search only
		log.Printf("Failed to generate query embedding, using keyword search only: %v", err)
		return cp.storage.KeywordSearch(query, 20)
	}

	// Perform hybrid search
	return cp.storage.HybridSearch(queryEmbedding, query)
}

// KeywordSearch performs only keyword-based search (fallback)
func (cp *ContentProcessor) KeywordSearch(query string) ([]*storage.SearchResult, error) {
	return cp.storage.KeywordSearch(query, 20)
}
