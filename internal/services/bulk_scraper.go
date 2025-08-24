package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bookmark-chat/internal/storage"
)

// ScrapingStatus represents the current state of bulk scraping
type ScrapingStatus string

const (
	StatusIdle      ScrapingStatus = "idle"
	StatusRunning   ScrapingStatus = "running"
	StatusPaused    ScrapingStatus = "paused"
	StatusCompleted ScrapingStatus = "completed"
	StatusStopped   ScrapingStatus = "stopped"
)

// BookmarkScrapingStatus represents the status of individual bookmark scraping
type BookmarkScrapingStatus string

const (
	BookmarkNotScraped  BookmarkScrapingStatus = "not-scraped"
	BookmarkInProgress  BookmarkScrapingStatus = "in-progress"
	BookmarkScraped     BookmarkScrapingStatus = "scraped"
	BookmarkError       BookmarkScrapingStatus = "error"
)

// BulkScrapingStatus represents the overall scraping status
type BulkScrapingStatus struct {
	Status           ScrapingStatus                       `json:"status"`
	Current          int                                  `json:"current"`
	Total            int                                  `json:"total"`
	Progress         float64                              `json:"progress"`
	CurrentURL       string                               `json:"current_url,omitempty"`
	BookmarkStatuses map[string]BookmarkScrapingProgress  `json:"bookmark_statuses,omitempty"`
}

// BookmarkScrapingProgress represents individual bookmark progress
type BookmarkScrapingProgress struct {
	Status BookmarkScrapingStatus `json:"status"`
	Error  string                 `json:"error,omitempty"`
}

// BulkScraper manages bulk scraping operations
type BulkScraper struct {
	scraper  Scraper
	storage  *storage.Storage
	mu       sync.RWMutex
	
	// Current operation state
	status           ScrapingStatus
	bookmarkIDs      []string
	current          int
	total            int
	currentURL       string
	bookmarkStatuses map[string]BookmarkScrapingProgress
	
	// Control channels
	pauseChan   chan struct{}
	resumeChan  chan struct{}
	stopChan    chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewBulkScraper creates a new bulk scraper
func NewBulkScraper(scraper Scraper, storage *storage.Storage) *BulkScraper {
	return &BulkScraper{
		scraper:          scraper,
		storage:          storage,
		status:           StatusIdle,
		bookmarkStatuses: make(map[string]BookmarkScrapingProgress),
		pauseChan:        make(chan struct{}, 1),
		resumeChan:       make(chan struct{}, 1),
		stopChan:         make(chan struct{}, 1),
	}
}

// Start begins the bulk scraping process
func (bs *BulkScraper) Start(ctx context.Context, bookmarkIDs []string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.status == StatusRunning || bs.status == StatusPaused {
		return fmt.Errorf("scraping already in progress")
	}
	
	bs.bookmarkIDs = bookmarkIDs
	bs.current = 0
	bs.total = len(bookmarkIDs)
	bs.status = StatusRunning
	bs.bookmarkStatuses = make(map[string]BookmarkScrapingProgress)
	bs.ctx, bs.cancel = context.WithCancel(ctx)
	
	// Initialize all bookmarks as not-scraped
	for _, id := range bookmarkIDs {
		bs.bookmarkStatuses[id] = BookmarkScrapingProgress{
			Status: BookmarkNotScraped,
		}
	}
	
	// Start scraping in background
	go bs.scrapeAll()
	
	return nil
}

// Pause pauses the current scraping operation
func (bs *BulkScraper) Pause() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.status != StatusRunning {
		return fmt.Errorf("no running scraping process to pause")
	}
	
	bs.status = StatusPaused
	select {
	case bs.pauseChan <- struct{}{}:
	default:
	}
	
	return nil
}

// Resume resumes a paused scraping operation
func (bs *BulkScraper) Resume() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.status != StatusPaused {
		return fmt.Errorf("no paused scraping process to resume")
	}
	
	bs.status = StatusRunning
	select {
	case bs.resumeChan <- struct{}{}:
	default:
	}
	
	return nil
}

// Stop stops the current scraping operation
func (bs *BulkScraper) Stop() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.status != StatusRunning && bs.status != StatusPaused {
		return fmt.Errorf("no scraping process to stop")
	}
	
	bs.status = StatusStopped
	if bs.cancel != nil {
		bs.cancel()
	}
	
	select {
	case bs.stopChan <- struct{}{}:
	default:
	}
	
	return nil
}

// GetStatus returns the current scraping status
func (bs *BulkScraper) GetStatus() BulkScrapingStatus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	progress := 0.0
	if bs.total > 0 {
		progress = float64(bs.current) / float64(bs.total) * 100
	}
	
	return BulkScrapingStatus{
		Status:           bs.status,
		Current:          bs.current,
		Total:            bs.total,
		Progress:         progress,
		CurrentURL:       bs.currentURL,
		BookmarkStatuses: bs.bookmarkStatuses,
	}
}

// scrapeAll performs the actual bulk scraping
func (bs *BulkScraper) scrapeAll() {
	defer func() {
		bs.mu.Lock()
		if bs.status == StatusRunning {
			bs.status = StatusCompleted
		}
		bs.mu.Unlock()
	}()
	
	for i, bookmarkID := range bs.bookmarkIDs {
		// Check for stop signal
		select {
		case <-bs.ctx.Done():
			return
		case <-bs.stopChan:
			return
		default:
		}
		
		// Check for pause signal
		select {
		case <-bs.pauseChan:
			// Wait for resume or stop
			select {
			case <-bs.resumeChan:
				// Continue
			case <-bs.stopChan:
				return
			case <-bs.ctx.Done():
				return
			}
		default:
		}
		
		// Update current position
		bs.mu.Lock()
		bs.current = i + 1
		bs.mu.Unlock()
		
		// Get bookmark info
		bookmark, err := bs.storage.GetBookmark(bookmarkID)
		if err != nil {
			bs.updateBookmarkStatus(bookmarkID, BookmarkError, fmt.Sprintf("Failed to get bookmark: %v", err))
			continue
		}
		
		// Update current URL
		bs.mu.Lock()
		bs.currentURL = bookmark.URL
		bs.mu.Unlock()
		
		// Update status to in-progress
		bs.updateBookmarkStatus(bookmarkID, BookmarkInProgress, "")
		
		// Create scraper if needed (fallback if nil)
		scraper := bs.scraper
		if scraper == nil {
			scraperConfig := DefaultScraperConfig()
			var scraperErr error
			scraper, scraperErr = NewScraper(scraperConfig)
			if scraperErr != nil {
				bs.updateBookmarkStatus(bookmarkID, BookmarkError, fmt.Sprintf("Failed to create scraper: %v", scraperErr))
				continue
			}
		}
		
		// Scrape the bookmark
		scrapedContent, err := scraper.Scrape(bs.ctx, bookmark.URL, DefaultScrapeOptions())
		if err != nil || !scrapedContent.Success {
			errorMsg := "Failed to scrape content"
			if scrapedContent != nil && scrapedContent.Error != "" {
				errorMsg = scrapedContent.Error
			} else if err != nil {
				errorMsg = err.Error()
			}
			bs.updateBookmarkStatus(bookmarkID, BookmarkError, errorMsg)
			continue
		}
		
		// Update bookmark with scraped data
		bookmark.Title = scrapedContent.Title
		bookmark.Description = scrapedContent.Description
		bookmark.FaviconURL = scrapedContent.FaviconURL
		bookmark.UpdatedAt = time.Now()
		now := time.Now()
		bookmark.ScrapedAt = &now
		
		err = bs.storage.UpdateBookmark(bookmark)
		if err != nil {
			bs.updateBookmarkStatus(bookmarkID, BookmarkError, fmt.Sprintf("Failed to update bookmark: %v", err))
			continue
		}
		
		// Store the scraped content
		err = bs.storage.StoreContent(bookmark.ID, scrapedContent.Content, scrapedContent.CleanText)
		if err != nil {
			// Log error but don't fail the scraping
			fmt.Printf("Failed to store content for bookmark %s: %v\n", bookmark.ID, err)
		}
		
		// Mark as successfully scraped
		bs.updateBookmarkStatus(bookmarkID, BookmarkScraped, "")
	}
	
	// Update final position
	bs.mu.Lock()
	bs.current = bs.total
	bs.currentURL = ""
	bs.mu.Unlock()
}

// updateBookmarkStatus updates the status of a specific bookmark
func (bs *BulkScraper) updateBookmarkStatus(bookmarkID string, status BookmarkScrapingStatus, errorMsg string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	bs.bookmarkStatuses[bookmarkID] = BookmarkScrapingProgress{
		Status: status,
		Error:  errorMsg,
	}
}