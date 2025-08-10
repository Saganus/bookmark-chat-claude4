package main

import (
	"fmt"
	"os"
	"strings"

	"bookmark-chat/internal/services"
	"bookmark-chat/internal/services/parsers"
)

func main() {
	service := services.NewImportService()
	
	// Test Firefox bookmarks
	fmt.Println("Testing Firefox bookmark parsing...")
	testFile("test_firefox_bookmarks.html", service)
	
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
	
	// Test Chrome bookmarks
	fmt.Println("Testing Chrome bookmark parsing...")
	testFile("test_chrome_bookmarks.html", service)
}

func testFile(filename string, service *services.ImportService) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	importResult, parseResult, err := service.ImportBookmarksFromReader(file)
	if err != nil {
		fmt.Printf("Error parsing %s: %v\n", filename, err)
		return
	}

	// Print results
	fmt.Printf("Format: %s\n", parseResult.Source)
	fmt.Printf("Total bookmarks found: %d\n", parseResult.TotalCount)
	fmt.Printf("Number of folders: %d\n", len(parseResult.Folders))
	fmt.Printf("Import status: %s\n", importResult.Status)
	fmt.Printf("Successfully imported: %d\n", importResult.Statistics.SuccessfullyImported)
	fmt.Printf("Failed: %d\n", importResult.Statistics.Failed)

	fmt.Println("\nFirst 5 bookmarks:")
	for i, bookmark := range parseResult.Bookmarks {
		if i >= 5 {
			break
		}
		folderPath := strings.Join(bookmark.FolderPath, "/")
		if folderPath == "" {
			folderPath = "(root)"
		}
		fmt.Printf("  %d. [%s] %s\n     URL: %s\n", i+1, folderPath, bookmark.Title, bookmark.URL)
	}

	// Print folder structure overview
	if len(parseResult.Folders) > 0 {
		fmt.Println("\nFolder structure:")
		printFolderStructure(parseResult.Folders, "")
	}
}

func printFolderStructure(folders []*parsers.BookmarkFolder, indent string) {
	for _, folder := range folders {
		folderPath := strings.Join(folder.Path, "/")
		if folderPath == "" {
			folderPath = "(root)"
		}
		fmt.Printf("%s- %s (%d bookmarks)\n", indent, folder.Name, len(folder.Bookmarks))
		
		if len(folder.Subfolders) > 0 {
			printFolderStructure(folder.Subfolders, indent+"  ")
		}
	}
}