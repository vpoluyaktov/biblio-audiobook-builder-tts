package parser

import (
	"testing"
)

// TestJoinGenres tests the joinGenres helper function (test cases 8-12 from spec).
// The language parameter is set to "en" throughout because these tests use plain
// human-readable strings (not FB2 codes), so no translation occurs and the
// existing assertions remain valid.
func TestJoinGenres(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		// TC8: Empty slice
		{
			name:     "empty slice",
			input:    []string{},
			expected: "",
		},
		// TC9: All empty/whitespace strings
		{
			name:     "all empty strings",
			input:    []string{"", "  ", ""},
			expected: "",
		},
		// TC10: Special characters preserved
		{
			name:     "special characters preserved",
			input:    []string{"Science Fiction & Fantasy"},
			expected: "Science Fiction & Fantasy",
		},
		// TC11: Exactly 5 genres — all joined with "; "
		{
			name:     "exactly 5 genres",
			input:    []string{"Fiction", "Adventure", "Romance", "Mystery", "Thriller"},
			expected: "Fiction; Adventure; Romance; Mystery; Thriller",
		},
		// TC12: Case-insensitive dedup keeps first occurrence casing
		{
			name:     "case-insensitive dedup",
			input:    []string{"Sci-Fi", "sci-fi", "SCI-FI"},
			expected: "Sci-Fi",
		},
		// TC4 (adapter level): Duplicate genres mixed with unique
		{
			name:     "duplicates with unique entries",
			input:    []string{"Fiction", "fiction", "Fantasy", "Fiction"},
			expected: "Fiction; Fantasy",
		},
		// TC5 (adapter level): Whitespace-only entries filtered
		{
			name:     "whitespace entries filtered",
			input:    []string{"Fiction", "  ", "", "Adventure"},
			expected: "Fiction; Adventure",
		},
		// TC6 (adapter level): More than 5 genres truncated to first 5
		{
			name:     "more than 5 genres truncated",
			input:    []string{"Genre1", "Genre2", "Genre3", "Genre4", "Genre5", "Genre6", "Genre7", "Genre8"},
			expected: "Genre1; Genre2; Genre3; Genre4; Genre5",
		},
		// TC7 (adapter level): Single genre — no trailing separator
		{
			name:     "single genre no trailing separator",
			input:    []string{"Audiobook"},
			expected: "Audiobook",
		},
		// Additional: whitespace trimmed from entries
		{
			name:     "whitespace trimmed from entries",
			input:    []string{"  Fiction  ", " Adventure "},
			expected: "Fiction; Adventure",
		},
		// Additional: genres with slashes and parens preserved
		{
			name:     "genres with special chars preserved",
			input:    []string{"Non-Fiction/Reference", "History (Ancient)"},
			expected: "Non-Fiction/Reference; History (Ancient)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use "en" so that unknown strings (plain English) pass through unchanged.
			result := joinGenres(tt.input, "en")
			if result != tt.expected {
				t.Errorf("joinGenres(%v, \"en\") = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestJoinGenres_NilSlice ensures joinGenres handles a nil input gracefully.
func TestJoinGenres_NilSlice(t *testing.T) {
	result := joinGenres(nil, "en")
	if result != "" {
		t.Errorf("joinGenres(nil, \"en\") = %q, want %q", result, "")
	}
}

// TestJoinGenres_Exactly5AfterDedup verifies that dedup happens before the 5-limit.
// If there are 7 inputs but 3 are duplicates, the cap applies after dedup.
func TestJoinGenres_Exactly5AfterDedup(t *testing.T) {
	// 8 inputs, 3 are case-insensitive dups of earlier entries — yields 5 unique
	input := []string{
		"Action",
		"action",   // dup of Action
		"Drama",
		"DRAMA",    // dup of Drama
		"Comedy",
		"Thriller",
		"Horror",
		"Romance",
	}
	// After dedup: Action, Drama, Comedy, Thriller, Horror, Romance (6 unique)
	// After cap 5: Action, Drama, Comedy, Thriller, Horror
	expected := "Action; Drama; Comedy; Thriller; Horror"
	result := joinGenres(input, "en")
	if result != expected {
		t.Errorf("joinGenres dedup+cap: got %q, want %q", result, expected)
	}
}

// TestParseBook_GenreField verifies that the Book.Genre field exists on the struct
// (TC1-TC3 integration is an ebook-format-level test; here we verify the field is present).
func TestBook_GenreField(t *testing.T) {
	book := &Book{
		Title:  "Test",
		Author: "Author",
		Genre:  "Fiction; Adventure",
	}
	if book.Genre != "Fiction; Adventure" {
		t.Errorf("Book.Genre field not available or not set correctly: got %q", book.Genre)
	}
}
