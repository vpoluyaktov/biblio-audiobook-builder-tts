package opds

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client represents an OPDS catalog client
type Client struct {
	httpClient *http.Client
	username   string
	password   string
}

// NewClient creates a new OPDS client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithAuth creates a new OPDS client with HTTP Basic Auth credentials
func NewClientWithAuth(username, password string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		username: username,
		password: password,
	}
}

// SetAuth sets the authentication credentials for the client
func (c *Client) SetAuth(username, password string) {
	c.username = username
	c.password = password
}

// Feed represents an OPDS feed (Atom-based)
type Feed struct {
	XMLName  xml.Name `xml:"feed"`
	ID       string   `xml:"id"`
	Title    string   `xml:"title"`
	Updated  string   `xml:"updated"`
	Author   *Author  `xml:"author"`
	Links    []Link   `xml:"link"`
	Entries  []Entry  `xml:"entry"`
	NextLink string   `xml:"-"` // Populated after parsing
}

// Author represents an Atom author
type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri"`
}

// Link represents an Atom link
type Link struct {
	Rel         string `xml:"rel,attr"`
	Href        string `xml:"href,attr"`
	Type        string `xml:"type,attr"`
	Title       string `xml:"title,attr"`
	FacetGroup  string `xml:"facetGroup,attr"`
	Count       int    `xml:"count,attr"`
	ActiveFacet bool   `xml:"activeFacet,attr"`
}

// Entry represents an OPDS catalog entry (book or navigation)
type Entry struct {
	ID         string     `xml:"id"`
	Title      string     `xml:"title"`
	Updated    string     `xml:"updated"`
	Published  string     `xml:"published"`
	Authors    []Author   `xml:"author"`
	Summary    string     `xml:"summary"`
	Content    *Content   `xml:"content"`
	Links      []Link     `xml:"link"`
	Categories []Category `xml:"category"`
	Language   string     `xml:"language"`
	Rights     string     `xml:"rights"`
	Publisher  string     `xml:"publisher"`
	Issued     string     `xml:"issued"`
}

// Content represents entry content
type Content struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// Category represents a category/subject
type Category struct {
	Term   string `xml:"term,attr"`
	Label  string `xml:"label,attr"`
	Scheme string `xml:"scheme,attr"`
}

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
	Links       []NavLink      `json:"links,omitempty"`
}

// NavLink represents a navigation link
type NavLink struct {
	Rel   string `json:"rel"`
	Href  string `json:"href"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// FetchCatalog fetches and parses an OPDS catalog from a URL
func (c *Client) FetchCatalog(catalogURL string) (*CatalogResponse, error) {
	req, err := http.NewRequest("GET", catalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/atom+xml, application/xml, text/xml")
	req.Header.Set("User-Agent", "abb_tts OPDS Client/1.0")

	// Add Basic Auth if credentials are set
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return c.ParseCatalog(body, catalogURL)
}

// ParseCatalog parses OPDS XML into a CatalogResponse
func (c *Client) ParseCatalog(data []byte, baseURL string) (*CatalogResponse, error) {
	var feed Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse OPDS feed: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	response := &CatalogResponse{
		Title:   feed.Title,
		Entries: make([]CatalogEntry, 0, len(feed.Entries)),
		Links:   make([]NavLink, 0),
	}

	// Process feed links
	for _, link := range feed.Links {
		absURL := resolveURL(base, link.Href)

		if link.Rel == "next" {
			response.NextPageURL = absURL
		}

		// Add navigation links
		if link.Rel == "start" || link.Rel == "self" || link.Rel == "search" ||
			link.Rel == "http://opds-spec.org/facet" || strings.Contains(link.Rel, "subsection") {
			response.Links = append(response.Links, NavLink{
				Rel:   link.Rel,
				Href:  absURL,
				Title: link.Title,
				Type:  link.Type,
			})
		}
	}

	// Process entries
	for _, entry := range feed.Entries {
		catalogEntry := c.convertEntry(entry, base)
		response.Entries = append(response.Entries, catalogEntry)
	}

	return response, nil
}

// convertEntry converts an OPDS entry to a CatalogEntry
func (c *Client) convertEntry(entry Entry, base *url.URL) CatalogEntry {
	ce := CatalogEntry{
		ID:            entry.ID,
		Title:         entry.Title,
		Authors:       make([]string, 0, len(entry.Authors)),
		Language:      entry.Language,
		Published:     entry.Published,
		Rights:        entry.Rights,
		Publisher:     entry.Publisher,
		Categories:    make([]string, 0),
		DownloadLinks: make([]DownloadLink, 0),
	}

	// Extract summary
	if entry.Summary != "" {
		ce.Summary = entry.Summary
	} else if entry.Content != nil {
		ce.Summary = entry.Content.Value
	}

	// Extract authors
	for _, author := range entry.Authors {
		if author.Name != "" {
			ce.Authors = append(ce.Authors, author.Name)
		}
	}

	// Extract categories
	for _, cat := range entry.Categories {
		if cat.Label != "" {
			ce.Categories = append(ce.Categories, cat.Label)
		} else if cat.Term != "" {
			ce.Categories = append(ce.Categories, cat.Term)
		}
	}

	// Process links
	for _, link := range entry.Links {
		absURL := resolveURL(base, link.Href)

		switch {
		case link.Rel == "http://opds-spec.org/image" || link.Rel == "http://opds-spec.org/cover":
			ce.CoverURL = absURL
		case link.Rel == "http://opds-spec.org/image/thumbnail" || link.Rel == "http://opds-spec.org/thumbnail":
			ce.ThumbnailURL = absURL
		case link.Rel == "http://opds-spec.org/acquisition" ||
			link.Rel == "http://opds-spec.org/acquisition/open-access" ||
			strings.HasPrefix(link.Rel, "http://opds-spec.org/acquisition"):
			dl := DownloadLink{
				URL:    absURL,
				Type:   link.Type,
				Title:  link.Title,
				Format: detectFormat(link.Type),
			}
			ce.DownloadLinks = append(ce.DownloadLinks, dl)
		case link.Rel == "subsection" || link.Rel == "http://opds-spec.org/sort/popular" ||
			link.Rel == "http://opds-spec.org/sort/new" || link.Rel == "alternate" ||
			strings.Contains(link.Type, "navigation") || strings.Contains(link.Type, "opds-catalog"):
			// This is a navigation entry
			if ce.NavigationLink == "" {
				ce.NavigationLink = absURL
				ce.IsNavigation = true
			}
		case link.Rel == "" && strings.Contains(link.Type, "atom+xml"):
			// Links with no rel but atom+xml type are typically navigation links
			if ce.NavigationLink == "" && len(ce.DownloadLinks) == 0 {
				ce.NavigationLink = absURL
				ce.IsNavigation = true
			}
		}
	}

	// If no cover but has thumbnail, use thumbnail as cover
	if ce.CoverURL == "" && ce.ThumbnailURL != "" {
		ce.CoverURL = ce.ThumbnailURL
	}

	return ce
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

	return base.ResolveReference(ref).String()
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

// Search performs an OPDS search if the catalog supports it
func (c *Client) Search(searchURL, query string) (*CatalogResponse, error) {
	// Replace {searchTerms} placeholder with actual query
	searchURL = strings.ReplaceAll(searchURL, "{searchTerms}", url.QueryEscape(query))
	searchURL = strings.ReplaceAll(searchURL, "{startIndex?}", "")
	searchURL = strings.ReplaceAll(searchURL, "{count?}", "")

	return c.FetchCatalog(searchURL)
}

// DownloadBook downloads a book from the given URL
func (c *Client) DownloadBook(downloadURL string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "abb_tts OPDS Client/1.0")

	// Add Basic Auth if credentials are set
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download book: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read book data: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}
