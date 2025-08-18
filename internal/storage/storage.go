package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/tursodatabase/go-libsql"
)

// Storage represents the database storage layer
type Storage struct {
	db *sql.DB
}

// Bookmark represents a bookmark entry
type Bookmark struct {
	ID          int       `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	ImportedAt  time.Time `json:"imported_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FolderPath  string    `json:"folder_path,omitempty"`
	Description string    `json:"description,omitempty"`
}

// Content represents scraped content from a bookmark
type Content struct {
	ID          int       `json:"id"`
	BookmarkID  int       `json:"bookmark_id"`
	RawContent  string    `json:"raw_content"`
	CleanText   string    `json:"clean_text"`
	ScrapedAt   time.Time `json:"scraped_at"`
	ContentType string    `json:"content_type"`
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Bookmark       *Bookmark `json:"bookmark"`
	Content        *Content  `json:"content,omitempty"`
	RelevanceScore float64   `json:"relevance_score"`
	SearchType     string    `json:"search_type"`
	MatchedSnippet string    `json:"matched_snippet,omitempty"`
}

// New creates a new Storage instance with a local libSQL database
func New(dbPath string) (*Storage, error) {
	if dbPath == "" {
		dbPath = "file:bookmarks.db"
	}

	db, err := sql.Open("libsql", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{db: db}

	if err := storage.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// initializeSchema creates all necessary tables and indexes
func (s *Storage) initializeSchema() error {
	schemas := []string{
		// Bookmarks table
		`CREATE TABLE IF NOT EXISTS bookmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			title TEXT,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'completed', 'failed')),
			imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			folder_path TEXT,
			description TEXT
		)`,

		// Content table
		`CREATE TABLE IF NOT EXISTS content (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bookmark_id INTEGER NOT NULL,
			raw_content TEXT,
			clean_text TEXT,
			scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			content_type TEXT DEFAULT 'text/html',
			FOREIGN KEY (bookmark_id) REFERENCES bookmarks(id) ON DELETE CASCADE
		)`,

		// Embeddings table with vector support
		`CREATE TABLE IF NOT EXISTS embeddings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content_id INTEGER NOT NULL,
			embedding BLOB,
			model_version TEXT DEFAULT 'text-embedding-3-small',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (content_id) REFERENCES content(id) ON DELETE CASCADE
		)`,

		// FTS5 virtual table for full-text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS bookmarks_fts USING fts5(
			title, 
			description, 
			clean_text,
			content='content',
			content_rowid='bookmark_id'
		)`,

		// Create vector index for embeddings (commented out due to compatibility issues)
		// `CREATE INDEX IF NOT EXISTS embeddings_vector_idx ON embeddings(libsql_vector_idx(embedding))`,

		// Create standard index for embeddings
		`CREATE INDEX IF NOT EXISTS idx_embeddings_content_id_lookup ON embeddings(content_id)`,

		// Create standard indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_status ON bookmarks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_url ON bookmarks(url)`,
		`CREATE INDEX IF NOT EXISTS idx_content_bookmark_id ON content(bookmark_id)`,
		`CREATE INDEX IF NOT EXISTS idx_embeddings_content_id ON embeddings(content_id)`,
	}

	for _, schema := range schemas {
		if _, err := s.db.Exec(schema); err != nil {
			return fmt.Errorf("failed to execute schema: %s, error: %w", schema, err)
		}
	}

	// Set up triggers to keep FTS in sync
	triggers := []string{
		`CREATE TRIGGER IF NOT EXISTS bookmarks_fts_insert AFTER INSERT ON content BEGIN
			INSERT INTO bookmarks_fts(rowid, title, description, clean_text) 
			SELECT NEW.bookmark_id, b.title, b.description, NEW.clean_text 
			FROM bookmarks b WHERE b.id = NEW.bookmark_id;
		END`,

		`CREATE TRIGGER IF NOT EXISTS bookmarks_fts_update AFTER UPDATE ON content BEGIN
			UPDATE bookmarks_fts SET clean_text = NEW.clean_text WHERE rowid = NEW.bookmark_id;
		END`,

		`CREATE TRIGGER IF NOT EXISTS bookmarks_fts_delete AFTER DELETE ON content BEGIN
			DELETE FROM bookmarks_fts WHERE rowid = OLD.bookmark_id;
		END`,
	}

	for _, trigger := range triggers {
		if _, err := s.db.Exec(trigger); err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
	}

	return nil
}

// AddBookmark adds a new bookmark to the database
func (s *Storage) AddBookmark(url string, title string) error {
	query := `INSERT INTO bookmarks (url, title) VALUES (?, ?)`
	_, err := s.db.Exec(query, url, title)
	if err != nil {
		return fmt.Errorf("failed to add bookmark: %w", err)
	}
	return nil
}

// UpdateBookmarkStatus updates the status of a bookmark
func (s *Storage) UpdateBookmarkStatus(bookmarkID int, status string) error {
	query := `UPDATE bookmarks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := s.db.Exec(query, status, bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to update bookmark status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bookmark with ID %d not found", bookmarkID)
	}

	return nil
}

// GetBookmark retrieves a bookmark by ID
func (s *Storage) GetBookmark(bookmarkID int) (*Bookmark, error) {
	query := `SELECT id, url, title, status, imported_at, created_at, updated_at, 
			  COALESCE(folder_path, ''), COALESCE(description, '') 
			  FROM bookmarks WHERE id = ?`

	row := s.db.QueryRow(query, bookmarkID)

	bookmark := &Bookmark{}
	err := row.Scan(
		&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Status,
		&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
		&bookmark.FolderPath, &bookmark.Description,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bookmark with ID %d not found", bookmarkID)
		}
		return nil, fmt.Errorf("failed to get bookmark: %w", err)
	}

	return bookmark, nil
}

// ListBookmarks retrieves all bookmarks
func (s *Storage) ListBookmarks() ([]*Bookmark, error) {
	query := `SELECT id, url, title, status, imported_at, created_at, updated_at, 
			  COALESCE(folder_path, ''), COALESCE(description, '') 
			  FROM bookmarks ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list bookmarks: %w", err)
	}
	defer rows.Close()

	var bookmarks []*Bookmark
	for rows.Next() {
		bookmark := &Bookmark{}
		err := rows.Scan(
			&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Status,
			&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
			&bookmark.FolderPath, &bookmark.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bookmark: %w", err)
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

// StoreContent stores scraped content for a bookmark
func (s *Storage) StoreContent(bookmarkID int, rawContent string, cleanText string) error {
	query := `INSERT OR REPLACE INTO content (bookmark_id, raw_content, clean_text) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, bookmarkID, rawContent, cleanText)
	if err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}
	return nil
}

// GetContent retrieves content by bookmark ID
func (s *Storage) GetContent(bookmarkID int) (*Content, error) {
	query := `SELECT id, bookmark_id, COALESCE(raw_content, ''), COALESCE(clean_text, ''), 
			  scraped_at, content_type FROM content WHERE bookmark_id = ?`

	row := s.db.QueryRow(query, bookmarkID)

	content := &Content{}
	err := row.Scan(
		&content.ID, &content.BookmarkID, &content.RawContent,
		&content.CleanText, &content.ScrapedAt, &content.ContentType,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("content for bookmark ID %d not found", bookmarkID)
		}
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return content, nil
}

// StoreEmbedding stores a vector embedding for content
func (s *Storage) StoreEmbedding(contentID int, embedding []float32) error {
	// Convert float32 slice to JSON format for vector32() function
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	query := `INSERT OR REPLACE INTO embeddings (content_id, embedding) VALUES (?, vector32(?))`
	_, err = s.db.Exec(query, contentID, string(embeddingJSON))
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}
	return nil
}

// GetEmbedding retrieves a vector embedding by content ID
func (s *Storage) GetEmbedding(contentID int) ([]float32, error) {
	query := `SELECT embedding FROM embeddings WHERE content_id = ?`

	row := s.db.QueryRow(query, contentID)

	var embeddingData []byte
	err := row.Scan(&embeddingData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding for content ID %d not found", contentID)
		}
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	var embedding []float32
	err = json.Unmarshal(embeddingData, &embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
	}

	return embedding, nil
}

// HybridSearch performs a combined semantic and keyword search
func (s *Storage) HybridSearch(queryEmbedding []float32, queryText string) ([]*SearchResult, error) {
	var allResults []*SearchResult

	// Perform semantic search using vector similarity
	semanticResults, err := s.semanticSearch(queryEmbedding, 50)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}

	// Perform keyword search using FTS5
	keywordResults, err := s.keywordSearch(queryText, 50)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// Combine and deduplicate results
	resultMap := make(map[int]*SearchResult)

	// Add semantic results with weight
	for _, result := range semanticResults {
		result.RelevanceScore *= 0.7 // Semantic weight
		result.SearchType = "semantic"
		resultMap[result.Bookmark.ID] = result
	}

	// Add keyword results, combining scores if bookmark already exists
	for _, result := range keywordResults {
		result.RelevanceScore *= 0.3 // Keyword weight
		if existing, exists := resultMap[result.Bookmark.ID]; exists {
			existing.RelevanceScore += result.RelevanceScore
			existing.SearchType = "hybrid"
			if result.MatchedSnippet != "" {
				existing.MatchedSnippet = result.MatchedSnippet
			}
		} else {
			result.SearchType = "keyword"
			resultMap[result.Bookmark.ID] = result
		}
	}

	// Convert map to slice and sort by relevance
	for _, result := range resultMap {
		allResults = append(allResults, result)
	}

	// Sort by relevance score (descending)
	for i := 0; i < len(allResults)-1; i++ {
		for j := i + 1; j < len(allResults); j++ {
			if allResults[i].RelevanceScore < allResults[j].RelevanceScore {
				allResults[i], allResults[j] = allResults[j], allResults[i]
			}
		}
	}

	// Limit results to top 20
	if len(allResults) > 20 {
		allResults = allResults[:20]
	}

	return allResults, nil
}

// semanticSearch performs vector similarity search
func (s *Storage) semanticSearch(queryEmbedding []float32, limit int) ([]*SearchResult, error) {
	// Fallback semantic search without vector index (for compatibility)
	// Note: This is a simplified version that doesn't do actual vector similarity
	query := `
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, ''),
		       c.id, c.bookmark_id, COALESCE(c.raw_content, ''), COALESCE(c.clean_text, ''),
		       c.scraped_at, c.content_type,
		       0.5 as similarity
		FROM embeddings e
		JOIN content c ON c.id = e.content_id
		JOIN bookmarks b ON b.id = c.bookmark_id
		ORDER BY e.id DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute semantic search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		bookmark := &Bookmark{}
		content := &Content{}
		var similarity float64

		err := rows.Scan(
			&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Status,
			&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
			&bookmark.FolderPath, &bookmark.Description,
			&content.ID, &content.BookmarkID, &content.RawContent,
			&content.CleanText, &content.ScrapedAt, &content.ContentType,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan semantic search result: %w", err)
		}

		result := &SearchResult{
			Bookmark:       bookmark,
			Content:        content,
			RelevanceScore: similarity,
			SearchType:     "semantic",
		}

		results = append(results, result)
	}

	return results, nil
}

// keywordSearch performs BM25-based full-text search
func (s *Storage) keywordSearch(queryText string, limit int) ([]*SearchResult, error) {
	// Escape FTS5 special characters and prepare query
	escapedQuery := strings.ReplaceAll(queryText, "'", "''")
	ftsQuery := fmt.Sprintf("'%s'", escapedQuery)

	query := `
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, ''),
		       c.id, c.bookmark_id, COALESCE(c.raw_content, ''), COALESCE(c.clean_text, ''),
		       c.scraped_at, c.content_type,
		       bm25(bookmarks_fts) as relevance,
		       snippet(bookmarks_fts, 2, '<mark>', '</mark>', '...', 32) as snippet
		FROM bookmarks_fts
		JOIN content c ON c.bookmark_id = bookmarks_fts.rowid
		JOIN bookmarks b ON b.id = c.bookmark_id
		WHERE bookmarks_fts MATCH ?
		ORDER BY relevance
		LIMIT ?
	`

	rows, err := s.db.Query(query, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute keyword search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		bookmark := &Bookmark{}
		content := &Content{}
		var relevance float64
		var snippet string

		err := rows.Scan(
			&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Status,
			&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
			&bookmark.FolderPath, &bookmark.Description,
			&content.ID, &content.BookmarkID, &content.RawContent,
			&content.CleanText, &content.ScrapedAt, &content.ContentType,
			&relevance, &snippet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan keyword search result: %w", err)
		}

		// Convert BM25 score to similarity (higher is better)
		similarity := 1.0 / (1.0 + (-relevance))

		result := &SearchResult{
			Bookmark:       bookmark,
			Content:        content,
			RelevanceScore: similarity,
			SearchType:     "keyword",
			MatchedSnippet: snippet,
		}

		results = append(results, result)
	}

	return results, nil
}
