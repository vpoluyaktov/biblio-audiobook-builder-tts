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
			name:     "Single uppercase letter",
			input:    "Буква A в тексте",
			expected: "Буква эй в тексте",
		},
		{
			name:     "Two-letter uppercase abbreviation",
			input:    "Адрес IP был заблокирован",
			expected: "Адрес ай пи был заблокирован",
		},
		{
			name:     "Three-letter uppercase abbreviation",
			input:    "Агентство NSA занимается разведкой",
			expected: "Агентство эн эс эй занимается разведкой",
		},
		{
			name:     "Multiple uppercase abbreviations",
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
			name:     "TSAC uppercase abbreviation",
			input:    "TSAC",
			expected: "ти эс эй си",
		},
		{
			name:     "TSAC with lowercase word",
			input:    "TSAC desent",
			expected: "ти эс эй си десент",
		},
		{
			name:     "Mixed case word - Microsoft",
			input:    "Компания Microsoft работает",
			expected: "Компания микрософт работает",
		},
		{
			name:     "Mixed case word - iPhone",
			input:    "Устройство iPhone популярно",
			expected: "Устройство ифоне популярно",
		},
		{
			name:     "All caps and mixed case",
			input:    "Компания IBM и Microsoft работают с NASA",
			expected: "Компания ай би эм и микрософт работают с эн эй эс эй",
		},
		{
			name:     "Lowercase word - http",
			input:    "Слово http в нижнем регистре",
			expected: "Слово хттп в нижнем регистре",
		},
		{
			name:     "Lowercase word - desent",
			input:    "Слово desent",
			expected: "Слово десент",
		},
		{
			name:     "Mixed case - TsAc",
			input:    "Слово TsAc",
			expected: "Слово тсак",
		},
		{
			name:     "English word with ch",
			input:    "Слово chat",
			expected: "Слово чат",
		},
		{
			name:     "English word with sh",
			input:    "Слово shop",
			expected: "Слово шоп",
		},
		{
			name:     "English word with th",
			input:    "Слово think",
			expected: "Слово тинк",
		},
		{
			name:     "English word with ph",
			input:    "Слово phone",
			expected: "Слово фоне",
		},
		{
			name:     "English word with ck",
			input:    "Слово check",
			expected: "Слово чек",
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
