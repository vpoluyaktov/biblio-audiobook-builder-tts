package opds

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// OPDS2Feed represents an OPDS 2.0 feed (JSON-based)
type OPDS2Feed struct {
	Metadata     OPDS2Metadata      `json:"metadata"`
	Links        []OPDS2Link        `json:"links,omitempty"`
	Navigation   []OPDS2Navigation  `json:"navigation,omitempty"`
	Groups       []OPDS2Group       `json:"groups,omitempty"`
	Publications []OPDS2Publication `json:"publications,omitempty"`
}

// OPDS2Metadata represents metadata for an OPDS 2.0 feed
type OPDS2Metadata struct {
	Title         string `json:"title"`
	NumberOfItems int    `json:"numberOfItems,omitempty"`
	ItemsPerPage  int    `json:"itemsPerPage,omitempty"`
	CurrentPage   int    `json:"currentPage,omitempty"`
}

// OPDS2Link represents a link in OPDS 2.0
type OPDS2Link struct {
	Href      string `json:"href"`
	Rel       string `json:"rel,omitempty"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Templated bool   `json:"templated,omitempty"`
}

// OPDS2Navigation represents a navigation entry
type OPDS2Navigation struct {
	Href  string `json:"href"`
	Title string `json:"title"`
	Type  string `json:"type,omitempty"`
	Rel   string `json:"rel,omitempty"`
}

// OPDS2Group represents a group of publications
type OPDS2Group struct {
	Metadata     OPDS2Metadata      `json:"metadata"`
	Links        []OPDS2Link        `json:"links,omitempty"`
	Navigation   []OPDS2Navigation  `json:"navigation,omitempty"`
	Publications []OPDS2Publication `json:"publications,omitempty"`
}

// OPDS2Publication represents a publication in OPDS 2.0
type OPDS2Publication struct {
	Metadata OPDS2PubMetadata `json:"metadata"`
	Links    []OPDS2Link      `json:"links,omitempty"`
	Images   []OPDS2Image     `json:"images,omitempty"`
}

// OPDS2PubMetadata represents publication metadata
type OPDS2PubMetadata struct {
	Type          string      `json:"@type,omitempty"`
	Title         string      `json:"title"`
	Author        interface{} `json:"author,omitempty"` // Can be string or object
	Description   string      `json:"description,omitempty"`
	Identifier    string      `json:"identifier,omitempty"`
	Language      string      `json:"language,omitempty"`
	Published     string      `json:"published,omitempty"`
	NumberOfPages int         `json:"numberOfPages,omitempty"`
	Publisher     string      `json:"publisher,omitempty"`
	Subject       interface{} `json:"subject,omitempty"` // Can be string or array
}

// OPDS2Image represents an image in OPDS 2.0
type OPDS2Image struct {
	Href   string `json:"href"`
	Type   string `json:"type,omitempty"`
	Rel    string `json:"rel,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ParseOPDS2Catalog parses OPDS 2.0 JSON into a CatalogResponse
func (c *Client) ParseOPDS2Catalog(data []byte, baseURL string) (*CatalogResponse, error) {
	var feed OPDS2Feed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse OPDS 2.0 feed: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	response := &CatalogResponse{
		Title:   feed.Metadata.Title,
		Entries: make([]CatalogEntry, 0),
		Links:   make([]NavLink, 0),
	}

	// Process feed-level links
	searchInfo := &SearchInfo{}
	for _, link := range feed.Links {
		// For templated URLs, preserve the template as-is (don't resolve)
		var absURL string
		if link.Templated {
			absURL = link.Href
		} else {
			absURL = resolveURL(base, link.Href)
		}

		if link.Rel == "next" {
			response.NextPageURL = absURL
		}
		if link.Rel == "previous" || link.Rel == "prev" {
			response.PrevPageURL = absURL
		}

		// Extract search links
		if link.Rel == "search" {
			if link.Templated {
				searchInfo.SearchTemplateURL = absURL
				searchInfo.Supported = true
			}
		}

		// Add navigation links
		if link.Rel == "start" || link.Rel == "self" || link.Rel == "search" {
			response.Links = append(response.Links, NavLink{
				Rel:   link.Rel,
				Href:  absURL,
				Title: link.Title,
				Type:  link.Type,
			})
		}
	}

	if searchInfo.Supported {
		response.SearchInfo = searchInfo
	}

	// Process navigation entries
	for _, nav := range feed.Navigation {
		absURL := resolveURL(base, nav.Href)
		entry := CatalogEntry{
			Title:          nav.Title,
			NavigationLink: absURL,
			IsNavigation:   true,
		}
		response.Entries = append(response.Entries, entry)
	}

	// Process groups
	for _, group := range feed.Groups {
		// Add group navigation if present
		for _, nav := range group.Navigation {
			absURL := resolveURL(base, nav.Href)
			entry := CatalogEntry{
				Title:          nav.Title,
				NavigationLink: absURL,
				IsNavigation:   true,
			}
			response.Entries = append(response.Entries, entry)
		}

		// Add group publications
		for _, pub := range group.Publications {
			entry := c.convertOPDS2Publication(pub, base)
			response.Entries = append(response.Entries, entry)
		}
	}

	// Process feed-level publications
	for _, pub := range feed.Publications {
		entry := c.convertOPDS2Publication(pub, base)
		response.Entries = append(response.Entries, entry)
	}

	return response, nil
}

// convertOPDS2Publication converts an OPDS 2.0 publication to a CatalogEntry
func (c *Client) convertOPDS2Publication(pub OPDS2Publication, base *url.URL) CatalogEntry {
	entry := CatalogEntry{
		ID:            pub.Metadata.Identifier,
		Title:         pub.Metadata.Title,
		Authors:       make([]string, 0),
		Summary:       pub.Metadata.Description,
		Language:      pub.Metadata.Language,
		Published:     pub.Metadata.Published,
		Publisher:     pub.Metadata.Publisher,
		Categories:    make([]string, 0),
		DownloadLinks: make([]DownloadLink, 0),
	}

	// Extract authors - handle both string and object formats
	switch author := pub.Metadata.Author.(type) {
	case string:
		if author != "" {
			entry.Authors = append(entry.Authors, author)
		}
	case map[string]interface{}:
		if name, ok := author["name"].(string); ok && name != "" {
			entry.Authors = append(entry.Authors, name)
		}
	case []interface{}:
		for _, a := range author {
			if authorStr, ok := a.(string); ok && authorStr != "" {
				entry.Authors = append(entry.Authors, authorStr)
			} else if authorMap, ok := a.(map[string]interface{}); ok {
				if name, ok := authorMap["name"].(string); ok && name != "" {
					entry.Authors = append(entry.Authors, name)
				}
			}
		}
	}

	// Extract subjects/categories
	switch subject := pub.Metadata.Subject.(type) {
	case string:
		if subject != "" {
			entry.Categories = append(entry.Categories, subject)
		}
	case []interface{}:
		for _, s := range subject {
			if subjectStr, ok := s.(string); ok && subjectStr != "" {
				entry.Categories = append(entry.Categories, subjectStr)
			} else if subjectMap, ok := s.(map[string]interface{}); ok {
				if name, ok := subjectMap["name"].(string); ok && name != "" {
					entry.Categories = append(entry.Categories, name)
				}
			}
		}
	}

	// Process images
	for _, img := range pub.Images {
		absURL := resolveURL(base, img.Href)
		if img.Rel == "cover" || strings.Contains(img.Rel, "cover") {
			entry.CoverURL = absURL
		}
		// Use first image as thumbnail if no specific thumbnail
		if entry.ThumbnailURL == "" && img.Href != "" {
			entry.ThumbnailURL = absURL
		}
	}

	// Process links
	for _, link := range pub.Links {
		absURL := resolveURL(base, link.Href)

		// Check for acquisition links
		if strings.Contains(link.Rel, "acquisition") ||
			strings.Contains(link.Rel, "borrow") ||
			strings.Contains(link.Rel, "buy") {

			// Determine format from type
			format := detectFormat(link.Type)

			dl := DownloadLink{
				URL:    absURL,
				Type:   link.Type,
				Title:  link.Title,
				Format: format,
			}
			entry.DownloadLinks = append(entry.DownloadLinks, dl)
		}

		// Check for navigation links
		if link.Rel == "alternate" || link.Rel == "self" {
			if strings.Contains(link.Type, "opds") || strings.Contains(link.Type, "json") {
				if entry.NavigationLink == "" && len(entry.DownloadLinks) == 0 {
					entry.NavigationLink = absURL
				}
			}
		}
	}

	return entry
}
