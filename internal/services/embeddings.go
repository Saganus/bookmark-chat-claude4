package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingService handles generating embeddings via OpenAI API
type EmbeddingService struct {
	client *openai.Client
	model  string
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService() (*EmbeddingService, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	client := openai.NewClient(apiKey)

	return &EmbeddingService{
		client: client,
		model:  "text-embedding-3-small", // 1536 dimensions, optimized for retrieval
	}, nil
}

// GenerateEmbedding creates an embedding for the given text
func (es *EmbeddingService) GenerateEmbedding(text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	resp, err := es.client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(es.model),
		Input: []string{text},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

// GenerateBatchEmbeddings creates embeddings for multiple texts in a single API call
func (es *EmbeddingService) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// OpenAI has limits on batch size, so split if needed
	const maxBatchSize = 2048
	if len(texts) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(texts), maxBatchSize)
	}

	resp, err := es.client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(es.model),
		Input: texts,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create batch embeddings: %w", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// GetModelInfo returns information about the embedding model being used
func (es *EmbeddingService) GetModelInfo() (string, int) {
	// text-embedding-3-small has 1536 dimensions
	return es.model, 1536
}

// estimateTokenCount provides a rough estimate of token count for text
// OpenAI's rule of thumb: ~4 characters per token for English text
func (es *EmbeddingService) estimateTokenCount(text string) int {
	return utf8.RuneCountInString(text) / 4
}

// ChunkText splits text into chunks using recursive character splitting
// This is optimized for web content with HTML-aware separators
func (es *EmbeddingService) ChunkText(text string, maxTokens int) []string {
	if text == "" {
		return []string{}
	}

	// If text is already small enough, return as-is
	if es.estimateTokenCount(text) <= maxTokens {
		return []string{text}
	}

	// HTML-aware separators in order of preference
	separators := []string{
		"\n\n", // Paragraph breaks
		"\n",   // Line breaks
		". ",   // Sentence endings
		"! ",   // Exclamation endings
		"? ",   // Question endings
		"; ",   // Semicolon breaks
		", ",   // Comma breaks
		" ",    // Word breaks
		"",     // Character breaks (last resort)
	}

	return es.recursiveSplit(text, separators, maxTokens)
}

// recursiveSplit implements the recursive character splitting algorithm
func (es *EmbeddingService) recursiveSplit(text string, separators []string, maxTokens int) []string {
	// Base case: if text is small enough, return it
	if es.estimateTokenCount(text) <= maxTokens {
		return []string{strings.TrimSpace(text)}
	}

	// Try each separator in order
	for _, separator := range separators {
		if separator == "" {
			// Last resort: split by character count
			return es.splitByCharacterCount(text, maxTokens)
		}

		if strings.Contains(text, separator) {
			parts := strings.Split(text, separator)
			return es.mergeParts(parts, separator, separators, maxTokens)
		}
	}

	// Should never reach here, but handle gracefully
	return es.splitByCharacterCount(text, maxTokens)
}

// mergeParts combines split parts while respecting token limits
func (es *EmbeddingService) mergeParts(parts []string, separator string, separators []string, maxTokens int) []string {
	var result []string
	var currentChunk string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to add this part to current chunk
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += separator
		}
		testChunk += part

		if es.estimateTokenCount(testChunk) <= maxTokens {
			// Part fits, add it to current chunk
			currentChunk = testChunk
		} else {
			// Part doesn't fit
			if currentChunk != "" {
				// Save current chunk and start new one
				result = append(result, strings.TrimSpace(currentChunk))
				currentChunk = part
			} else {
				// Even single part is too big, split it recursively
				subChunks := es.recursiveSplit(part, separators, maxTokens)
				result = append(result, subChunks...)
				currentChunk = ""
			}
		}
	}

	// Don't forget the last chunk
	if currentChunk != "" {
		result = append(result, strings.TrimSpace(currentChunk))
	}

	return result
}

// splitByCharacterCount splits text by approximate character count
func (es *EmbeddingService) splitByCharacterCount(text string, maxTokens int) []string {
	// Estimate max characters based on token limit
	maxChars := maxTokens * 4 // 4 chars per token estimate

	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += maxChars {
		end := i + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		chunks = append(chunks, strings.TrimSpace(chunk))
	}

	return chunks
}

// GenerateEmbeddingWithChunking generates embeddings for text, chunking if necessary
func (es *EmbeddingService) GenerateEmbeddingWithChunking(text string) ([][]float32, []string, error) {
	// Split text into chunks
	chunks := es.ChunkText(text, 6000) // Conservative limit under 8192

	if len(chunks) == 0 {
		return nil, nil, fmt.Errorf("no chunks generated from text")
	}

	// Generate embeddings for all chunks
	embeddings, err := es.GenerateBatchEmbeddings(chunks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate chunk embeddings: %w", err)
	}

	return embeddings, chunks, nil
}
