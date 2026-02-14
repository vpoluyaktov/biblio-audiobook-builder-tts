package normalize

import (
	"testing"
)

func TestEnglishCardinal(t *testing.T) {
	conv := &EnglishConverter{}
	ctx := Context{Form: Cardinal}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "zero"},
		{1, "one"},
		{5, "five"},
		{10, "ten"},
		{11, "eleven"},
		{12, "twelve"},
		{13, "thirteen"},
		{19, "nineteen"},
		{20, "twenty"},
		{21, "twenty-one"},
		{42, "forty-two"},
		{99, "ninety-nine"},
		{100, "one hundred"},
		{101, "one hundred one"},
		{110, "one hundred ten"},
		{111, "one hundred eleven"},
		{123, "one hundred twenty-three"},
		{200, "two hundred"},
		{999, "nine hundred ninety-nine"},
		{1000, "one thousand"},
		{1001, "one thousand one"},
		{1234, "one thousand two hundred thirty-four"},
		{10000, "ten thousand"},
		{12345, "twelve thousand three hundred forty-five"},
		{100000, "one hundred thousand"},
		{123456, "one hundred twenty-three thousand four hundred fifty-six"},
		{1000000, "one million"},
		{1000001, "one million one"},
		{1234567, "one million two hundred thirty-four thousand five hundred sixty-seven"},
		{1000000000, "one billion"},
		{1000000000000, "one trillion"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Cardinal) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestEnglishNormalizeAbbreviations(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "USSR abbreviation",
			input:    "USSR collapsed in 1991",
			expected: "U ES ES AR collapsed in 1991",
		},
		{
			name:     "mixed text",
			input:    "The CIA and FBI shared reports",
			expected: "The CEE I A and EF BEE I shared reports",
		},
		{
			name:     "single uppercase letter unchanged",
			input:    "Plan A was selected",
			expected: "Plan A was selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.NormalizeAbbreviations(tt.input, "en")
			if result != tt.expected {
				t.Errorf("NormalizeAbbreviations(%q, en) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEnglishOrdinal(t *testing.T) {
	conv := &EnglishConverter{}
	ctx := Context{Form: Ordinal}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "zeroth"},
		{1, "first"},
		{2, "second"},
		{3, "third"},
		{4, "fourth"},
		{5, "fifth"},
		{9, "ninth"},
		{10, "tenth"},
		{11, "eleventh"},
		{12, "twelfth"},
		{13, "thirteenth"},
		{19, "nineteenth"},
		{20, "twentieth"},
		{21, "twenty-first"},
		{22, "twenty-second"},
		{23, "twenty-third"},
		{30, "thirtieth"},
		{42, "forty-second"},
		{99, "ninety-ninth"},
		{100, "one hundredth"},
		{101, "one hundred first"},
		{110, "one hundred tenth"},
		{111, "one hundred eleventh"},
		{123, "one hundred twenty-third"},
		{200, "two hundredth"},
		{1000, "one thousandth"},
		{1001, "one thousand first"},
		{1234, "one thousand two hundred thirty-fourth"},
		{1000000, "one millionth"},
		{1000001, "one million first"},
		{1000000000, "one billionth"},
		{1000000000000, "one trillionth"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Ordinal) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestEnglishNegative(t *testing.T) {
	conv := &EnglishConverter{}

	tests := []struct {
		n        int64
		form     Form
		expected string
	}{
		{-1, Cardinal, "minus one"},
		{-42, Cardinal, "minus forty-two"},
		{-1, Ordinal, "minus first"},
		{-42, Ordinal, "minus forty-second"},
	}

	for _, tt := range tests {
		ctx := Context{Form: tt.form}
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, %v) = %q, want %q", tt.n, tt.form, result, tt.expected)
			}
		})
	}
}

func TestEnglishConverterInterface(t *testing.T) {
	conv := &EnglishConverter{}

	if conv.LanguageCode() != "en" {
		t.Errorf("LanguageCode() = %q, want %q", conv.LanguageCode(), "en")
	}

	if conv.LanguageName() != "English" {
		t.Errorf("LanguageName() = %q, want %q", conv.LanguageName(), "English")
	}

	if conv.SupportsContext() {
		t.Error("SupportsContext() = true, want false")
	}
}

func TestEnglishRegistration(t *testing.T) {
	conv := Get("en")
	if conv == nil {
		t.Fatal("English converter not registered")
	}

	if conv.LanguageCode() != "en" {
		t.Errorf("Registered converter LanguageCode() = %q, want %q", conv.LanguageCode(), "en")
	}
}

func BenchmarkEnglishCardinal(b *testing.B) {
	conv := &EnglishConverter{}
	ctx := Context{Form: Cardinal}

	b.Run("small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conv.ToWords(42, ctx)
		}
	})

	b.Run("medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conv.ToWords(123456, ctx)
		}
	})

	b.Run("large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conv.ToWords(999999999999, ctx)
		}
	})
}

func BenchmarkEnglishOrdinal(b *testing.B) {
	conv := &EnglishConverter{}
	ctx := Context{Form: Ordinal}

	b.Run("small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conv.ToWords(42, ctx)
		}
	})

	b.Run("medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conv.ToWords(123456, ctx)
		}
	})

	b.Run("large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conv.ToWords(999999999999, ctx)
		}
	})
}

func TestEnglishRomanNumerals(t *testing.T) {
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

func TestEnglishFindRomanNumerals(t *testing.T) {
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
			name:  "Multiple Roman numerals",
			input: "Part I and Part II",
			expected: []RomanMatch{
				{Start: 5, End: 6, Roman: "I", Value: 1, WordBefore: "Part", WordAfter: "and"},
				{Start: 16, End: 18, Roman: "II", Value: 2, WordBefore: "Part", WordAfter: ""},
			},
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
			}
		})
	}
}

func TestEnglishDetermineRomanPosition(t *testing.T) {
	nounDB := NewNounDatabase()

	tests := []struct {
		name         string
		match        RomanMatch
		expectedPos  RomanPosition
		expectedNoun bool
	}{
		{
			name:         "Part I - after noun",
			match:        RomanMatch{Roman: "I", Value: 1, WordBefore: "Part", WordAfter: ""},
			expectedPos:  RomanAfterNoun,
			expectedNoun: true,
		},
		{
			name:         "I Chapter - before noun",
			match:        RomanMatch{Roman: "I", Value: 1, WordBefore: "", WordAfter: "Chapter"},
			expectedPos:  RomanBeforeNoun,
			expectedNoun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nounInfo := DetermineRomanPosition(tt.match, "en", nounDB)
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
