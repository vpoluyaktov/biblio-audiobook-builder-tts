package parser

import (
	"testing"
)

// TestIsRussian tests the isRussian language normalization helper.
// Spec: Section 2 "Language Normalization Rules" and Section 4 tests 21-30.
func TestIsRussian(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Standard ISO 639-1
		{name: "bare ru", input: "ru", expected: true},
		// With BCP-47 region code, hyphen separator
		{name: "ru-RU locale", input: "ru-RU", expected: true},
		// With underscore separator
		{name: "ru_RU underscore", input: "ru_RU", expected: true},
		// Uppercase
		{name: "RU uppercase", input: "RU", expected: true},
		// Full English name
		{name: "Russian full name", input: "Russian", expected: true},
		// Cyrillic language name
		{name: "русский cyrillic", input: "русский", expected: true},
		// English
		{name: "en english", input: "en", expected: false},
		// Empty string
		{name: "empty string", input: "", expected: false},
		// German
		{name: "de german", input: "de", expected: false},
		// Romanian — starts with "ru" substring but must NOT match
		// because it is "rum", not "ru" followed by separator or end-of-string.
		{name: "rum romanian not russian", input: "rum", expected: false},
		// Whitespace variants normalise without matching
		{name: "space only", input: "   ", expected: false},
		// Mixed-case full name
		{name: "RUSSIAN all caps", input: "RUSSIAN", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRussian(tt.input)
			if got != tt.expected {
				t.Errorf("isRussian(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestMapGenreName tests MapGenreName(code, language string) string.
// Spec: Section 4 tests 1-12, plus additional edge cases.
func TestMapGenreName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		language string
		expected string
	}{
		// TC1: Basic Russian mapping
		{
			name:     "det_espionage russian basic",
			code:     "det_espionage",
			language: "ru",
			expected: "Шпионский детектив",
		},
		// TC2: Basic English mapping
		{
			name:     "det_espionage english basic",
			code:     "det_espionage",
			language: "en",
			expected: "Espionage",
		},
		// TC3: Locale with region code normalises to Russian
		{
			name:     "det_espionage ru-RU locale",
			code:     "det_espionage",
			language: "ru-RU",
			expected: "Шпионский детектив",
		},
		// TC4: Uppercase language code normalises to Russian
		{
			name:     "det_espionage RU uppercase",
			code:     "det_espionage",
			language: "RU",
			expected: "Шпионский детектив",
		},
		// TC5: Unknown code passes through unchanged (Russian)
		{
			name:     "unknown code passthrough russian",
			code:     "unknown_xyz",
			language: "ru",
			expected: "unknown_xyz",
		},
		// TC6: Unknown code passes through unchanged (English)
		{
			name:     "unknown code passthrough english",
			code:     "unknown_xyz",
			language: "en",
			expected: "unknown_xyz",
		},
		// TC7: Empty language defaults to English
		{
			name:     "empty language defaults to english",
			code:     "det_espionage",
			language: "",
			expected: "Espionage",
		},
		// TC8: Full English name normalisation → Russian
		{
			name:     "Russian full name normalisation",
			code:     "det_espionage",
			language: "Russian",
			expected: "Шпионский детектив",
		},
		// TC9: Cyrillic language name normalisation → Russian
		{
			name:     "русский cyrillic normalisation",
			code:     "det_espionage",
			language: "русский",
			expected: "Шпионский детектив",
		},
		// TC10: Another Russian mapping — sf_fantasy
		{
			name:     "sf_fantasy russian",
			code:     "sf_fantasy",
			language: "ru",
			expected: "Фэнтези",
		},
		// TC11: Another English mapping — prose_classic
		{
			name:     "prose_classic english",
			code:     "prose_classic",
			language: "en",
			expected: "Classic Prose",
		},
		// TC12: Non-Russian, non-English language falls back to English
		{
			name:     "sf_fantasy german falls back to english",
			code:     "sf_fantasy",
			language: "de",
			expected: "Fantasy",
		},
		// Extra: "rum" must not be treated as Russian (Romanian ISO 639-2)
		{
			name:     "rum not russian passthrough",
			code:     "rum",
			language: "rum",
			expected: "rum",
		},
		// Extra: sf_fantasy English
		{
			name:     "sf_fantasy english",
			code:     "sf_fantasy",
			language: "en",
			expected: "Fantasy",
		},
		// Extra: French falls back to English
		{
			name:     "det_espionage french falls back to english",
			code:     "det_espionage",
			language: "fr",
			expected: "Espionage",
		},
		// Extra: ru_RU with underscore separator
		{
			name:     "det_espionage ru_RU underscore",
			code:     "det_espionage",
			language: "ru_RU",
			expected: "Шпионский детектив",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapGenreName(tt.code, tt.language)
			if got != tt.expected {
				t.Errorf("MapGenreName(%q, %q) = %q, want %q", tt.code, tt.language, got, tt.expected)
			}
		})
	}
}

// TestJoinGenresLanguageAware tests the language-aware joinGenres(genres []string, language string) string.
// Spec: Section 4 tests 13-20.
func TestJoinGenresLanguageAware(t *testing.T) {
	tests := []struct {
		name     string
		genres   []string
		language string
		expected string
	}{
		// TC13: Empty genres list → empty string
		{
			name:     "empty genres list",
			genres:   []string{},
			language: "ru",
			expected: "",
		},
		// TC14: Single known genre in Russian — no trailing comma
		{
			name:     "single genre russian",
			genres:   []string{"det_espionage"},
			language: "ru",
			expected: "Шпионский детектив",
		},
		// TC15: Known + unknown mixed in Russian
		{
			name:     "known and unknown mixed russian",
			genres:   []string{"det_espionage", "unknown_xyz"},
			language: "ru",
			expected: "Шпионский детектив; unknown_xyz",
		},
		// TC16: Duplicate codes deduplicated (English)
		{
			name:     "duplicate codes deduplicated english",
			genres:   []string{"sf_fantasy", "sf_fantasy"},
			language: "en",
			expected: "Fantasy",
		},
		// TC17: Max 5 genres enforced — 6 inputs yields 5 outputs (unknown codes pass through)
		{
			name:     "max 5 genres enforced",
			genres:   []string{"a", "b", "c", "d", "e", "f"},
			language: "en",
			expected: "a; b; c; d; e",
		},
		// TC18: Case-insensitive dedup on raw code
		{
			name:     "case insensitive dedup on raw code",
			genres:   []string{"det_espionage", "DET_ESPIONAGE"},
			language: "en",
			expected: "Espionage",
		},
		// TC19: Whitespace trimmed before lookup
		{
			name:     "whitespace trimmed before lookup",
			genres:   []string{"  det_espionage  "},
			language: "en",
			expected: "Espionage",
		},
		// TC20: Empty and blank genres skipped
		{
			name:     "empty and blank genres skipped",
			genres:   []string{"", " ", "det_espionage"},
			language: "ru",
			expected: "Шпионский детектив",
		},
		// Extra: nil slice returns empty string
		{
			name:     "nil slice returns empty string",
			genres:   nil,
			language: "ru",
			expected: "",
		},
		// Extra: Multiple known genres in Russian
		{
			name:     "multiple known genres russian",
			genres:   []string{"det_espionage", "sf_fantasy"},
			language: "ru",
			expected: "Шпионский детектив; Фэнтези",
		},
		// Extra: Multiple known genres in English
		{
			name:     "multiple known genres english",
			genres:   []string{"det_espionage", "sf_fantasy"},
			language: "en",
			expected: "Espionage; Fantasy",
		},
		// Extra: 5 or more known FB2 codes — exactly 5 selected from longer list
		{
			name:     "exactly 5 from 6 known codes english",
			genres:   []string{"det_espionage", "sf_fantasy", "prose_classic", "poetry", "thriller", "humor"},
			language: "en",
			expected: "Espionage; Fantasy; Classic Prose; Poetry; Thriller",
		},
		// Extra: Language fallback — French uses English names
		{
			name:     "french falls back to english names",
			genres:   []string{"sf_fantasy"},
			language: "fr",
			expected: "Fantasy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinGenres(tt.genres, tt.language)
			if got != tt.expected {
				t.Errorf("joinGenres(%v, %q) = %q, want %q", tt.genres, tt.language, got, tt.expected)
			}
		})
	}
}
