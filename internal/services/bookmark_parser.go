package services

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"bookmark-chat/internal/services/parsers"
)

// BookmarkParserService handles parsing of bookmark files from different browsers
type BookmarkParserService struct {
	parsers []parsers.BookmarkParser
}

// NewBookmarkParserService creates a new bookmark parser service with all available parsers
func NewBookmarkParserService() *BookmarkParserService {
	return &BookmarkParserService{
		parsers: []parsers.BookmarkParser{
			parsers.NewFirefoxParser(),
			parsers.NewChromeParser(),
		},
	}
}

// ParseBookmarkFile attempts to parse a bookmark file by auto-detecting the format
func (s *BookmarkParserService) ParseBookmarkFile(reader io.Reader) (*parsers.ParseResult, error) {
	// Read the content to allow multiple parser attempts
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read bookmark file: %w", err)
	}

	// Try each parser until one succeeds
	for _, parser := range s.parsers {
		// Create a new reader for each parser attempt
		contentReader := bytes.NewReader(content)
		
		// Check if this parser can handle the format
		if parser.ValidateFormat(contentReader) {
			// Reset reader and parse
			contentReader = bytes.NewReader(content)
			result, err := parser.ParseFile(contentReader)
			if err != nil {
				// Log the error but try the next parser
				continue
			}
			return result, nil
		}
	}

	// If no parser could handle the format, try to determine what it might be
	contentStr := string(content)
	if len(contentStr) > 1024 {
		contentStr = contentStr[:1024] // Only check first 1KB for format hints
	}

	var formatHint string
	if strings.Contains(contentStr, "DOCTYPE NETSCAPE-Bookmark-file-1") {
		if strings.Contains(contentStr, "Bookmarks Menu") {
			formatHint = "Firefox (but validation failed)"
		} else if strings.Contains(contentStr, "<H1>Bookmarks</H1>") {
			formatHint = "Chrome (but validation failed)"
		} else {
			formatHint = "Netscape bookmark format (unknown variant)"
		}
	} else if strings.Contains(contentStr, "<") && strings.Contains(contentStr, ">") {
		formatHint = "HTML (but not recognized bookmark format)"
	} else {
		formatHint = "Unknown format"
	}

	return nil, fmt.Errorf("unsupported bookmark format detected: %s", formatHint)
}

// GetSupportedFormats returns a list of supported bookmark formats
func (s *BookmarkParserService) GetSupportedFormats() []string {
	formats := make([]string, len(s.parsers))
	for i, parser := range s.parsers {
		formats[i] = parser.GetSupportedFormat()
	}
	return formats
}

// ConvertToAPIFormat converts ParseResult to the format expected by the API
func (s *BookmarkParserService) ConvertToAPIFormat(result *parsers.ParseResult) *parsers.ImportResult {
	importResult := &parsers.ImportResult{
		Status: "success",
		Statistics: parsers.ImportStatistics{
			TotalFound:           result.TotalCount,
			SuccessfullyImported: result.TotalCount - len(result.Errors),
			Failed:               len(result.Errors),
			Duplicates:           0, // TODO: Implement duplicate detection
		},
	}

	// Convert parse errors to import errors
	for _, parseErr := range result.Errors {
		importResult.Errors = append(importResult.Errors, parsers.ImportError{
			URL:   parseErr.URL,
			Error: parseErr.Message,
		})
	}

	// Set status based on results
	if len(result.Errors) == 0 {
		importResult.Status = "success"
	} else if len(result.Errors) == result.TotalCount {
		importResult.Status = "failed"
	} else {
		importResult.Status = "partial"
	}

	return importResult
}

// FlattenFolderStructure extracts all bookmarks from the folder hierarchy into a flat list
func (s *BookmarkParserService) FlattenFolderStructure(folders []*parsers.BookmarkFolder) []parsers.Bookmark {
	var allBookmarks []parsers.Bookmark
	
	for _, folder := range folders {
		// Add bookmarks from this folder
		allBookmarks = append(allBookmarks, folder.Bookmarks...)
		
		// Recursively add bookmarks from subfolders
		if len(folder.Subfolders) > 0 {
			subfoldersBookmarks := s.FlattenFolderStructure(folder.Subfolders)
			allBookmarks = append(allBookmarks, subfoldersBookmarks...)
		}
	}
	
	return allBookmarks
}

// BuildFolderPathString converts folder path array to a string representation
func (s *BookmarkParserService) BuildFolderPathString(folderPath []string) string {
	if len(folderPath) == 0 {
		return ""
	}
	return strings.Join(folderPath, "/")
}