# Bookmark Storage Layer

A complete database storage layer for the bookmark chat system using libSQL with vector extensions for semantic search capabilities.

## Overview

This storage layer provides a local-only database solution using libSQL (SQLite-compatible) with advanced features including:

- **Vector embeddings storage** with F32_BLOB for efficient semantic search
- **Full-text search** using SQLite FTS5 extension with BM25 ranking
- **Hybrid search** combining semantic and keyword search with weighted scoring
- **Batch operations** for efficient bulk processing
- **Comprehensive error handling** with detailed error messages
- **Transaction support** for data consistency

## Architecture

### Database Schema

The storage layer uses four main tables:

1. **bookmarks** - Core bookmark metadata
2. **content** - Scraped and cleaned content from bookmark URLs  
3. **embeddings** - Vector representations using F32_BLOB(1536)
4. **bookmarks_fts** - FTS5 virtual table for full-text search

### Key Features

- **Vector Search**: Uses libSQL's vector extensions with cosine similarity
- **Keyword Search**: FTS5 with BM25 scoring and snippet generation
- **Hybrid Scoring**: Combines semantic (70%) and keyword (30%) relevance
- **Batch Processing**: Efficient bulk operations with transactions
- **Auto-sync FTS**: Triggers keep full-text index synchronized

## Usage

### Basic Setup

```go
import "github.com/bookmark-chat-claude4/internal/storage"

// Initialize storage with local database
store, err := storage.New("file:bookmarks.db")
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

### Core Operations

#### Bookmark Management
```go
// Add a new bookmark
err := store.AddBookmark("https://example.com", "Example Site")

// Get bookmark by ID
bookmark, err := store.GetBookmark(1)

// Update bookmark status
err := store.UpdateBookmarkStatus(1, "completed")

// List all bookmarks
bookmarks, err := store.ListBookmarks()

// Delete bookmark and all associated data
err := store.DeleteBookmark(1)
```

#### Content Management
```go
// Store scraped content
err := store.StoreContent(bookmarkID, rawHTML, cleanText)

// Retrieve content
content, err := store.GetContent(bookmarkID)
```

#### Embedding Management
```go
// Store vector embedding (1536 dimensions for text-embedding-3-small)
embedding := []float32{0.1, 0.2, 0.3, ...} // 1536 dimensions
err := store.StoreEmbedding(contentID, embedding)

// Retrieve embedding
embedding, err := store.GetEmbedding(contentID)
```

### Advanced Features

#### Hybrid Search
```go
// Combines semantic and keyword search
queryEmbedding := generateEmbedding("Go programming language")
results, err := store.HybridSearch(queryEmbedding, "Go programming language")

for _, result := range results {
    fmt.Printf("Title: %s, Score: %.3f, Type: %s\n", 
        result.Bookmark.Title, result.RelevanceScore, result.SearchType)
    if result.MatchedSnippet != "" {
        fmt.Printf("Snippet: %s\n", result.MatchedSnippet)
    }
}
```

#### Batch Operations
```go
batchOps := store.NewBatchOperations()

// Batch add bookmarks
bookmarks := []struct{URL, Title string}{
    {"https://site1.com", "Site 1"},
    {"https://site2.com", "Site 2"},
}
err := batchOps.BatchAddBookmarks(bookmarks)

// Batch store embeddings
embeddings := []struct{ContentID int; Embedding []float32}{
    {1, embedding1},
    {2, embedding2},
}
err := batchOps.BatchStoreEmbeddings(embeddings)
```

#### Filtered Search
```go
opts := storage.SearchOptions{
    Status:        "completed",
    FolderPath:    "tech",
    CreatedAfter:  time.Now().AddDate(0, -1, 0), // Last month
    Limit:         50,
}
results, err := store.SearchBookmarksWithFilters(opts)
```

#### Database Statistics
```go
stats, err := store.GetStats()
fmt.Printf("Total bookmarks: %d\n", stats["total_bookmarks"])
fmt.Printf("With embeddings: %d\n", stats["bookmarks_with_embeddings"])
```

## Data Models

### Bookmark
```go
type Bookmark struct {
    ID          int       `json:"id"`
    URL         string    `json:"url"`
    Title       string    `json:"title"`
    Status      string    `json:"status"`          // pending, completed, failed
    ImportedAt  time.Time `json:"imported_at"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    FolderPath  string    `json:"folder_path,omitempty"`
    Description string    `json:"description,omitempty"`
}
```

### Content
```go
type Content struct {
    ID          int       `json:"id"`
    BookmarkID  int       `json:"bookmark_id"`
    RawContent  string    `json:"raw_content"`     // Original HTML
    CleanText   string    `json:"clean_text"`      // Processed text
    ScrapedAt   time.Time `json:"scraped_at"`
    ContentType string    `json:"content_type"`
}
```

### SearchResult
```go
type SearchResult struct {
    Bookmark        *Bookmark `json:"bookmark"`
    Content         *Content  `json:"content,omitempty"`
    RelevanceScore  float64   `json:"relevance_score"`    // 0.0 - 1.0
    SearchType      string    `json:"search_type"`        // semantic, keyword, hybrid
    MatchedSnippet  string    `json:"matched_snippet,omitempty"`
}
```

## Performance Considerations

### Indexing Strategy
- Vector index on embeddings for fast semantic search
- FTS5 index for efficient keyword search  
- Standard B-tree indexes on frequently queried columns
- Foreign key constraints with CASCADE delete

### Query Optimization
- Hybrid search limits results to top 50 per method before combination
- Pagination support through SearchOptions.Limit
- Efficient batch operations using prepared statements
- Connection pooling via sql.DB

### Memory Usage
- Vector embeddings stored as F32_BLOB (1536 × 4 bytes = 6KB per embedding)
- FTS5 index size approximately 30-40% of content size
- Prepared statements cached for repeated operations

## Integration with OpenAI

### Embedding Generation
```go
// Example integration with OpenAI API
func generateEmbedding(text string, apiKey string) ([]float32, error) {
    client := openai.NewClient(apiKey)
    resp, err := client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
        Model: openai.TextEmbedding3Small,
        Input: []string{text},
    })
    if err != nil {
        return nil, err
    }
    return resp.Data[0].Embedding, nil
}

// Store in database
embedding, err := generateEmbedding(content.CleanText, apiKey)
if err != nil {
    return err
}
return store.StoreEmbedding(content.ID, embedding)
```

### Chunking Long Content
```go
// For content exceeding token limits (8191 for text-embedding-3-small)
func chunkContent(text string, maxTokens int) []string {
    // Implementation depends on tokenizer
    // Split text into chunks under token limit
    // Return array of text chunks
}

// Store embeddings for each chunk
for i, chunk := range chunks {
    embedding, err := generateEmbedding(chunk, apiKey)
    if err != nil {
        continue
    }
    // Store with chunk index in separate table or modify schema
}
```

## Error Handling

The storage layer provides comprehensive error handling:

- **Connection errors**: Database file permissions, disk space
- **Constraint violations**: Duplicate URLs, invalid foreign keys  
- **Data validation**: Invalid embedding dimensions, status values
- **Transaction failures**: Rollback on partial batch operations
- **Resource cleanup**: Proper connection and statement cleanup

All errors are wrapped with context using `fmt.Errorf` with `%w` verb for error chain inspection.

## Testing

Run the comprehensive test suite:

```bash
go test ./internal/storage -v
```

Benchmark performance:
```bash
go test ./internal/storage -bench=. -benchmem
```

See `storage_test.go` for complete test coverage including:
- Unit tests for all CRUD operations
- Integration tests for hybrid search  
- Error condition testing
- Performance benchmarks

## Example Application

See `examples/storage_example.go` for a complete demonstration of all features:

```bash
cd examples
go run storage_example.go
```

This creates a sample database with bookmarks, content, and embeddings, then demonstrates all search capabilities.

## Dependencies

- `github.com/tursodatabase/libsql-client-go/libsql` - libSQL Go driver
- Standard library: `database/sql`, `encoding/json`, `fmt`, `strings`, `time`

No external dependencies beyond the libSQL driver, keeping the storage layer lightweight and focused.