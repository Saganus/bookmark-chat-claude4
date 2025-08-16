package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

type HTMLScraper struct {
	client      *http.Client
	rateLimiter *rate.Limiter
	mu          sync.RWMutex
}

func NewHTMLScraper() *HTMLScraper {
	return &HTMLScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: rate.NewLimiter(rate.Limit(2.0), 1),
	}
}

func (s *HTMLScraper) SetRateLimit(requestsPerSecond float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimiter = rate.NewLimiter(rate.Limit(requestsPerSecond), 1)
}

func (s *HTMLScraper) Scrape(ctx context.Context, url string, options ScrapeOptions) (*ScrapedContent, error) {
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(options.RetryDelay):
			}
		}

		content, err := s.scrapeOnce(ctx, url, options)
		if err == nil {
			return content, nil
		}
		lastErr = err
	}

	return &ScrapedContent{
		URL:       url,
		Success:   false,
		Error:     lastErr.Error(),
		ScrapedAt: time.Now(),
	}, lastErr
}

func (s *HTMLScraper) ScrapeMultiple(ctx context.Context, urls []string, options ScrapeOptions) ([]*ScrapedContent, error) {
	results := make([]*ScrapedContent, len(urls))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for i, url := range urls {
		wg.Add(1)
		go func(index int, u string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, _ := s.Scrape(ctx, u, options)
			results[index] = result
		}(i, url)
	}

	wg.Wait()
	return results, nil
}

func (s *HTMLScraper) scrapeOnce(ctx context.Context, url string, options ScrapeOptions) (*ScrapedContent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", options.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	if !options.FollowRedirects {
		s.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("non-HTML content type: %s", contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	content := s.extractContent(doc, url)
	content.URL = url
	content.ScrapedAt = time.Now()
	content.Success = true

	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	content.Headers = headers

	return content, nil
}

func (s *HTMLScraper) extractContent(doc *goquery.Document, baseURL string) *ScrapedContent {
	content := &ScrapedContent{}

	content.Title = s.extractTitle(doc)
	content.Description = s.extractDescription(doc)
	content.FaviconURL = s.extractFavicon(doc, baseURL)
	content.Content = s.extractMainContent(doc)
	content.CleanText = s.cleanText(content.Content)

	return content
}

func (s *HTMLScraper) extractTitle(doc *goquery.Document) string {
	if title := doc.Find("meta[property='og:title']").AttrOr("content", ""); title != "" {
		return strings.TrimSpace(title)
	}
	if title := doc.Find("meta[name='twitter:title']").AttrOr("content", ""); title != "" {
		return strings.TrimSpace(title)
	}
	if title := doc.Find("title").Text(); title != "" {
		return strings.TrimSpace(title)
	}
	if title := doc.Find("h1").First().Text(); title != "" {
		return strings.TrimSpace(title)
	}
	return ""
}

func (s *HTMLScraper) extractDescription(doc *goquery.Document) string {
	if desc := doc.Find("meta[property='og:description']").AttrOr("content", ""); desc != "" {
		return strings.TrimSpace(desc)
	}
	if desc := doc.Find("meta[name='twitter:description']").AttrOr("content", ""); desc != "" {
		return strings.TrimSpace(desc)
	}
	if desc := doc.Find("meta[name='description']").AttrOr("content", ""); desc != "" {
		return strings.TrimSpace(desc)
	}
	return ""
}

func (s *HTMLScraper) extractFavicon(doc *goquery.Document, baseURL string) string {
	selectors := []string{
		"link[rel='icon']",
		"link[rel='shortcut icon']",
		"link[rel='apple-touch-icon']",
		"link[rel='apple-touch-icon-precomposed']",
	}

	for _, selector := range selectors {
		if href := doc.Find(selector).AttrOr("href", ""); href != "" {
			if strings.HasPrefix(href, "http") {
				return href
			}
			if strings.HasPrefix(href, "//") {
				return "https:" + href
			}
			if strings.HasPrefix(href, "/") {
				return baseURL + href
			}
			return baseURL + "/" + href
		}
	}

	return baseURL + "/favicon.ico"
}

func (s *HTMLScraper) extractMainContent(doc *goquery.Document) string {
	removeSelectors := []string{
		"script", "style", "nav", "header", "footer", "aside",
		".sidebar", ".navigation", ".menu", ".ads", ".advertisement",
		".social", ".share", ".comments", ".popup", ".modal",
	}

	for _, selector := range removeSelectors {
		doc.Find(selector).Remove()
	}

	mainSelectors := []string{
		"main", "article", ".content", ".main-content", ".post-content",
		".entry-content", ".article-content", "#content", "#main",
	}

	for _, selector := range mainSelectors {
		if content := doc.Find(selector).First(); content.Length() > 0 {
			return strings.TrimSpace(content.Text())
		}
	}

	doc.Find("header, nav, footer, aside").Remove()
	return strings.TrimSpace(doc.Find("body").Text())
}

func (s *HTMLScraper) cleanText(text string) string {
	whitespaceRegex := regexp.MustCompile(`\s+`)
	text = whitespaceRegex.ReplaceAllString(text, " ")

	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) > 3 {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}
