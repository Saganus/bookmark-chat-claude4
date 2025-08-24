package services

import (
	"context"
	"fmt"
	"os"

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
