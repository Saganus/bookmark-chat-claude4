package services

import (
	"context"
	"fmt"
)

type FirecrawlScraper struct {
	apiKey  string
	baseURL string
}

func NewFirecrawlScraper(apiKey string) *FirecrawlScraper {
	return &FirecrawlScraper{
		apiKey:  apiKey,
		baseURL: "https://api.firecrawl.dev/v1",
	}
}

func (f *FirecrawlScraper) Scrape(ctx context.Context, url string, options ScrapeOptions) (*ScrapedContent, error) {
	return nil, fmt.Errorf("firecrawl scraper not implemented yet")
}

func (f *FirecrawlScraper) ScrapeMultiple(ctx context.Context, urls []string, options ScrapeOptions) ([]*ScrapedContent, error) {
	return nil, fmt.Errorf("firecrawl scraper not implemented yet")
}

func (f *FirecrawlScraper) SetRateLimit(requestsPerSecond float64) {
}

type ScraperConfig struct {
	Type            ScraperType `json:"type"`
	FirecrawlAPIKey string      `json:"firecrawl_api_key,omitempty"`
	RateLimitRPS    float64     `json:"rate_limit_rps"`
}

func NewScraper(config ScraperConfig) (Scraper, error) {
	switch config.Type {
	case ScraperTypeHTML:
		scraper := NewHTMLScraper()
		if config.RateLimitRPS > 0 {
			scraper.SetRateLimit(config.RateLimitRPS)
		}
		return scraper, nil
	case ScraperTypeFirecrawl:
		if config.FirecrawlAPIKey == "" {
			return nil, fmt.Errorf("firecrawl API key is required")
		}
		scraper := NewFirecrawlScraper(config.FirecrawlAPIKey)
		if config.RateLimitRPS > 0 {
			scraper.SetRateLimit(config.RateLimitRPS)
		}
		return scraper, nil
	default:
		return nil, fmt.Errorf("unsupported scraper type: %s", config.Type)
	}
}

func DefaultScraperConfig() ScraperConfig {
	return ScraperConfig{
		Type:         ScraperTypeHTML,
		RateLimitRPS: 2.0,
	}
}
