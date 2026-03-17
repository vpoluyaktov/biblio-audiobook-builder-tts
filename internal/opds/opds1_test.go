package opds

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseOPDS1Catalog_NavigationFeed(t *testing.T) {
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
	catalog, err := client.ParseOPDS1Catalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseOPDS1Catalog failed: %v", err)
	}

	if catalog.Title != "Test Catalog" {
		t.Errorf("expected title 'Test Catalog', got '%s'", catalog.Title)
	}

	if len(catalog.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog.Entries))
	}

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

func TestParseOPDS1Catalog_AcquisitionFeed(t *testing.T) {
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
	catalog, err := client.ParseOPDS1Catalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseOPDS1Catalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]

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

	if len(entry.Authors) != 2 {
		t.Fatalf("expected 2 authors, got %d", len(entry.Authors))
	}
	if entry.Authors[0] != "John Doe" {
		t.Errorf("expected first author 'John Doe', got '%s'", entry.Authors[0])
	}
	if entry.Authors[1] != "Jane Smith" {
		t.Errorf("expected second author 'Jane Smith', got '%s'", entry.Authors[1])
	}

	if len(entry.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(entry.Categories))
	}
	if entry.Categories[0] != "Fiction" {
		t.Errorf("expected first category 'Fiction', got '%s'", entry.Categories[0])
	}

	if entry.CoverURL != "http://example.com/covers/123.jpg" {
		t.Errorf("expected cover URL 'http://example.com/covers/123.jpg', got '%s'", entry.CoverURL)
	}
	if entry.ThumbnailURL != "http://example.com/thumbs/123.jpg" {
		t.Errorf("expected thumbnail URL 'http://example.com/thumbs/123.jpg', got '%s'", entry.ThumbnailURL)
	}

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

	if entry.IsNavigation {
		t.Error("expected entry to not be navigation")
	}
}

func TestParseOPDS1Catalog_Pagination(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:page1</id>
  <title>Page 1</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <link rel="next" href="/page2"/>
  <link rel="previous" href="/page0"/>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseOPDS1Catalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseOPDS1Catalog failed: %v", err)
	}

	if catalog.NextPageURL != "http://example.com/page2" {
		t.Errorf("expected next page URL 'http://example.com/page2', got '%s'", catalog.NextPageURL)
	}
	if catalog.PrevPageURL != "http://example.com/page0" {
		t.Errorf("expected prev page URL 'http://example.com/page0', got '%s'", catalog.PrevPageURL)
	}
}

func TestParseOPDS1Catalog_ContentAsSummary(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>Test</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <entry>
    <id>tag:book1</id>
    <title>Book with Content</title>
    <content type="text">This is content used as summary</content>
  </entry>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseOPDS1Catalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseOPDS1Catalog failed: %v", err)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	if catalog.Entries[0].Summary != "This is content used as summary" {
		t.Errorf("expected summary from content, got '%s'", catalog.Entries[0].Summary)
	}
}

func TestSearchWithGutenbergFormat(t *testing.T) {
	gutenbergResponse := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:opensearch="http://a9.com/-/spec/opensearch/1.1/">
  <id>https://www.gutenberg.org/ebooks/search.opds/?query=sherlock</id>
  <title>Search Results for 'sherlock'</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <opensearch:totalResults>42</opensearch:totalResults>
  <entry>
    <id>urn:gutenberg:1661</id>
    <title>The Adventures of Sherlock Holmes</title>
    <author><name>Arthur Conan Doyle</name></author>
    <link rel="http://opds-spec.org/acquisition" href="/ebooks/1661.epub" type="application/epub+zip"/>
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
	searchURL := server.URL + "?query={searchTerms}"
	catalog, err := client.Search(searchURL, "sherlock")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if catalog.Title != "Search Results for 'sherlock'" {
		t.Errorf("expected title 'Search Results for 'sherlock'', got '%s'", catalog.Title)
	}

	if len(catalog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
	}

	entry := catalog.Entries[0]
	if entry.Title != "The Adventures of Sherlock Holmes" {
		t.Errorf("expected title 'The Adventures of Sherlock Holmes', got '%s'", entry.Title)
	}
	if len(entry.Authors) != 1 || entry.Authors[0] != "Arthur Conan Doyle" {
		t.Errorf("expected author 'Arthur Conan Doyle', got %v", entry.Authors)
	}
}

func TestSearchLinkExtraction(t *testing.T) {
	feedWithSearch := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:root</id>
  <title>Test Catalog</title>
  <updated>2024-01-01T00:00:00Z</updated>
  <link rel="search" href="/opensearch.xml" type="application/opensearchdescription+xml"/>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(feedWithSearch))
	}))
	defer server.Close()

	client := NewClient()
	catalog, err := client.FetchCatalog(server.URL)
	if err != nil {
		t.Fatalf("FetchCatalog failed: %v", err)
	}

	if catalog.SearchInfo == nil {
		t.Fatal("expected SearchInfo to be populated")
	}

	if !catalog.SearchInfo.Supported {
		t.Error("expected search to be supported")
	}

	if catalog.SearchInfo.OpenSearchURL != server.URL+"/opensearch.xml" {
		t.Errorf("expected OpenSearch URL '%s/opensearch.xml', got '%s'",
			server.URL, catalog.SearchInfo.OpenSearchURL)
	}
}

func TestOpenSearchDescriptionParsing(t *testing.T) {
	osdXML := `<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
   <ShortName>Gutenberg</ShortName>
   <Description>Search Project Gutenberg</Description>
   <Url type="application/atom+xml" template="https://www.gutenberg.org/ebooks/search.opds/?query={searchTerms}"/>
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

	if osd.Description != "Search Project Gutenberg" {
		t.Errorf("expected Description 'Search Project Gutenberg', got '%s'", osd.Description)
	}

	if len(osd.URLs) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(osd.URLs))
	}

	if osd.URLs[0].Template != "https://www.gutenberg.org/ebooks/search.opds/?query={searchTerms}" {
		t.Errorf("unexpected template: %s", osd.URLs[0].Template)
	}

	template := osd.GetAtomSearchTemplate()
	if template != "https://www.gutenberg.org/ebooks/search.opds/?query={searchTerms}" {
		t.Errorf("unexpected template from GetAtomSearchTemplate: %s", template)
	}
}
