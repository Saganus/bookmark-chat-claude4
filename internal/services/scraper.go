package services

import (
	"context"
	"time"
)

type ScrapedContent struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	CleanText   string            `json:"clean_text"`
	Description string            `json:"description"`
	FaviconURL  string            `json:"favicon_url"`
	Headers     map[string]string `json:"headers"`
	ScrapedAt   time.Time         `json:"scraped_at"`
	Success     bool              `json:"success"`
	Error       string            `json:"error,omitempty"`
}

type ScrapeOptions struct {
	UserAgent       string        `json:"user_agent"`
	Timeout         time.Duration `json:"timeout"`
	FollowRedirects bool          `json:"follow_redirects"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	ExtractImages   bool          `json:"extract_images"`
	ExtractLinks    bool          `json:"extract_links"`
}

type Scraper interface {
	Scrape(ctx context.Context, url string, options ScrapeOptions) (*ScrapedContent, error)
	ScrapeMultiple(ctx context.Context, urls []string, options ScrapeOptions) ([]*ScrapedContent, error)
	SetRateLimit(requestsPerSecond float64)
}

type ScraperType string

const (
	ScraperTypeHTML      ScraperType = "html"
	ScraperTypeFirecrawl ScraperType = "firecrawl"
)

func DefaultScrapeOptions() ScrapeOptions {
	return ScrapeOptions{
		UserAgent:       "BookmarkChat/1.0 (+https://github.com/user/bookmark-chat)",
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		ExtractImages:   false,
		ExtractLinks:    false,
	}
}
