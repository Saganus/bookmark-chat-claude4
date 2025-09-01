package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bookmark-chat/internal/services/parsers"
	"github.com/google/uuid"
	_ "github.com/tursodatabase/go-libsql"
)

// Storage represents the database storage layer
type Storage struct {
	db *sql.DB
}

// Bookmark represents a bookmark entry
type Bookmark struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	ImportedAt  time.Time  `json:"imported_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ScrapedAt   *time.Time `json:"scraped_at,omitempty"`
	FolderID    *string    `json:"folder_id,omitempty"`
	FolderPath  string     `json:"folder_path,omitempty"`
	FaviconURL  string     `json:"favicon_url,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
}

// BookmarkFolder represents a folder in the bookmark hierarchy
type BookmarkFolder struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	ParentID   *string           `json:"parent_id,omitempty"`
	Path       string            `json:"path"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Bookmarks  []*Bookmark       `json:"bookmarks,omitempty"`
	Subfolders []*BookmarkFolder `json:"subfolders,omitempty"`
}

// Content represents scraped content from a bookmark
type Content struct {
	ID          int       `json:"id"`
	BookmarkID  string    `json:"bookmark_id"`
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

	// Add WAL mode and connection settings for better concurrency handling
	if !strings.Contains(dbPath, "?") {
		dbPath += "?_journal=WAL&_timeout=10000&_sync=NORMAL&_cache_size=1000"
	}

	db, err := sql.Open("libsql", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for better concurrency
	db.SetMaxOpenConns(1)     // Single connection to avoid SQLite lock issues
	db.SetMaxIdleConns(1)     // Keep one idle connection
	db.SetConnMaxLifetime(0)  // Don't expire connections

	storage := &Storage{db: db}

	if err := storage.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Apply categorization migration
	if err := storage.applyCategorization(); err != nil {
		return nil, fmt.Errorf("failed to apply categorization migration: %w", err)
	}

	return storage, nil
}

// retryWithBackoff executes a function with exponential backoff for database lock errors
func (s *Storage) retryWithBackoff(operation func() error) error {
	maxRetries := 5
	baseDelay := 100 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		
		// Check if it's a database lock error
		if strings.Contains(err.Error(), "database is locked") || 
		   strings.Contains(err.Error(), "SQLite failure") {
			if attempt < maxRetries-1 {
				// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
				delay := baseDelay * time.Duration(1<<attempt)
				time.Sleep(delay)
				continue
			}
		}
		
		return err
	}
	
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// GetDB returns the underlying database connection (for testing)
func (s *Storage) GetDB() *sql.DB {
	return s.db
}

// initializeSchema creates all necessary tables and indexes
func (s *Storage) initializeSchema() error {
	schemas := []string{
		// Folders table for hierarchical structure
		`CREATE TABLE IF NOT EXISTS folders (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id TEXT,
			path TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE CASCADE
		)`,

		// Bookmarks table
		`CREATE TABLE IF NOT EXISTS bookmarks (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			title TEXT,
			description TEXT,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'completed', 'failed')),
			imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			scraped_at TIMESTAMP,
			folder_id TEXT,
			folder_path TEXT,
			favicon_url TEXT,
			tags TEXT, -- JSON array of tags
			FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE SET NULL
		)`,

		// Content table
		`CREATE TABLE IF NOT EXISTS content (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bookmark_id TEXT NOT NULL,
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
			chunk_index INTEGER DEFAULT 0,
			chunk_text TEXT,
			embedding BLOB,
			model_version TEXT DEFAULT 'text-embedding-3-small',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (content_id) REFERENCES content(id) ON DELETE CASCADE
		)`,

		// FTS5 virtual table for bookmarks full-text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS bookmarks_fts USING fts5(
			title, 
			description
		)`,

		// FTS5 virtual table for content full-text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS content_fts USING fts5(
			clean_text
		)`,

		// Create standard indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path)`,
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_status ON bookmarks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_url ON bookmarks(url)`,
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_folder_id ON bookmarks(folder_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_bookmark_id ON content(bookmark_id)`,
		`CREATE INDEX IF NOT EXISTS idx_embeddings_content_id ON embeddings(content_id)`,
		`CREATE INDEX IF NOT EXISTS idx_embeddings_content_chunk ON embeddings(content_id, chunk_index)`,
	}

	for _, schema := range schemas {
		if _, err := s.db.Exec(schema); err != nil {
			return fmt.Errorf("failed to execute schema: %s, error: %w", schema, err)
		}
	}

	// Note: We'll manually manage FTS table synchronization to avoid trigger conflicts
	// FTS tables will be updated explicitly when content changes

	return nil
}

// ImportBookmarks imports bookmarks and folders from a parse result
func (s *Storage) ImportBookmarks(parseResult *parsers.ParseResult) (*ImportResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	result := &ImportResult{
		TotalFound:           parseResult.TotalCount,
		SuccessfullyImported: 0,
		Failed:               0,
		Duplicates:           0,
		ImportedFolders:      []*BookmarkFolder{},
		ImportedBookmarks:    []*Bookmark{},
		Errors:               []string{},
	}

	// Create folder hierarchy first
	folderMap := make(map[string]string) // path -> folder ID mapping
	for _, folder := range parseResult.Folders {
		if err := s.createFolderHierarchy(tx, folder, nil, folderMap); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create folder %s: %v", folder.Name, err))
			continue
		}
	}

	// Import bookmarks
	for _, bookmark := range parseResult.Bookmarks {
		bookmarkID := uuid.New().String()
		var folderID *string

		// Find folder ID if bookmark has a folder path
		if len(bookmark.FolderPath) > 0 {
			folderPath := strings.Join(bookmark.FolderPath, "/")
			if fID, exists := folderMap[folderPath]; exists {
				folderID = &fID
			}
		}

		// Convert tags to JSON
		var tagsJSON string
		if len(bookmark.FolderPath) > 0 {
			tags := []string{} // Could be extended to include actual tags from parsing
			if tagsBytes, err := json.Marshal(tags); err == nil {
				tagsJSON = string(tagsBytes)
			}
		}

		// Check for duplicates
		var existingID string
		err := tx.QueryRow("SELECT id FROM bookmarks WHERE url = ?", bookmark.URL).Scan(&existingID)
		if err == nil {
			result.Duplicates++
			continue
		} else if err != sql.ErrNoRows {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Error checking duplicate for %s: %v", bookmark.URL, err))
			continue
		}

		// Insert bookmark
		folderPath := strings.Join(bookmark.FolderPath, "/")
		_, err = tx.Exec(`
			INSERT INTO bookmarks (id, url, title, description, folder_id, folder_path, favicon_url, tags, imported_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			bookmarkID, bookmark.URL, bookmark.Title, "", folderID, folderPath, bookmark.Icon, tagsJSON, bookmark.DateAdded)

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert bookmark %s: %v", bookmark.URL, err))
			continue
		}

		result.SuccessfullyImported++

		// Create bookmark object for result
		dbBookmark := &Bookmark{
			ID:         bookmarkID,
			URL:        bookmark.URL,
			Title:      bookmark.Title,
			Status:     "pending",
			ImportedAt: bookmark.DateAdded,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			FolderID:   folderID,
			FolderPath: folderPath,
			FaviconURL: bookmark.Icon,
			Tags:       []string{},
		}
		result.ImportedBookmarks = append(result.ImportedBookmarks, dbBookmark)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// createFolderHierarchy recursively creates folder hierarchy
func (s *Storage) createFolderHierarchy(tx *sql.Tx, folder *parsers.BookmarkFolder, parentID *string, folderMap map[string]string) error {
	folderID := uuid.New().String()
	folderPath := strings.Join(folder.Path, "/")

	// Insert folder
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO folders (id, name, parent_id, path)
		VALUES (?, ?, ?, ?)`,
		folderID, folder.Name, parentID, folderPath)

	if err != nil {
		return fmt.Errorf("failed to insert folder: %w", err)
	}

	// Store in map for bookmark reference
	folderMap[folderPath] = folderID

	// Recursively create subfolders
	for _, subfolder := range folder.Subfolders {
		if err := s.createFolderHierarchy(tx, subfolder, &folderID, folderMap); err != nil {
			return err
		}
	}

	return nil
}

// GetBookmark retrieves a bookmark by ID
func (s *Storage) GetBookmark(bookmarkID string) (*Bookmark, error) {
	query := `SELECT id, url, title, description, status, imported_at, created_at, updated_at, 
			  scraped_at, folder_id, COALESCE(folder_path, ''), COALESCE(favicon_url, ''), COALESCE(tags, '[]')
			  FROM bookmarks WHERE id = ?`

	row := s.db.QueryRow(query, bookmarkID)

	bookmark := &Bookmark{}
	var tagsJSON string
	err := row.Scan(
		&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Description, &bookmark.Status,
		&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
		&bookmark.ScrapedAt, &bookmark.FolderID, &bookmark.FolderPath, &bookmark.FaviconURL, &tagsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bookmark with ID %s not found", bookmarkID)
		}
		return nil, fmt.Errorf("failed to get next row: %w", err)
	}

	// Parse tags JSON
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &bookmark.Tags); err != nil {
			bookmark.Tags = []string{}
		}
	}

	return bookmark, nil
}

// ListBookmarks retrieves all bookmarks
func (s *Storage) ListBookmarks() ([]*Bookmark, error) {
	query := `SELECT id, url, title, description, status, imported_at, created_at, updated_at, 
			  scraped_at, folder_id, COALESCE(folder_path, ''), COALESCE(favicon_url, ''), COALESCE(tags, '[]')
			  FROM bookmarks ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list bookmarks: %w", err)
	}
	defer rows.Close()

	var bookmarks []*Bookmark
	for rows.Next() {
		bookmark := &Bookmark{}
		var tagsJSON string
		err := rows.Scan(
			&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Description, &bookmark.Status,
			&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
			&bookmark.ScrapedAt, &bookmark.FolderID, &bookmark.FolderPath, &bookmark.FaviconURL, &tagsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bookmark: %w", err)
		}

		// Parse tags JSON
		if tagsJSON != "" {
			if err := json.Unmarshal([]byte(tagsJSON), &bookmark.Tags); err != nil {
				bookmark.Tags = []string{}
			}
		}

		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

// GetBookmarksWithFolders retrieves all bookmarks organized by folders
func (s *Storage) GetBookmarksWithFolders() ([]*BookmarkFolder, error) {
	// Get all folders
	folders, err := s.getFolderHierarchy()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder hierarchy: %w", err)
	}

	// Get all bookmarks and organize them by folder
	bookmarks, err := s.ListBookmarks()
	if err != nil {
		return nil, fmt.Errorf("failed to list bookmarks: %w", err)
	}

	// Create a map to quickly find folders by ID
	folderMap := make(map[string]*BookmarkFolder)
	var rootFolders []*BookmarkFolder

	for _, folder := range folders {
		folderMap[folder.ID] = folder
		if folder.ParentID == nil {
			rootFolders = append(rootFolders, folder)
		}
	}

	// Organize bookmarks by folder
	for _, bookmark := range bookmarks {
		if bookmark.FolderID != nil {
			if folder, exists := folderMap[*bookmark.FolderID]; exists {
				folder.Bookmarks = append(folder.Bookmarks, bookmark)
			}
		}
	}

	return rootFolders, nil
}

// getFolderHierarchy retrieves all folders and builds the hierarchy
func (s *Storage) getFolderHierarchy() ([]*BookmarkFolder, error) {
	query := `SELECT id, name, parent_id, path, created_at, updated_at FROM folders ORDER BY path`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query folders: %w", err)
	}
	defer rows.Close()

	var folders []*BookmarkFolder
	folderMap := make(map[string]*BookmarkFolder)

	for rows.Next() {
		folder := &BookmarkFolder{}
		err := rows.Scan(
			&folder.ID, &folder.Name, &folder.ParentID, &folder.Path,
			&folder.CreatedAt, &folder.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}

		folder.Bookmarks = []*Bookmark{}
		folder.Subfolders = []*BookmarkFolder{}
		folders = append(folders, folder)
		folderMap[folder.ID] = folder
	}

	// Build hierarchy
	for _, folder := range folders {
		if folder.ParentID != nil {
			if parent, exists := folderMap[*folder.ParentID]; exists {
				parent.Subfolders = append(parent.Subfolders, folder)
			}
		}
	}

	return folders, nil
}

// UpdateBookmarkStatus updates the status of a bookmark
func (s *Storage) UpdateBookmarkStatus(bookmarkID string, status string) error {
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
		return fmt.Errorf("bookmark with ID %s not found", bookmarkID)
	}

	return nil
}

// UpdateBookmark updates a bookmark's metadata
func (s *Storage) UpdateBookmark(bookmark *Bookmark) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the bookmark
	query := `
		UPDATE bookmarks 
		SET title = ?, description = ?, favicon_url = ?, updated_at = ?, scraped_at = ?
		WHERE id = ?
	`
	result, err := tx.Exec(query, bookmark.Title, bookmark.Description, bookmark.FaviconURL,
		bookmark.UpdatedAt, bookmark.ScrapedAt, bookmark.ID)
	if err != nil {
		return fmt.Errorf("failed to update bookmark: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bookmark with ID %s not found", bookmark.ID)
	}

	// Get the bookmark's rowid to update FTS
	var rowid int64
	err = tx.QueryRow("SELECT rowid FROM bookmarks WHERE id = ?", bookmark.ID).Scan(&rowid)
	if err != nil {
		return fmt.Errorf("failed to get bookmark rowid: %w", err)
	}

	// Update or insert into FTS table
	_, err = tx.Exec("INSERT OR REPLACE INTO bookmarks_fts(rowid, title, description) VALUES (?, ?, ?)",
		rowid, bookmark.Title, bookmark.Description)
	if err != nil {
		return fmt.Errorf("failed to update bookmarks FTS: %w", err)
	}

	return tx.Commit()
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	TotalFound           int               `json:"total_found"`
	SuccessfullyImported int               `json:"successfully_imported"`
	Failed               int               `json:"failed"`
	Duplicates           int               `json:"duplicates"`
	ImportedFolders      []*BookmarkFolder `json:"imported_folders"`
	ImportedBookmarks    []*Bookmark       `json:"imported_bookmarks"`
	Errors               []string          `json:"errors"`
}

// StoreContent stores scraped content for a bookmark
func (s *Storage) StoreContent(bookmarkID string, rawContent string, cleanText string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete any existing content for this bookmark
	_, err = tx.Exec("DELETE FROM content WHERE bookmark_id = ?", bookmarkID)
	if err != nil {
		// Log but don't fail - the content might not exist yet
	}

	// Insert new content
	query := `INSERT INTO content (bookmark_id, raw_content, clean_text, scraped_at, content_type) 
	          VALUES (?, ?, ?, CURRENT_TIMESTAMP, 'text/html')`
	result, err := tx.Exec(query, bookmarkID, rawContent, cleanText)
	if err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}

	// Get the content ID for FTS update
	contentID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get content ID: %w", err)
	}

	// Update content FTS table
	_, err = tx.Exec("INSERT INTO content_fts(rowid, clean_text) VALUES (?, ?)", contentID, cleanText)
	if err != nil {
		return fmt.Errorf("failed to update content FTS: %w", err)
	}

	return tx.Commit()
}

// GetContent retrieves content by bookmark ID
func (s *Storage) GetContent(bookmarkID string) (*Content, error) {
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
			return nil, fmt.Errorf("content for bookmark ID %s not found", bookmarkID)
		}
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return content, nil
}

// StoreEmbedding stores a vector embedding for content (single chunk, index 0)
func (s *Storage) StoreEmbedding(contentID int, embedding []float32) error {
	return s.StoreChunkEmbedding(contentID, 0, embedding, "")
}

// StoreChunkEmbedding stores a vector embedding for a specific chunk of content
func (s *Storage) StoreChunkEmbedding(contentID int, chunkIndex int, embedding []float32, chunkText string) error {
	fmt.Printf("[StoreChunkEmbedding] Starting with contentID=%d, chunkIndex=%d, embedding length=%d\n", contentID, chunkIndex, len(embedding))

	// Convert float32 slice to JSON format for vector32() function
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	fmt.Printf("[StoreChunkEmbedding] JSON marshaled, length=%d bytes\n", len(embeddingJSON))

	query := `INSERT OR REPLACE INTO embeddings (content_id, chunk_index, chunk_text, embedding) VALUES (?, ?, ?, vector32(?))`
	fmt.Printf("[StoreChunkEmbedding] Executing query for chunk %d\n", chunkIndex)

	result, err := s.db.Exec(query, contentID, chunkIndex, chunkText, string(embeddingJSON))
	if err != nil {
		fmt.Printf("[StoreChunkEmbedding] ❌ Query execution failed: %v\n", err)
		return fmt.Errorf("failed to store chunk embedding: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	fmt.Printf("[StoreChunkEmbedding] ✓ Query successful: rows affected=%d, last insert ID=%d\n", rowsAffected, lastInsertID)

	return nil
}

// StoreMultipleChunkEmbeddings stores embeddings for multiple chunks in a transaction
func (s *Storage) StoreMultipleChunkEmbeddings(contentID int, embeddings [][]float32, chunks []string) error {
	if len(embeddings) != len(chunks) {
		return fmt.Errorf("embeddings count (%d) does not match chunks count (%d)", len(embeddings), len(chunks))
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing embeddings for this content
	_, err = tx.Exec(`DELETE FROM embeddings WHERE content_id = ?`, contentID)
	if err != nil {
		return fmt.Errorf("failed to clear existing embeddings: %w", err)
	}

	// Insert new chunk embeddings
	query := `INSERT INTO embeddings (content_id, chunk_index, chunk_text, embedding) VALUES (?, ?, ?, vector32(?))`
	for i, embedding := range embeddings {
		embeddingJSON, err := json.Marshal(embedding)
		if err != nil {
			return fmt.Errorf("failed to marshal embedding for chunk %d: %w", i, err)
		}

		_, err = tx.Exec(query, contentID, i, chunks[i], string(embeddingJSON))
		if err != nil {
			return fmt.Errorf("failed to store embedding for chunk %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit chunk embeddings transaction: %w", err)
	}

	fmt.Printf("[StoreMultipleChunkEmbeddings] ✓ Stored %d chunk embeddings for content %d\n", len(embeddings), contentID)
	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// Normalize keyword scores to 0-1 range using max BM25 score from results
	s.normalizeBM25Scores(keywordResults)

	// Combine and deduplicate results
	resultMap := make(map[string]*SearchResult)

	// Add semantic results with rebalanced weight and threshold
	for _, result := range semanticResults {
		// Apply minimum threshold for semantic results (0.3 = 30% similarity)
		if result.RelevanceScore < 0.3 {
			continue
		}
		
		result.RelevanceScore *= 0.4 // Rebalanced semantic weight (was 0.7, now 0.4)
		result.SearchType = "semantic"
		
		// Apply exact word match boost
		s.applyExactMatchBoost(result, queryText)
		
		// Apply field-specific boosting (title, URL)
		s.applyFieldSpecificBoost(result, queryText)
		
		resultMap[result.Bookmark.ID] = result
	}

	// Add keyword results, combining scores if bookmark already exists
	for _, result := range keywordResults {
		// Apply minimum threshold for keyword results (0.15 normalized BM25)
		if result.RelevanceScore < 0.15 {
			continue
		}
		
		result.RelevanceScore *= 0.6 // Rebalanced keyword weight (was 0.3, now 0.6)
		
		// Apply exact word match boost before combining
		s.applyExactMatchBoost(result, queryText)
		
		// Apply field-specific boosting (title, URL)
		s.applyFieldSpecificBoost(result, queryText)
		
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

// normalizeBM25Scores normalizes BM25 scores to 0-1 range based on the maximum score in results
func (s *Storage) normalizeBM25Scores(results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	
	// Find the maximum BM25 score
	maxScore := 0.0
	for _, result := range results {
		if result.RelevanceScore > maxScore {
			maxScore = result.RelevanceScore
		}
	}
	
	// Avoid division by zero
	if maxScore <= 0 {
		return
	}
	
	// Normalize all scores to 0-1 range
	for _, result := range results {
		result.RelevanceScore = result.RelevanceScore / maxScore
	}
}

// applyExactMatchBoost boosts scores for exact word matches in title, description, or content
func (s *Storage) applyExactMatchBoost(result *SearchResult, queryText string) {
	if queryText == "" {
		return
	}
	
	queryWords := strings.Fields(strings.ToLower(queryText))
	boostApplied := false
	
	// Check title for exact word matches
	titleLower := strings.ToLower(result.Bookmark.Title)
	for _, word := range queryWords {
		if strings.Contains(titleLower, word) {
			result.RelevanceScore *= 1.5 // 50% boost for title matches
			boostApplied = true
			break
		}
	}
	
	// Check description for exact word matches (if not already boosted)
	if !boostApplied && result.Bookmark.Description != "" {
		descLower := strings.ToLower(result.Bookmark.Description)
		for _, word := range queryWords {
			if strings.Contains(descLower, word) {
				result.RelevanceScore *= 1.3 // 30% boost for description matches
				boostApplied = true
				break
			}
		}
	}
	
	// Check content for exact word matches (if not already boosted)
	if !boostApplied && result.Content != nil && result.Content.CleanText != "" {
		contentLower := strings.ToLower(result.Content.CleanText)
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				result.RelevanceScore *= 1.2 // 20% boost for content matches
				break
			}
		}
	}
}

// applyFieldSpecificBoost applies high-priority boosts for exact matches in title and URL
func (s *Storage) applyFieldSpecificBoost(result *SearchResult, queryText string) {
	if queryText == "" {
		return
	}
	
	queryLower := strings.ToLower(queryText)
	titleLower := strings.ToLower(result.Bookmark.Title)
	urlLower := strings.ToLower(result.Bookmark.URL)
	
	// Check for exact title match (case-insensitive)
	if strings.Contains(titleLower, queryLower) {
		result.RelevanceScore *= 3.0 // 3x boost for exact title matches
		return // Don't apply URL boost if title already matched
	}
	
	// Check for URL matches (domain, path, or query parameters)
	if strings.Contains(urlLower, queryLower) {
		result.RelevanceScore *= 2.0 // 2x boost for URL matches
		return
	}
	
	// Check for individual word matches in title (less aggressive than exact match)
	queryWords := strings.Fields(queryLower)
	titleWords := strings.Fields(titleLower)
	
	titleMatches := 0
	for _, queryWord := range queryWords {
		for _, titleWord := range titleWords {
			if queryWord == titleWord {
				titleMatches++
				break
			}
		}
	}
	
	// Apply graduated boost based on word matches in title
	if titleMatches > 0 {
		matchRatio := float64(titleMatches) / float64(len(queryWords))
		if matchRatio >= 0.5 { // 50% or more words match
			result.RelevanceScore *= 2.0 // 2x boost for high word match ratio
		} else if matchRatio >= 0.25 { // 25% or more words match
			result.RelevanceScore *= 1.5 // 1.5x boost for moderate word match ratio
		}
	}
}

// semanticSearch performs vector similarity search using libSQL vector functions
func (s *Storage) semanticSearch(queryEmbedding []float32, limit int) ([]*SearchResult, error) {
	// Convert query embedding to JSON for vector32() function
	queryEmbeddingJSON, err := json.Marshal(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query embedding: %w", err)
	}

	query := `
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, ''),
		       c.id, c.bookmark_id, COALESCE(c.raw_content, ''), COALESCE(c.clean_text, ''),
		       c.scraped_at, c.content_type,
		       vector_distance_cos(e.embedding, vector32(?)) as similarity
		FROM embeddings e
		JOIN content c ON c.id = e.content_id
		JOIN bookmarks b ON b.id = c.bookmark_id
		WHERE vector_distance_cos(e.embedding, vector32(?)) < 1.0
		ORDER BY similarity ASC
		LIMIT ?
	`

	rows, err := s.db.Query(query, string(queryEmbeddingJSON), string(queryEmbeddingJSON), limit)
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

		// Convert cosine distance to similarity score (1 - distance)
		similarityScore := 1.0 - similarity

		result := &SearchResult{
			Bookmark:       bookmark,
			Content:        content,
			RelevanceScore: similarityScore,
			SearchType:     "semantic",
		}

		results = append(results, result)
	}

	return results, nil
}

// KeywordSearch performs only keyword-based search (public method)
func (s *Storage) KeywordSearch(queryText string, limit int) ([]*SearchResult, error) {
	return s.keywordSearch(queryText, limit)
}

// keywordSearch performs BM25-based full-text search
func (s *Storage) keywordSearch(queryText string, limit int) ([]*SearchResult, error) {
	// Escape FTS5 special characters and prepare query
	escapedQuery := strings.ReplaceAll(queryText, "'", "''")
	// Don't add quotes here - they'll be added by the SQL query
	ftsQuery := escapedQuery

	// Use UNION to combine bookmark title/description matches with content matches
	query := `
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, ''),
		       COALESCE(c.id, 0), COALESCE(c.bookmark_id, ''), COALESCE(c.raw_content, ''), COALESCE(c.clean_text, ''),
		       COALESCE(c.scraped_at, b.created_at), COALESCE(c.content_type, 'text/html'),
		       bm25(bookmarks_fts) as relevance,
		       '' as snippet
		FROM bookmarks_fts
		JOIN bookmarks b ON b.rowid = bookmarks_fts.rowid
		LEFT JOIN content c ON c.bookmark_id = b.id
		WHERE bookmarks_fts MATCH ?
		
		UNION
		
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, ''),
		       c.id, c.bookmark_id, c.raw_content, c.clean_text,
		       c.scraped_at, c.content_type,
		       bm25(content_fts) as relevance,
		       snippet(content_fts, 0, '<mark>', '</mark>', '...', 32) as snippet
		FROM content_fts
		JOIN content c ON c.id = content_fts.rowid
		JOIN bookmarks b ON b.id = c.bookmark_id
		WHERE content_fts MATCH ?
		
		ORDER BY relevance
		LIMIT ?
	`

	rows, err := s.db.Query(query, ftsQuery, ftsQuery, limit)
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

		// Convert BM25 score to similarity (higher is better for BM25)
		similarity := relevance

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
