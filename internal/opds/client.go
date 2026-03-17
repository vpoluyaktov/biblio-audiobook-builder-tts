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

// OpenSearch-specific types

// OpenSearchDescription represents an OpenSearch description document
type OpenSearchDescription struct {
	XMLName     xml.Name        `xml:"OpenSearchDescription"`
	ShortName   string          `xml:"ShortName"`
	Description string          `xml:"Description"`
	URLs        []OpenSearchURL `xml:"Url"`
}

// OpenSearchURL represents a URL template in OpenSearch
type OpenSearchURL struct {
	Type     string `xml:"type,attr"`
	Template string `xml:"template,attr"`
	Rel      string `xml:"rel,attr"`
}

// FetchCatalog fetches and parses an OPDS catalog from a URL
func (c *Client) FetchCatalog(catalogURL string) (*CatalogResponse, error) {
	req, err := http.NewRequest("GET", catalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/opds+json, application/atom+xml, application/xml, text/xml")
	req.Header.Set("User-Agent", "BiblioHub Audiobook Builder OPDS Client/1.0")

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

// ParseCatalog parses OPDS feed (XML or JSON) into a CatalogResponse
// Routes to the appropriate parser based on format detection
func (c *Client) ParseCatalog(data []byte, baseURL string) (*CatalogResponse, error) {
	// Detect format: OPDS 2.0 (JSON) or OPDS 1.x (XML)
	if isOPDS2(data) {
		return c.ParseOPDS2Catalog(data, baseURL)
	}

	// Parse as OPDS 1.x (Atom/XML)
	return c.ParseOPDS1Catalog(data, baseURL)
}

// Search performs an OPDS search if the catalog supports it
func (c *Client) Search(searchURL, query string) (*CatalogResponse, error) {
	// Replace {searchTerms} placeholder with actual query
	searchURL = strings.ReplaceAll(searchURL, "{searchTerms}", url.QueryEscape(query))
	searchURL = strings.ReplaceAll(searchURL, "{startIndex?}", "")
	searchURL = strings.ReplaceAll(searchURL, "{count?}", "")

	return c.FetchCatalog(searchURL)
}

// FetchOpenSearchDescription fetches and parses an OpenSearch description document
func (c *Client) FetchOpenSearchDescription(openSearchURL string) (*OpenSearchDescription, error) {
	req, err := http.NewRequest("GET", openSearchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/opensearchdescription+xml, application/xml, text/xml")
	req.Header.Set("User-Agent", "BiblioHub Audiobook Builder OPDS Client/1.0")

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenSearch description: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenSearch description returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var osd OpenSearchDescription
	if err := xml.Unmarshal(body, &osd); err != nil {
		return nil, fmt.Errorf("failed to parse OpenSearch description: %w", err)
	}

	return &osd, nil
}

// GetAtomSearchTemplate extracts the Atom/OPDS search URL template from an OpenSearch description
func (osd *OpenSearchDescription) GetAtomSearchTemplate() string {
	for _, u := range osd.URLs {
		// Look for atom+xml type (OPDS search results)
		if strings.Contains(u.Type, "atom+xml") {
			return u.Template
		}
	}
	// Fallback: return first URL if no atom type found
	if len(osd.URLs) > 0 {
		return osd.URLs[0].Template
	}
	return ""
}

// GetSearchURLByTitle extracts a specific search URL template by its title attribute
func (osd *OpenSearchDescription) GetSearchURLByTitle(title string) string {
	for _, u := range osd.URLs {
		if strings.Contains(u.Type, "atom+xml") && strings.Contains(strings.ToLower(u.Rel), "results") {
			if strings.Contains(strings.ToLower(u.Rel), strings.ToLower(title)) {
				return u.Template
			}
			// Also check the title attribute if present
			if urlTitle := u.Rel; strings.Contains(strings.ToLower(urlTitle), strings.ToLower(title)) {
				return u.Template
			}
		}
	}
	return ""
}

// PopulateSearchURLs extracts all search URL templates from OpenSearch description
func (osd *OpenSearchDescription) PopulateSearchURLs(searchInfo *SearchInfo) {
	fmt.Printf("Debug: PopulateSearchURLs - Processing %d URLs from OpenSearch descriptor\n", len(osd.URLs))
	for _, u := range osd.URLs {
		fmt.Printf("Debug: URL Type=%s, Template=%s, Rel=%s\n", u.Type, u.Template, u.Rel)
		if !strings.Contains(u.Type, "atom+xml") {
			continue
		}

		// Check URL path to determine search type
		template := u.Template
		if strings.Contains(template, "/search/authors") {
			searchInfo.AuthorSearchURL = template
			fmt.Printf("Debug: Set AuthorSearchURL to %s\n", template)
		} else if strings.Contains(template, "/search/series") {
			searchInfo.SeriesSearchURL = template
			fmt.Printf("Debug: Set SeriesSearchURL to %s\n", template)
		} else if strings.Contains(template, "/search?") || strings.HasSuffix(template, "/search") {
			// Default book/title search
			if searchInfo.TitleSearchURL == "" {
				searchInfo.TitleSearchURL = template
				fmt.Printf("Debug: Set TitleSearchURL to %s\n", template)
			}
		}
	}
}

// GetSearchTemplate returns the search URL template for a catalog.
// It first checks for a direct Atom search template, then falls back to fetching OpenSearch description.
func (c *Client) GetSearchTemplate(searchInfo *SearchInfo) (string, error) {
	return c.GetSearchTemplateByType(searchInfo, "")
}

// GetSearchTemplateByType returns the search URL template for a specific search type.
// searchType can be "title", "author", "series", or "" for default search.
// For feeds like FreeLib that support field-specific search, using "title" returns
// actual book results instead of navigation categories.
func (c *Client) GetSearchTemplateByType(searchInfo *SearchInfo, searchType string) (string, error) {
	if searchInfo == nil || !searchInfo.Supported {
		return "", fmt.Errorf("search not supported")
	}

	// Check for field-specific search templates first
	switch searchType {
	case "title":
		if searchInfo.TitleSearchURL != "" {
			return searchInfo.TitleSearchURL, nil
		}
	case "author":
		if searchInfo.AuthorSearchURL != "" {
			return searchInfo.AuthorSearchURL, nil
		}
	case "series":
		if searchInfo.SeriesSearchURL != "" {
			return searchInfo.SeriesSearchURL, nil
		}
	}

	// For default search, prefer title search if available (returns actual books)
	// This handles FreeLib-style feeds where q= returns navigation categories
	if searchType == "" && searchInfo.TitleSearchURL != "" {
		return searchInfo.TitleSearchURL, nil
	}

	// Use direct Atom search template
	if searchInfo.SearchTemplateURL != "" {
		return searchInfo.SearchTemplateURL, nil
	}

	// Fall back to OpenSearch description - fetch and populate all search URLs
	if searchInfo.OpenSearchURL != "" {
		osd, err := c.FetchOpenSearchDescription(searchInfo.OpenSearchURL)
		if err != nil {
			return "", fmt.Errorf("failed to get OpenSearch description: %w", err)
		}

		// Populate all search URLs from OpenSearch description
		osd.PopulateSearchURLs(searchInfo)

		// Now try again with the populated URLs
		switch searchType {
		case "title":
			if searchInfo.TitleSearchURL != "" {
				return searchInfo.TitleSearchURL, nil
			}
		case "author":
			if searchInfo.AuthorSearchURL != "" {
				return searchInfo.AuthorSearchURL, nil
			}
		case "series":
			if searchInfo.SeriesSearchURL != "" {
				return searchInfo.SeriesSearchURL, nil
			}
		}

		// Fallback to generic search template
		template := osd.GetAtomSearchTemplate()
		if template == "" {
			return "", fmt.Errorf("no search URL template found in OpenSearch description")
		}
		return template, nil
	}

	return "", fmt.Errorf("no search URL available")
}

// DownloadBook downloads a book from the given URL
func (c *Client) DownloadBook(downloadURL string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "BiblioHub Audiobook Builder OPDS Client/1.0")

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
