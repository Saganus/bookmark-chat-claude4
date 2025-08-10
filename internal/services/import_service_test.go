package services

import (
	"os"
	"strings"
	"testing"

	"bookmark-chat/internal/services/parsers"
)

func TestImportService_ImportBookmarksFromReader_Firefox(t *testing.T) {
	service := NewImportService()
	
	file, err := os.Open("../../test_firefox_bookmarks.html")
	if err != nil {
		t.Fatalf("Failed to open test_firefox_bookmarks.html: %v", err)
	}
	defer file.Close()

	importResult, parseResult, err := service.ImportBookmarksFromReader(file)
	if err != nil {
		t.Fatalf("Failed to parse Firefox bookmarks: %v", err)
	}

	// Validate parse result
	validateFirefoxParseResult(t, parseResult)
	
	// Validate import result
	validateImportResult(t, importResult, parseResult)
	
	// Test folder structure
	validateFirefoxFolderStructure(t, parseResult.Folders)
	
	// Test bookmark details
	validateFirefoxBookmarkDetails(t, parseResult.Bookmarks)
}

func TestImportService_ImportBookmarksFromReader_Chrome(t *testing.T) {
	service := NewImportService()
	
	file, err := os.Open("../../test_chrome_bookmarks.html")
	if err != nil {
		t.Fatalf("Failed to open test_chrome_bookmarks.html: %v", err)
	}
	defer file.Close()

	importResult, parseResult, err := service.ImportBookmarksFromReader(file)
	if err != nil {
		t.Fatalf("Failed to parse Chrome bookmarks: %v", err)
	}

	// Validate parse result
	validateChromeParseResult(t, parseResult)
	
	// Validate import result
	validateImportResult(t, importResult, parseResult)
	
	// Test folder structure
	validateChromeFolderStructure(t, parseResult.Folders)
	
	// Test bookmark details
	validateChromeBookmarkDetails(t, parseResult.Bookmarks)
}

func TestImportService_InvalidFile(t *testing.T) {
	service := NewImportService()
	
	// Test with non-existent file
	_, _, err := service.ImportBookmarksFromReader(strings.NewReader("invalid content"))
	if err == nil {
		t.Error("Expected error when parsing invalid content, got nil")
	}
}

func TestImportService_GetSupportedFormats(t *testing.T) {
	service := NewImportService()
	formats := service.GetSupportedFormats()
	
	if len(formats) == 0 {
		t.Error("Expected at least one supported format")
	}
	
	// Should support common browser formats
	hasFirefox := false
	hasChrome := false
	for _, format := range formats {
		if strings.Contains(strings.ToLower(format), "firefox") {
			hasFirefox = true
		}
		if strings.Contains(strings.ToLower(format), "chrome") {
			hasChrome = true
		}
	}
	
	if !hasFirefox {
		t.Error("Expected Firefox format support")
	}
	if !hasChrome {
		t.Error("Expected Chrome format support")
	}
}

// Helper functions for validation

func validateFirefoxParseResult(t *testing.T, result *parsers.ParseResult) {
	t.Helper()
	
	if result == nil {
		t.Fatal("ParseResult is nil")
	}
	
	if result.Source != "Firefox" {
		t.Errorf("Expected source 'Firefox', got '%s'", result.Source)
	}
	
	if result.TotalCount <= 0 {
		t.Errorf("Expected positive bookmark count, got %d", result.TotalCount)
	}
	
	if len(result.Bookmarks) != result.TotalCount {
		t.Errorf("Bookmark count mismatch: TotalCount=%d, len(Bookmarks)=%d", 
			result.TotalCount, len(result.Bookmarks))
	}
	
	if len(result.Folders) == 0 {
		t.Error("Expected at least one folder")
	}
	
	t.Logf("Firefox parsing results: Format=%s, Total=%d, Folders=%d", 
		result.Source, result.TotalCount, len(result.Folders))
}

func validateChromeParseResult(t *testing.T, result *parsers.ParseResult) {
	t.Helper()
	
	if result == nil {
		t.Fatal("ParseResult is nil")
	}
	
	if result.Source != "Chrome" {
		t.Errorf("Expected source 'Chrome', got '%s'", result.Source)
	}
	
	if result.TotalCount <= 0 {
		t.Errorf("Expected positive bookmark count, got %d", result.TotalCount)
	}
	
	if len(result.Bookmarks) != result.TotalCount {
		t.Errorf("Bookmark count mismatch: TotalCount=%d, len(Bookmarks)=%d", 
			result.TotalCount, len(result.Bookmarks))
	}
	
	t.Logf("Chrome parsing results: Format=%s, Total=%d, Folders=%d", 
		result.Source, result.TotalCount, len(result.Folders))
}

func validateImportResult(t *testing.T, importResult *parsers.ImportResult, parseResult *parsers.ParseResult) {
	t.Helper()
	
	if importResult == nil {
		t.Fatal("ImportResult is nil")
	}
	
	if importResult.Status == "" {
		t.Error("Expected non-empty status")
	}
	
	// Statistics should match parse results for basic imports
	if importResult.Statistics.TotalFound != parseResult.TotalCount {
		t.Errorf("Statistics mismatch: TotalFound=%d, ParseResult.TotalCount=%d", 
			importResult.Statistics.TotalFound, parseResult.TotalCount)
	}
	
	// For basic parsing without database, all should be "successfully imported"
	if importResult.Statistics.SuccessfullyImported != parseResult.TotalCount {
		t.Errorf("Expected all bookmarks to be successfully imported: got %d, expected %d", 
			importResult.Statistics.SuccessfullyImported, parseResult.TotalCount)
	}
	
	t.Logf("Import results: Status=%s, Imported=%d, Failed=%d", 
		importResult.Status, importResult.Statistics.SuccessfullyImported, importResult.Statistics.Failed)
}

func validateFirefoxFolderStructure(t *testing.T, folders []*parsers.BookmarkFolder) {
	t.Helper()
	
	if len(folders) == 0 {
		t.Error("Expected at least one folder for Firefox bookmarks")
		return
	}
	
	// Firefox parser creates a single root "Bookmarks" folder containing all bookmarks
	// The hierarchical structure is preserved in each bookmark's FolderPath field
	if len(folders) != 1 {
		t.Errorf("Expected exactly 1 root folder for Firefox bookmarks, got %d", len(folders))
		return
	}
	
	rootFolder := folders[0]
	if rootFolder.Name != "Bookmarks" {
		t.Errorf("Expected root folder name 'Bookmarks', got '%s'", rootFolder.Name)
	}
	
	// Validate that all bookmarks are in the root folder
	expectedBookmarkCount := 24 // Based on test file
	if len(rootFolder.Bookmarks) != expectedBookmarkCount {
		t.Errorf("Expected %d bookmarks in root folder, got %d", expectedBookmarkCount, len(rootFolder.Bookmarks))
	}
	
	// Validate that folder paths are preserved in individual bookmarks
	folderPathCounts := make(map[string]int)
	for _, bookmark := range rootFolder.Bookmarks {
		folderPath := strings.Join(bookmark.FolderPath, "/")
		if folderPath == "" {
			folderPath = "(root)"
		}
		folderPathCounts[folderPath]++
	}
	
	// Expected folder paths based on test file structure
	expectedFolderPaths := map[string]int{
		"Bookmarks Toolbar":          3,
		"Technology/Databases":       3,
		"Technology/AI & Machine Learning": 4,
		"Technology/Web Development": 3,
		"Science & Reference":        4,
		"Tools & Platforms":         4,
		"Industry & News":           3,
	}
	
	// Verify expected folder paths
	for expectedPath, expectedCount := range expectedFolderPaths {
		if actualCount, found := folderPathCounts[expectedPath]; !found {
			t.Errorf("Expected to find bookmarks in folder path '%s'", expectedPath)
		} else if actualCount != expectedCount {
			t.Errorf("Expected %d bookmarks in folder path '%s', got %d", expectedCount, expectedPath, actualCount)
		}
	}
	
	// Log folder structure for debugging
	t.Logf("Root folder: %s (Path: %v) - %d bookmarks, %d subfolders", 
		rootFolder.Name, rootFolder.Path, len(rootFolder.Bookmarks), len(rootFolder.Subfolders))
	t.Logf("Folder path distribution: %+v", folderPathCounts)
}

func validateChromeFolderStructure(t *testing.T, folders []*parsers.BookmarkFolder) {
	t.Helper()
	
	if len(folders) == 0 {
		t.Error("Expected at least one folder for Chrome bookmarks")
		return
	}
	
	// Chrome parser creates a single root "Bookmarks" folder containing all bookmarks
	// The hierarchical structure is preserved in each bookmark's FolderPath field
	if len(folders) != 1 {
		t.Errorf("Expected exactly 1 root folder for Chrome bookmarks, got %d", len(folders))
		return
	}
	
	rootFolder := folders[0]
	if rootFolder.Name != "Bookmarks" {
		t.Errorf("Expected root folder name 'Bookmarks', got '%s'", rootFolder.Name)
	}
	
	// Validate that all bookmarks are in the root folder
	expectedBookmarkCount := 3 // Based on test file: 2 in Bookmarks Bar + 1 at root level
	if len(rootFolder.Bookmarks) != expectedBookmarkCount {
		t.Errorf("Expected %d bookmarks in root folder, got %d", expectedBookmarkCount, len(rootFolder.Bookmarks))
	}
	
	// Validate that folder paths are preserved in individual bookmarks
	folderPathCounts := make(map[string]int)
	for _, bookmark := range rootFolder.Bookmarks {
		folderPath := strings.Join(bookmark.FolderPath, "/")
		if folderPath == "" {
			folderPath = "(root)"
		}
		folderPathCounts[folderPath]++
	}
	
	// Expected folder paths based on Chrome test file structure
	expectedFolderPaths := map[string]int{
		"Bookmarks Bar": 2, // 2 bookmarks in Bookmarks Bar
		"(root)":        1, // 1 bookmark at root level
	}
	
	// Verify expected folder paths
	for expectedPath, expectedCount := range expectedFolderPaths {
		if actualCount, found := folderPathCounts[expectedPath]; !found {
			t.Errorf("Expected to find bookmarks in folder path '%s'", expectedPath)
		} else if actualCount != expectedCount {
			t.Errorf("Expected %d bookmarks in folder path '%s', got %d", expectedCount, expectedPath, actualCount)
		}
	}
	
	// Log folder structure for debugging
	t.Logf("Root folder: %s (Path: %v) - %d bookmarks, %d subfolders", 
		rootFolder.Name, rootFolder.Path, len(rootFolder.Bookmarks), len(rootFolder.Subfolders))
	t.Logf("Folder path distribution: %+v", folderPathCounts)
}

func validateFirefoxBookmarkDetails(t *testing.T, bookmarks []parsers.Bookmark) {
	t.Helper()
	
	if len(bookmarks) == 0 {
		t.Error("Expected at least one bookmark")
		return
	}
	
	// Log first 5 bookmarks for verification (matching test_parser.go behavior)
	t.Log("First 5 Firefox bookmarks:")
	for i, bookmark := range bookmarks {
		if i >= 5 {
			break
		}
		
		// Validate required fields
		if bookmark.URL == "" {
			t.Errorf("Bookmark %d has empty URL", i+1)
		}
		if bookmark.Title == "" {
			t.Errorf("Bookmark %d has empty Title", i+1)
		}
		
		folderPath := strings.Join(bookmark.FolderPath, "/")
		if folderPath == "" {
			folderPath = "(root)"
		}
		
		t.Logf("  %d. [%s] %s\n     URL: %s", i+1, folderPath, bookmark.Title, bookmark.URL)
		
		// Test for expected bookmarks from test file
		if bookmark.URL == "https://en.wikipedia.org/wiki/Machine_learning" {
			if bookmark.Title != "Machine learning - Wikipedia" {
				t.Errorf("Expected title 'Machine learning - Wikipedia', got '%s'", bookmark.Title)
			}
		}
	}
}

func validateChromeBookmarkDetails(t *testing.T, bookmarks []parsers.Bookmark) {
	t.Helper()
	
	if len(bookmarks) == 0 {
		t.Error("Expected at least one bookmark")
		return
	}
	
	// Log first 5 bookmarks for verification (matching test_parser.go behavior)  
	t.Log("First 5 Chrome bookmarks:")
	for i, bookmark := range bookmarks {
		if i >= 5 {
			break
		}
		
		// Validate required fields
		if bookmark.URL == "" {
			t.Errorf("Bookmark %d has empty URL", i+1)
		}
		if bookmark.Title == "" {
			t.Errorf("Bookmark %d has empty Title", i+1)
		}
		
		folderPath := strings.Join(bookmark.FolderPath, "/")
		if folderPath == "" {
			folderPath = "(root)"
		}
		
		t.Logf("  %d. [%s] %s\n     URL: %s", i+1, folderPath, bookmark.Title, bookmark.URL)
		
		// Test for expected bookmarks from test file  
		if bookmark.URL == "https://golang.org/doc/" {
			if !strings.Contains(bookmark.Title, "Go") {
				t.Errorf("Expected title to contain 'Go', got '%s'", bookmark.Title)
			}
		}
	}
}

// Benchmark tests to ensure performance
func BenchmarkImportService_Firefox(b *testing.B) {
	service := NewImportService()
	
	for i := 0; i < b.N; i++ {
		file, err := os.Open("../../test_firefox_bookmarks.html")
		if err != nil {
			b.Fatalf("Failed to open test file: %v", err)
		}
		
		_, _, err = service.ImportBookmarksFromReader(file)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
		
		file.Close()
	}
}

func BenchmarkImportService_Chrome(b *testing.B) {
	service := NewImportService()
	
	for i := 0; i < b.N; i++ {
		file, err := os.Open("../../test_chrome_bookmarks.html")
		if err != nil {
			b.Fatalf("Failed to open test file: %v", err)
		}
		
		_, _, err = service.ImportBookmarksFromReader(file)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
		
		file.Close()
	}
}