package opds

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

// Feed represents an OPDS 1.x feed (Atom-based)
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

// ParseOPDS1Catalog parses OPDS 1.x (Atom/XML) into a CatalogResponse
func (c *Client) ParseOPDS1Catalog(data []byte, baseURL string) (*CatalogResponse, error) {
	var feed Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse OPDS 1.x feed: %w", err)
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

	// Extract search info from feed links
	searchInfo := &SearchInfo{}

	// Process feed links
	for _, link := range feed.Links {
		absURL := resolveURL(base, link.Href)

		if link.Rel == "next" {
			response.NextPageURL = absURL
		}
		if link.Rel == "previous" || link.Rel == "prev" {
			response.PrevPageURL = absURL
		}

		// Extract search links
		if link.Rel == "search" {
			if strings.Contains(link.Type, "opensearchdescription") {
				// OpenSearch description document URL
				searchInfo.OpenSearchURL = absURL
				searchInfo.Supported = true
			} else if strings.Contains(link.Type, "atom+xml") {
				// Direct Atom search template (FreeLib style)
				searchInfo.SearchTemplateURL = absURL
				searchInfo.Supported = true

				// Check if this template supports field-specific search (like FreeLib)
				// Template format: /search?q={searchTerms}&author={atom:author}&title={atom:title}
				if strings.Contains(link.Href, "{atom:title}") {
					// Extract title search URL - build clean URL with just title parameter
					titleURL := extractFieldSearchURL(link.Href, "title")
					searchInfo.TitleSearchURL = resolveURL(base, titleURL)
				}
				if strings.Contains(link.Href, "{atom:author}") {
					// Extract author search URL - build clean URL with just author parameter
					authorURL := extractFieldSearchURL(link.Href, "author")
					searchInfo.AuthorSearchURL = resolveURL(base, authorURL)
				}
				if strings.Contains(link.Href, "{atom:series}") {
					// Extract series search URL - build clean URL with just series parameter
					seriesURL := extractFieldSearchURL(link.Href, "series")
					searchInfo.SeriesSearchURL = resolveURL(base, seriesURL)
				}
			}
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

	// Set search info if search is supported
	if searchInfo.Supported {
		// If we have an OpenSearch URL, fetch it now to populate all search URLs
		if searchInfo.OpenSearchURL != "" {
			osd, err := c.FetchOpenSearchDescription(searchInfo.OpenSearchURL)
			if err != nil {
				// Log the error but continue - we'll try again later if needed
				fmt.Printf("Warning: Failed to fetch OpenSearch descriptor from %s: %v\n", searchInfo.OpenSearchURL, err)
			} else {
				// Populate all search URL types from the OpenSearch description
				osd.PopulateSearchURLs(searchInfo)
				fmt.Printf("Debug: Populated search URLs - Title: %s, Author: %s, Series: %s\n",
					searchInfo.TitleSearchURL, searchInfo.AuthorSearchURL, searchInfo.SeriesSearchURL)
			}
		}
		response.SearchInfo = searchInfo
	}

	// Process entries
	for _, entry := range feed.Entries {
		catalogEntry := c.convertOPDS1Entry(entry, base)
		response.Entries = append(response.Entries, catalogEntry)
	}

	return response, nil
}

// convertOPDS1Entry converts an OPDS 1.x entry to a CatalogEntry
func (c *Client) convertOPDS1Entry(entry Entry, base *url.URL) CatalogEntry {
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
			link.Rel == "http://opds-spec.org/sort/new" ||
			strings.Contains(link.Type, "navigation"):
			// This is a navigation entry
			if ce.NavigationLink == "" {
				ce.NavigationLink = absURL
				ce.IsNavigation = true
			}
		case link.Rel == "" && (strings.Contains(link.Type, "atom+xml") || strings.Contains(link.Type, "opds-catalog")):
			// Links with no rel but atom+xml or opds-catalog type are typically navigation links (FreeLib style)
			if ce.NavigationLink == "" && len(ce.DownloadLinks) == 0 {
				ce.NavigationLink = absURL
				ce.IsNavigation = true
			}
		case link.Rel == "alternate" && strings.Contains(link.Type, "opds-catalog"):
			// Alternate links with opds-catalog type lead to book acquisition pages (Project Gutenberg style)
			// Only treat as navigation if no download links exist yet
			if ce.NavigationLink == "" && len(ce.DownloadLinks) == 0 {
				ce.NavigationLink = absURL
				// Don't set IsNavigation=true - this is a book detail page, not a folder
			}
		}
	}

	// If no cover but has thumbnail, use thumbnail as cover
	if ce.CoverURL == "" && ce.ThumbnailURL != "" {
		ce.CoverURL = ce.ThumbnailURL
	}

	return ce
}

// extractFieldSearchURL extracts a clean field-specific search URL from a template
// Input: /search?q={searchTerms}&author={atom:author}&title={atom:title}
// Output for field="title": /search?title={searchTerms}
func extractFieldSearchURL(templateURL, field string) string {
	// Parse the URL to extract the base path
	u, err := url.Parse(templateURL)
	if err != nil {
		return ""
	}

	// Build a clean URL with just the specified field
	return fmt.Sprintf("%s?%s={searchTerms}", u.Path, field)
}
