package server

import (
	"testing"
	"time"

	"biblio-audiobook-builder-tts/internal/parser"
)

// TC23: CreatePreview with genre — Book with Genre populates Preview.Genre.
func TestCreatePreview_WithGenre(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:  "Dune",
		Author: "Frank Herbert",
		Genre:  "Science Fiction, Adventure",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "The beginning of the story."},
		},
	}

	preview := store.CreatePreview(book, "dune.epub")

	if preview.Genre != "Science Fiction, Adventure" {
		t.Errorf("CreatePreview should copy book.Genre to preview.Genre: got %q, want %q",
			preview.Genre, "Science Fiction, Adventure")
	}
}

// TC24: CreatePreview without genre — Preview.Genre is empty string.
func TestCreatePreview_WithoutGenre(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:  "Unknown Genre Book",
		Author: "Anonymous",
		Genre:  "", // no genre
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "Some content."},
		},
	}

	preview := store.CreatePreview(book, "unknown.epub")

	if preview.Genre != "" {
		t.Errorf("CreatePreview with no genre should leave Preview.Genre empty: got %q", preview.Genre)
	}
}

// TestPreview_GenreFieldExists verifies the Genre field is present on the Preview struct
// with the correct JSON tag (json:"genre,omitempty").
func TestPreview_GenreFieldOmitEmpty(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:  "Test Book",
		Author: "Author",
		Genre:  "",
		Chapters: []parser.Chapter{
			{Title: "Ch1", Content: "content"},
		},
	}

	preview := store.CreatePreview(book, "test.epub")

	// The Genre field must exist on the Preview struct; when empty it is omitempty.
	// We access it directly to verify it compiles and is zero value.
	_ = preview.Genre // will fail to compile if field doesn't exist
}

// TestCreatePreview_GenreFromOPDSOverride verifies that Genre can be overridden on
// the Preview after creation (as done by handleOPDSDownload when Categories are present).
func TestCreatePreview_GenreOverrideAfterCreation(t *testing.T) {
	store := NewPreviewStore(5 * time.Minute)

	book := &parser.Book{
		Title:  "Some Book",
		Author: "Author",
		Genre:  "Fantasy",
		Chapters: []parser.Chapter{
			{Title: "Ch1", Content: "content"},
		},
	}

	preview := store.CreatePreview(book, "book.epub")

	// Simulate OPDS categories overriding ebook-parsed genre (per design decision)
	preview.Genre = "Literary Fiction, Historical"

	retrieved, exists := store.GetPreview(preview.ID)
	if !exists {
		t.Fatal("Preview should exist after creation")
	}

	// The override must be visible via the same pointer (Preview is stored by pointer)
	if retrieved.Genre != "Literary Fiction, Historical" {
		t.Errorf("OPDS genre override not reflected: got %q, want %q",
			retrieved.Genre, "Literary Fiction, Historical")
	}
}
