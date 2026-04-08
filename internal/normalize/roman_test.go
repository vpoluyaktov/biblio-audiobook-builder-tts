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
			name:  "Simple Roman numeral II",
			input: "Test II here",
			expected: []RomanMatch{
				{Start: 5, End: 7, Roman: "II", Value: 2, WordBefore: "Test", WordAfter: "here"},
			},
		},
		{
			name:  "Roman numeral at start",
			input: "III test",
			expected: []RomanMatch{
				{Start: 0, End: 3, Roman: "III", Value: 3, WordBefore: "", WordAfter: "test"},
			},
		},
		{
			name:  "Roman numeral at end",
			input: "test IV",
			expected: []RomanMatch{
				{Start: 5, End: 7, Roman: "IV", Value: 4, WordBefore: "test", WordAfter: ""},
			},
		},
		{
			name:  "Multiple Roman numerals",
			input: "test V and VI",
			expected: []RomanMatch{
				{Start: 5, End: 6, Roman: "V", Value: 5, WordBefore: "test", WordAfter: "and"},
				{Start: 11, End: 13, Roman: "VI", Value: 6, WordBefore: "and", WordAfter: ""},
			},
		},
		{
			name:  "Standalone I (found but filtered later by noun database)",
			input: "I am here",
			expected: []RomanMatch{
				{Start: 0, End: 1, Roman: "I", Value: 1, WordBefore: "", WordAfter: "am"},
			},
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

func TestProcessRomanNumeralsRussian(t *testing.T) {
	nounDB := NewNounDatabase()
	converter := GetOrDefault("ru")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Глава III - ordinal feminine",
			input:    "Глава III",
			expected: "Глава третья",
		},
		{
			name:     "Часть I - ordinal feminine",
			input:    "Часть I",
			expected: "Часть первая",
		},
		{
			name:     "Том IV - ordinal masculine",
			input:    "Том IV",
			expected: "Том четвёртый",
		},
		{
			name:     "Раздел II - ordinal masculine",
			input:    "Раздел II",
			expected: "Раздел второй",
		},
		{
			name:     "Действие V - ordinal neuter",
			input:    "Действие V",
			expected: "Действие пятое",
		},
		{
			name:     "Roman before noun: III Глава - ordinal feminine",
			input:    "III Глава",
			expected: "третья Глава",
		},
		{
			name:     "No context noun - not converted",
			input:    "Текст III здесь",
			expected: "Текст III здесь",
		},
		{
			name:     "Multiple Roman numerals with context",
			input:    "Часть I и Глава III",
			expected: "Часть первая и Глава третья",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessRomanNumerals(tt.input, "ru", converter, nounDB)
			if result != tt.expected {
				t.Errorf("ProcessRomanNumerals(%q) = %q, want %q", tt.input, result, tt.expected)
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
			name:         "No context - word before unknown",
			match:        RomanMatch{Roman: "V", Value: 5, WordBefore: "unknown", WordAfter: ""},
			lang:         "en",
			expectedPos:  RomanAlone,
			expectedNoun: false,
		},
		{
			name:         "No context - word after unknown",
			match:        RomanMatch{Roman: "V", Value: 5, WordBefore: "", WordAfter: "unknown"},
			lang:         "en",
			expectedPos:  RomanAlone,
			expectedNoun: false,
		},
		{
			name:         "No context - empty",
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
