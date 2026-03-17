package opds

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Client creation and configuration tests

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

// FetchCatalog tests

func TestFetchCatalog_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test</title>
</feed>`))
	}))
	defer server.Close()

	client := NewClientWithAuth("user", "pass")
	catalog, err := client.FetchCatalog(server.URL)
	if err != nil {
		t.Fatalf("FetchCatalog failed: %v", err)
	}
	if catalog.Title != "Test" {
		t.Errorf("expected title 'Test', got '%s'", catalog.Title)
	}
}

func TestFetchCatalog_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.FetchCatalog(server.URL)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// Search tests

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query != "test query" {
			t.Errorf("expected query 'test query', got '%s'", query)
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Search Results</title>
</feed>`))
	}))
	defer server.Close()

	client := NewClient()
	searchURL := server.URL + "?q={searchTerms}"
	catalog, err := client.Search(searchURL, "test query")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if catalog.Title != "Search Results" {
		t.Errorf("expected title 'Search Results', got '%s'", catalog.Title)
	}
}

func TestSearchWithAuthCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Authenticated Search</title>
</feed>`))
	}))
	defer server.Close()

	client := NewClientWithAuth("testuser", "secret")
	searchURL := server.URL + "?q={searchTerms}"
	catalog, err := client.Search(searchURL, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if catalog.Title != "Authenticated Search" {
		t.Errorf("expected title 'Authenticated Search', got '%s'", catalog.Title)
	}
}

// DownloadBook tests

func TestDownloadBook(t *testing.T) {
	bookContent := []byte("fake epub content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/epub+zip")
		w.Write(bookContent)
	}))
	defer server.Close()

	client := NewClient()
	data, contentType, err := client.DownloadBook(server.URL)
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
		w.Write([]byte("book content"))
	}))
	defer server.Close()

	client := NewClientWithAuth("user", "pass")
	data, _, err := client.DownloadBook(server.URL)
	if err != nil {
		t.Fatalf("DownloadBook failed: %v", err)
	}
	if string(data) != "book content" {
		t.Errorf("unexpected content: %s", data)
	}
}

// ParseCatalog routing tests

func TestParseCatalog_RoutesToOPDS1(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:test</id>
  <title>OPDS 1.x Feed</title>
  <updated>2024-01-01T00:00:00Z</updated>
</feed>`)

	client := NewClient()
	catalog, err := client.ParseCatalog(xmlData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if catalog.Title != "OPDS 1.x Feed" {
		t.Errorf("expected title 'OPDS 1.x Feed', got '%s'", catalog.Title)
	}
}

func TestParseCatalog_RoutesToOPDS2(t *testing.T) {
	jsonData := []byte(`{
		"metadata": {
			"title": "OPDS 2.0 Feed"
		},
		"publications": []
	}`)

	client := NewClient()
	catalog, err := client.ParseCatalog(jsonData, "http://example.com/opds")
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if catalog.Title != "OPDS 2.0 Feed" {
		t.Errorf("expected title 'OPDS 2.0 Feed', got '%s'", catalog.Title)
	}
}
