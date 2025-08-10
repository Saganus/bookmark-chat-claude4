package parsers

import (
	"io"
	"time"
)

// Bookmark represents a parsed bookmark with its metadata
type Bookmark struct {
	URL        string
	Title      string
	DateAdded  time.Time
	FolderPath []string // Hierarchical path like ["Technology", "Databases"]
	Icon       string   // Base64 encoded icon data or URL
}

// BookmarkFolder represents a folder in the bookmark hierarchy
type BookmarkFolder struct {
	Name       string
	Path       []string
	Bookmarks  []Bookmark
	Subfolders []*BookmarkFolder
}

// ParseResult contains the complete parsing result
type ParseResult struct {
	Source     string           // "Firefox" or "Chrome"
	ParsedAt   time.Time
	Bookmarks  []Bookmark       // Flattened list of all bookmarks
	Folders    []*BookmarkFolder // Hierarchical folder structure
	TotalCount int
	Errors     []ParseError
}

// ParseError represents an error that occurred during parsing
type ParseError struct {
	URL     string
	Message string
	Line    int
}

// BookmarkParser defines the interface for bookmark file parsers
type BookmarkParser interface {
	// ParseFile parses a bookmark file from the given reader
	ParseFile(reader io.Reader) (*ParseResult, error)
	
	// GetSupportedFormat returns the format name this parser supports
	GetSupportedFormat() string
	
	// ValidateFormat checks if the given content matches this parser's format
	ValidateFormat(reader io.Reader) bool
}

// ImportStatistics holds statistics for the import process
type ImportStatistics struct {
	TotalFound           int
	SuccessfullyImported int
	Failed               int
	Duplicates           int
}

// ImportResult contains the complete import result
type ImportResult struct {
	Status     string           // "success", "partial", "failed"
	Statistics ImportStatistics
	Errors     []ImportError
}

// ImportError represents an error during the import process
type ImportError struct {
	URL   string
	Error string
}