package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"bookmark-chat/internal/services"
	"bookmark-chat/internal/services/parsers"
	"bookmark-chat/internal/storage"
)

func main() {
	// Check if API key is set
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable must be set")
	}

	fmt.Println("=== OpenAI Embeddings Test ===\n")

	// Initialize storage
	store, err := storage.New("file:test_embeddings.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize content processor (includes embedding service)
	processor, err := services.NewContentProcessor(store)
	if err != nil {
		log.Fatalf("Failed to create content processor: %v", err)
	}

	fmt.Println("1. Testing embedding generation...")

	// Test embedding generation
	testTexts := []string{
		"Go is a programming language developed by Google",
		"Python is great for data science and machine learning",
		"JavaScript runs in web browsers and Node.js",
		"Database algorithms are essential for efficient queries",
	}

	for i, text := range testTexts {
		fmt.Printf("Generating embedding for text %d...\n", i+1)

		embedding, err := processor.GenerateQueryEmbedding(text)
		if err != nil {
			log.Printf("Failed to generate embedding: %v", err)
			continue
		}

		fmt.Printf("✓ Generated embedding with %d dimensions\n", len(embedding))

		// Insert a test bookmark using the proper parser types
		result, err := store.ImportBookmarks(&parsers.ParseResult{
			Source:   "test",
			ParsedAt: time.Now(),
			Bookmarks: []parsers.Bookmark{{
				URL:       fmt.Sprintf("https://example.com/test-%d", i+1),
				Title:     fmt.Sprintf("Test Article %d", i+1),
				DateAdded: time.Now(),
			}},
			TotalCount: 1,
		})
		if err == nil && len(result.ImportedBookmarks) > 0 {
			bookmarkID := result.ImportedBookmarks[0].ID

			// Store content
			err = store.StoreContent(bookmarkID, fmt.Sprintf("<html><body>%s</body></html>", text), text)
			if err == nil {
				// Get content to get the content ID
				content, err := store.GetContent(bookmarkID)
				if err == nil {
					// Store embedding
					err = store.StoreEmbedding(content.ID, embedding)
					if err == nil {
						fmt.Printf("✓ Stored content and embedding for test %d\n", i+1)
					} else {
						log.Printf("Failed to store embedding: %v", err)
					}
				} else {
					log.Printf("Failed to get content: %v", err)
				}
			} else {
				log.Printf("Failed to store content: %v", err)
			}
		} else {
			log.Printf("Failed to import bookmark: %v", err)
		}
	}

	fmt.Println("\n2. Testing semantic search...")

	// Test semantic search
	query := "algorithm"
	fmt.Printf("Searching for: '%s'\n", query)

	results, err := processor.HybridSearch(query)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d results:\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. %s (Score: %.3f, Type: %s)\n",
				i+1, result.Bookmark.Title, result.RelevanceScore, result.SearchType)
			if result.Content != nil && len(result.Content.CleanText) > 0 {
				text := result.Content.CleanText
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				fmt.Printf("     Content: %s\n", text)
			}
		}
	}

	fmt.Println("\n3. Testing with exact SQL query...")

	// Test the raw SQL query
	queryEmbedding, err := processor.GenerateQueryEmbedding(query)
	if err != nil {
		log.Printf("Failed to generate query embedding: %v", err)
	} else {
		fmt.Printf("✓ Generated query embedding with %d dimensions\n", len(queryEmbedding))

		// Test semantic search directly
		semanticResults, err := store.HybridSearch(queryEmbedding, query)
		if err != nil {
			log.Printf("Direct semantic search failed: %v", err)
		} else {
			fmt.Printf("Direct search found %d results\n", len(semanticResults))
		}
	}

	fmt.Println("\n=== Test Complete ===")
	fmt.Println("Database file created: test_embeddings.db")
	fmt.Println("Clean up by running: rm test_embeddings.db")
}
