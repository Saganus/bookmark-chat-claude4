package parsers

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ChromeParser implements BookmarkParser for Chrome HTML bookmark exports
type ChromeParser struct{}

// NewChromeParser creates a new Chrome parser
func NewChromeParser() *ChromeParser {
	return &ChromeParser{}
}

// GetSupportedFormat returns the format name
func (p *ChromeParser) GetSupportedFormat() string {
	return "Chrome"
}

// ValidateFormat checks if the content looks like a Chrome bookmark export
func (p *ChromeParser) ValidateFormat(reader io.Reader) bool {
	content := make([]byte, 1024)
	n, _ := reader.Read(content)
	contentStr := string(content[:n])
	
	// Chrome exports contain DOCTYPE NETSCAPE-Bookmark-file-1 and H1 with "Bookmarks" (not "Bookmarks Menu")
	return strings.Contains(contentStr, "DOCTYPE NETSCAPE-Bookmark-file-1") &&
		strings.Contains(contentStr, "<H1>Bookmarks</H1>")
}

// ParseFile parses Chrome bookmark HTML export format
func (p *ChromeParser) ParseFile(reader io.Reader) (*ParseResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read Chrome bookmarks file: %w", err)
	}

	return p.ParseChromeBookmarks(content)
}

// ParseChromeBookmarks parses Chrome bookmark HTML export format
func (p *ChromeParser) ParseChromeBookmarks(data []byte) (*ParseResult, error) {
	reader := strings.NewReader(string(data))
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Chrome bookmarks HTML: %w", err)
	}

	result := &ParseResult{
		Source:   "Chrome",
		ParsedAt: time.Now(),
	}

	// Process the entire document recursively to find ALL bookmarks
	var allBookmarks []Bookmark
	p.parseNodeRecursively(doc, []string{}, &allBookmarks)

	// Create a single root folder containing all bookmarks
	rootFolder := &BookmarkFolder{
		Name:      "Bookmarks",
		Path:      []string{},
		Bookmarks: allBookmarks,
	}

	result.Folders = []*BookmarkFolder{rootFolder}
	result.Bookmarks = allBookmarks
	result.TotalCount = len(allBookmarks)

	return result, nil
}

// parseNodeRecursively processes HTML nodes recursively to find all bookmarks
func (p *ChromeParser) parseNodeRecursively(n *html.Node, currentPath []string, bookmarks *[]Bookmark) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "dl":
			// Process a definition list - this contains bookmarks and folders
			p.processDL(n, currentPath, bookmarks)
			return
		case "a":
			// Sometimes bookmarks appear outside of DL
			bookmark := p.extractChromeBookmark(n, currentPath)
			if bookmark.URL != "" {
				*bookmarks = append(*bookmarks, bookmark)
			}
			return
		}
	}

	// Continue recursing through children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.parseNodeRecursively(c, currentPath, bookmarks)
	}
}

// processDL processes a DL (definition list) element
func (p *ChromeParser) processDL(dl *html.Node, currentPath []string, bookmarks *[]Bookmark) {
	// Process all children of the DL
	for child := dl.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			switch child.Data {
			case "dt":
				// Process a DT element
				p.processDT(child, currentPath, bookmarks)
			case "dl":
				// Sometimes DL elements are direct children (nested folders)
				p.processDL(child, currentPath, bookmarks)
			}
		}
	}
}

// processDT processes a DT (definition term) element
func (p *ChromeParser) processDT(dt *html.Node, currentPath []string, bookmarks *[]Bookmark) {
	// Check what's inside this DT
	for child := dt.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			switch child.Data {
			case "h3":
				// This is a folder header
				folderName := p.getTextContent(child)
				newPath := append(append([]string{}, currentPath...), folderName)

				// Look for DL elements as siblings of the current DT
				// Chrome structure: DT(H3) followed by DL
				for dtSibling := dt.NextSibling; dtSibling != nil; dtSibling = dtSibling.NextSibling {
					if dtSibling.Type == html.ElementNode && dtSibling.Data == "dl" {
						p.processDL(dtSibling, newPath, bookmarks)
						break // Process only the first DL sibling
					}
				}

				// Also check for direct DL children (alternate structure)
				for sibling := child.NextSibling; sibling != nil; sibling = sibling.NextSibling {
					if sibling.Type == html.ElementNode && sibling.Data == "dl" {
						p.processDL(sibling, newPath, bookmarks)
					}
				}
			case "a":
				// This is a bookmark
				bookmark := p.extractChromeBookmark(child, currentPath)
				if bookmark.URL != "" {
					*bookmarks = append(*bookmarks, bookmark)
				}
			}
		}
	}
}

func (p *ChromeParser) extractChromeBookmark(aNode *html.Node, folderPath []string) Bookmark {
	bookmark := Bookmark{
		FolderPath: folderPath,
	}

	// Extract URL and other attributes
	for _, attr := range aNode.Attr {
		switch strings.ToLower(attr.Key) {
		case "href":
			bookmark.URL = attr.Val
		case "add_date":
			if timestamp, err := strconv.ParseInt(attr.Val, 10, 64); err == nil {
				bookmark.DateAdded = time.Unix(timestamp, 0)
			}
		case "icon":
			bookmark.Icon = attr.Val
		}
	}

	// Extract title from text content
	bookmark.Title = p.getTextContent(aNode)

	return bookmark
}

func (p *ChromeParser) getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(p.getTextContent(c))
	}
	return strings.TrimSpace(text.String())
}