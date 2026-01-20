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
