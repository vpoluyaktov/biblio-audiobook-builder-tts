package normalize

import (
	"testing"
)

func TestProcessorEnglish(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple number",
			input:    "I have 5 apples",
			expected: "I have five apples",
		},
		{
			name:     "chapter ordinal",
			input:    "Chapter 5",
			expected: "Chapter fifth",
		},
		{
			name:     "page ordinal",
			input:    "Page 42",
			expected: "Page forty-second",
		},
		{
			name:     "dollars cardinal",
			input:    "100 dollars",
			expected: "one hundred dollars",
		},
		{
			name:     "years cardinal",
			input:    "25 years old",
			expected: "twenty-five years old",
		},
		{
			name:     "multiple numbers",
			input:    "Chapter 3 has 15 pages",
			expected: "Chapter third has fifteen pages",
		},
		{
			name:     "no numbers",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "large number",
			input:    "Population is 1000000",
			expected: "Population is one million",
		},
		{
			name:     "negative number",
			input:    "Temperature is -5 degrees",
			expected: "Temperature is minus five degrees",
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

func TestProcessorRussian(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "chapter ordinal feminine",
			input:    "Глава 5",
			expected: "Глава пятая",
		},
		{
			name:     "chapter ordinal feminine 1",
			input:    "Глава 1",
			expected: "Глава первая",
		},
		{
			name:     "chapter ordinal feminine 2",
			input:    "Глава 2",
			expected: "Глава вторая",
		},
		{
			name:     "volume ordinal masculine",
			input:    "Том 3",
			expected: "Том третий",
		},
		{
			name:     "rubles cardinal masculine",
			input:    "100 рублей",
			expected: "сто рублей",
		},
		{
			name:     "page ordinal feminine",
			input:    "Страница 42",
			expected: "Страница сорок вторая",
		},
		{
			name:     "part ordinal feminine",
			input:    "Часть 1",
			expected: "Часть первая",
		},
		{
			name:     "years cardinal",
			input:    "5 лет",
			expected: "пять лет",
		},
		{
			name:     "multiple numbers",
			input:    "Глава 3 содержит 15 страниц",
			expected: "Глава третья содержит пятнадцать страниц",
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

func TestProcessorWithContext(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		lang     string
		ctx      Context
		expected string
	}{
		{
			name:     "english ordinal",
			input:    "Number 5",
			lang:     "en",
			ctx:      Context{Form: Ordinal, Gender: Masculine},
			expected: "Number fifth",
		},
		{
			name:     "english cardinal",
			input:    "Number 5",
			lang:     "en",
			ctx:      Context{Form: Cardinal, Gender: Masculine},
			expected: "Number five",
		},
		{
			name:     "russian ordinal feminine",
			input:    "Номер 5",
			lang:     "ru",
			ctx:      Context{Form: Ordinal, Gender: Feminine},
			expected: "Номер пятая",
		},
		{
			name:     "russian ordinal masculine",
			input:    "Номер 5",
			lang:     "ru",
			ctx:      Context{Form: Ordinal, Gender: Masculine},
			expected: "Номер пятый",
		},
		{
			name:     "russian cardinal feminine",
			input:    "Номер 1",
			lang:     "ru",
			ctx:      Context{Form: Cardinal, Gender: Feminine},
			expected: "Номер одна",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ProcessWithContext(tt.input, tt.lang, tt.ctx)
			if result != tt.expected {
				t.Errorf("ProcessWithContext(%q, %q, %v) = %q, want %q",
					tt.input, tt.lang, tt.ctx, result, tt.expected)
			}
		})
	}
}

func TestNormalizeChapter(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		lang     string
		expected string
	}{
		{
			name:     "english chapter",
			input:    "Chapter 5",
			lang:     "en",
			expected: "Chapter fifth",
		},
		{
			name:     "russian chapter",
			input:    "Глава 5",
			lang:     "ru",
			expected: "Глава пятая",
		},
		{
			name:     "english chapter 1",
			input:    "Chapter 1",
			lang:     "en",
			expected: "Chapter first",
		},
		{
			name:     "russian chapter 1",
			input:    "Глава 1",
			lang:     "ru",
			expected: "Глава первая",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.NormalizeChapter(tt.input, tt.lang)
			if result != tt.expected {
				t.Errorf("NormalizeChapter(%q, %q) = %q, want %q",
					tt.input, tt.lang, result, tt.expected)
			}
		})
	}
}

func TestExtractNumbers(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected []int64
	}{
		{
			name:     "single number",
			input:    "Chapter 5",
			expected: []int64{5},
		},
		{
			name:     "multiple numbers",
			input:    "From 10 to 20",
			expected: []int64{10, 20},
		},
		{
			name:     "no numbers",
			input:    "Hello world",
			expected: []int64{},
		},
		{
			name:     "negative number",
			input:    "Temperature -5",
			expected: []int64{-5},
		},
		{
			name:     "large numbers",
			input:    "1000000 and 999",
			expected: []int64{1000000, 999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ExtractNumbers(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractNumbers(%q) returned %d numbers, want %d",
					tt.input, len(result), len(tt.expected))
				return
			}
			for i, n := range result {
				if n != tt.expected[i] {
					t.Errorf("ExtractNumbers(%q)[%d] = %d, want %d",
						tt.input, i, n, tt.expected[i])
				}
			}
		})
	}
}

func TestHasNumbers(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		input    string
		expected bool
	}{
		{"Chapter 5", true},
		{"Hello world", false},
		{"100 dollars", true},
		{"", false},
		{"abc123def", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := p.HasNumbers(tt.input)
			if result != tt.expected {
				t.Errorf("HasNumbers(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReplaceNumber(t *testing.T) {
	tests := []struct {
		n        int64
		lang     string
		ctx      Context
		expected string
	}{
		{5, "en", Context{Form: Cardinal}, "five"},
		{5, "en", Context{Form: Ordinal}, "fifth"},
		{5, "ru", Context{Form: Cardinal, Gender: Masculine}, "пять"},
		{5, "ru", Context{Form: Ordinal, Gender: Feminine}, "пятая"},
		{1, "ru", Context{Form: Cardinal, Gender: Feminine}, "одна"},
		{2, "ru", Context{Form: Cardinal, Gender: Feminine}, "две"},
	}

	for _, tt := range tests {
		result := ReplaceNumber(tt.n, tt.lang, tt.ctx)
		if result != tt.expected {
			t.Errorf("ReplaceNumber(%d, %q, %v) = %q, want %q",
				tt.n, tt.lang, tt.ctx, result, tt.expected)
		}
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input    string
		lang     string
		expected string
	}{
		{"Chapter 5", "en", "Chapter fifth"},
		{"Глава 5", "ru", "Глава пятая"},
		{"100 dollars", "en", "one hundred dollars"},
	}

	for _, tt := range tests {
		result := NormalizeText(tt.input, tt.lang)
		if result != tt.expected {
			t.Errorf("NormalizeText(%q, %q) = %q, want %q",
				tt.input, tt.lang, result, tt.expected)
		}
	}
}

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello world", []string{"Hello", "world"}},
		{"Chapter 5!", []string{"Chapter", "5", "!"}},
		{"one-two", []string{"one-two"}},
		{"it's", []string{"it's"}},
		{"a, b, c", []string{"a", ",", "b", ",", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SplitIntoWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitIntoWords(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, w := range result {
				if w != tt.expected[i] {
					t.Errorf("SplitIntoWords(%q)[%d] = %q, want %q",
						tt.input, i, w, tt.expected[i])
				}
			}
		})
	}
}

func TestNounDatabaseLookup(t *testing.T) {
	db := NewNounDatabase()

	tests := []struct {
		lang     string
		noun     string
		wantOk   bool
		wantForm Form
		wantGend Gender
	}{
		{"en", "chapter", true, Ordinal, Masculine},
		{"en", "Chapter", true, Ordinal, Masculine},
		{"en", "chapters", true, Ordinal, Masculine},
		{"en", "dollar", true, Cardinal, Masculine},
		{"en", "dollars", true, Cardinal, Masculine},
		{"en", "unknown", false, Cardinal, Masculine},
		{"ru", "глава", true, Ordinal, Feminine},
		{"ru", "Глава", true, Ordinal, Feminine},
		{"ru", "главы", true, Ordinal, Feminine},
		{"ru", "том", true, Ordinal, Masculine},
		{"ru", "рубль", true, Cardinal, Masculine},
		{"ru", "рублей", true, Cardinal, Masculine},
		{"ru", "неизвестно", false, Cardinal, Masculine},
	}

	for _, tt := range tests {
		t.Run(tt.lang+"_"+tt.noun, func(t *testing.T) {
			info, ok := db.Lookup(tt.lang, tt.noun)
			if ok != tt.wantOk {
				t.Errorf("Lookup(%q, %q) ok = %v, want %v", tt.lang, tt.noun, ok, tt.wantOk)
				return
			}
			if ok {
				if info.TriggerForm != tt.wantForm {
					t.Errorf("Lookup(%q, %q).TriggerForm = %v, want %v",
						tt.lang, tt.noun, info.TriggerForm, tt.wantForm)
				}
				if info.Gender != tt.wantGend {
					t.Errorf("Lookup(%q, %q).Gender = %v, want %v",
						tt.lang, tt.noun, info.Gender, tt.wantGend)
				}
			}
		})
	}
}

func BenchmarkProcess(b *testing.B) {
	p := NewProcessor()

	b.Run("english_short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p.Process("Chapter 5", "en")
		}
	})

	b.Run("english_long", func(b *testing.B) {
		text := "Chapter 1 has 25 pages. Chapter 2 has 30 pages. Total: 55 pages."
		for i := 0; i < b.N; i++ {
			p.Process(text, "en")
		}
	})

	b.Run("russian_short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p.Process("Глава 5", "ru")
		}
	})

	b.Run("russian_long", func(b *testing.B) {
		text := "Глава 1 содержит 25 страниц. Глава 2 содержит 30 страниц. Всего: 55 страниц."
		for i := 0; i < b.N; i++ {
			p.Process(text, "ru")
		}
	})
}

func TestRussianOrdinalSuffixes(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "year with -м suffix",
			input:    "В 1996-м году",
			expected: "В одна тысяча девятьсот девяносто шестом году",
		},
		{
			name:     "year with -го suffix",
			input:    "до 1996-го года",
			expected: "до одна тысяча девятьсот девяносто шестого года",
		},
		{
			name:     "ordinal with -й suffix masculine",
			input:    "это был 5-й раз",
			expected: "это был пятый раз",
		},
		{
			name:     "ordinal with -я suffix feminine",
			input:    "это была 5-я попытка",
			expected: "это была пятая попытка",
		},
		{
			name:     "ordinal with -е suffix neuter",
			input:    "это было 5-е место",
			expected: "это было пятое место",
		},
		{
			name:     "multiple ordinal suffixes",
			input:    "с 1990-го по 2000-й год",
			expected: "с одна тысяча девятьсот девяностого по две тысячный год",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Process(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRussianDateFormat(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "day of March",
			input:    "25 марта",
			expected: "двадцать пятое марта",
		},
		{
			name:     "day of January",
			input:    "1 января",
			expected: "первое января",
		},
		{
			name:     "day of December",
			input:    "31 декабря",
			expected: "тридцать первое декабря",
		},
		{
			name:     "full date with year",
			input:    "25 марта 1996 года",
			expected: "двадцать пятое марта одна тысяча девятьсот девяносто шестого года",
		},
		{
			name:     "date in sentence",
			input:    "Это случилось 10 мая",
			expected: "Это случилось десятое мая",
		},
		{
			name:     "year nominative",
			input:    "1996 год",
			expected: "одна тысяча девятьсот девяносто шестой год",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Process(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
