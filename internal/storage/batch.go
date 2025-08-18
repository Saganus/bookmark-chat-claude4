package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// BatchOperations provides batch processing capabilities for efficiency
type BatchOperations struct {
	storage *Storage
}

// NewBatchOperations creates a new batch operations instance
func (s *Storage) NewBatchOperations() *BatchOperations {
	return &BatchOperations{storage: s}
}

// BatchAddBookmarks adds multiple bookmarks in a single transaction
func (bo *BatchOperations) BatchAddBookmarks(bookmarks []struct {
	URL   string
	Title string
}) error {
	tx, err := bo.storage.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO bookmarks (url, title) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, bookmark := range bookmarks {
		_, err := stmt.Exec(bookmark.URL, bookmark.Title)
		if err != nil {
			return fmt.Errorf("failed to insert bookmark %s: %w", bookmark.URL, err)
		}
	}

	return tx.Commit()
}

// BatchStoreEmbeddings stores multiple embeddings in a single transaction
func (bo *BatchOperations) BatchStoreEmbeddings(embeddings []struct {
	ContentID int
	Embedding []float32
}) error {
	tx, err := bo.storage.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO embeddings (content_id, embedding) VALUES (?, vector32(?))`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, emb := range embeddings {
		embeddingJSON, err := json.Marshal(emb.Embedding)
		if err != nil {
			return fmt.Errorf("failed to marshal embedding for content %d: %w", emb.ContentID, err)
		}

		_, err = stmt.Exec(emb.ContentID, string(embeddingJSON))
		if err != nil {
			return fmt.Errorf("failed to insert embedding for content %d: %w", emb.ContentID, err)
		}
	}

	return tx.Commit()
}

// GetBookmarksWithoutEmbeddings returns bookmarks that don't have embeddings yet
func (s *Storage) GetBookmarksWithoutEmbeddings(limit int) ([]*Bookmark, error) {
	query := `
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, '')
		FROM bookmarks b
		JOIN content c ON c.bookmark_id = b.id
		LEFT JOIN embeddings e ON e.content_id = c.id
		WHERE e.id IS NULL AND c.clean_text IS NOT NULL AND c.clean_text != ''
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmarks without embeddings: %w", err)
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

// GetStats returns database statistics
func (s *Storage) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	queries := map[string]string{
		"total_bookmarks":        "SELECT COUNT(*) FROM bookmarks",
		"completed_bookmarks":    "SELECT COUNT(*) FROM bookmarks WHERE status = 'completed'",
		"pending_bookmarks":      "SELECT COUNT(*) FROM bookmarks WHERE status = 'pending'",
		"failed_bookmarks":       "SELECT COUNT(*) FROM bookmarks WHERE status = 'failed'",
		"total_content_entries":  "SELECT COUNT(*) FROM content",
		"total_embeddings":       "SELECT COUNT(*) FROM embeddings",
		"bookmarks_with_content": "SELECT COUNT(DISTINCT bookmark_id) FROM content WHERE clean_text IS NOT NULL",
		"bookmarks_with_embeddings": `
			SELECT COUNT(DISTINCT c.bookmark_id) 
			FROM content c 
			JOIN embeddings e ON e.content_id = c.id
		`,
	}

	for statName, query := range queries {
		var count int
		err := s.db.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s: %w", statName, err)
		}
		stats[statName] = count
	}

	return stats, nil
}

// SearchBookmarksWithFilters provides advanced search with filtering options
func (s *Storage) SearchBookmarksWithFilters(opts SearchOptions) ([]*SearchResult, error) {
	baseQuery := `
		SELECT b.id, b.url, b.title, b.status, b.imported_at, b.created_at, b.updated_at,
		       COALESCE(b.folder_path, ''), COALESCE(b.description, ''),
		       c.id, c.bookmark_id, COALESCE(c.raw_content, ''), COALESCE(c.clean_text, ''),
		       c.scraped_at, c.content_type
		FROM bookmarks b
		LEFT JOIN content c ON c.bookmark_id = b.id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	if opts.Status != "" {
		baseQuery += fmt.Sprintf(" AND b.status = ?%d", argIndex)
		args = append(args, opts.Status)
		argIndex++
	}

	if opts.FolderPath != "" {
		baseQuery += fmt.Sprintf(" AND b.folder_path LIKE ?%d", argIndex)
		args = append(args, "%"+opts.FolderPath+"%")
		argIndex++
	}

	if !opts.CreatedAfter.IsZero() {
		baseQuery += fmt.Sprintf(" AND b.created_at >= ?%d", argIndex)
		args = append(args, opts.CreatedAfter)
		argIndex++
	}

	if !opts.CreatedBefore.IsZero() {
		baseQuery += fmt.Sprintf(" AND b.created_at <= ?%d", argIndex)
		args = append(args, opts.CreatedBefore)
		argIndex++
	}

	// Add ordering and limits
	baseQuery += " ORDER BY b.created_at DESC"
	if opts.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT ?%d", argIndex)
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search bookmarks with filters: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		bookmark := &Bookmark{}
		content := &Content{}
		var contentID sql.NullInt64
		var bookmarkID sql.NullInt64
		var rawContent, cleanText sql.NullString
		var scrapedAt sql.NullTime
		var contentType sql.NullString

		err := rows.Scan(
			&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Status,
			&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
			&bookmark.FolderPath, &bookmark.Description,
			&contentID, &bookmarkID, &rawContent, &cleanText,
			&scrapedAt, &contentType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan filtered search result: %w", err)
		}

		result := &SearchResult{
			Bookmark:       bookmark,
			RelevanceScore: 1.0,
			SearchType:     "filtered",
		}

		// Add content if available
		if contentID.Valid {
			content.ID = int(contentID.Int64)
			content.BookmarkID = int(bookmarkID.Int64)
			content.RawContent = rawContent.String
			content.CleanText = cleanText.String
			content.ContentType = contentType.String
			if scrapedAt.Valid {
				content.ScrapedAt = scrapedAt.Time
			}
			result.Content = content
		}

		results = append(results, result)
	}

	return results, nil
}

// SearchOptions defines filtering options for bookmark searches
type SearchOptions struct {
	Status        string
	FolderPath    string
	CreatedAfter  time.Time
	CreatedBefore time.Time
	Limit         int
}

// DeleteBookmark removes a bookmark and all associated data
func (s *Storage) DeleteBookmark(bookmarkID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete from FTS table first
	_, err = tx.Exec("DELETE FROM bookmarks_fts WHERE rowid = ?", bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to delete from FTS table: %w", err)
	}

	// Delete embeddings (cascade will handle this, but explicit is better)
	_, err = tx.Exec(`
		DELETE FROM embeddings 
		WHERE content_id IN (SELECT id FROM content WHERE bookmark_id = ?)
	`, bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}

	// Delete content
	_, err = tx.Exec("DELETE FROM content WHERE bookmark_id = ?", bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}

	// Delete bookmark
	result, err := tx.Exec("DELETE FROM bookmarks WHERE id = ?", bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bookmark with ID %d not found", bookmarkID)
	}

	return tx.Commit()
}
