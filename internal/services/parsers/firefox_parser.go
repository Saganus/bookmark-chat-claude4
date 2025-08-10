package parsers

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// FirefoxParser implements BookmarkParser for Firefox HTML bookmark exports
type FirefoxParser struct{}

// NewFirefoxParser creates a new Firefox parser
func NewFirefoxParser() *FirefoxParser {
	return &FirefoxParser{}
}

// GetSupportedFormat returns the format name
func (p *FirefoxParser) GetSupportedFormat() string {
	return "Firefox"
}

// ValidateFormat checks if the content looks like a Firefox bookmark export
func (p *FirefoxParser) ValidateFormat(reader io.Reader) bool {
	content := make([]byte, 1024)
	n, _ := reader.Read(content)
	contentStr := string(content[:n])
	
	// Firefox exports contain DOCTYPE NETSCAPE-Bookmark-file-1 and H1 with "Bookmarks Menu"
	return strings.Contains(contentStr, "DOCTYPE NETSCAPE-Bookmark-file-1") &&
		strings.Contains(contentStr, "Bookmarks Menu")
}

// ParseFile parses Firefox bookmark HTML export format
func (p *FirefoxParser) ParseFile(reader io.Reader) (*ParseResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read Firefox bookmarks file: %w", err)
	}

	return p.ParseFirefoxBookmarks(content)
}

// ParseFirefoxBookmarks parses Firefox bookmark HTML export format
func (p *FirefoxParser) ParseFirefoxBookmarks(data []byte) (*ParseResult, error) {
	reader := strings.NewReader(string(data))
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Firefox bookmarks HTML: %w", err)
	}

	result := &ParseResult{
		Source:   "Firefox",
		ParsedAt: time.Now(),
	}

	// Process the entire document recursively to find ALL bookmarks
	// This approach handles multiple DL elements throughout the document
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
func (p *FirefoxParser) parseNodeRecursively(n *html.Node, currentPath []string, bookmarks *[]Bookmark) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "dl":
			// Process a definition list - this contains bookmarks and folders
			p.processDL(n, currentPath, bookmarks)
			return
		case "a":
			// Sometimes bookmarks appear outside of DL
			bookmark := p.extractFirefoxBookmark(n, currentPath)
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
func (p *FirefoxParser) processDL(dl *html.Node, currentPath []string, bookmarks *[]Bookmark) {
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
func (p *FirefoxParser) processDT(dt *html.Node, currentPath []string, bookmarks *[]Bookmark) {
	// Check what's inside this DT
	for child := dt.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			switch child.Data {
			case "h3":
				// This is a folder header
				folderName := p.getTextContent(child)
				newPath := append(append([]string{}, currentPath...), folderName)

				// Look for DL elements in two places:
				// 1. As siblings of H3 within the same DT
				for sibling := child.NextSibling; sibling != nil; sibling = sibling.NextSibling {
					if sibling.Type == html.ElementNode && sibling.Data == "dl" {
						// Direct pattern: DT → H3 → DL
						p.processDL(sibling, newPath, bookmarks)
					}
				}

				// 2. In DD element that is sibling of this DT
				for dtSibling := dt.NextSibling; dtSibling != nil; dtSibling = dtSibling.NextSibling {
					if dtSibling.Type == html.ElementNode && dtSibling.Data == "dd" {
						// DD pattern: DT(H3) → DD → DL
						for ddChild := dtSibling.FirstChild; ddChild != nil; ddChild = ddChild.NextSibling {
							if ddChild.Type == html.ElementNode && ddChild.Data == "dl" {
								p.processDL(ddChild, newPath, bookmarks)
							}
						}
						break // Only process the first DD sibling
					}
				}
			case "a":
				// This is a bookmark
				bookmark := p.extractFirefoxBookmark(child, currentPath)
				if bookmark.URL != "" {
					*bookmarks = append(*bookmarks, bookmark)
				}
			}
		}
	}
}

func (p *FirefoxParser) extractFirefoxBookmark(aNode *html.Node, folderPath []string) Bookmark {
	bookmark := Bookmark{
		FolderPath: folderPath,
	}

	// Extract URL and other attributes
	for _, attr := range aNode.Attr {
		switch attr.Key {
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

func (p *FirefoxParser) getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(p.getTextContent(c))
	}
	return strings.TrimSpace(text.String())
}