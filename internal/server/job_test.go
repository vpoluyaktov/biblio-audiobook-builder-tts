package server

import (
	"testing"

	"biblio-audiobook-builder-tts/internal/parser"
)

// TestNewJob_GenreParameter verifies the updated NewJob signature accepts a genre parameter
// and stores it in BookGenre.
func TestNewJob_GenreParameter(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "Fiction, Adventure", 1.0, 1.0, false)
	if job.BookGenre != "Fiction, Adventure" {
		t.Errorf("NewJob BookGenre: got %q, want %q", job.BookGenre, "Fiction, Adventure")
	}
}

// TestNewJob_EmptyGenre verifies NewJob accepts an empty genre without issue.
func TestNewJob_EmptyGenre(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "", 1.0, 1.0, false)
	if job.BookGenre != "" {
		t.Errorf("NewJob empty BookGenre: got %q, want %q", job.BookGenre, "")
	}
}

// TC17: SetBook with genre when BookGenre is empty — BookGenre populated from book.Genre.
func TestSetBook_SetsGenreWhenEmpty(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "", 1.0, 1.0, false)

	book := &parser.Book{
		Title:  "My Book",
		Author: "Some Author",
		Genre:  "Science Fiction",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "content"},
		},
	}

	job.SetBook(book)

	if job.BookGenre != "Science Fiction" {
		t.Errorf("SetBook should populate BookGenre from book.Genre when empty: got %q, want %q",
			job.BookGenre, "Science Fiction")
	}
}

// TC18: SetBook with genre when BookGenre already set — BookGenre must NOT be overwritten.
func TestSetBook_DoesNotOverwriteExistingGenre(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "Fantasy", 1.0, 1.0, false)

	book := &parser.Book{
		Title:  "My Book",
		Author: "Some Author",
		Genre:  "Science Fiction",
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "content"},
		},
	}

	job.SetBook(book)

	if job.BookGenre != "Fantasy" {
		t.Errorf("SetBook should NOT overwrite existing BookGenre: got %q, want %q",
			job.BookGenre, "Fantasy")
	}
}

// TC19: Clone includes BookGenre — verify Clone() copies BookGenre to JobDTO.
func TestClone_IncludesBookGenre(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "Horror, Mystery", 1.0, 1.0, false)

	dto := job.Clone()

	if dto.BookGenre != "Horror, Mystery" {
		t.Errorf("Clone() should copy BookGenre to JobDTO: got %q, want %q",
			dto.BookGenre, "Horror, Mystery")
	}
}

// TestClone_EmptyGenrePreserved verifies Clone preserves empty BookGenre (not nil or changed).
func TestClone_EmptyGenrePreserved(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "", 1.0, 1.0, false)

	dto := job.Clone()

	if dto.BookGenre != "" {
		t.Errorf("Clone() should preserve empty BookGenre: got %q", dto.BookGenre)
	}
}

// TestSetBook_NilBook verifies SetBook handles nil book without panicking.
func TestSetBook_NilBook(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "Fantasy", 1.0, 1.0, false)

	// Should not panic
	job.SetBook(nil)

	// Genre should remain unchanged
	if job.BookGenre != "Fantasy" {
		t.Errorf("SetBook(nil) should not change BookGenre: got %q, want %q",
			job.BookGenre, "Fantasy")
	}
}

// TestSetBook_BookWithEmptyGenre verifies that when book.Genre is empty,
// an existing BookGenre is left intact.
func TestSetBook_BookWithEmptyGenreDoesNotClearExisting(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "Fantasy", 1.0, 1.0, false)

	book := &parser.Book{
		Title:  "My Book",
		Author: "Some Author",
		Genre:  "", // empty
		Chapters: []parser.Chapter{
			{Title: "Chapter 1", Content: "content"},
		},
	}

	job.SetBook(book)

	if job.BookGenre != "Fantasy" {
		t.Errorf("SetBook with empty book.Genre should not change existing BookGenre: got %q, want %q",
			job.BookGenre, "Fantasy")
	}
}
