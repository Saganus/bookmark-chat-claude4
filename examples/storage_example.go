package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/bookmark-chat-claude4/internal/storage"
)

func main() {
	// Initialize storage with a local database file
	store, err := storage.New("file:example_bookmarks.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	fmt.Println("=== Bookmark Storage Layer Example ===\n")

	// 1. Add some sample bookmarks
	fmt.Println("1. Adding sample bookmarks...")
	bookmarks := []struct {
		URL   string
		Title string
	}{
		{"https://golang.org", "The Go Programming Language"},
		{"https://echo.labstack.com", "Echo - High performance, minimalist Go web framework"},
		{"https://docs.turso.tech", "Turso Documentation"},
		{"https://openai.com/blog/embeddings", "OpenAI Embeddings"},
		{"https://sqlite.org/fts5.html", "SQLite FTS5 Extension"},
	}

	for _, bookmark := range bookmarks {
		err := store.AddBookmark(bookmark.URL, bookmark.Title)
		if err != nil {
			log.Printf("Failed to add bookmark %s: %v", bookmark.URL, err)
		} else {
			fmt.Printf("✓ Added: %s\n", bookmark.Title)
		}
	}

	// 2. List all bookmarks
	fmt.Println("\n2. Listing all bookmarks...")
	allBookmarks, err := store.ListBookmarks()
	if err != nil {
		log.Fatalf("Failed to list bookmarks: %v", err)
	}

	for _, bookmark := range allBookmarks {
		fmt.Printf("ID: %d, URL: %s, Title: %s, Status: %s\n",
			bookmark.ID, bookmark.URL, bookmark.Title, bookmark.Status)
	}

	// 3. Add content for some bookmarks
	fmt.Println("\n3. Adding sample content...")
	sampleContent := map[int]struct {
		Raw   string
		Clean string
	}{
		1: {
			Raw:   "<html><body><h1>Go Programming Language</h1><p>Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.</p></body></html>",
			Clean: "Go Programming Language. Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
		},
		2: {
			Raw:   "<html><body><h1>Echo Framework</h1><p>High performance, extensible, minimalist Go web framework.</p></body></html>",
			Clean: "Echo Framework. High performance, extensible, minimalist Go web framework.",
		},
		3: {
			Raw:   "<html><body><h1>Turso Documentation</h1><p>Turso is a SQLite-compatible database built on libSQL, offering edge replicas and vector extensions.</p></body></html>",
			Clean: "Turso Documentation. Turso is a SQLite-compatible database built on libSQL, offering edge replicas and vector extensions.",
		},
	}

	for bookmarkID, content := range sampleContent {
		err := store.StoreContent(bookmarkID, content.Raw, content.Clean)
		if err != nil {
			log.Printf("Failed to store content for bookmark %d: %v", bookmarkID, err)
		} else {
			fmt.Printf("✓ Stored content for bookmark %d\n", bookmarkID)
		}
	}

	// 4. Update bookmark statuses
	fmt.Println("\n4. Updating bookmark statuses...")
	for bookmarkID := range sampleContent {
		err := store.UpdateBookmarkStatus(bookmarkID, "completed")
		if err != nil {
			log.Printf("Failed to update status for bookmark %d: %v", bookmarkID, err)
		} else {
			fmt.Printf("✓ Updated status for bookmark %d to completed\n", bookmarkID)
		}
	}

	// 5. Generate and store sample embeddings
	fmt.Println("\n5. Generating and storing sample embeddings...")
	for bookmarkID := range sampleContent {
		content, err := store.GetContent(bookmarkID)
		if err != nil {
			log.Printf("Failed to get content for bookmark %d: %v", bookmarkID, err)
			continue
		}

		// Generate a mock embedding (in real usage, you'd use OpenAI API)
		embedding := generateMockEmbedding(1536)

		err = store.StoreEmbedding(content.ID, embedding)
		if err != nil {
			log.Printf("Failed to store embedding for content %d: %v", content.ID, err)
		} else {
			fmt.Printf("✓ Stored embedding for content %d (bookmark %d)\n", content.ID, bookmarkID)
		}
	}

	// 6. Perform hybrid search
	fmt.Println("\n6. Performing hybrid search...")

	// Mock query embedding
	queryEmbedding := generateMockEmbedding(1536)
	queryText := "Go programming language"

	results, err := store.HybridSearch(queryEmbedding, queryText)
	if err != nil {
		log.Printf("Hybrid search failed: %v", err)
	} else {
		fmt.Printf("Found %d results for query '%s':\n", len(results), queryText)
		for i, result := range results {
			fmt.Printf("  %d. %s (Score: %.3f, Type: %s)\n",
				i+1, result.Bookmark.Title, result.RelevanceScore, result.SearchType)
			if result.MatchedSnippet != "" {
				fmt.Printf("     Snippet: %s\n", result.MatchedSnippet)
			}
		}
	}

	// 7. Test batch operations
	fmt.Println("\n7. Testing batch operations...")
	batchOps := store.NewBatchOperations()

	newBookmarks := []struct {
		URL   string
		Title string
	}{
		{"https://example.com/1", "Example Site 1"},
		{"https://example.com/2", "Example Site 2"},
		{"https://example.com/3", "Example Site 3"},
	}

	err = batchOps.BatchAddBookmarks(newBookmarks)
	if err != nil {
		log.Printf("Batch add failed: %v", err)
	} else {
		fmt.Printf("✓ Batch added %d bookmarks\n", len(newBookmarks))
	}

	// 8. Get database statistics
	fmt.Println("\n8. Database statistics:")
	stats, err := store.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		for statName, value := range stats {
			fmt.Printf("  %s: %d\n", statName, value)
		}
	}

	// 9. Test filtered search
	fmt.Println("\n9. Testing filtered search...")
	searchOpts := storage.SearchOptions{
		Status: "completed",
		Limit:  10,
	}

	filteredResults, err := store.SearchBookmarksWithFilters(searchOpts)
	if err != nil {
		log.Printf("Filtered search failed: %v", err)
	} else {
		fmt.Printf("Found %d completed bookmarks:\n", len(filteredResults))
		for _, result := range filteredResults {
			fmt.Printf("  - %s (%s)\n", result.Bookmark.Title, result.Bookmark.URL)
		}
	}

	// 10. Demonstrate error handling
	fmt.Println("\n10. Testing error handling...")

	// Try to get a non-existent bookmark
	_, err = store.GetBookmark(9999)
	if err != nil {
		fmt.Printf("✓ Expected error for non-existent bookmark: %v\n", err)
	}

	// Try to update status of non-existent bookmark
	err = store.UpdateBookmarkStatus(9999, "completed")
	if err != nil {
		fmt.Printf("✓ Expected error for non-existent bookmark update: %v\n", err)
	}

	fmt.Println("\n=== Storage Layer Example Complete ===")
	fmt.Println("Database file created: example_bookmarks.db")
	fmt.Printf("Clean up by running: rm example_bookmarks.db\n")
}

// generateMockEmbedding creates a mock embedding vector for testing
func generateMockEmbedding(dimension int) []float32 {
	embedding := make([]float32, dimension)
	for i := range embedding {
		embedding[i] = rand.Float32()*2 - 1 // Random values between -1 and 1
	}
	return embedding
}

// Example of how to integrate with OpenAI for real embeddings
func generateRealEmbedding(text string, apiKey string) ([]float32, error) {
	// This is pseudocode - you would use the actual OpenAI Go client
	// client := openai.NewClient(apiKey)
	// resp, err := client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
	//     Model: openai.TextEmbedding3Small,
	//     Input: []string{text},
	// })
	// if err != nil {
	//     return nil, err
	// }
	// return resp.Data[0].Embedding, nil

	return nil, fmt.Errorf("not implemented - use actual OpenAI client")
}
