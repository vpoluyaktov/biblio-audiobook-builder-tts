package server

import (
	"testing"

	"biblio-audiobook-builder-tts/internal/parser"
)

// resolveGenreForTest replicates the 3-tier genre resolution logic described in
// FEATURE_GENRE.md §File 8 (worker.go), so unit tests can exercise the logic without
// running the full buildM4B pipeline (which requires ffmpeg).
//
// This function MUST match the logic the Backend Developer puts in buildM4B:
//  1. Use job.BookGenre if set.
//  2. Otherwise fall back to book.Genre.
//  3. Otherwise fall back to the constant "Audiobook".
func resolveGenreForTest(job *Job, book *parser.Book) string {
	genre := job.BookGenre
	if genre == "" && book != nil && book.Genre != "" {
		genre = book.Genre
	}
	if genre == "" {
		genre = "Audiobook"
	}
	return genre
}

// TestResolveGenre exercises the 3-tier fallback logic (TC13-TC16).
func TestResolveGenre(t *testing.T) {
	tests := []struct {
		name        string
		jobGenre    string // Job.BookGenre
		bookGenre   string // parser.Book.Genre
		expectedOut string
	}{
		// TC13: Job has BookGenre and book also has Genre — job.BookGenre wins.
		{
			name:        "TC13: job BookGenre wins over book Genre",
			jobGenre:    "Fantasy",
			bookGenre:   "Science Fiction",
			expectedOut: "Fantasy",
		},
		// TC14: Job has empty BookGenre, book has Genre — book.Genre used.
		{
			name:        "TC14: empty job BookGenre falls back to book Genre",
			jobGenre:    "",
			bookGenre:   "Science Fiction",
			expectedOut: "Science Fiction",
		},
		// TC15: Both empty — fallback to "Audiobook".
		{
			name:        "TC15: both empty falls back to Audiobook",
			jobGenre:    "",
			bookGenre:   "",
			expectedOut: "Audiobook",
		},
		// TC16: Job has BookGenre, book Genre is empty — job.BookGenre used.
		{
			name:        "TC16: job BookGenre used when book Genre is empty",
			jobGenre:    "Horror",
			bookGenre:   "",
			expectedOut: "Horror",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", tt.jobGenre, 1.0, 1.0, false)

			book := &parser.Book{
				Title:  "Test Book",
				Author: "Author",
				Genre:  tt.bookGenre,
				Chapters: []parser.Chapter{
					{Title: "Ch1", Content: "text"},
				},
			}

			got := resolveGenreForTest(job, book)
			if got != tt.expectedOut {
				t.Errorf("resolveGenre(%q, %q) = %q, want %q",
					tt.jobGenre, tt.bookGenre, got, tt.expectedOut)
			}
		})
	}
}

// TestResolveGenre_NilBook verifies the fallback when book is nil.
// This should not happen in normal flow, but the logic must not panic.
func TestResolveGenre_NilBook(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "", 1.0, 1.0, false)

	got := resolveGenreForTest(job, nil)
	if got != "Audiobook" {
		t.Errorf("resolveGenre with nil book: got %q, want %q", got, "Audiobook")
	}
}

// TestResolveGenre_MultipleGenresPreserved verifies that multi-genre strings pass through untouched.
func TestResolveGenre_MultipleGenresPreserved(t *testing.T) {
	job := NewJob("test.epub", "/tmp/test.epub", "espeak", "en", "en", "Fiction, Adventure, Mystery", 1.0, 1.0, false)
	book := &parser.Book{Genre: "Other"}

	got := resolveGenreForTest(job, book)
	if got != "Fiction, Adventure, Mystery" {
		t.Errorf("resolveGenre multi-genre: got %q, want %q", got, "Fiction, Adventure, Mystery")
	}
}
