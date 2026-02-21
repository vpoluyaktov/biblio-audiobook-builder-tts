package sanitize

import (
	"testing"
)

func TestLatinToRussianConverter_ConvertLatinInRussianText(t *testing.T) {
	converter := NewLatinToRussianConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Two-letter abbreviation",
			input:    "Адрес IP был заблокирован",
			expected: "Адрес ай пи был заблокирован",
		},
		{
			name:     "Three-letter abbreviation",
			input:    "Агентство NSA занимается разведкой",
			expected: "Агентство эн эс эй занимается разведкой",
		},
		{
			name:     "Multiple abbreviations",
			input:    "FBI и CIA работают с NSA",
			expected: "эф би ай и си ай эй работают с эн эс эй",
		},
		{
			name:     "USB device",
			input:    "Подключите USB устройство",
			expected: "Подключите ю эс би устройство",
		},
		{
			name:     "HTTP protocol",
			input:    "Протокол HTTP используется везде",
			expected: "Протокол эйч ти ти пи используется везде",
		},
		{
			name:     "CPU and GPU",
			input:    "Процессор CPU и видеокарта GPU",
			expected: "Процессор си пи ю и видеокарта джи пи ю",
		},
		{
			name:     "DVD and CD",
			input:    "Диск DVD или CD",
			expected: "Диск ди ви ди или си ди",
		},
		{
			name:     "TV channel",
			input:    "Канал TV показывает новости",
			expected: "Канал ти ви показывает новости",
		},
		{
			name:     "AI technology",
			input:    "Технология AI развивается",
			expected: "Технология эй ай развивается",
		},
		{
			name:     "VR and AR",
			input:    "Виртуальная реальность VR и дополненная AR",
			expected: "Виртуальная реальность ви ар и дополненная эй ар",
		},
		{
			name:     "At start of sentence",
			input:    "FBI расследует дело",
			expected: "эф би ай расследует дело",
		},
		{
			name:     "At end of sentence",
			input:    "Это работа CIA",
			expected: "Это работа си ай эй",
		},
		{
			name:     "No conversion in English context",
			input:    "The FBI investigates crimes",
			expected: "The FBI investigates crimes",
		},
		{
			name:     "No conversion for single letter",
			input:    "Буква A в тексте",
			expected: "Буква A в тексте",
		},
		{
			name:     "No conversion for lowercase",
			input:    "Слово http в нижнем регистре",
			expected: "Слово http в нижнем регистре",
		},
		{
			name:     "Mixed Cyrillic and Latin text",
			input:    "Компания IBM и Microsoft работают с NASA",
			expected: "Компания ай би эм и Microsoft работают с эн эй эс эй",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertLatinInRussianText(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertLatinInRussianText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLatinToRussianConverter_ContextDetection(t *testing.T) {
	converter := NewLatinToRussianConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Mostly Russian context",
			input:    "В России FBI работает с местными властями",
			expected: "В России эф би ай работает с местными властями",
		},
		{
			name:     "Mostly English context - no conversion",
			input:    "The FBI works in Russia sometimes",
			expected: "The FBI works in Russia sometimes",
		},
		{
			name:     "Balanced context with 20% Cyrillic",
			input:    "FBI CIA NSA работают",
			expected: "эф би ай си ай эй эн эс эй работают",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertLatinInRussianText(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertLatinInRussianText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
