package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"bookmark-chat/internal/storage"
	"github.com/sashabaranov/go-openai"
)

// CategorizationService handles AI-powered bookmark categorization
type CategorizationService struct {
	storage      *storage.Storage
	openaiClient *openai.Client
	model        string
}

// Message represents a chat message for OpenAI API
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewCategorizationService creates a new categorization service
func NewCategorizationService(storage *storage.Storage) (*CategorizationService, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required for categorization")
	}

	client := openai.NewClient(apiKey)
	model := os.Getenv("CATEGORIZATION_MODEL")
	if model == "" {
		model = "gpt-4o-mini" // Cost-effective model for categorization
	}

	return &CategorizationService{
		storage:      storage,
		openaiClient: client,
		model:        model,
	}, nil
}

// CategorizeBookmark uses GPT to categorize a bookmark based on its content
func (cs *CategorizationService) CategorizeBookmark(ctx context.Context, bookmarkID string) (*storage.CategorizationResult, error) {
	// Get bookmark with content
	bookmark, err := cs.storage.GetBookmarkWithContent(ctx, bookmarkID)
	if err != nil {
		return nil, fmt.Errorf("get bookmark: %w", err)
	}

	// Get existing categories for context
	categories, err := cs.storage.GetCategories(ctx)
	if err != nil {
		// Continue without existing categories
		categories = []storage.Category{}
	}

	// Build categorization prompt
	prompt := cs.buildCategorizationPrompt(bookmark, categories)

	// Call OpenAI API
	response, err := cs.createChatCompletion(ctx, []Message{
		{Role: "system", Content: "You are an expert bookmark categorization assistant. Analyze the provided bookmark and respond only with valid JSON in the exact format requested. Do not include any explanatory text outside the JSON."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("openai completion: %w", err)
	}

	// Parse response
	var result storage.CategorizationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("parse categorization response: %w", err)
	}

	// Validate result
	if result.PrimaryCategory == "" {
		return nil, fmt.Errorf("categorization result missing primary category")
	}
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 1 {
		result.ConfidenceScore = 0.5 // Default confidence
	}

	// Save categorization result
	if err := cs.storage.SaveCategorizationResult(ctx, bookmarkID, result); err != nil {
		return nil, fmt.Errorf("save categorization: %w", err)
	}

	return &result, nil
}

// BulkCategorize processes multiple bookmarks with rate limiting
func (cs *CategorizationService) BulkCategorize(ctx context.Context, bookmarkIDs []string, autoApply bool, confidenceThreshold float64) ([]storage.CategorizationResult, error) {
	results := make([]storage.CategorizationResult, 0, len(bookmarkIDs))
	appliedCount := 0
	
	// Rate limiting: 30 requests per minute for OpenAI API
	rateLimiter := time.NewTicker(2 * time.Second) // ~30 per minute
	defer rateLimiter.Stop()

	for i, id := range bookmarkIDs {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-rateLimiter.C:
			// Rate limited - proceed with request
		}

		result, err := cs.CategorizeBookmark(ctx, id)
		if err != nil {
			// Log error but continue with other bookmarks
			fmt.Printf("Failed to categorize bookmark %s: %v\n", id, err)
			continue
		}
		
		results = append(results, *result)
		
		// Auto-apply if confidence is high enough
		if autoApply && result.ConfidenceScore >= confidenceThreshold {
			if err := cs.storage.ApproveCategorizationResult(ctx, id); err != nil {
				fmt.Printf("Failed to approve categorization for bookmark %s: %v\n", id, err)
			} else {
				appliedCount++
			}
		}

		// Progress logging
		if (i+1)%5 == 0 || i == len(bookmarkIDs)-1 {
			fmt.Printf("Categorized %d/%d bookmarks (applied: %d)\n", i+1, len(bookmarkIDs), appliedCount)
		}
	}
	
	return results, nil
}

// createChatCompletion creates a chat completion using OpenAI API
func (cs *CategorizationService) createChatCompletion(ctx context.Context, messages []Message) (string, error) {
	// Convert our Message type to OpenAI's ChatCompletionMessage
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := openai.ChatCompletionRequest{
		Model:       cs.model,
		Messages:    openaiMessages,
		MaxTokens:   500,
		Temperature: 0.3, // Lower temperature for more consistent categorization
		TopP:        0.9,
	}

	resp, err := cs.openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("chat completion request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// buildCategorizationPrompt creates a detailed prompt for categorization
func (cs *CategorizationService) buildCategorizationPrompt(bookmark *storage.Bookmark, existingCategories []storage.Category) string {
	// Build list of existing category names
	var categoryNames []string
	for _, cat := range existingCategories {
		if cat.ParentCategory != nil {
			categoryNames = append(categoryNames, fmt.Sprintf("%s/%s", *cat.ParentCategory, cat.Name))
		} else {
			categoryNames = append(categoryNames, cat.Name)
		}
	}

	// Limit existing categories to most relevant ones
	if len(categoryNames) > 20 {
		categoryNames = categoryNames[:20]
	}

	existingCategoriesStr := "None"
	if len(categoryNames) > 0 {
		existingCategoriesStr = strings.Join(categoryNames, ", ")
	}

	// Truncate content for prompt efficiency
	content := bookmark.Description
	if len(content) > 2000 {
		content = content[:2000] + "..."
	}

	prompt := fmt.Sprintf(`Analyze this bookmark and suggest categories:

URL: %s
Title: %s
Folder Path: %s
Content: %s

Existing categories in the system: %s

Generate a JSON response with this exact structure:
{
    "primary_category": "Most relevant category name",
    "secondary_categories": ["Secondary category 1", "Secondary category 2"],
    "tags": ["specific", "descriptive", "tags", "for", "searching"],
    "confidence_score": 0.85,
    "reasoning": "Brief explanation"
}

Guidelines:
- Use existing categories when appropriate, or create new ones when necessary
- Primary category should be the most relevant single category
- Secondary categories should be 0-3 additional relevant categories
- Tags should be 3-8 specific keywords from the content for enhanced searchability
- Confidence score between 0 and 1 based on how clear the categorization is
- Keep category names 2-4 words maximum
- Reasoning should be 1-2 sentences explaining the categorization
- Focus on the main topic/domain of the bookmark

Common category examples: Technology, Programming, Web Development, Data Science, Business, Finance, Education, Health, News, Entertainment, Travel, Food, Sports, Art, Science, Documentation, Tools, Tutorials, Reference`,
		bookmark.URL,
		bookmark.Title,
		bookmark.FolderPath,
		content,
		existingCategoriesStr)

	return prompt
}

// GetUncategorizedBookmarks returns bookmarks that need categorization
func (cs *CategorizationService) GetUncategorizedBookmarks(ctx context.Context, limit int) ([]string, error) {
	return cs.storage.GetBookmarksNeedingCategorization(ctx, limit)
}