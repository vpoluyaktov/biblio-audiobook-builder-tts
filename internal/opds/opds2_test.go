package opds

import (
	"testing"
)

func TestParseOPDS2Catalog_InternetArchive(t *testing.T) {
	jsonData := []byte(`{
		"metadata": {
			"title": "Archive.org"
		},
		"links": [
			{
				"href": "https://opds.prod.archive.org/catalog?short_query=BRS19",
				"rel": "self",
				"type": "application/opds+json"
			},
			{
				"href": "https://opds.prod.archive.org/search{?query}",
				"rel": "search",
				"templated": true,
				"type": "application/opds+json"
			}
		],
		"navigation": [
			{
				"href": "https://opds.prod.archive.org/catalog?short_query=LCW25",
				"title": "LCP Books Ready to Borrow",
				"type": "application/opds+json"
			},
			{
				"href": "https://opds.prod.archive.org/catalog?short_query=MOS12",
				"title": "Modern Books",
				"type": "application/opds+json"
			}
		],
		"groups": [
			{
				"metadata": {
					"title": "Available Now",
					"numberOfItems": 1684432
				},
				"publications": [
					{
						"metadata": {
							"@type": "http://schema.org/Book",
							"title": "Test Book",
							"author": "Test Author",
							"description": "A test book description",
							"identifier": "https://archive.org/details/testbook",
							"language": "en",
							"published": "2020-09-28T01:25:02Z"
						},
						"images": [
							{
								"href": "https://archive.org/download/testbook/__ia_thumb.jpg",
								"rel": "cover",
								"type": "image/jpeg",
								"width": 800,
								"height": 1400
							}
						],
						"links": [
							{
								"href": "https://archive.org/services/loans/loan/?opds=1&identifier=testbook&action=webpub",
								"rel": "http://opds-spec.org/acquisition/borrow",
								"type": "application/opds-publication+json"
							}
						]
					}
				]
			}
		]
	}`)

	client := NewClient()
	catalog, err := client.ParseCatalog(jsonData, "https://archive.org/services/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if catalog.Title != "Archive.org" {
		t.Errorf("expected title 'Archive.org', got '%s'", catalog.Title)
	}

	if catalog.SearchInfo == nil || !catalog.SearchInfo.Supported {
		t.Error("expected search to be supported")
	}

	if catalog.SearchInfo.SearchTemplateURL != "https://opds.prod.archive.org/search{?query}" {
		t.Errorf("expected search template URL 'https://opds.prod.archive.org/search{?query}', got '%s'", 
			catalog.SearchInfo.SearchTemplateURL)
	}

	if len(catalog.Entries) < 3 {
		t.Fatalf("expected at least 3 entries (2 navigation + 1 publication), got %d", len(catalog.Entries))
	}

	navCount := 0
	pubCount := 0
	for _, entry := range catalog.Entries {
		if entry.IsNavigation {
			navCount++
		} else if len(entry.DownloadLinks) > 0 || entry.Title == "Test Book" {
			pubCount++
		}
	}

	if navCount < 2 {
		t.Errorf("expected at least 2 navigation entries, got %d", navCount)
	}

	if pubCount < 1 {
		t.Errorf("expected at least 1 publication entry, got %d", pubCount)
	}

	var testBook *CatalogEntry
	for i, entry := range catalog.Entries {
		if entry.Title == "Test Book" {
			testBook = &catalog.Entries[i]
			break
		}
	}

	if testBook == nil {
		t.Fatal("could not find 'Test Book' entry")
	}

	if len(testBook.Authors) != 1 || testBook.Authors[0] != "Test Author" {
		t.Errorf("expected author 'Test Author', got %v", testBook.Authors)
	}

	if testBook.Summary != "A test book description" {
		t.Errorf("expected summary 'A test book description', got '%s'", testBook.Summary)
	}

	if testBook.Language != "en" {
		t.Errorf("expected language 'en', got '%s'", testBook.Language)
	}

	if testBook.CoverURL != "https://archive.org/download/testbook/__ia_thumb.jpg" {
		t.Errorf("expected cover URL 'https://archive.org/download/testbook/__ia_thumb.jpg', got '%s'", 
			testBook.CoverURL)
	}

	if len(testBook.DownloadLinks) < 1 {
		t.Errorf("expected at least 1 download link, got %d", len(testBook.DownloadLinks))
	}
}

func TestIsOPDS2_JSON(t *testing.T) {
	jsonData := []byte(`{"metadata": {"title": "Test"}}`)
	if !isOPDS2(jsonData) {
		t.Error("expected isOPDS2 to return true for JSON data")
	}
}

func TestIsOPDS2_XML(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`)
	if isOPDS2(xmlData) {
		t.Error("expected isOPDS2 to return false for XML data")
	}
}

func TestParseOPDS2Catalog_WithAuthors(t *testing.T) {
	tests := []struct {
		name           string
		authorField    string
		expectedAuthor string
	}{
		{
			name:           "String author",
			authorField:    `"author": "John Doe"`,
			expectedAuthor: "John Doe",
		},
		{
			name:           "Object author",
			authorField:    `"author": {"name": "Jane Smith"}`,
			expectedAuthor: "Jane Smith",
		},
		{
			name:           "Array of strings",
			authorField:    `"author": ["Alice", "Bob"]`,
			expectedAuthor: "Alice",
		},
		{
			name:           "Array of objects",
			authorField:    `"author": [{"name": "Charlie"}, {"name": "Dave"}]`,
			expectedAuthor: "Charlie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := []byte(`{
				"metadata": {"title": "Test Feed"},
				"publications": [{
					"metadata": {
						"title": "Test Book",
						` + tt.authorField + `
					}
				}]
			}`)

			client := NewClient()
			catalog, err := client.ParseCatalog(jsonData, "https://example.com/opds")
			if err != nil {
				t.Fatalf("ParseCatalog failed: %v", err)
			}

			if len(catalog.Entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(catalog.Entries))
			}

			if len(catalog.Entries[0].Authors) == 0 {
				t.Fatal("expected at least one author")
			}

			if catalog.Entries[0].Authors[0] != tt.expectedAuthor {
				t.Errorf("expected first author '%s', got '%s'", 
					tt.expectedAuthor, catalog.Entries[0].Authors[0])
			}
		})
	}
}
