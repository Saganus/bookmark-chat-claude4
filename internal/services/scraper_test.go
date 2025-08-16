package services

import (
	"context"
	"testing"
	"time"
)

func TestHTMLScraper_Scrape(t *testing.T) {
	scraper := NewHTMLScraper()
	options := DefaultScrapeOptions()
	options.Timeout = 10 * time.Second

	testURL := "https://example.com"
	ctx := context.Background()

	content, err := scraper.Scrape(ctx, testURL, options)
	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	if !content.Success {
		t.Fatalf("Scraping was not successful: %s", content.Error)
	}

	if content.URL != testURL {
		t.Errorf("Expected URL %s, got %s", testURL, content.URL)
	}

	if content.Title == "" {
		t.Error("Expected title to be extracted")
	}

	if content.CleanText == "" {
		t.Error("Expected clean text to be extracted")
	}

	if content.ScrapedAt.IsZero() {
		t.Error("Expected scraped_at timestamp to be set")
	}
}

func TestScraperFactory(t *testing.T) {
	config := DefaultScraperConfig()
	scraper, err := NewScraper(config)
	if err != nil {
		t.Fatalf("Failed to create scraper: %v", err)
	}

	if scraper == nil {
		t.Fatal("Expected scraper to be created")
	}

	_, ok := scraper.(*HTMLScraper)
	if !ok {
		t.Error("Expected HTML scraper to be created by default")
	}
}

func TestDefaultScrapeOptions(t *testing.T) {
	options := DefaultScrapeOptions()

	if options.UserAgent == "" {
		t.Error("Expected default user agent to be set")
	}

	if options.Timeout == 0 {
		t.Error("Expected default timeout to be set")
	}

	if options.MaxRetries == 0 {
		t.Error("Expected default max retries to be set")
	}

	if !options.FollowRedirects {
		t.Error("Expected follow redirects to be true by default")
	}
}
