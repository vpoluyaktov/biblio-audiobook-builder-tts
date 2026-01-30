package normalize

import (
	"testing"
)

func TestParseRoman(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"I", 1},
		{"II", 2},
		{"III", 3},
		{"IV", 4},
		{"V", 5},
		{"VI", 6},
		{"VII", 7},
		{"VIII", 8},
		{"IX", 9},
		{"X", 10},
		{"XI", 11},
		{"XII", 12},
		{"XIV", 14},
		{"XV", 15},
		{"XIX", 19},
		{"XX", 20},
		{"XXI", 21},
		{"XXX", 30},
		{"XL", 40},
		{"L", 50},
		{"LX", 60},
		{"XC", 90},
		{"C", 100},
		{"CD", 400},
		{"D", 500},
		{"CM", 900},
		{"M", 1000},
		{"MCMXCIX", 1999},
		{"MMXXIV", 2024},
		{"i", 1},   // lowercase
		{"iv", 4},  // lowercase
		{"", 0},    // empty
		{"ABC", 0}, // invalid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseRoman(tt.input)
			if result != tt.expected {
				t.Errorf("ParseRoman(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidRoman(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"I", true},
		{"II", true},
		{"III", true},
		{"IV", true},
		{"V", true},
		{"IX", true},
		{"X", true},
		{"L", true},
		{"C", true},
		{"D", true},
		{"M", true},
		{"MCMXCIX", true},
		{"", false},
		{"ABC", false},
		{"IIII", false}, // invalid: 4 consecutive I's
		// Note: VV technically parses to 10 but is non-standard; we allow it for simplicity
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsValidRoman(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidRoman(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindRomanNumerals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []RomanMatch
	}{
		{
			name:  "Part I",
			input: "Part I",
			expected: []RomanMatch{
				{Start: 5, End: 6, Roman: "I", Value: 1, WordBefore: "Part", WordAfter: ""},
			},
		},
		{
			name:  "Chapter VII",
			input: "Chapter VII",
			expected: []RomanMatch{
				{Start: 8, End: 11, Roman: "VII", Value: 7, WordBefore: "Chapter", WordAfter: ""},
			},
		},
		{
			name:  "I Глава (Roman before noun)",
			input: "I Глава",
			expected: []RomanMatch{
				{Start: 0, End: 1, Roman: "I", Value: 1, WordBefore: "", WordAfter: "Глава"},
			},
		},
		{
			name:  "Глава III",
			input: "Глава III",
			expected: []RomanMatch{
				{Start: 11, End: 14, Roman: "III", Value: 3, WordBefore: "Глава", WordAfter: ""},
			},
		},
		{
			name:  "Multiple Roman numerals",
			input: "Part I and Part II",
			expected: []RomanMatch{
				{Start: 5, End: 6, Roman: "I", Value: 1, WordBefore: "Part", WordAfter: "and"},
				{Start: 16, End: 18, Roman: "II", Value: 2, WordBefore: "Part", WordAfter: ""},
			},
		},
		{
			name:  "Standalone I without context (found but filtered later)",
			input: "I am here",
			expected: []RomanMatch{
				{Start: 0, End: 1, Roman: "I", Value: 1, WordBefore: "", WordAfter: "am"},
			}, // FindRomanNumerals returns it, but FilterRomanNumerals will remove it
		},
		{
			name:     "No Roman numerals",
			input:    "Hello world",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindRomanNumerals(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("FindRomanNumerals(%q) returned %d matches, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, match := range result {
				exp := tt.expected[i]
				if match.Roman != exp.Roman || match.Value != exp.Value {
					t.Errorf("FindRomanNumerals(%q)[%d] = {Roman: %q, Value: %d}, want {Roman: %q, Value: %d}",
						tt.input, i, match.Roman, match.Value, exp.Roman, exp.Value)
				}
				if match.WordBefore != exp.WordBefore || match.WordAfter != exp.WordAfter {
					t.Errorf("FindRomanNumerals(%q)[%d] context = {Before: %q, After: %q}, want {Before: %q, After: %q}",
						tt.input, i, match.WordBefore, match.WordAfter, exp.WordBefore, exp.WordAfter)
				}
			}
		})
	}
}

func TestProcessRomanNumeralsEnglish(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Part I - cardinal",
			input:    "Part I",
			expected: "Part one",
		},
		{
			name:     "Part IV - cardinal",
			input:    "Part IV",
			expected: "Part four",
		},
		{
			name:     "Chapter VII - cardinal",
			input:    "Chapter VII",
			expected: "Chapter seven",
		},
		{
			name:     "Chapter X - cardinal",
			input:    "Chapter X",
			expected: "Chapter ten",
		},
		{
			name:     "I Chapter - ordinal",
			input:    "I Chapter",
			expected: "first Chapter",
		},
		{
			name:     "III Part - ordinal",
			input:    "III Part",
			expected: "third Part",
		},
		{
			name:     "Multiple parts",
			input:    "Part I, Part II, Part III",
			expected: "Part one, Part two, Part three",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Process(tt.input, "en")
			if result != tt.expected {
				t.Errorf("Process(%q, en) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessRomanNumeralsRussian(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Часть I - cardinal feminine",
			input:    "Часть I",
			expected: "Часть одна",
		},
		{
			name:     "Часть II - cardinal feminine",
			input:    "Часть II",
			expected: "Часть две",
		},
		{
			name:     "Глава III - cardinal feminine",
			input:    "Глава III",
			expected: "Глава три",
		},
		{
			name:     "Глава IX - cardinal feminine",
			input:    "Глава IX",
			expected: "Глава девять",
		},
		{
			name:     "Том V - cardinal masculine",
			input:    "Том V",
			expected: "Том пять",
		},
		{
			name:     "I Глава - ordinal feminine",
			input:    "I Глава",
			expected: "первая Глава",
		},
		{
			name:     "II Часть - ordinal feminine",
			input:    "II Часть",
			expected: "вторая Часть",
		},
		{
			name:     "III Том - ordinal masculine",
			input:    "III Том",
			expected: "третий Том",
		},
		{
			name:     "IX Глава - ordinal feminine",
			input:    "IX Глава",
			expected: "девятая Глава",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Process(%q, ru) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetermineRomanPosition(t *testing.T) {
	nounDB := NewNounDatabase()

	tests := []struct {
		name         string
		match        RomanMatch
		lang         string
		expectedPos  RomanPosition
		expectedNoun bool
	}{
		{
			name:         "Part I - after noun",
			match:        RomanMatch{Roman: "I", Value: 1, WordBefore: "Part", WordAfter: ""},
			lang:         "en",
			expectedPos:  RomanAfterNoun,
			expectedNoun: true,
		},
		{
			name:         "I Chapter - before noun",
			match:        RomanMatch{Roman: "I", Value: 1, WordBefore: "", WordAfter: "Chapter"},
			lang:         "en",
			expectedPos:  RomanBeforeNoun,
			expectedNoun: true,
		},
		{
			name:         "Глава III - after noun",
			match:        RomanMatch{Roman: "III", Value: 3, WordBefore: "Глава", WordAfter: ""},
			lang:         "ru",
			expectedPos:  RomanAfterNoun,
			expectedNoun: true,
		},
		{
			name:         "I Глава - before noun",
			match:        RomanMatch{Roman: "I", Value: 1, WordBefore: "", WordAfter: "Глава"},
			lang:         "ru",
			expectedPos:  RomanBeforeNoun,
			expectedNoun: true,
		},
		{
			name:         "No context",
			match:        RomanMatch{Roman: "V", Value: 5, WordBefore: "", WordAfter: ""},
			lang:         "en",
			expectedPos:  RomanAlone,
			expectedNoun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nounInfo := DetermineRomanPosition(tt.match, tt.lang, nounDB)
			if pos != tt.expectedPos {
				t.Errorf("DetermineRomanPosition() position = %v, want %v", pos, tt.expectedPos)
			}
			hasNoun := nounInfo != nil
			if hasNoun != tt.expectedNoun {
				t.Errorf("DetermineRomanPosition() hasNoun = %v, want %v", hasNoun, tt.expectedNoun)
			}
		})
	}
}
