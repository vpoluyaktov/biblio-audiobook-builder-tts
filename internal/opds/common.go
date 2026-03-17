package opds

import (
	"encoding/json"
	"net/url"
	"strings"
)

// CatalogEntry is a simplified representation for the UI
type CatalogEntry struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Authors        []string       `json:"authors"`
	Summary        string         `json:"summary"`
	Language       string         `json:"language"`
	Published      string         `json:"published"`
	Rights         string         `json:"rights"`
	Publisher      string         `json:"publisher"`
	CoverURL       string         `json:"cover_url"`
	ThumbnailURL   string         `json:"thumbnail_url"`
	DownloadLinks  []DownloadLink `json:"download_links"`
	NavigationLink string         `json:"navigation_link,omitempty"`
	IsNavigation   bool           `json:"is_navigation"`
	Categories     []string       `json:"categories"`
}

// DownloadLink represents a book download option
type DownloadLink struct {
	URL    string `json:"url"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Format string `json:"format"` // epub, fb2, pdf, etc.
}

// CatalogResponse is the API response for catalog browsing
type CatalogResponse struct {
	Title       string         `json:"title"`
	Entries     []CatalogEntry `json:"entries"`
	NextPageURL string         `json:"next_page_url,omitempty"`
	PrevPageURL string         `json:"prev_page_url,omitempty"`
	Links       []NavLink      `json:"links,omitempty"`
	SearchInfo  *SearchInfo    `json:"search_info,omitempty"`
}

// NavLink represents a navigation link
type NavLink struct {
	Rel   string `json:"rel"`
	Href  string `json:"href"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// SearchInfo contains search capability information for a catalog
type SearchInfo struct {
	// SearchTemplateURL is the direct Atom search URL template (if available in feed)
	SearchTemplateURL string `json:"search_template_url,omitempty"`
	// OpenSearchURL is the URL to the OpenSearch description document
	OpenSearchURL string `json:"opensearch_url,omitempty"`
	// Supported indicates if search is available
	Supported bool `json:"supported"`
	// TitleSearchURL is a URL template for title-specific search (for feeds like FreeLib)
	TitleSearchURL string `json:"title_search_url,omitempty"`
	// AuthorSearchURL is a URL template for author-specific search
	AuthorSearchURL string `json:"author_search_url,omitempty"`
	// SeriesSearchURL is a URL template for series-specific search
	SeriesSearchURL string `json:"series_search_url,omitempty"`
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(base *url.URL, href string) string {
	if href == "" {
		return ""
	}

	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	resolved := base.ResolveReference(ref).String()

	// Fix invalid port :0 that some OPDS feeds return (e.g., FreeLib)
	// Replace "http://host:0/" with "https://host/"
	resolved = fixInvalidPort(resolved)

	return resolved
}

// fixInvalidPort fixes URLs with invalid port :0 by removing it and using https
func fixInvalidPort(urlStr string) string {
	if strings.Contains(urlStr, ":0/") {
		// Replace http://host:0/ with https://host/
		urlStr = strings.Replace(urlStr, ":0/", "/", 1)
		// Also upgrade to https if it was http
		if strings.HasPrefix(urlStr, "http://") {
			urlStr = strings.Replace(urlStr, "http://", "https://", 1)
		}
	}
	return urlStr
}

// detectFormat detects the book format from MIME type
func detectFormat(mimeType string) string {
	mimeType = strings.ToLower(mimeType)

	switch {
	case strings.Contains(mimeType, "epub"):
		return "epub"
	case strings.Contains(mimeType, "fb2") || strings.Contains(mimeType, "fictionbook"):
		return "fb2"
	case strings.Contains(mimeType, "pdf"):
		return "pdf"
	case strings.Contains(mimeType, "mobi") || strings.Contains(mimeType, "x-mobipocket"):
		return "mobi"
	case strings.Contains(mimeType, "azw"):
		return "azw"
	case strings.Contains(mimeType, "text/plain"):
		return "txt"
	case strings.Contains(mimeType, "text/html"):
		return "html"
	default:
		return "unknown"
	}
}

// isOPDS2 checks if the data is OPDS 2.0 (JSON) format
func isOPDS2(data []byte) bool {
	// Trim whitespace
	trimmed := strings.TrimSpace(string(data))

	// Check if it starts with { (JSON)
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}

	// Try to parse as JSON and check for OPDS 2.0 structure
	var test map[string]interface{}
	if err := json.Unmarshal(data, &test); err != nil {
		return false
	}

	// OPDS 2.0 feeds typically have "metadata" or "navigation" or "publications" fields
	_, hasMetadata := test["metadata"]
	_, hasNavigation := test["navigation"]
	_, hasPublications := test["publications"]
	_, hasGroups := test["groups"]

	return hasMetadata || hasNavigation || hasPublications || hasGroups
}
