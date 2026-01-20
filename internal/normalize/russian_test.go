package normalize

import (
	"testing"
)

func TestRussianCardinalMasculine(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Cardinal, Gender: Masculine}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "ноль"},
		{1, "один"},
		{2, "два"},
		{3, "три"},
		{4, "четыре"},
		{5, "пять"},
		{9, "девять"},
		{10, "десять"},
		{11, "одиннадцать"},
		{12, "двенадцать"},
		{13, "тринадцать"},
		{19, "девятнадцать"},
		{20, "двадцать"},
		{21, "двадцать один"},
		{22, "двадцать два"},
		{30, "тридцать"},
		{40, "сорок"},
		{42, "сорок два"},
		{50, "пятьдесят"},
		{99, "девяносто девять"},
		{100, "сто"},
		{101, "сто один"},
		{111, "сто одиннадцать"},
		{123, "сто двадцать три"},
		{200, "двести"},
		{300, "триста"},
		{400, "четыреста"},
		{500, "пятьсот"},
		{999, "девятьсот девяносто девять"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Cardinal, Masculine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianCardinalFeminine(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Cardinal, Gender: Feminine}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "ноль"},
		{1, "одна"},
		{2, "две"},
		{3, "три"},
		{21, "двадцать одна"},
		{22, "двадцать две"},
		{31, "тридцать одна"},
		{42, "сорок две"},
		{101, "сто одна"},
		{102, "сто две"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Cardinal, Feminine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianCardinalNeuter(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Cardinal, Gender: Neuter}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "ноль"},
		{1, "одно"},
		{2, "два"},
		{21, "двадцать одно"},
		{101, "сто одно"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Cardinal, Neuter) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianCardinalThousands(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Cardinal, Gender: Masculine}

	tests := []struct {
		n        int64
		expected string
	}{
		{1000, "одна тысяча"},
		{2000, "две тысячи"},
		{3000, "три тысячи"},
		{4000, "четыре тысячи"},
		{5000, "пять тысяч"},
		{10000, "десять тысяч"},
		{11000, "одиннадцать тысяч"},
		{21000, "двадцать одна тысяча"},
		{22000, "двадцать две тысячи"},
		{100000, "сто тысяч"},
		{123456, "сто двадцать три тысячи четыреста пятьдесят шесть"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Cardinal, Masculine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianCardinalMillions(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Cardinal, Gender: Masculine}

	tests := []struct {
		n        int64
		expected string
	}{
		{1000000, "один миллион"},
		{2000000, "два миллиона"},
		{5000000, "пять миллионов"},
		{11000000, "одиннадцать миллионов"},
		{21000000, "двадцать один миллион"},
		{1000000000, "один миллиард"},
		{2000000000, "два миллиарда"},
		{5000000000, "пять миллиардов"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Cardinal, Masculine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianOrdinalMasculine(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Ordinal, Gender: Masculine}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "нулевой"},
		{1, "первый"},
		{2, "второй"},
		{3, "третий"},
		{4, "четвёртый"},
		{5, "пятый"},
		{6, "шестой"},
		{7, "седьмой"},
		{8, "восьмой"},
		{9, "девятый"},
		{10, "десятый"},
		{11, "одиннадцатый"},
		{12, "двенадцатый"},
		{19, "девятнадцатый"},
		{20, "двадцатый"},
		{21, "двадцать первый"},
		{22, "двадцать второй"},
		{30, "тридцатый"},
		{40, "сороковой"},
		{42, "сорок второй"},
		{50, "пятидесятый"},
		{99, "девяносто девятый"},
		{100, "сотый"},
		{101, "сто первый"},
		{200, "двухсотый"},
		{123, "сто двадцать третий"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Ordinal, Masculine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianOrdinalFeminine(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Ordinal, Gender: Feminine}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "нулевая"},
		{1, "первая"},
		{2, "вторая"},
		{3, "третья"},
		{4, "четвёртая"},
		{5, "пятая"},
		{10, "десятая"},
		{11, "одиннадцатая"},
		{20, "двадцатая"},
		{21, "двадцать первая"},
		{22, "двадцать вторая"},
		{100, "сотая"},
		{101, "сто первая"},
		{123, "сто двадцать третья"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Ordinal, Feminine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianOrdinalNeuter(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Ordinal, Gender: Neuter}

	tests := []struct {
		n        int64
		expected string
	}{
		{0, "нулевое"},
		{1, "первое"},
		{2, "второе"},
		{3, "третье"},
		{10, "десятое"},
		{21, "двадцать первое"},
		{100, "сотое"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Ordinal, Neuter) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianOrdinalThousands(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Ordinal, Gender: Masculine}

	tests := []struct {
		n        int64
		expected string
	}{
		{1000, "тысячный"},
		{2000, "две тысячный"},
		{1001, "одна тысяча первый"},
		{1010, "одна тысяча десятый"},
		{1100, "одна тысяча сотый"},
		{2001, "две тысячи первый"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Ordinal, Masculine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianOrdinalMillions(t *testing.T) {
	conv := &RussianConverter{}
	ctx := Context{Form: Ordinal, Gender: Masculine}

	tests := []struct {
		n        int64
		expected string
	}{
		{1000000, "миллионный"},
		{2000000, "два миллионный"},
		{1000001, "один миллион первый"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, Ordinal, Masculine) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestRussianNegative(t *testing.T) {
	conv := &RussianConverter{}

	tests := []struct {
		n        int64
		form     Form
		gender   Gender
		expected string
	}{
		{-1, Cardinal, Masculine, "минус один"},
		{-42, Cardinal, Masculine, "минус сорок два"},
		{-1, Cardinal, Feminine, "минус одна"},
		{-1, Ordinal, Masculine, "минус первый"},
		{-1, Ordinal, Feminine, "минус первая"},
	}

	for _, tt := range tests {
		ctx := Context{Form: tt.form, Gender: tt.gender}
		t.Run(tt.expected, func(t *testing.T) {
			result := conv.ToWords(tt.n, ctx)
			if result != tt.expected {
				t.Errorf("ToWords(%d, %v, %v) = %q, want %q", tt.n, tt.form, tt.gender, result, tt.expected)
			}
		})
	}
}

func TestRussianConverterInterface(t *testing.T) {
	conv := &RussianConverter{}

	if conv.LanguageCode() != "ru" {
		t.Errorf("LanguageCode() = %q, want %q", conv.LanguageCode(), "ru")
	}

	if conv.LanguageName() != "Russian" {
		t.Errorf("LanguageName() = %q, want %q", conv.LanguageName(), "Russian")
	}

	if !conv.SupportsContext() {
		t.Error("SupportsContext() = false, want true")
	}
}

func TestRussianRegistration(t *testing.T) {
	conv := Get("ru")
	if conv == nil {
		t.Fatal("Russian converter not registered")
	}

	if conv.LanguageCode() != "ru" {
		t.Errorf("Registered converter LanguageCode() = %q, want %q", conv.LanguageCode(), "ru")
	}
}

func BenchmarkRussianCardinal(b *testing.B) {
	conv := &RussianConverter{}
	ctx := Context{Form: Cardinal, Gender: Masculine}

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
			conv.ToWords(999999999, ctx)
		}
	})
}

func BenchmarkRussianOrdinal(b *testing.B) {
	conv := &RussianConverter{}
	ctx := Context{Form: Ordinal, Gender: Feminine}

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
		// Currency tests
		{
			name:     "1 ruble",
			input:    "1 рубль",
			expected: "один рубль",
		},
		{
			name:     "2 rubles",
			input:    "2 рубля",
			expected: "два рубля",
		},
		{
			name:     "5 rubles",
			input:    "5 рублей",
			expected: "пять рублей",
		},
		{
			name:     "21 rubles",
			input:    "21 рубль",
			expected: "двадцать один рубль",
		},
		{
			name:     "1 dollar",
			input:    "1 доллар",
			expected: "один доллар",
		},
		{
			name:     "1 kopeck feminine",
			input:    "1 копейка",
			expected: "одна копейка",
		},
		{
			name:     "2 kopecks feminine",
			input:    "2 копейки",
			expected: "две копейки",
		},
		// Time tests
		{
			name:     "1 hour",
			input:    "1 час",
			expected: "один час",
		},
		{
			name:     "2 hours",
			input:    "2 часа",
			expected: "два часа",
		},
		{
			name:     "5 hours",
			input:    "5 часов",
			expected: "пять часов",
		},
		{
			name:     "1 minute feminine",
			input:    "1 минута",
			expected: "одна минута",
		},
		{
			name:     "2 minutes feminine",
			input:    "2 минуты",
			expected: "две минуты",
		},
		{
			name:     "1 second feminine",
			input:    "1 секунда",
			expected: "одна секунда",
		},
		{
			name:     "1 day",
			input:    "1 день",
			expected: "один день",
		},
		{
			name:     "1 week feminine",
			input:    "1 неделя",
			expected: "одна неделя",
		},
		// Measurement tests
		{
			name:     "1 kilometer",
			input:    "1 километр",
			expected: "один километр",
		},
		{
			name:     "2 kilometers",
			input:    "2 километра",
			expected: "два километра",
		},
		{
			name:     "100 meters",
			input:    "100 метров",
			expected: "сто метров",
		},
		{
			name:     "1 liter",
			input:    "1 литр",
			expected: "один литр",
		},
		{
			name:     "1 kilogram",
			input:    "1 килограмм",
			expected: "один килограмм",
		},
		// People and quantities
		{
			name:     "1 person",
			input:    "1 человек",
			expected: "один человек",
		},
		{
			name:     "2 people",
			input:    "2 человека",
			expected: "два человека",
		},
		{
			name:     "5 people",
			input:    "5 человек",
			expected: "пять человек",
		},
		{
			name:     "1 time",
			input:    "1 раз",
			expected: "один раз",
		},
		{
			name:     "2 times",
			input:    "2 раза",
			expected: "два раза",
		},
		{
			name:     "10 percent",
			input:    "10 процентов",
			expected: "десять процентов",
		},
		{
			name:     "1 piece feminine",
			input:    "1 штука",
			expected: "одна штука",
		},
		// Ordinal triggers
		{
			name:     "book ordinal feminine",
			input:    "Книга 7",
			expected: "Книга седьмая",
		},
		{
			name:     "series ordinal feminine",
			input:    "Серия 12",
			expected: "Серия двенадцатая",
		},
		{
			name:     "episode ordinal masculine",
			input:    "Эпизод 4",
			expected: "Эпизод четвёртый",
		},
		{
			name:     "act ordinal masculine",
			input:    "Акт 2",
			expected: "Акт второй",
		},
		{
			name:     "scene ordinal feminine",
			input:    "Сцена 3",
			expected: "Сцена третья",
		},
		{
			name:     "place ordinal neuter",
			input:    "Место 1",
			expected: "Место первое",
		},
		{
			name:     "level ordinal masculine",
			input:    "Уровень 10",
			expected: "Уровень десятый",
		},
		// Large numbers
		{
			name:     "large number with rubles",
			input:    "1000000 рублей",
			expected: "один миллион рублей",
		},
		{
			name:     "chapter 100",
			input:    "Глава 100",
			expected: "Глава сотая",
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
		{
			name:     "decade with -х suffix and гг abbreviation",
			input:    "и до конца 90-х гг.",
			expected: "и до конца девяностых годов",
		},
		{
			name:     "decade 80s",
			input:    "и до конца 80-х гг.",
			expected: "и до конца восьмидесятых годов",
		},
		{
			name:     "decade 70s",
			input:    "и до конца 70-х гг.",
			expected: "и до конца семидесятых годов",
		},
		{
			name:     "decade 2000s",
			input:    "и до конца 2000-х гг.",
			expected: "и до конца двухтысячных годов",
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
			name:     "date in sentence with genitive trigger",
			input:    "Это случилось 10 мая",
			expected: "Это случилось десятого мая",
		},
		{
			name:     "date after preposition до",
			input:    "до 15 января",
			expected: "до пятнадцатого января",
		},
		{
			name:     "date after preposition после",
			input:    "после 1 марта",
			expected: "после первого марта",
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

func TestGenderFromNounEnding(t *testing.T) {
	tests := []struct {
		word     string
		expected Gender
	}{
		// Feminine endings (-а, -я)
		{"собака", Feminine},
		{"кошка", Feminine},
		{"земля", Feminine},
		{"станция", Feminine},
		{"Москва", Feminine},

		// Neuter endings (-о, -е, -ё)
		{"окно", Neuter},
		{"море", Neuter},
		{"здание", Neuter},
		{"бельё", Neuter},

		// Masculine endings (consonants, -й)
		{"стол", Masculine},
		{"дом", Masculine},
		{"музей", Masculine},
		{"герой", Masculine},
		{"компьютер", Masculine},

		// Soft sign (-ь) - ambiguous, defaults to masculine
		{"день", Masculine},
		{"словарь", Masculine},

		// Empty string
		{"", Masculine},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			result := GenderFromNounEnding(tt.word)
			if result != tt.expected {
				t.Errorf("GenderFromNounEnding(%q) = %v, want %v", tt.word, result, tt.expected)
			}
		})
	}
}

func TestRussianFallbackGenderDetection(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Singular nominative forms
		{
			name:     "unknown feminine noun ending in -а",
			input:    "1 собака",
			expected: "одна собака",
		},
		{
			name:     "unknown neuter noun ending in -о",
			input:    "1 окно",
			expected: "одно окно",
		},
		{
			name:     "unknown masculine noun ending in consonant",
			input:    "1 стол",
			expected: "один стол",
		},
		{
			name:     "unknown feminine noun ending in -я",
			input:    "1 земля",
			expected: "одна земля",
		},
		{
			name:     "unknown neuter noun ending in -е",
			input:    "1 море",
			expected: "одно море",
		},
		{
			name:     "unknown masculine noun ending in -й",
			input:    "1 музей",
			expected: "один музей",
		},
		// Plural forms (genitive singular after 2-4)
		{
			name:     "feminine plural ending in -и (2-4)",
			input:    "2 собаки",
			expected: "две собаки",
		},
		{
			name:     "feminine plural ending in -и (3)",
			input:    "3 кошки",
			expected: "три кошки",
		},
		{
			name:     "neuter plural ending in -а (2-4)",
			input:    "2 окна",
			expected: "два окна",
		},
		{
			name:     "neuter plural ending in -я (2-4)",
			input:    "2 моря",
			expected: "два моря",
		},
		// Masculine genitive plural (after 5+)
		{
			name:     "masculine plural ending in -ов",
			input:    "5 столов",
			expected: "пять столов",
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
