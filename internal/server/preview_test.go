package server

import (
	"testing"
	"time"

	"abb_tts/internal/parser"
)

func TestPreviewStore_CreatePreview(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:       "Test Book",
		Author:      "Test Author",
		Description: "A test description",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "This is the first chapter with some words."},
			{Title: "Chapter 2", Content: "This is the second chapter with more words here."},
		},
	}

	preview := store.CreatePreview(book, "test.epub")

	if preview.ID == "" {
		t.Error("Preview ID should not be empty")
	}
	if preview.BookTitle != "Test Book" {
		t.Errorf("Expected title 'Test Book', got '%s'", preview.BookTitle)
	}
	if preview.BookAuthor != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", preview.BookAuthor)
	}
	if preview.TotalChapters != 2 {
		t.Errorf("Expected 2 chapters, got %d", preview.TotalChapters)
	}
	if preview.TotalWords == 0 {
		t.Error("Total words should not be 0")
	}
	if preview.TotalCharacters == 0 {
		t.Error("Total characters should not be 0")
	}
	if preview.EstimatedDurationMinutes < 0 {
		t.Error("Estimated duration should not be negative")
	}
	if len(preview.CostEstimates) == 0 {
		t.Error("Cost estimates should not be empty")
	}

	// Check that espeak is free
	if espeak, ok := preview.CostEstimates["espeak"]; ok {
		if espeak.Cost != 0 {
			t.Errorf("espeak should be free, got cost %f", espeak.Cost)
		}
	} else {
		t.Error("espeak should be in cost estimates")
	}
}

func TestPreviewStore_GetPreview(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:  "Test Book",
		Author: "Test Author",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "Content here."},
		},
	}

	created := store.CreatePreview(book, "test.epub")

	// Should be able to retrieve it
	retrieved, exists := store.GetPreview(created.ID)
	if !exists {
		t.Error("Preview should exist")
	}
	if retrieved.ID != created.ID {
		t.Error("Retrieved preview ID should match created ID")
	}

	// Non-existent preview
	_, exists = store.GetPreview("non-existent-id")
	if exists {
		t.Error("Non-existent preview should not exist")
	}
}

func TestPreviewStore_DeletePreview(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:  "Test Book",
		Author: "Test Author",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "Content here."},
		},
	}

	preview := store.CreatePreview(book, "test.epub")

	// Delete it
	store.DeletePreview(preview.ID)

	// Should no longer exist
	_, exists := store.GetPreview(preview.ID)
	if exists {
		t.Error("Deleted preview should not exist")
	}
}

func TestPreviewStore_CoverImage(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	coverData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes

	book := &parser.Book{
		Title:      "Test Book",
		Author:     "Test Author",
		CoverImage: coverData,
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "Content here."},
		},
	}

	preview := store.CreatePreview(book, "test.epub")

	if preview.CoverImageURL == "" {
		t.Error("Cover image URL should be set when cover exists")
	}

	// Retrieve cover
	cover, exists := store.GetCover(preview.ID)
	if !exists {
		t.Error("Cover should exist")
	}
	if len(cover) != len(coverData) {
		t.Errorf("Cover data length mismatch: got %d, want %d", len(cover), len(coverData))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		minutes  int
		expected string
	}{
		{0, "0m"},
		{30, "30m"},
		{59, "59m"},
		{60, "1h"},
		{90, "1h 30m"},
		{120, "2h"},
		{150, "2h 30m"},
		{813, "13h 33m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.minutes)
		if result != tt.expected {
			t.Errorf("formatDuration(%d) = %s, want %s", tt.minutes, result, tt.expected)
		}
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"hello  world", 2}, // double space
		{"hello\nworld", 2}, // newline
		{"hello\tworld", 2}, // tab
		{"one two three four five", 5},
	}

	for _, tt := range tests {
		result := splitWords(tt.input)
		if len(result) != tt.expected {
			t.Errorf("splitWords(%q) = %d words, want %d", tt.input, len(result), tt.expected)
		}
	}
}

func TestCostEstimates(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	// Create a book with 1 million characters
	content := make([]byte, 1000000)
	for i := range content {
		content[i] = 'a'
	}

	book := &parser.Book{
		Title:  "Large Book",
		Author: "Author",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: string(content)},
		},
	}

	preview := store.CreatePreview(book, "large.epub")

	// Check Google WaveNet cost (should be ~$16 for 1M chars)
	if wavenet, ok := preview.CostEstimates["google_wavenet"]; ok {
		if wavenet.Cost < 15.9 || wavenet.Cost > 16.1 {
			t.Errorf("google_wavenet cost for 1M chars should be ~$16, got $%.2f", wavenet.Cost)
		}
	}

	// Check espeak is still free
	if espeak, ok := preview.CostEstimates["espeak"]; ok {
		if espeak.Cost != 0 {
			t.Errorf("espeak should be free regardless of size, got $%.2f", espeak.Cost)
		}
	}
}
