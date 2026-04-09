package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"biblio-audiobook-builder-tts/internal/config"
	"biblio-audiobook-builder-tts/internal/parser"
	"biblio-audiobook-builder-tts/internal/tts"
)

// newTestServer creates a minimal Server instance suitable for unit tests.
// It uses no database, no auth, and a real PreviewStore.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		DefaultProvider:   "espeak",
		DefaultVoice:      "en",
		DefaultSpeed:      1.0,
		DefaultPitch:      1.0,
		AuthMode:          "", // no auth
	}
	svc := tts.NewService(cfg)
	srv := New(":0", cfg, svc)
	return srv
}

// TestHandleOPDSConvert_PassesGenreToJob verifies TC22:
// handleOPDSConvert creates a job whose BookGenre matches preview.Genre.
func TestHandleOPDSConvert_PassesGenreToJob(t *testing.T) {
	srv := newTestServer(t)

	// Pre-populate a preview with a genre and a non-empty FilePath so the
	// handler does not reject it.  We use a temp file path that exists.
	tmpFile := t.TempDir() + "/book.epub"

	preview := &Preview{
		ID:         "preview-genre-test",
		FileName:   "book.epub",
		FilePath:   tmpFile,
		BookTitle:  "Test Book",
		BookAuthor: "Author",
		Genre:      "Science Fiction, Fantasy",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * time.Minute),
		CostEstimates: map[string]CostEstimate{
			"espeak": {Cost: 0, Currency: "USD", Note: "Free"},
		},
	}
	srv.previewStore.mu.Lock()
	srv.previewStore.previews[preview.ID] = preview
	srv.previewStore.mu.Unlock()

	reqBody, _ := json.Marshal(map[string]interface{}{
		"preview_id": "preview-genre-test",
		"provider":   "espeak",
		"voice":      "en",
		"language":   "en",
		"speed":      1.0,
		"pitch":      1.0,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/opds/convert", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleOPDSConvert(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("handleOPDSConvert returned %d, want %d; body: %s",
			w.Code, http.StatusCreated, w.Body.String())
	}

	var dto JobDTO
	if err := json.NewDecoder(w.Body).Decode(&dto); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if dto.BookGenre != "Science Fiction, Fantasy" {
		t.Errorf("Job BookGenre from OPDS convert: got %q, want %q",
			dto.BookGenre, "Science Fiction, Fantasy")
	}
}

// TestHandleOPDSConvert_EmptyGenreAllowed verifies that a preview with no genre
// creates a job with empty BookGenre (genre resolution falls back to "Audiobook" at
// build time, not at job creation time).
func TestHandleOPDSConvert_EmptyGenreAllowed(t *testing.T) {
	srv := newTestServer(t)

	tmpFile := t.TempDir() + "/book.epub"

	preview := &Preview{
		ID:         "preview-no-genre",
		FileName:   "book.epub",
		FilePath:   tmpFile,
		BookTitle:  "No Genre Book",
		BookAuthor: "Author",
		Genre:      "", // no genre
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * time.Minute),
		CostEstimates: map[string]CostEstimate{
			"espeak": {Cost: 0, Currency: "USD"},
		},
	}
	srv.previewStore.mu.Lock()
	srv.previewStore.previews[preview.ID] = preview
	srv.previewStore.mu.Unlock()

	reqBody, _ := json.Marshal(map[string]interface{}{
		"preview_id": "preview-no-genre",
		"provider":   "espeak",
		"voice":      "en",
		"language":   "en",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/opds/convert", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleOPDSConvert(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("handleOPDSConvert with no genre returned %d, want %d; body: %s",
			w.Code, http.StatusCreated, w.Body.String())
	}

	var dto JobDTO
	if err := json.NewDecoder(w.Body).Decode(&dto); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if dto.BookGenre != "" {
		t.Errorf("Job BookGenre should be empty when preview has no genre: got %q", dto.BookGenre)
	}
}

// TestOPDSDownloadRequestHasCategories verifies TC20/TC21:
// The request struct for handleOPDSDownload has a Categories field.
// We test this by sending a request with categories and verifying the preview
// returned by the handler carries the genre derived from those categories.
//
// NOTE: This test requires a real EPUB file to be downloaded, which is impractical
// in unit tests. Instead we test the joinOPDSCategories helper which encapsulates
// the same logic applied to req.Categories.
func TestJoinOPDSCategories(t *testing.T) {
	tests := []struct {
		name       string
		categories []string
		expected   string
	}{
		{
			name:       "single category",
			categories: []string{"Fiction"},
			expected:   "Fiction",
		},
		{
			name:       "multiple categories joined",
			categories: []string{"Fiction", "Adventure", "Classic"},
			expected:   "Fiction, Adventure, Classic",
		},
		{
			name:       "empty categories returns empty",
			categories: []string{},
			expected:   "",
		},
		{
			name:       "nil categories returns empty",
			categories: nil,
			expected:   "",
		},
		{
			name:       "whitespace entries filtered",
			categories: []string{"Fiction", "  ", "", "History"},
			expected:   "Fiction, History",
		},
		{
			name:       "case-insensitive dedup",
			categories: []string{"Fiction", "fiction", "FICTION", "Adventure"},
			expected:   "Fiction, Adventure",
		},
		{
			name:       "more than 5 truncated",
			categories: []string{"A", "B", "C", "D", "E", "F", "G"},
			expected:   "A, B, C, D, E",
		},
		{
			name:       "special chars preserved",
			categories: []string{"Science Fiction & Fantasy"},
			expected:   "Science Fiction & Fantasy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinOPDSCategories(tt.categories)
			if result != tt.expected {
				t.Errorf("joinOPDSCategories(%v) = %q, want %q",
					tt.categories, result, tt.expected)
			}
		})
	}
}

// TestHandleOPDSConvert_PreviewGenreFromBook verifies TC21:
// When no OPDS categories are provided, genre comes from the ebook parse result
// (which is already stored in preview.Genre at download time).
func TestHandleOPDSConvert_PreviewGenreFromBook(t *testing.T) {
	srv := newTestServer(t)

	tmpFile := t.TempDir() + "/book.epub"

	// Simulate a preview that was created from an ebook with genre metadata
	preview := &Preview{
		ID:         "preview-ebook-genre",
		FileName:   "book.epub",
		FilePath:   tmpFile,
		BookTitle:  "Ebook With Genre",
		BookAuthor: "Author",
		Genre:      "Historical Fiction", // from ebook metadata
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * time.Minute),
		CostEstimates: map[string]CostEstimate{
			"espeak": {Cost: 0, Currency: "USD"},
		},
	}
	srv.previewStore.mu.Lock()
	srv.previewStore.previews[preview.ID] = preview
	srv.previewStore.mu.Unlock()

	reqBody, _ := json.Marshal(map[string]interface{}{
		"preview_id": "preview-ebook-genre",
		"provider":   "espeak",
		"voice":      "en",
		"language":   "en",
		// No categories field — simulates non-OPDS upload path
	})

	req := httptest.NewRequest(http.MethodPost, "/api/opds/convert", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleOPDSConvert(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("handleOPDSConvert returned %d; body: %s", w.Code, w.Body.String())
	}

	var dto JobDTO
	if err := json.NewDecoder(w.Body).Decode(&dto); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if dto.BookGenre != "Historical Fiction" {
		t.Errorf("Job BookGenre from ebook genre: got %q, want %q",
			dto.BookGenre, "Historical Fiction")
	}
}

// TestOPDSRoutingSmoke verifies HTTP method handling for OPDS genre-related endpoints.
func TestOPDSRoutingSmoke(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantNotAllowed bool // true if we expect 405 Method Not Allowed
	}{
		{
			name:           "GET /api/opds/download not allowed",
			method:         http.MethodGet,
			path:           "/api/opds/download",
			wantNotAllowed: true,
		},
		{
			name:           "GET /api/opds/convert not allowed",
			method:         http.MethodGet,
			path:           "/api/opds/convert",
			wantNotAllowed: true,
		},
		{
			name:           "POST /api/opds/download accepted (not 405)",
			method:         http.MethodPost,
			path:           "/api/opds/download",
			wantNotAllowed: false,
		},
		{
			name:           "POST /api/opds/convert accepted (not 405)",
			method:         http.MethodPost,
			path:           "/api/opds/convert",
			wantNotAllowed: false,
		},
	}

	srv := newTestServer(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodPost {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte("{}")))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()

			// Call handler directly based on path
			switch tt.path {
			case "/api/opds/download":
				srv.handleOPDSDownload(w, req)
			case "/api/opds/convert":
				srv.handleOPDSConvert(w, req)
			}

			if tt.wantNotAllowed {
				if w.Code != http.StatusMethodNotAllowed {
					t.Errorf("%s %s: got %d, want %d",
						tt.method, tt.path, w.Code, http.StatusMethodNotAllowed)
				}
			} else {
				if w.Code == http.StatusMethodNotAllowed {
					t.Errorf("%s %s: got 405 Method Not Allowed (should be accepted)",
						tt.method, tt.path)
				}
			}
		})
	}
}

// Compile-time check: verify Genre is present on Preview struct.
var _ = Preview{Genre: "Fiction"}

// Compile-time check: verify BookGenre is present on Job and JobDTO structs.
var _ = Job{BookGenre: ""}
var _ = JobDTO{BookGenre: ""}

// Compile-time check: verify Book.Genre is present on parser.Book.
var _ = parser.Book{Genre: ""}
