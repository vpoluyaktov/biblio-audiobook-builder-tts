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
