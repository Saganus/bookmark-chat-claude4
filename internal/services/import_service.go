package services

import (
	"fmt"
	"io"
	"mime/multipart"

	"bookmark-chat/internal/services/parsers"
)

// ImportService handles the complete bookmark import process
type ImportService struct {
	parserService *BookmarkParserService
}

// NewImportService creates a new import service
func NewImportService() *ImportService {
	return &ImportService{
		parserService: NewBookmarkParserService(),
	}
}

// ImportBookmarksFromFile handles the complete import process from an uploaded file
func (s *ImportService) ImportBookmarksFromFile(fileHeader *multipart.FileHeader) (*parsers.ImportResult, *parsers.ParseResult, error) {
	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Parse the bookmark file
	parseResult, err := s.parserService.ParseBookmarkFile(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse bookmark file: %w", err)
	}

	// Convert to API format
	importResult := s.parserService.ConvertToAPIFormat(parseResult)

	// TODO: In the future, this is where we would:
	// 1. Check for duplicates against existing bookmarks in database
	// 2. Store bookmarks in database
	// 3. Queue URLs for scraping
	// 4. Generate embeddings
	// 5. Update statistics with actual import results

	return importResult, parseResult, nil
}

// ImportBookmarksFromReader handles import from an io.Reader (useful for testing)
func (s *ImportService) ImportBookmarksFromReader(reader io.Reader) (*parsers.ImportResult, *parsers.ParseResult, error) {
	// Parse the bookmark file
	parseResult, err := s.parserService.ParseBookmarkFile(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse bookmark file: %w", err)
	}

	// Convert to API format
	importResult := s.parserService.ConvertToAPIFormat(parseResult)

	return importResult, parseResult, nil
}

// GetSupportedFormats returns the list of supported bookmark formats
func (s *ImportService) GetSupportedFormats() []string {
	return s.parserService.GetSupportedFormats()
}

// ValidateFile performs basic validation on the uploaded file
func (s *ImportService) ValidateFile(fileHeader *multipart.FileHeader) error {
	// Check file size (limit to 50MB)
	const maxFileSize = 50 * 1024 * 1024 // 50MB
	if fileHeader.Size > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", fileHeader.Size, maxFileSize)
	}

	// Check file extension (allow .html, .htm)
	filename := fileHeader.Filename
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	// Basic extension check - both Firefox and Chrome export as HTML
	if !(len(filename) > 4 && (filename[len(filename)-5:] == ".html" || filename[len(filename)-4:] == ".htm")) {
		return fmt.Errorf("unsupported file extension: expected .html or .htm")
	}

	return nil
}

// GetImportPreview provides a preview of what would be imported without actually importing
func (s *ImportService) GetImportPreview(reader io.Reader) (*ImportPreview, error) {
	parseResult, err := s.parserService.ParseBookmarkFile(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bookmark file for preview: %w", err)
	}

	preview := &ImportPreview{
		Format:          parseResult.Source,
		TotalBookmarks:  parseResult.TotalCount,
		FolderCount:     len(parseResult.Folders),
		SampleBookmarks: make([]BookmarkPreview, 0),
		FolderStructure: s.buildFolderPreview(parseResult.Folders),
	}

	// Add up to 5 sample bookmarks
	sampleCount := min(5, len(parseResult.Bookmarks))
	for i := 0; i < sampleCount; i++ {
		bookmark := parseResult.Bookmarks[i]
		preview.SampleBookmarks = append(preview.SampleBookmarks, BookmarkPreview{
			Title:      bookmark.Title,
			URL:        bookmark.URL,
			FolderPath: s.parserService.BuildFolderPathString(bookmark.FolderPath),
		})
	}

	return preview, nil
}

// ImportPreview represents a preview of what will be imported
type ImportPreview struct {
	Format          string            `json:"format"`
	TotalBookmarks  int               `json:"total_bookmarks"`
	FolderCount     int               `json:"folder_count"`
	SampleBookmarks []BookmarkPreview `json:"sample_bookmarks"`
	FolderStructure []FolderPreview   `json:"folder_structure"`
}

// BookmarkPreview represents a preview of a single bookmark
type BookmarkPreview struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	FolderPath string `json:"folder_path"`
}

// FolderPreview represents a preview of a folder structure
type FolderPreview struct {
	Name           string          `json:"name"`
	Path           string          `json:"path"`
	BookmarkCount  int             `json:"bookmark_count"`
	SubfolderCount int             `json:"subfolder_count"`
	Subfolders     []FolderPreview `json:"subfolders,omitempty"`
}

// buildFolderPreview builds a preview of the folder structure
func (s *ImportService) buildFolderPreview(folders []*parsers.BookmarkFolder) []FolderPreview {
	previews := make([]FolderPreview, len(folders))
	
	for i, folder := range folders {
		previews[i] = FolderPreview{
			Name:           folder.Name,
			Path:           s.parserService.BuildFolderPathString(folder.Path),
			BookmarkCount:  len(folder.Bookmarks),
			SubfolderCount: len(folder.Subfolders),
		}
		
		// Recursively build subfolder previews (limit depth to avoid huge responses)
		if len(folder.Subfolders) > 0 && len(folder.Path) < 5 {
			previews[i].Subfolders = s.buildFolderPreview(folder.Subfolders)
		}
	}
	
	return previews
}

// Helper function for min (since Go doesn't have a built-in generic min)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}