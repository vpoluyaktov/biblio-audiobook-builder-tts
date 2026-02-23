package normalize

import (
	"testing"
)

// TestRussianGenderDetection tests gender detection for Russian nouns after numbers
func TestRussianGenderDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Masculine nouns with genitive singular ending in -а
		{
			name:     "52 children masculine",
			input:    "52 ребёнка",
			expected: "пятьдесят два ребёнка",
		},
		{
			name:     "2 children masculine",
			input:    "2 ребёнка",
			expected: "два ребёнка",
		},
		{
			name:     "22 children masculine",
			input:    "22 ребёнка",
			expected: "двадцать два ребёнка",
		},
		{
			name:     "32 children masculine",
			input:    "32 ребёнка",
			expected: "тридцать два ребёнка",
		},
		{
			name:     "42 children masculine",
			input:    "42 ребёнка",
			expected: "сорок два ребёнка",
		},
		{
			name:     "102 children masculine",
			input:    "102 ребёнка",
			expected: "сто два ребёнка",
		},
		{
			name:     "5 children genitive plural",
			input:    "5 детей",
			expected: "пять детей",
		},
		{
			name:     "10 children genitive plural",
			input:    "10 детей",
			expected: "десять детей",
		},
		{
			name:     "52 children genitive plural",
			input:    "52 детей",
			expected: "пятьдесят два детей",
		},

		// Other masculine nouns in genitive singular
		{
			name:     "2 tables masculine",
			input:    "2 стола",
			expected: "два стола",
		},
		{
			name:     "3 houses masculine",
			input:    "3 дома",
			expected: "три дома",
		},
		{
			name:     "4 cities masculine",
			input:    "4 города",
			expected: "четыре города",
		},
		{
			name:     "22 rubles masculine",
			input:    "22 рубля",
			expected: "двадцать два рубля",
		},

		// Feminine nouns - should use две/три/четыре
		{
			name:     "2 books feminine",
			input:    "2 книги",
			expected: "две книги",
		},
		{
			name:     "3 cats feminine",
			input:    "3 кошки",
			expected: "три кошки",
		},
		{
			name:     "4 girls feminine",
			input:    "4 девочки",
			expected: "четыре девочки",
		},
		{
			name:     "22 kopecks feminine",
			input:    "22 копейки",
			expected: "двадцать две копейки",
		},
		{
			name:     "32 minutes feminine",
			input:    "32 минуты",
			expected: "тридцать две минуты",
		},
		{
			name:     "42 weeks feminine",
			input:    "42 недели",
			expected: "сорок две недели",
		},

		// Neuter nouns - should use два/три/четыре
		{
			name:     "2 windows neuter",
			input:    "2 окна",
			expected: "два окна",
		},
		{
			name:     "3 seas neuter",
			input:    "3 моря",
			expected: "три моря",
		},
		{
			name:     "4 fields neuter",
			input:    "4 поля",
			expected: "четыре поля",
		},

		// Numbers 5+ use genitive plural (same form for all genders)
		{
			name:     "5 books",
			input:    "5 книг",
			expected: "пять книг",
		},
		{
			name:     "10 rubles",
			input:    "10 рублей",
			expected: "десять рублей",
		},
		{
			name:     "100 meters",
			input:    "100 метров",
			expected: "сто метров",
		},

		// Edge cases with 1
		{
			name:     "1 child masculine",
			input:    "1 ребёнок",
			expected: "один ребёнок",
		},
		{
			name:     "1 book feminine",
			input:    "1 книга",
			expected: "одна книга",
		},
		{
			name:     "1 window neuter",
			input:    "1 окно",
			expected: "одно окно",
		},
		{
			name:     "21 child masculine",
			input:    "21 ребёнок",
			expected: "двадцать один ребёнок",
		},
		{
			name:     "21 book feminine",
			input:    "21 книга",
			expected: "двадцать одна книга",
		},

		// Numbers 11-19 use genitive plural
		{
			name:     "11 children",
			input:    "11 детей",
			expected: "одиннадцать детей",
		},
		{
			name:     "15 books",
			input:    "15 книг",
			expected: "пятнадцать книг",
		},
		{
			name:     "19 rubles",
			input:    "19 рублей",
			expected: "девятнадцать рублей",
		},
	}

	p := NewProcessor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Process(%q, ru) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGenderFromNounEnding tests the heuristic gender detection
// Note: The heuristic is conservative and defaults to feminine for -а endings.
// Masculine genitive singular forms should be added to the noun database (ru.csv).
func TestGenderFromNounEnding(t *testing.T) {
	tests := []struct {
		noun     string
		expected Gender
	}{
		// Masculine genitive singular ending in -а - these should be in noun database
		// The heuristic defaults to feminine for safety
		{"ребёнка", Feminine}, // child (genitive singular) - will be masculine via database
		{"стола", Feminine},   // table (genitive singular) - will be masculine via database
		{"дома", Feminine},    // house (genitive singular) - will be masculine via database
		{"города", Feminine},  // city (genitive singular) - will be masculine via database
		{"рубля", Feminine},   // ruble (genitive singular) - will be masculine via database

		// Feminine nominative ending in -ка, -га, -ха, -ча, -ща, -жа, -ша
		{"кошка", Feminine}, // cat
		{"книга", Feminine}, // book
		{"муха", Feminine},  // fly
		{"дача", Feminine},  // dacha
		{"роща", Feminine},  // grove
		{"лужа", Feminine},  // puddle
		{"каша", Feminine},  // porridge

		// Feminine genitive singular ending in -и, -ы
		{"книги", Feminine},  // book (genitive singular)
		{"собаки", Feminine}, // dog (genitive singular)
		{"воды", Feminine},   // water (genitive singular)
		{"горы", Feminine},   // mountain (genitive singular)

		// Neuter genitive singular
		{"окна", Neuter}, // window (genitive singular)
		{"моря", Neuter}, // sea (genitive singular)
		{"поля", Neuter}, // field (genitive singular)

		// Masculine genitive plural ending in -ов, -ев, -ей
		{"столов", Masculine}, // tables (genitive plural)
		{"музеев", Masculine}, // museums (genitive plural)
		{"врачей", Masculine}, // doctors (genitive plural)

		// Consonant endings (masculine nominative)
		{"стол", Masculine},  // table
		{"дом", Masculine},   // house
		{"город", Masculine}, // city
		{"рубль", Masculine}, // ruble

		// Feminine nominative ending in -а, -я (not after specific consonants)
		{"вода", Feminine},  // water
		{"земля", Feminine}, // earth
		{"идея", Feminine},  // idea

		// Neuter nominative ending in -о, -е, -ё
		{"окно", Neuter},  // window
		{"море", Neuter},  // sea
		{"ружьё", Neuter}, // gun
	}

	for _, tt := range tests {
		t.Run(tt.noun, func(t *testing.T) {
			result := GenderFromNounEnding(tt.noun)
			if result != tt.expected {
				t.Errorf("GenderFromNounEnding(%q) = %v, want %v", tt.noun, result, tt.expected)
			}
		})
	}
}

// TestRussianNumbersWithContext tests Russian number conversion with explicit context
func TestRussianNumbersWithContext(t *testing.T) {
	tests := []struct {
		name     string
		number   int64
		ctx      Context
		expected string
	}{
		// Cardinal numbers with different genders
		{
			name:     "1 masculine",
			number:   1,
			ctx:      Context{Form: Cardinal, Gender: Masculine},
			expected: "один",
		},
		{
			name:     "1 feminine",
			number:   1,
			ctx:      Context{Form: Cardinal, Gender: Feminine},
			expected: "одна",
		},
		{
			name:     "1 neuter",
			number:   1,
			ctx:      Context{Form: Cardinal, Gender: Neuter},
			expected: "одно",
		},
		{
			name:     "2 masculine",
			number:   2,
			ctx:      Context{Form: Cardinal, Gender: Masculine},
			expected: "два",
		},
		{
			name:     "2 feminine",
			number:   2,
			ctx:      Context{Form: Cardinal, Gender: Feminine},
			expected: "две",
		},
		{
			name:     "2 neuter",
			number:   2,
			ctx:      Context{Form: Cardinal, Gender: Neuter},
			expected: "два",
		},
		{
			name:     "52 masculine",
			number:   52,
			ctx:      Context{Form: Cardinal, Gender: Masculine},
			expected: "пятьдесят два",
		},
		{
			name:     "52 feminine",
			number:   52,
			ctx:      Context{Form: Cardinal, Gender: Feminine},
			expected: "пятьдесят две",
		},
		{
			name:     "22 masculine",
			number:   22,
			ctx:      Context{Form: Cardinal, Gender: Masculine},
			expected: "двадцать два",
		},
		{
			name:     "22 feminine",
			number:   22,
			ctx:      Context{Form: Cardinal, Gender: Feminine},
			expected: "двадцать две",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceNumber(tt.number, "ru", tt.ctx)
			if result != tt.expected {
				t.Errorf("ReplaceNumber(%d, ru, %v) = %q, want %q",
					tt.number, tt.ctx, result, tt.expected)
			}
		})
	}
}
