package storage

import (
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {
	// Create temporary database for testing
	dbPath := "file:test_bookmarks.db"
	defer os.Remove("test_bookmarks.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	t.Run("AddBookmark", testAddBookmark(store))
	t.Run("GetBookmark", testGetBookmark(store))
	t.Run("ListBookmarksReturnsEmpty", testListBookmarksEmpty(store))
	t.Run("UpdateBookmarkStatus", testUpdateBookmarkStatus(store))
	t.Run("StoreAndGetContent", testStoreAndGetContent(store))
	t.Run("StoreAndGetEmbedding", testStoreAndGetEmbedding(store))
	t.Run("HybridSearch", testHybridSearch(store))
	t.Run("BatchOperations", testBatchOperations(store))
	t.Run("GetStats", testGetStats(store))
	t.Run("SearchWithFilters", testSearchWithFilters(store))
	t.Run("DeleteBookmark", testDeleteBookmark(store))
	t.Run("ErrorHandling", testErrorHandling(store))
}

func testAddBookmark(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		err := store.AddBookmark("https://example.com", "Example Site")
		if err != nil {
			t.Errorf("Failed to add bookmark: %v", err)
		}

		// Test duplicate URL (should fail due to UNIQUE constraint)
		err = store.AddBookmark("https://example.com", "Duplicate Site")
		if err == nil {
			t.Error("Expected error for duplicate URL, but got none")
		}
	}
}

func testGetBookmark(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Add a bookmark first
		err := store.AddBookmark("https://test.com", "Test Site")
		if err != nil {
			t.Fatalf("Failed to add bookmark: %v", err)
		}

		// Get the bookmark
		bookmark, err := store.GetBookmark(1)
		if err != nil {
			t.Errorf("Failed to get bookmark: %v", err)
		}

		if bookmark.URL != "https://test.com" {
			t.Errorf("Expected URL 'https://test.com', got '%s'", bookmark.URL)
		}

		if bookmark.Title != "Test Site" {
			t.Errorf("Expected title 'Test Site', got '%s'", bookmark.Title)
		}

		if bookmark.Status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", bookmark.Status)
		}
	}
}

func testListBookmarksEmpty(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Create a fresh database for this test
		tempStore, err := New("file:empty_test.db")
		if err != nil {
			t.Fatalf("Failed to create temp storage: %v", err)
		}
		defer tempStore.Close()
		defer os.Remove("empty_test.db")

		bookmarks, err := tempStore.ListBookmarksWithoutEmbeddings(10)
		if err != nil {
			t.Errorf("Failed to list bookmarks: %v", err)
		}

		if len(bookmarks) != 0 {
			t.Errorf("Expected 0 bookmarks, got %d", len(bookmarks))
		}
	}
}

func testUpdateBookmarkStatus(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Add a bookmark first
		err := store.AddBookmark("https://status-test.com", "Status Test")
		if err != nil {
			t.Fatalf("Failed to add bookmark: %v", err)
		}

		// Update status
		err = store.UpdateBookmarkStatus(1, "completed")
		if err != nil {
			t.Errorf("Failed to update bookmark status: %v", err)
		}

		// Verify status was updated
		bookmark, err := store.GetBookmark(1)
		if err != nil {
			t.Errorf("Failed to get bookmark: %v", err)
		}

		if bookmark.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", bookmark.Status)
		}
	}
}

func testStoreAndGetContent(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Add a bookmark first
		err := store.AddBookmark("https://content-test.com", "Content Test")
		if err != nil {
			t.Fatalf("Failed to add bookmark: %v", err)
		}

		rawContent := "<html><body>Test content</body></html>"
		cleanText := "Test content"

		// Store content
		err = store.StoreContent(1, rawContent, cleanText)
		if err != nil {
			t.Errorf("Failed to store content: %v", err)
		}

		// Get content
		content, err := store.GetContent(1)
		if err != nil {
			t.Errorf("Failed to get content: %v", err)
		}

		if content.RawContent != rawContent {
			t.Errorf("Expected raw content '%s', got '%s'", rawContent, content.RawContent)
		}

		if content.CleanText != cleanText {
			t.Errorf("Expected clean text '%s', got '%s'", cleanText, content.CleanText)
		}
	}
}

func testStoreAndGetEmbedding(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Add bookmark and content first
		err := store.AddBookmark("https://embedding-test.com", "Embedding Test")
		if err != nil {
			t.Fatalf("Failed to add bookmark: %v", err)
		}

		err = store.StoreContent(1, "<html><body>Embedding test</body></html>", "Embedding test")
		if err != nil {
			t.Fatalf("Failed to store content: %v", err)
		}

		// Generate test embedding
		embedding := make([]float32, 1536)
		for i := range embedding {
			embedding[i] = rand.Float32()
		}

		// Store embedding
		err = store.StoreEmbedding(1, embedding)
		if err != nil {
			t.Errorf("Failed to store embedding: %v", err)
		}

		// Get embedding
		retrievedEmbedding, err := store.GetEmbedding(1)
		if err != nil {
			t.Errorf("Failed to get embedding: %v", err)
		}

		if len(retrievedEmbedding) != len(embedding) {
			t.Errorf("Expected embedding length %d, got %d", len(embedding), len(retrievedEmbedding))
		}

		// Check a few values (due to float precision, we'll check approximate equality)
		for i := 0; i < 10; i++ {
			diff := embedding[i] - retrievedEmbedding[i]
			if diff < -0.001 || diff > 0.001 {
				t.Errorf("Embedding value mismatch at index %d: expected %f, got %f", i, embedding[i], retrievedEmbedding[i])
			}
		}
	}
}

func testHybridSearch(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Setup test data
		testBookmarks := []struct {
			URL     string
			Title   string
			Content string
		}{
			{"https://go1.com", "Go Programming", "Go is a programming language developed by Google"},
			{"https://python1.com", "Python Guide", "Python is a high-level programming language"},
			{"https://js1.com", "JavaScript Tutorial", "JavaScript is a scripting language for web development"},
		}

		for i, bookmark := range testBookmarks {
			err := store.AddBookmark(bookmark.URL, bookmark.Title)
			if err != nil {
				t.Fatalf("Failed to add bookmark %d: %v", i, err)
			}

			err = store.StoreContent(i+1, "<html><body>"+bookmark.Content+"</body></html>", bookmark.Content)
			if err != nil {
				t.Fatalf("Failed to store content %d: %v", i, err)
			}

			// Generate mock embedding
			embedding := make([]float32, 1536)
			for j := range embedding {
				embedding[j] = rand.Float32()
			}

			err = store.StoreEmbedding(i+1, embedding)
			if err != nil {
				t.Fatalf("Failed to store embedding %d: %v", i, err)
			}
		}

		// Perform search
		queryEmbedding := make([]float32, 1536)
		for i := range queryEmbedding {
			queryEmbedding[i] = rand.Float32()
		}

		results, err := store.HybridSearch(queryEmbedding, "programming language")
		if err != nil {
			t.Errorf("Hybrid search failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected search results, got none")
		}

		// Verify result structure
		for _, result := range results {
			if result.Bookmark == nil {
				t.Error("Search result missing bookmark")
			}
			if result.RelevanceScore <= 0 {
				t.Error("Search result has invalid relevance score")
			}
			if result.SearchType == "" {
				t.Error("Search result missing search type")
			}
		}
	}
}

func testBatchOperations(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		batchOps := store.NewBatchOperations()

		bookmarks := []struct {
			URL   string
			Title string
		}{
			{"https://batch1.com", "Batch Test 1"},
			{"https://batch2.com", "Batch Test 2"},
			{"https://batch3.com", "Batch Test 3"},
		}

		err := batchOps.BatchAddBookmarks(bookmarks)
		if err != nil {
			t.Errorf("Batch add bookmarks failed: %v", err)
		}

		// Verify bookmarks were added
		allBookmarks, err := store.ListBookmarks()
		if err != nil {
			t.Errorf("Failed to list bookmarks: %v", err)
		}

		if len(allBookmarks) < len(bookmarks) {
			t.Errorf("Expected at least %d bookmarks, got %d", len(bookmarks), len(allBookmarks))
		}
	}
}

func testGetStats(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		stats, err := store.GetStats()
		if err != nil {
			t.Errorf("Failed to get stats: %v", err)
		}

		expectedStats := []string{
			"total_bookmarks",
			"completed_bookmarks",
			"pending_bookmarks",
			"failed_bookmarks",
			"total_content_entries",
			"total_embeddings",
			"bookmarks_with_content",
			"bookmarks_with_embeddings",
		}

		for _, statName := range expectedStats {
			if _, exists := stats[statName]; !exists {
				t.Errorf("Missing stat: %s", statName)
			}
		}

		if stats["total_bookmarks"] < 0 {
			t.Error("Total bookmarks should not be negative")
		}
	}
}

func testSearchWithFilters(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Add a bookmark with specific status
		err := store.AddBookmark("https://filter-test.com", "Filter Test")
		if err != nil {
			t.Fatalf("Failed to add bookmark: %v", err)
		}

		err = store.UpdateBookmarkStatus(1, "completed")
		if err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		// Search with filters
		opts := SearchOptions{
			Status: "completed",
			Limit:  10,
		}

		results, err := store.SearchBookmarksWithFilters(opts)
		if err != nil {
			t.Errorf("Filtered search failed: %v", err)
		}

		// Verify all results have the correct status
		for _, result := range results {
			if result.Bookmark.Status != "completed" {
				t.Errorf("Expected status 'completed', got '%s'", result.Bookmark.Status)
			}
		}
	}
}

func testDeleteBookmark(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Add a bookmark to delete
		err := store.AddBookmark("https://delete-test.com", "Delete Test")
		if err != nil {
			t.Fatalf("Failed to add bookmark: %v", err)
		}

		// Delete the bookmark
		err = store.DeleteBookmark(1)
		if err != nil {
			t.Errorf("Failed to delete bookmark: %v", err)
		}

		// Verify bookmark is deleted
		_, err = store.GetBookmark(1)
		if err == nil {
			t.Error("Expected error when getting deleted bookmark, but got none")
		}
	}
}

func testErrorHandling(store *Storage) func(*testing.T) {
	return func(t *testing.T) {
		// Test getting non-existent bookmark
		_, err := store.GetBookmark(9999)
		if err == nil {
			t.Error("Expected error for non-existent bookmark, got none")
		}

		// Test updating non-existent bookmark
		err = store.UpdateBookmarkStatus(9999, "completed")
		if err == nil {
			t.Error("Expected error for non-existent bookmark update, got none")
		}

		// Test getting content for non-existent bookmark
		_, err = store.GetContent(9999)
		if err == nil {
			t.Error("Expected error for non-existent content, got none")
		}

		// Test getting embedding for non-existent content
		_, err = store.GetEmbedding(9999)
		if err == nil {
			t.Error("Expected error for non-existent embedding, got none")
		}

		// Test deleting non-existent bookmark
		err = store.DeleteBookmark(9999)
		if err == nil {
			t.Error("Expected error for deleting non-existent bookmark, got none")
		}
	}
}

func BenchmarkAddBookmark(b *testing.B) {
	store, err := New("file:benchmark.db")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()
	defer os.Remove("benchmark.db")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.AddBookmark("https://example.com/"+string(rune(i)), "Benchmark Test "+string(rune(i)))
	}
}

func BenchmarkHybridSearch(b *testing.B) {
	store, err := New("file:search_benchmark.db")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()
	defer os.Remove("search_benchmark.db")

	// Setup test data
	for i := 0; i < 100; i++ {
		store.AddBookmark("https://example.com/"+string(rune(i)), "Test Bookmark "+string(rune(i)))
		store.StoreContent(i+1, "<html><body>Test content "+string(rune(i))+"</body></html>", "Test content "+string(rune(i)))

		embedding := make([]float32, 1536)
		for j := range embedding {
			embedding[j] = rand.Float32()
		}
		store.StoreEmbedding(i+1, embedding)
	}

	queryEmbedding := make([]float32, 1536)
	for i := range queryEmbedding {
		queryEmbedding[i] = rand.Float32()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.HybridSearch(queryEmbedding, "test content")
	}
}
