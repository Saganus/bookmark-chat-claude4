package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Category represents a bookmark category
type Category struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	ParentCategory *string   `json:"parent_category,omitempty"`
	Color          string    `json:"color"`
	UsageCount     int       `json:"usage_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CategorizationResult represents the AI categorization result
type CategorizationResult struct {
	PrimaryCategory      string   `json:"primary_category"`
	SecondaryCategories  []string `json:"secondary_categories"`
	Tags                 []string `json:"tags"`
	ConfidenceScore     float64  `json:"confidence_score"`
	Reasoning           string   `json:"reasoning"`
}

// BookmarkCategorization represents the full categorization state of a bookmark
type BookmarkCategorization struct {
	BookmarkID          string               `json:"bookmark_id"`
	Categorization      CategorizationResult `json:"categorization"`
	UserApproved        bool                 `json:"user_approved"`
	CategorizationDate  time.Time           `json:"categorization_date"`
}

// applyCategorization runs the categorization migration
func (s *Storage) applyCategorization() error {
	// Check if migration was already applied by looking for categories table
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='categories'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for categories table: %w", err)
	}

	if count > 0 {
		// Migration already applied, check for new columns in bookmarks table
		var columnCount int
		err = s.db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('bookmarks') 
			WHERE name IN ('categorization_date', 'categorization_confidence', 'categorization_status')
		`).Scan(&columnCount)
		
		if err == nil && columnCount >= 3 {
			return nil // All migration changes already applied
		}
	}

	// Read and execute migration
	migrationPath := "internal/storage/migrations/003_add_categorization.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Split SQL statements and execute them one by one
	statements := strings.Split(string(migrationSQL), ";")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		_, err = s.db.Exec(statement)
		if err != nil {
			// Ignore errors for ALTER TABLE on existing columns
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("failed to execute migration statement '%s': %w", statement, err)
		}
	}

	return nil
}

// SaveCategorizationResult stores AI categorization suggestions
func (s *Storage) SaveCategorizationResult(ctx context.Context, bookmarkID string, result CategorizationResult) error {
	return s.retryWithBackoff(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback()

		// Create or get primary category
		primaryCatID, err := s.getOrCreateCategory(tx, result.PrimaryCategory, nil)
		if err != nil {
			return fmt.Errorf("create primary category: %w", err)
		}

		// Clear existing categories for this bookmark
		_, err = tx.ExecContext(ctx, "DELETE FROM bookmark_categories WHERE bookmark_id = ?", bookmarkID)
		if err != nil {
			return fmt.Errorf("clear existing categories: %w", err)
		}

		// Link primary category to bookmark
		_, err = tx.ExecContext(ctx, `
			INSERT INTO bookmark_categories (bookmark_id, category_id, is_primary, confidence_score, user_approved)
			VALUES (?, ?, TRUE, ?, FALSE)
		`, bookmarkID, primaryCatID, result.ConfidenceScore)
		if err != nil {
			return fmt.Errorf("link primary category: %w", err)
		}

		// Handle secondary categories
		for _, catName := range result.SecondaryCategories {
			catID, err := s.getOrCreateCategory(tx, catName, nil)
			if err != nil {
				continue
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO bookmark_categories (bookmark_id, category_id, is_primary, confidence_score, user_approved)
				VALUES (?, ?, FALSE, ?, FALSE)
			`, bookmarkID, catID, result.ConfidenceScore)
			if err != nil {
				// Continue on error for secondary categories
				continue
			}
		}

		// Clear existing tags for this bookmark
		_, err = tx.ExecContext(ctx, "DELETE FROM bookmark_tags WHERE bookmark_id = ?", bookmarkID)
		if err != nil {
			return fmt.Errorf("clear existing tags: %w", err)
		}

		// Handle tags
		for _, tagName := range result.Tags {
			tagID, err := s.getOrCreateTag(tx, tagName)
			if err != nil {
				continue
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO bookmark_tags (bookmark_id, tag_id)
				VALUES (?, ?)
			`, bookmarkID, tagID)
			if err != nil {
				// Continue on error for tags
				continue
			}
		}

		// Update bookmark metadata
		_, err = tx.ExecContext(ctx, `
			UPDATE bookmarks 
			SET categorization_date = CURRENT_TIMESTAMP,
				categorization_confidence = ?,
				categorization_status = 'completed'
			WHERE id = ?
		`, result.ConfidenceScore, bookmarkID)
		if err != nil {
			return fmt.Errorf("update bookmark metadata: %w", err)
		}

		return tx.Commit()
	})
}

// GetCategories returns all categories with usage stats
func (s *Storage) GetCategories(ctx context.Context) ([]Category, error) {
	query := `
		SELECT id, name, parent_category, COALESCE(color, ''), usage_count, created_at, updated_at
		FROM categories
		ORDER BY usage_count DESC, name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.ParentCategory, &cat.Color,
			&cat.UsageCount, &cat.CreatedAt, &cat.UpdatedAt)
		if err != nil {
			continue
		}
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

// GetBookmarksNeedingCategorization returns bookmarks without categories
func (s *Storage) GetBookmarksNeedingCategorization(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT b.id 
		FROM bookmarks b
		LEFT JOIN bookmark_categories bc ON b.id = bc.bookmark_id
		WHERE bc.bookmark_id IS NULL 
		   OR b.categorization_status = 'pending'
		   OR b.categorization_status IS NULL
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// GetBookmarkWithContent retrieves a bookmark with its content for categorization
func (s *Storage) GetBookmarkWithContent(ctx context.Context, bookmarkID string) (*Bookmark, error) {
	query := `
		SELECT b.id, b.url, b.title, COALESCE(b.description, ''), b.status, b.imported_at, 
			   b.created_at, b.updated_at, b.scraped_at, b.folder_id, 
			   COALESCE(b.folder_path, ''), COALESCE(b.favicon_url, ''), 
			   COALESCE(b.tags, '[]'), COALESCE(c.clean_text, '')
		FROM bookmarks b
		LEFT JOIN content c ON c.bookmark_id = b.id
		WHERE b.id = ?
	`

	row := s.db.QueryRowContext(ctx, query, bookmarkID)

	bookmark := &Bookmark{}
	var tagsJSON, content string
	err := row.Scan(
		&bookmark.ID, &bookmark.URL, &bookmark.Title, &bookmark.Description, &bookmark.Status,
		&bookmark.ImportedAt, &bookmark.CreatedAt, &bookmark.UpdatedAt,
		&bookmark.ScrapedAt, &bookmark.FolderID, &bookmark.FolderPath, &bookmark.FaviconURL, 
		&tagsJSON, &content,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bookmark with ID %s not found", bookmarkID)
		}
		return nil, fmt.Errorf("failed to get bookmark: %w", err)
	}

	// Parse tags JSON
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &bookmark.Tags); err != nil {
			bookmark.Tags = []string{}
		}
	}

	// Add content to description if available for categorization
	if content != "" {
		if bookmark.Description != "" {
			bookmark.Description += "\n\nContent: " + content[:min(len(content), 1000)]
		} else {
			bookmark.Description = "Content: " + content[:min(len(content), 1000)]
		}
	}

	return bookmark, nil
}

// ApproveCategorizationResult marks a categorization as user-approved
func (s *Storage) ApproveCategorizationResult(ctx context.Context, bookmarkID string) error {
	return s.retryWithBackoff(func() error {
		_, err := s.db.ExecContext(ctx, `
			UPDATE bookmark_categories 
			SET user_approved = TRUE 
			WHERE bookmark_id = ?
		`, bookmarkID)
		if err != nil {
			return fmt.Errorf("failed to approve categorization: %w", err)
		}
		return nil
	})
}

// Helper functions
func (s *Storage) getOrCreateCategory(tx *sql.Tx, name string, parent *string) (int, error) {
	// First try to get existing category
	var id int
	err := tx.QueryRow("SELECT id FROM categories WHERE name = ? AND COALESCE(parent_category, '') = COALESCE(?, '')", 
		name, parent).Scan(&id)
	if err == nil {
		// Update usage count
		tx.Exec("UPDATE categories SET usage_count = usage_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
		return id, nil
	}

	// Create new category
	result, err := tx.Exec(`
		INSERT INTO categories (name, parent_category, color, usage_count) 
		VALUES (?, ?, ?, 1)
	`, name, parent, s.generateCategoryColor(name))
	if err != nil {
		return 0, err
	}

	insertID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(insertID), nil
}

func (s *Storage) getOrCreateTag(tx *sql.Tx, name string) (int, error) {
	// First try to get existing tag
	var id int
	err := tx.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id)
	if err == nil {
		// Update usage count
		tx.Exec("UPDATE tags SET usage_count = usage_count + 1 WHERE id = ?", id)
		return id, nil
	}

	// Create new tag
	result, err := tx.Exec("INSERT INTO tags (name, usage_count) VALUES (?, 1)", name)
	if err != nil {
		return 0, err
	}

	insertID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(insertID), nil
}

// generateCategoryColor creates a consistent color for a category based on its name
func (s *Storage) generateCategoryColor(name string) string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FECA57",
		"#FF9FF3", "#54A0FF", "#5F27CD", "#00D2D3", "#FF9F43",
		"#C7ECEE", "#DDA0DD", "#98D8C8", "#F7DC6F", "#BB8FCE",
	}
	
	hash := 0
	for _, char := range name {
		hash = int(char) + ((hash << 5) - hash)
	}
	
	if hash < 0 {
		hash = -hash
	}
	
	return colors[hash%len(colors)]
}

