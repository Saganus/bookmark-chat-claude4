package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bookmark-chat/internal/services"
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

	fmt.Println("1. Testing embedding generation...")

	// Initialize embedding service directly
	embeddingService, err := services.NewEmbeddingService()
	if err != nil {
		log.Fatalf("Failed to create embedding service: %v", err)
	}

	// Test embedding generation
	testText := "This is a test about database algorithms and efficient query processing"
	fmt.Printf("Generating embedding for: '%s'\n", testText)

	embedding, err := embeddingService.GenerateEmbedding(testText)
	if err != nil {
		log.Fatalf("Failed to generate embedding: %v", err)
	}

	fmt.Printf("✓ Generated embedding with %d dimensions\n", len(embedding))
	fmt.Printf("First few values: [%.6f, %.6f, %.6f, ...]\n",
		embedding[0], embedding[1], embedding[2])

	fmt.Println("\n2. Testing query embedding...")

	query := "algorithm"
	fmt.Printf("Generating query embedding for: '%s'\n", query)

	queryEmbedding, err := embeddingService.GenerateEmbedding(query)
	if err != nil {
		log.Fatalf("Failed to generate query embedding: %v", err)
	}

	fmt.Printf("✓ Generated query embedding with %d dimensions\n", len(queryEmbedding))

	fmt.Println("\n3. Manual storage test with detailed logging...")

	// Check if libSQL vector functions are available
	fmt.Println("3.1 Testing vector function availability...")
	var vectorTest string
	err = store.GetDB().QueryRow("SELECT vector32('[1.0, 2.0, 3.0]')").Scan(&vectorTest)
	if err != nil {
		log.Printf("⚠️  vector32() function not available: %v", err)
		log.Printf("This might be a libSQL vector extension issue")
	} else {
		fmt.Printf("✓ vector32() function is available\n")
	}

	// Create a fake bookmark entry by directly inserting into the database
	fmt.Println("3.2 Inserting test bookmark...")
	result, err := store.GetDB().Exec(`
		INSERT INTO bookmarks (id, url, title, status, imported_at, created_at, updated_at)
		VALUES ('test-001', 'https://example.com/algorithm-guide', 'Algorithm Guide', 'completed', datetime('now'), datetime('now'), datetime('now'))
	`)
	if err != nil {
		log.Printf("❌ Failed to insert test bookmark: %v", err)
		return
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✓ Inserted test bookmark (rows affected: %d)\n", rowsAffected)
	}

	// Verify bookmark was inserted
	fmt.Println("3.3 Verifying bookmark insertion...")
	var bookmarkCount int
	err = store.GetDB().QueryRow("SELECT COUNT(*) FROM bookmarks WHERE id = 'test-001'").Scan(&bookmarkCount)
	if err != nil {
		log.Printf("❌ Failed to verify bookmark: %v", err)
		return
	}
	fmt.Printf("✓ Bookmark verification: %d bookmark(s) found\n", bookmarkCount)

	if bookmarkCount == 0 {
		log.Printf("❌ Bookmark was not inserted successfully")
		return
	}

	// Store content
	fmt.Println("3.4 Storing content...")
	err = store.StoreContent("test-001",
		"<html><body><h1>Algorithm Guide</h1><p>This guide covers various algorithms including sorting, searching, and graph algorithms.</p></body></html>",
		"Algorithm Guide. This guide covers various algorithms including sorting, searching, and graph algorithms.")

	if err != nil {
		log.Printf("❌ Failed to store content: %v", err)
		return
	} else {
		fmt.Printf("✓ Content stored successfully\n")
	}

	// Verify content was stored
	fmt.Println("3.5 Verifying content storage...")
	var contentCount int
	err = store.GetDB().QueryRow("SELECT COUNT(*) FROM content WHERE bookmark_id = 'test-001'").Scan(&contentCount)
	if err != nil {
		log.Printf("❌ Failed to verify content: %v", err)
		return
	}
	fmt.Printf("✓ Content verification: %d content record(s) found\n", contentCount)

	// Get content to get the content ID
	fmt.Println("3.6 Retrieving content for embedding storage...")
	content, err := store.GetContent("test-001")
	if err != nil {
		log.Printf("❌ Failed to get content: %v", err)
		return
	} else {
		fmt.Printf("✓ Retrieved content with ID: %d\n", content.ID)
		fmt.Printf("   Content length: %d characters\n", len(content.CleanText))
	}

	// Store embedding with detailed logging
	fmt.Println("3.7 Storing embedding...")
	fmt.Printf("   Embedding dimensions: %d\n", len(embedding))
	fmt.Printf("   Content ID: %d\n", content.ID)
	fmt.Printf("   First few embedding values: [%.6f, %.6f, %.6f, ...]\n",
		embedding[0], embedding[1], embedding[2])

	err = store.StoreEmbedding(content.ID, embedding)
	if err != nil {
		log.Printf("❌ Failed to store embedding: %v", err)
		return
	} else {
		fmt.Printf("✓ Embedding stored successfully\n")
	}

	// Verify embedding was stored
	fmt.Println("3.8 Verifying embedding storage...")
	var embeddingCount int
	err = store.GetDB().QueryRow("SELECT COUNT(*) FROM embeddings WHERE content_id = ?", content.ID).Scan(&embeddingCount)
	if err != nil {
		log.Printf("❌ Failed to verify embedding: %v", err)
		return
	}
	fmt.Printf("✓ Embedding verification: %d embedding(s) found for content ID %d\n", embeddingCount, content.ID)

	// Check total embeddings count
	var totalEmbeddings int
	err = store.GetDB().QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&totalEmbeddings)
	if err != nil {
		log.Printf("❌ Failed to count total embeddings: %v", err)
	} else {
		fmt.Printf("✓ Total embeddings in database: %d\n", totalEmbeddings)
	}

	// If vector32() function failed, try storing embedding as raw JSON (fallback)
	if totalEmbeddings == 0 {
		fmt.Println("\n3.9 Fallback: Trying to store embedding without vector32()...")

		embeddingJSON, _ := json.Marshal(embedding)
		fallbackQuery := `INSERT OR REPLACE INTO embeddings (content_id, embedding) VALUES (?, ?)`
		result, err := store.GetDB().Exec(fallbackQuery, content.ID, embeddingJSON)
		if err != nil {
			log.Printf("❌ Fallback storage also failed: %v", err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			fmt.Printf("✓ Fallback storage successful (rows affected: %d)\n", rowsAffected)

			// Check count again
			err = store.GetDB().QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&totalEmbeddings)
			if err != nil {
				log.Printf("❌ Failed to count embeddings after fallback: %v", err)
			} else {
				fmt.Printf("✓ Total embeddings after fallback: %d\n", totalEmbeddings)
			}
		}
	}

	fmt.Println("\n4. Database schema inspection...")

	// Check the embeddings table structure
	fmt.Println("4.1 Checking embeddings table schema...")
	rows, err := store.GetDB().Query("PRAGMA table_info(embeddings)")
	if err != nil {
		log.Printf("❌ Failed to get table info: %v", err)
	} else {
		defer rows.Close()
		fmt.Println("   Embeddings table structure:")
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, dfltValue, pk interface{}
			rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			fmt.Printf("     - %s: %s\n", name, dataType)
		}
	}

	// Check what's actually in the embeddings table
	fmt.Println("4.2 Raw embeddings table content...")
	rows, err = store.GetDB().Query("SELECT id, content_id, model_version, created_at FROM embeddings LIMIT 5")
	if err != nil {
		log.Printf("❌ Failed to query embeddings: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var id, contentID int
			var modelVersion, createdAt string
			rows.Scan(&id, &contentID, &modelVersion, &createdAt)
			fmt.Printf("   Row %d: id=%d, content_id=%d, model=%s, created=%s\n",
				count+1, id, contentID, modelVersion, createdAt)
			count++
		}
		if count == 0 {
			fmt.Println("   No rows found in embeddings table")
		}
	}

	fmt.Println("\n5. Testing semantic search query...")

	// Test the raw SQL semantic search
	results, err := store.HybridSearch(queryEmbedding, query)
	if err != nil {
		log.Printf("Hybrid search failed: %v", err)

		// Try keyword search as fallback
		keywordResults, err := store.KeywordSearch(query, 10)
		if err != nil {
			log.Printf("Keyword search also failed: %v", err)
		} else {
			fmt.Printf("Keyword search found %d results\n", len(keywordResults))
		}
	} else {
		fmt.Printf("✓ Hybrid search found %d results\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. %s (Score: %.3f, Type: %s)\n",
				i+1, result.Bookmark.Title, result.RelevanceScore, result.SearchType)
		}
	}

	fmt.Println("\n=== Test Complete ===")
	fmt.Println("Database file created: test_embeddings.db")
	fmt.Println("Clean up by running: rm test_embeddings.db")
}
