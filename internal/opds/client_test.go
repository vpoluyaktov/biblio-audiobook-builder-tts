package opds

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNewClientWithAuth(t *testing.T) {
	client := NewClientWithAuth("user", "pass")
	if client == nil {
		t.Fatal("NewClientWithAuth returned nil")
	}
	if client.username != "user" {
		t.Errorf("expected username 'user', got '%s'", client.username)
	}
	if client.password != "pass" {
		t.Errorf("expected password 'pass', got '%s'", client.password)
	}
}

func TestSetAuth(t *testing.T) {
	client := NewClient()
	client.SetAuth("testuser", "testpass")
	if client.username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", client.username)
	}
	if client.password != "testpass" {
		t.Errorf("expected password 'testpass', got '%s'", client.password)
	}
}

func TestParseCatalog_NavigationFeed(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:root</id>
  <title>Test Catalog</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <link rel="self" href="/opds" type="application/atom+xml"/>
  <link rel="start" href="/opds" type="application/atom+xml"/>
  <link rel="search" href="/search" type="application/opensearchdescription+xml"/>
  <entry>
    <id>tag:authors</id>
    <title>By Authors</title>
    <content type="text">Browse by author</content>
    <link type="application/atom+xml;profile=opds-catalog" href="/authors"/>
  </entry>
  <entry>
    <id>tag:genres</id>
    <title>By Genre</title>
    <content type="text">Browse by genre</content>
    <link rel="subsection" type="application/atom+xml" href="/genres"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if catalog.Title != "Test Catalog" {
		t.Errorf("expected title 'Test Catalog', got '%s'", catalog.Title)
	}

	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}

	// First entry - navigation via type attribute
	entry1 := catalog.Entries[0]
	if entry1.Title != "By Authors" {
		t.Errorf("expected title 'By Authors', got '%s'", entry1.Title)
	}
	if !entry1.IsNavigation {
		t.Error("expected entry1 to be navigation")
	}
	if entry1.NavigationLink != "http://example.com/authors" {
		t.Errorf("expected navigation link 'http://example.com/authors', got '%s'", entry1.NavigationLink)
	}

	// Second entry - navigation via rel attribute
	entry2 := catalog.Entries[1]
	if entry2.Title != "By Genre" {
		t.Errorf("expected title 'By Genre', got '%s'", entry2.Title)
	}
	if !entry2.IsNavigation {
		t.Error("expected entry2 to be navigation")
	}
	if entry2.NavigationLink != "http://example.com/genres" {
		t.Errorf("expected navigation link 'http://example.com/genres', got '%s'", entry2.NavigationLink)
	}
}

func TestParseCatalog_AcquisitionFeed(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/terms/">
  <id>tag:books</id>
  <title>Books</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <entry>
    <id>urn:book:123</id>
    <title>Test Book</title>
    <author><name>John Doe</name></author>
    <author><name>Jane Smith</name></author>
    <summary>A test book summary</summary>
    <dc:language>en</dc:language>
    <dc:publisher>Test Publisher</dc:publisher>
    <category term="fiction" label="Fiction"/>
    <category term="adventure"/>
    <link rel="http://opds-spec.org/image" href="/covers/123.jpg" type="image/jpeg"/>
    <link rel="http://opds-spec.org/image/thumbnail" href="/thumbs/123.jpg" type="image/jpeg"/>
    <link rel="http://opds-spec.org/acquisition" href="/download/123.epub" type="application/epub+zip" title="EPUB"/>
    <link rel="http://opds-spec.org/acquisition" href="/download/123.fb2" type="application/fb2+xml" title="FB2"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]

	// Check basic metadata
	if entry.Title != "Test Book" {
		t.Errorf("expected title 'Test Book', got '%s'", entry.Title)
	}
	if entry.Summary != "A test book summary" {
		t.Errorf("expected summary 'A test book summary', got '%s'", entry.Summary)
	}
	if entry.Language != "en" {
		t.Errorf("expected language 'en', got '%s'", entry.Language)
	}
	if entry.Publisher != "Test Publisher" {
		t.Errorf("expected publisher 'Test Publisher', got '%s'", entry.Publisher)
	}

	// Check authors
	if len(entry.Authors) != 2 {
		t.Fatalf("expected 2 authors, got %d", len(entry.Authors))
	}
	if entry.Authors[0] != "John Doe" {
		t.Errorf("expected first author 'John Doe', got '%s'", entry.Authors[0])
	}
	if entry.Authors[1] != "Jane Smith" {
		t.Errorf("expected second author 'Jane Smith', got '%s'", entry.Authors[1])
	}

	// Check categories
	if len(entry.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(entry.Categories))
	}
	if entry.Categories[0] != "Fiction" {
		t.Errorf("expected first category 'Fiction', got '%s'", entry.Categories[0])
	}

	// Check cover images
	if entry.CoverURL != "http://example.com/covers/123.jpg" {
		t.Errorf("expected cover URL 'http://example.com/covers/123.jpg', got '%s'", entry.CoverURL)
	}
	if entry.ThumbnailURL != "http://example.com/thumbs/123.jpg" {
		t.Errorf("expected thumbnail URL 'http://example.com/thumbs/123.jpg', got '%s'", entry.ThumbnailURL)
	}

	// Check download links
	if len(entry.DownloadLinks) != 2 {
		t.Fatalf("expected 2 download links, got %d", len(entry.DownloadLinks))
	}

	epub := entry.DownloadLinks[0]
	if epub.Format != "epub" {
		t.Errorf("expected format 'epub', got '%s'", epub.Format)
	}
	if epub.URL != "http://example.com/download/123.epub" {
		t.Errorf("expected URL 'http://example.com/download/123.epub', got '%s'", epub.URL)
	}

	fb2 := entry.DownloadLinks[1]
	if fb2.Format != "fb2" {
		t.Errorf("expected format 'fb2', got '%s'", fb2.Format)
	}

	// Should not be navigation
	if entry.IsNavigation {
		t.Error("expected entry to not be navigation")
	}
}

func TestParseCatalog_Pagination(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:page1</id>
  <title>Page 1</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <link rel="next" href="/opds?page=2" type="application/atom+xml"/>
  <link rel="self" href="/opds?page=1" type="application/atom+xml"/>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if catalog.NextPageURL != "http://example.com/opds?page=2" {
		t.Errorf("expected next page URL 'http://example.com/opds?page=2', got '%s'", catalog.NextPageURL)
	}
}

func TestParseCatalog_ContentAsSummary(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <entry>
    <id>1</id>
    <title>Entry with content</title>
    <content type="text">This is the content used as summary</content>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	if catalog.Entries[0].Summary != "This is the content used as summary" {
		t.Errorf("expected summary from content, got '%s'", catalog.Entries[0].Summary)
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"application/epub+zip", "epub"},
		{"application/fb2+xml", "fb2"},
		{"application/x-fictionbook+xml", "fb2"},
		{"application/pdf", "pdf"},
		{"application/x-mobipocket-ebook", "mobi"},
		{"text/plain", "txt"},
		{"text/html", "html"},
		{"application/octet-stream", "unknown"},
	}

	for _, tt := range tests {
		result := detectFormat(tt.mimeType)
		if result != tt.expected {
			t.Errorf("detectFormat(%s) = %s, expected %s", tt.mimeType, result, tt.expected)
		}
	}
}

func TestFetchCatalog_WithAuth(t *testing.T) {
	// Create test server that requires auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:auth</id>
  <title>Authenticated Feed</title>
  <updated>2024-01-01T00:00:00Z</updated>
</feed>`))
	}))
	defer server.Close()

	// Test without auth - should fail
	client := NewClient()
	_, err := client.FetchCatalog(server.URL)
	if err == nil {
		t.Error("expected error without auth")
	}

	// Test with auth - should succeed
	clientWithAuth := NewClientWithAuth("testuser", "testpass")
	catalog, err := clientWithAuth.FetchCatalog(server.URL)
	if err != nil {
		t.Fatalf("FetchCatalog with auth failed: %v", err)
	}
	if catalog.Title != "Authenticated Feed" {
		t.Errorf("expected title 'Authenticated Feed', got '%s'", catalog.Title)
	}
}

func TestFetchCatalog_Error(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.FetchCatalog(server.URL)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFixInvalidPort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://lib.e-books.plus:0/opds/1/page=2", "https://lib.e-books.plus/opds/1/page=2"},
		{"http://example.com:0/path", "https://example.com/path"},
		{"https://example.com/normal", "https://example.com/normal"},
		{"http://example.com:8080/path", "http://example.com:8080/path"},
	}

	for _, tt := range tests {
		result := fixInvalidPort(tt.input)
		if result != tt.expected {
			t.Errorf("fixInvalidPort(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base     string
		href     string
		expected string
	}{
		{"http://example.com/opds", "/authors", "http://example.com/authors"},
		{"http://example.com/opds/", "authors", "http://example.com/opds/authors"},
		{"http://example.com/opds", "http://other.com/feed", "http://other.com/feed"},
		{"http://example.com/opds", "", ""},
	}

	for _, tt := range tests {
		base, _ := parseURL(tt.base)
		result := resolveURL(base, tt.href)
		if result != tt.expected {
			t.Errorf("resolveURL(%s, %s) = %s, expected %s", tt.base, tt.href, result, tt.expected)
		}
	}
}

func parseURL(s string) (*url.URL, error) {
	return url.Parse(s)
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query != "test query" {
			t.Errorf("expected query 'test query', got '%s'", query)
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:search</id>
  <title>Search Results</title>
  <updated>2024-01-01T00:00:00Z</updated>
</feed>`))
	}))
	defer server.Close()

	client := NewClient()
	catalog, err := client.Search(server.URL+"/search?q={searchTerms}", "test query")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if catalog.Title != "Search Results" {
		t.Errorf("expected title 'Search Results', got '%s'", catalog.Title)
	}
}

func TestDownloadBook(t *testing.T) {
	bookContent := []byte("fake epub content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/epub+zip")
		w.Write(bookContent)
	}))
	defer server.Close()

	client := NewClient()
	data, contentType, err := client.DownloadBook(server.URL + "/book.epub")
	if err != nil {
		t.Fatalf("DownloadBook failed: %v", err)
	}

	if string(data) != string(bookContent) {
		t.Errorf("expected content '%s', got '%s'", bookContent, data)
	}
	if contentType != "application/epub+zip" {
		t.Errorf("expected content type 'application/epub+zip', got '%s'", contentType)
	}
}

func TestDownloadBook_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/epub+zip")
		w.Write([]byte("authenticated content"))
	}))
	defer server.Close()

	// Without auth
	client := NewClient()
	_, _, err := client.DownloadBook(server.URL)
	if err == nil {
		t.Error("expected error without auth")
	}

	// With auth
	clientWithAuth := NewClientWithAuth("user", "pass")
	data, _, err := clientWithAuth.DownloadBook(server.URL)
	if err != nil {
		t.Fatalf("DownloadBook with auth failed: %v", err)
	}
	if string(data) != "authenticated content" {
		t.Errorf("expected 'authenticated content', got '%s'", data)
	}
}

// Tests for OPDS navigation link detection

func TestParseEntry_NavigationLink_Subsection(t *testing.T) {
	// Test that subsection rel is detected as navigation
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:category</id>
    <title>Fiction</title>
    <link rel="subsection" href="/opds/fiction" type="application/atom+xml"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	if !entry.IsNavigation {
		t.Error("entry with subsection rel should be navigation")
	}
	if entry.NavigationLink == "" {
		t.Error("navigation link should not be empty")
	}
}

func TestParseEntry_NavigationLink_FreeLibStyle(t *testing.T) {
	// Test FreeLib-style navigation: empty rel with opds-catalog type
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:category</id>
    <title>Authors</title>
    <link href="/opds/authors" type="application/atom+xml;profile=opds-catalog"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	if !entry.IsNavigation {
		t.Error("entry with empty rel and opds-catalog type should be navigation")
	}
	if entry.NavigationLink == "" {
		t.Error("navigation link should not be empty")
	}
}

func TestParseEntry_AcquisitionLink_ProjectGutenbergStyle(t *testing.T) {
	// Test Project Gutenberg style: alternate rel with opds-catalog type should NOT be navigation folder
	// It's a book detail page, not a folder
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opds="http://opds-spec.org/2010/catalog">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:book1</id>
    <title>A Great Book</title>
    <author><name>John Doe</name></author>
    <link rel="alternate" href="/ebooks/12345.opds" type="application/atom+xml;profile=opds-catalog"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	// Should NOT be marked as navigation (folder) - it's a book detail page
	if entry.IsNavigation {
		t.Error("entry with alternate rel and opds-catalog type should NOT be marked as navigation folder")
	}
	// But should have a navigation link to the book detail page
	if entry.NavigationLink == "" {
		t.Error("navigation link should be set for book detail page")
	}
}

func TestParseEntry_AcquisitionLink_DirectDownload(t *testing.T) {
	// Test direct acquisition links (epub, mobi, etc.)
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opds="http://opds-spec.org/2010/catalog">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:book1</id>
    <title>A Great Book</title>
    <link rel="http://opds-spec.org/acquisition" href="/download/book.epub" type="application/epub+zip"/>
    <link rel="http://opds-spec.org/acquisition" href="/download/book.mobi" type="application/x-mobipocket-ebook"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	if entry.IsNavigation {
		t.Error("entry with acquisition links should not be navigation")
	}
	if len(entry.DownloadLinks) != 2 {
		t.Errorf("expected 2 download links, got %d", len(entry.DownloadLinks))
	}
}

func TestParseEntry_MixedLinks(t *testing.T) {
	// Test entry with both acquisition and navigation links
	// Acquisition links should take precedence
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opds="http://opds-spec.org/2010/catalog">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:book1</id>
    <title>A Great Book</title>
    <link rel="http://opds-spec.org/acquisition/open-access" href="/download/book.epub" type="application/epub+zip"/>
    <link rel="alternate" href="/ebooks/12345.opds" type="application/atom+xml;profile=opds-catalog"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	// Should have download links
	if len(entry.DownloadLinks) != 1 {
		t.Errorf("expected 1 download link, got %d", len(entry.DownloadLinks))
	}
	// Should NOT be marked as navigation since it has download links
	if entry.IsNavigation {
		t.Error("entry with acquisition links should not be marked as navigation")
	}
}

func TestParseEntry_NavigationType(t *testing.T) {
	// Test navigation type in link type attribute
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:nav</id>
    <title>Browse by Author</title>
    <link href="/opds/authors" type="application/atom+xml;type=navigation"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	if !entry.IsNavigation {
		t.Error("entry with navigation type should be navigation")
	}
}

func TestParseEntry_SortLinks(t *testing.T) {
	// Test OPDS sort links (popular, new)
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <entry>
    <id>tag:popular</id>
    <title>Most Popular</title>
    <link rel="http://opds-spec.org/sort/popular" href="/opds/popular" type="application/atom+xml"/>
  </entry>
  <entry>
    <id>tag:new</id>
    <title>New Releases</title>
    <link rel="http://opds-spec.org/sort/new" href="/opds/new" type="application/atom+xml"/>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}

	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}

	for _, entry := range catalog.Entries {
		if !entry.IsNavigation {
			t.Errorf("entry '%s' with sort link should be navigation", entry.Title)
		}
	}
}

// TestSearchTemplateSubstitution tests that search URL templates are correctly processed
func TestSearchTemplateSubstitution(t *testing.T) {
	tests := []struct {
		name        string
		searchURL   string
		query       string
		expectedURL string
	}{
		{
			name:        "Gutenberg style - simple searchTerms",
			searchURL:   "http://example.com/search.opds/?query={searchTerms}",
			query:       "sherlock holmes",
			expectedURL: "http://example.com/search.opds/?query=sherlock+holmes",
		},
		{
			name:        "FreeLib style - with optional params",
			searchURL:   "http://example.com/opds/1/search?q={searchTerms}&author={atom:author}&title={atom:title}",
			query:       "толстой",
			expectedURL: "http://example.com/opds/1/search?q=%D1%82%D0%BE%D0%BB%D1%81%D1%82%D0%BE%D0%B9&author={atom:author}&title={atom:title}",
		},
		{
			name:        "With startIndex and count placeholders",
			searchURL:   "http://example.com/search?q={searchTerms}&start={startIndex?}&count={count?}",
			query:       "test",
			expectedURL: "http://example.com/search?q=test&start=&count=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate what Search() does
			result := tt.searchURL
			result = strings.ReplaceAll(result, "{searchTerms}", urlEncodeTest(tt.query))
			result = strings.ReplaceAll(result, "{startIndex?}", "")
			result = strings.ReplaceAll(result, "{count?}", "")

			if result != tt.expectedURL {
				t.Errorf("expected URL:\n%s\ngot:\n%s", tt.expectedURL, result)
			}
		})
	}
}

// urlEncodeTest is a helper to URL-encode a string for tests
func urlEncodeTest(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "+"),
		"толстой", "%D1%82%D0%BE%D0%BB%D1%81%D1%82%D0%BE%D0%B9")
}

// TestSearchWithGutenbergFormat tests search with Project Gutenberg style feed
func TestSearchWithGutenbergFormat(t *testing.T) {
	gutenbergResponse := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opensearch="http://a9.com/-/spec/opensearch/1.1/">
  <id>https://www.gutenberg.org/ebooks/search.opds/?query=sherlock</id>
  <title>Project Gutenberg: Search Results</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <opensearch:totalResults>50</opensearch:totalResults>
  <opensearch:itemsPerPage>25</opensearch:itemsPerPage>
  <link rel="next" href="/ebooks/search.opds/?query=sherlock&amp;start_index=26"/>
  <entry>
    <id>https://www.gutenberg.org/ebooks/244.opds</id>
    <title>A Study in Scarlet</title>
    <author><name>Arthur Conan Doyle</name></author>
    <link rel="alternate" href="/ebooks/244.opds" type="application/atom+xml;profile=opds-catalog"/>
  </entry>
  <entry>
    <id>https://www.gutenberg.org/ebooks/1661.opds</id>
    <title>The Adventures of Sherlock Holmes</title>
    <author><name>Arthur Conan Doyle</name></author>
    <link rel="alternate" href="/ebooks/1661.opds" type="application/atom+xml;profile=opds-catalog"/>
  </entry>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != "sherlock" {
			t.Errorf("expected query 'sherlock', got '%s'", query)
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(gutenbergResponse))
	}))
	defer server.Close()

	client := NewClient()
	catalog, err := client.Search(server.URL+"/search.opds/?query={searchTerms}", "sherlock")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if catalog.Title != "Project Gutenberg: Search Results" {
		t.Errorf("expected title 'Project Gutenberg: Search Results', got '%s'", catalog.Title)
	}

	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	if entry.Title != "A Study in Scarlet" {
		t.Errorf("expected title 'A Study in Scarlet', got '%s'", entry.Title)
	}
	if len(entry.Authors) != 1 || entry.Authors[0] != "Arthur Conan Doyle" {
		t.Errorf("expected author 'Arthur Conan Doyle', got %v", entry.Authors)
	}

	if catalog.NextPageURL == "" {
		t.Error("expected next page URL to be set")
	}
}

// TestSearchWithFreeLibFormat tests search with FreeLib style feed
func TestSearchWithFreeLibFormat(t *testing.T) {
	freelibResponse := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/terms/">
  <id>tag:search</id>
  <title>FB2</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <link rel="start" href="/opds/1"/>
  <link rel="search" type="application/opensearchdescription+xml" href="/opds/1/opensearch.xml"/>
  <link rel="search" type="application/atom+xml" href="/opds/1/search?q={searchTerms}"/>
  <entry>
    <id>tag:search:authors</id>
    <title>Поиск авторов</title>
    <link type="application/atom+xml;profile=opds-catalog" href="/opds/1/search?author=толстой"/>
  </entry>
  <entry>
    <id>tag:search:title</id>
    <title>Поиск книг по названию</title>
    <link type="application/atom+xml;profile=opds-catalog" href="/opds/1/search?title=толстой"/>
  </entry>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			t.Error("expected 'q' parameter to be set")
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(freelibResponse))
	}))
	defer server.Close()

	client := NewClient()
	catalog, err := client.Search(server.URL+"/opds/1/search?q={searchTerms}", "толстой")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if catalog.Title != "FB2" {
		t.Errorf("expected title 'FB2', got '%s'", catalog.Title)
	}

	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}

	for _, entry := range catalog.Entries {
		if !entry.IsNavigation {
			t.Errorf("expected entry '%s' to be navigation", entry.Title)
		}
	}
}

// TestSearchLinkExtraction tests that search links are correctly extracted from feed
func TestSearchLinkExtraction(t *testing.T) {
	feedWithSearch := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:root</id>
  <title>Test Catalog</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <link rel="search" type="application/opensearchdescription+xml" href="/opensearch.xml"/>
  <link rel="search" type="application/atom+xml" href="/search?q={searchTerms}"/>
  <entry>
    <id>tag:authors</id>
    <title>By Authors</title>
    <link type="application/atom+xml;profile=opds-catalog" href="/authors"/>
  </entry>
</feed>`

	client := NewClient()
	catalog, err := client.ParseCatalog([]byte(feedWithSearch), "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	foundOpenSearchLink := false
	foundAtomSearchLink := false

	for _, link := range catalog.Links {
		if link.Rel == "search" {
			if strings.Contains(link.Type, "opensearchdescription") {
				foundOpenSearchLink = true
				if link.Href != "http://example.com/opensearch.xml" {
					t.Errorf("expected opensearch href 'http://example.com/opensearch.xml', got '%s'", link.Href)
				}
			}
			if strings.Contains(link.Type, "atom+xml") {
				foundAtomSearchLink = true
			}
		}
	}

	if !foundOpenSearchLink {
		t.Error("expected to find OpenSearch description link")
	}
	if !foundAtomSearchLink {
		t.Error("expected to find Atom search link")
	}
}

// TestSearchWithAuthCredentials tests search with authentication
func TestSearchWithAuthCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:auth-search</id>
  <title>Authenticated Search Results</title>
  <updated>2024-01-01T00:00:00Z</updated>
</feed>`))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Search(server.URL+"/search?q={searchTerms}", "test")
	if err == nil {
		t.Error("expected error without auth")
	}

	clientWithAuth := NewClientWithAuth("testuser", "secret")
	catalog, err := clientWithAuth.Search(server.URL+"/search?q={searchTerms}", "test")
	if err != nil {
		t.Fatalf("Search with auth failed: %v", err)
	}
	if catalog.Title != "Authenticated Search Results" {
		t.Errorf("expected title 'Authenticated Search Results', got '%s'", catalog.Title)
	}
}

// TestOpenSearchDescriptionParsing tests parsing of OpenSearch description documents
func TestOpenSearchDescriptionParsing(t *testing.T) {
	osdXML := `<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
   <ShortName>Gutenberg</ShortName>
   <Description>Search the Project Gutenberg ebook catalog.</Description>
   <Url type="text/html" template="http://www.gutenberg.org/ebooks/search/?query={searchTerms}"/>
   <Url type="application/atom+xml" template="http://www.gutenberg.org/ebooks/search.opds/?query={searchTerms}"/>
</OpenSearchDescription>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/opensearchdescription+xml")
		w.Write([]byte(osdXML))
	}))
	defer server.Close()

	client := NewClient()
	osd, err := client.FetchOpenSearchDescription(server.URL)
	if err != nil {
		t.Fatalf("FetchOpenSearchDescription failed: %v", err)
	}

	if osd.ShortName != "Gutenberg" {
		t.Errorf("expected ShortName 'Gutenberg', got '%s'", osd.ShortName)
	}

	if len(osd.URLs) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(osd.URLs))
	}

	template := osd.GetAtomSearchTemplate()
	expectedTemplate := "http://www.gutenberg.org/ebooks/search.opds/?query={searchTerms}"
	if template != expectedTemplate {
		t.Errorf("expected template '%s', got '%s'", expectedTemplate, template)
	}
}

// TestSearchInfoExtraction tests that SearchInfo is correctly extracted from feeds
func TestSearchInfoExtraction(t *testing.T) {
	tests := []struct {
		name                 string
		feedXML              string
		expectSupported      bool
		expectOpenSearchURL  string
		expectSearchTemplate string
	}{
		{
			name: "Gutenberg style - OpenSearch only",
			feedXML: `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <link rel="search" type="application/opensearchdescription+xml" href="/opensearch.xml"/>
</feed>`,
			expectSupported:      true,
			expectOpenSearchURL:  "http://example.com/opensearch.xml",
			expectSearchTemplate: "",
		},
		{
			name: "FreeLib style - direct Atom template",
			feedXML: `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <link rel="search" type="application/atom+xml" href="/search?q={searchTerms}"/>
</feed>`,
			expectSupported:      true,
			expectOpenSearchURL:  "",
			expectSearchTemplate: "http://example.com/search?q={searchTerms}",
		},
		{
			name: "Both OpenSearch and Atom template",
			feedXML: `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <link rel="search" type="application/opensearchdescription+xml" href="/opensearch.xml"/>
  <link rel="search" type="application/atom+xml" href="/search?q={searchTerms}"/>
</feed>`,
			expectSupported:      true,
			expectOpenSearchURL:  "http://example.com/opensearch.xml",
			expectSearchTemplate: "http://example.com/search?q={searchTerms}",
		},
		{
			name: "No search support",
			feedXML: `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
</feed>`,
			expectSupported: false,
		},
	}

	client := NewClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog, err := client.ParseCatalog([]byte(tt.feedXML), "http://example.com/opds")
			if err != nil {
				t.Fatalf("ParseCatalog failed: %v", err)
			}

			if tt.expectSupported {
				if catalog.SearchInfo == nil {
					t.Fatal("expected SearchInfo to be set")
				}
				if !catalog.SearchInfo.Supported {
					t.Error("expected Supported to be true")
				}
				if catalog.SearchInfo.OpenSearchURL != tt.expectOpenSearchURL {
					t.Errorf("expected OpenSearchURL '%s', got '%s'", tt.expectOpenSearchURL, catalog.SearchInfo.OpenSearchURL)
				}
				if catalog.SearchInfo.SearchTemplateURL != tt.expectSearchTemplate {
					t.Errorf("expected SearchTemplateURL '%s', got '%s'", tt.expectSearchTemplate, catalog.SearchInfo.SearchTemplateURL)
				}
			} else {
				if catalog.SearchInfo != nil {
					t.Error("expected SearchInfo to be nil")
				}
			}
		})
	}
}

// TestGetSearchTemplate tests the GetSearchTemplate method
func TestGetSearchTemplate(t *testing.T) {
	osdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/opensearchdescription+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
   <ShortName>Test</ShortName>
   <Url type="application/atom+xml" template="http://example.com/search?q={searchTerms}"/>
</OpenSearchDescription>`))
	}))
	defer osdServer.Close()

	client := NewClient()

	searchInfo := &SearchInfo{
		Supported:         true,
		SearchTemplateURL: "http://direct.com/search?q={searchTerms}",
		OpenSearchURL:     osdServer.URL,
	}
	template, err := client.GetSearchTemplate(searchInfo)
	if err != nil {
		t.Fatalf("GetSearchTemplate failed: %v", err)
	}
	if template != "http://direct.com/search?q={searchTerms}" {
		t.Errorf("expected direct template, got '%s'", template)
	}

	searchInfo2 := &SearchInfo{
		Supported:     true,
		OpenSearchURL: osdServer.URL,
	}
	template2, err := client.GetSearchTemplate(searchInfo2)
	if err != nil {
		t.Fatalf("GetSearchTemplate with OpenSearch failed: %v", err)
	}
	if template2 != "http://example.com/search?q={searchTerms}" {
		t.Errorf("expected OpenSearch template, got '%s'", template2)
	}

	_, err = client.GetSearchTemplate(nil)
	if err == nil {
		t.Error("expected error with nil SearchInfo")
	}

	_, err = client.GetSearchTemplate(&SearchInfo{Supported: false})
	if err == nil {
		t.Error("expected error with unsupported SearchInfo")
	}
}
