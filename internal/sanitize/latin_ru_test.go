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
			name:     "Simple two-letter abbreviation",
			input:    "Самолет DC летел над городом",
			expected: "Самолет ди си летел над городом",
		},
		{
			name:     "Three-letter abbreviation",
			input:    "Агентство NSA занимается разведкой",
			expected: "Агентство эн эс эй занимается разведкой",
		},
		{
			name:     "Abbreviation with hyphen and number - DC-19",
			input:    "Модель DC-19 была популярна",
			expected: "Модель ди си 19 была популярна",
		},
		{
			name:     "Single letter with hyphen and number - L-3",
			input:    "Самолет L-3 взлетел",
			expected: "Самолет эл 3 взлетел",
		},
		{
			name:     "Multiple abbreviations in text",
			input:    "FBI и CIA работают с NSA",
			expected: "эф би ай и си ай эй работают с эн эс эй",
		},
		{
			name:     "IP address context",
			input:    "Адрес IP был заблокирован",
			expected: "Адрес ай пи был заблокирован",
		},
		{
			name:     "USB device",
			input:    "Подключите USB устройство",
			expected: "Подключите ю эс би устройство",
		},
		{
			name:     "Mixed case - should only match uppercase",
			input:    "Протокол HTTP используется везде",
			expected: "Протокол эйч ти ти пи используется везде",
		},
		{
			name:     "Aircraft model Ту-154 with Latin",
			input:    "Самолет Ту-154 и DC-10",
			expected: "Самолет Ту-154 и ди си 10",
		},
		{
			name:     "Boeing model",
			input:    "Боинг-747 и DC-7 в небе",
			expected: "Боинг-747 и ди си 7 в небе",
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
			name:     "Multiple hyphens - L-3 and DC-19",
			input:    "Модели L-3 и DC-19 устарели",
			expected: "Модели эл 3 и ди си 19 устарели",
		},
		{
			name:     "At start of sentence",
			input:    "FBI расследует дело",
			expected: "эф би ай расследует дело",
		},
		{
			name:     "At end of sentence",
			input:    "Это было сделано NSA",
			expected: "Это было сделано эн эс эй",
		},
		{
			name:     "Single letter A",
			input:    "Витамин A полезен",
			expected: "Витамин эй полезен",
		},
		{
			name:     "Single letter B",
			input:    "Гепатит B опасен",
			expected: "Гепатит би опасен",
		},
		{
			name:     "Single letter C",
			input:    "Витамин C важен",
			expected: "Витамин си важен",
		},
		{
			name:     "DNA and RNA",
			input:    "Молекулы DNA и RNA",
			expected: "Молекулы ди эн эй и ар эн эй",
		},
		{
			name:     "GPS navigation",
			input:    "Навигация GPS работает",
			expected: "Навигация джи пи эс работает",
		},
		{
			name:     "WiFi connection",
			input:    "Подключение WiFi активно",
			expected: "Подключение дабл ю ай эф ай активно",
		},
		{
			name:     "HTML and CSS",
			input:    "Код HTML и CSS",
			expected: "Код эйч ти эм эл и си эс эс",
		},
		{
			name:     "JSON format",
			input:    "Формат JSON популярен",
			expected: "Формат джей эс оу эн популярен",
		},
		{
			name:     "PDF document",
			input:    "Документ PDF открыт",
			expected: "Документ пи ди эф открыт",
		},
		{
			name:     "CEO and CFO",
			input:    "Директор CEO и финансист CFO",
			expected: "Директор си и оу и финансист си эф оу",
		},
		{
			name:     "PhD degree",
			input:    "Степень PhD получена",
			expected: "Степень пи эйч ди получена",
		},
		{
			name:     "OK response",
			input:    "Ответ OK получен",
			expected: "Ответ оу кей получен",
		},
		{
			name:     "VIP guest",
			input:    "Гость VIP прибыл",
			expected: "Гость ви ай пи прибыл",
		},
		{
			name:     "No Latin letters - pure Russian",
			input:    "Это просто русский текст без латиницы",
			expected: "Это просто русский текст без латиницы",
		},
		{
			name:     "No Latin letters - pure Cyrillic abbreviations",
			input:    "МВД и ФСБ работают вместе",
			expected: "МВД и ФСБ работают вместе",
		},
		{
			name:     "English text should not be converted",
			input:    "This is English text with FBI and CIA",
			expected: "This is English text with FBI and CIA",
		},
		{
			name:     "Mixed with numbers",
			input:    "Версия 5G и 4G",
			expected: "Версия 5 джи и 4 джи",
		},
		{
			name:     "Complex sentence",
			input:    "Агентство FBI использует технологию AI для анализа DNA",
			expected: "Агентство эф би ай использует технологию эй ай для анализа ди эн эй",
		},
		{
			name:     "With punctuation",
			input:    "USB, HDMI, и WiFi - все работает",
			expected: "ю эс би, эйч ди эм ай, и дабл ю ай эф ай - все работает",
		},
		{
			name:     "Lowercase should not match",
			input:    "Слово usb написано маленькими буквами",
			expected: "Слово usb написано маленькими буквами",
		},
		{
			name:     "AC and DC current",
			input:    "Ток AC и DC используются",
			expected: "Ток эй си и ди си используются",
		},
		{
			name:     "AM and FM radio",
			input:    "Радио AM и FM",
			expected: "Радио эй эм и эф эм",
		},
		{
			name:     "SSD and HDD",
			input:    "Диск SSD быстрее HDD",
			expected: "Диск эс эс ди быстрее эйч ди ди",
		},
		{
			name:     "HTTP and HTTPS",
			input:    "Протоколы HTTP и HTTPS",
			expected: "Протоколы эйч ти ти пи и эйч ти ти пи эс",
		},
		{
			name:     "SQL database",
			input:    "База данных SQL",
			expected: "База данных эс кью эл",
		},
		{
			name:     "API endpoint",
			input:    "Точка API доступна",
			expected: "Точка эй пи ай доступна",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertLatinInRussianText(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertLatinInRussianText(%q)\ngot:  %q\nwant: %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLatinToRussianConverter_EdgeCases(t *testing.T) {
	converter := NewLatinToRussianConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only spaces",
			input:    "   ",
			expected: "   ",
		},
		{
			name:     "Only Latin letters - no Russian context",
			input:    "FBI CIA NSA",
			expected: "FBI CIA NSA",
		},
		{
			name:     "Single Cyrillic word with Latin",
			input:    "USB порт",
			expected: "ю эс би порт",
		},
		{
			name:     "Latin at boundaries",
			input:    "FBI, CIA. NSA!",
			expected: "FBI, CIA. NSA!",
		},
		{
			name:     "Very long abbreviation",
			input:    "Аббревиатура ABCDEFGHIJ длинная",
			expected: "Аббревиатура эй би си ди и эф джи эйч ай джей длинная",
		},
		{
			name:     "Numbers only with hyphen",
			input:    "Номер 123-456",
			expected: "Номер 123-456",
		},
		{
			name:     "Mixed Cyrillic and Latin abbreviations",
			input:    "МВД и FBI работают",
			expected: "МВД и эф би ай работают",
		},
		{
			name:     "Hyphen without number",
			input:    "Код DC- был неполным",
			expected: "Код ди си- был неполным",
		},
		{
			name:     "Multiple hyphens",
			input:    "Модель DC-19-A устарела",
			expected: "Модель ди си 19-эй устарела",
		},
		{
			name:     "Special characters around abbreviation",
			input:    "(FBI) [CIA] {NSA} в России",
			expected: "(эф би ай) [си ай эй] {эн эс эй} в России",
		},
		{
			name:     "Newlines and tabs",
			input:    "Агентство\nFBI\tи\nCIA",
			expected: "Агентство\nэф би ай\tи\nси ай эй",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertLatinInRussianText(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertLatinInRussianText(%q)\ngot:  %q\nwant: %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLatinToRussianConverter_ContextDetection(t *testing.T) {
	converter := NewLatinToRussianConverter()

	tests := []struct {
		name          string
		input         string
		shouldConvert bool
		description   string
	}{
		{
			name:          "Russian context - should convert",
			input:         "Это русский текст с FBI аббревиатурой",
			shouldConvert: true,
			description:   "Latin abbreviation in Russian text should be converted",
		},
		{
			name:          "English context - should not convert",
			input:         "This is English text with FBI abbreviation",
			shouldConvert: false,
			description:   "Latin abbreviation in English text should not be converted",
		},
		{
			name:          "Mostly Russian - should convert",
			input:         "Длинный русский текст с множеством слов и только одна аббревиатура FBI в конце",
			shouldConvert: true,
			description:   "Latin in predominantly Russian text should convert",
		},
		{
			name:          "Mostly English - should not convert",
			input:         "Long English text with many words and only одно Russian word with FBI",
			shouldConvert: false,
			description:   "Latin in predominantly English text should not convert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertLatinInRussianText(tt.input)
			containsFBI := result != tt.input

			if tt.shouldConvert && !containsFBI {
				t.Errorf("%s: expected conversion but got none\ninput:  %q\noutput: %q", tt.description, tt.input, result)
			}
			if !tt.shouldConvert && containsFBI {
				t.Errorf("%s: expected no conversion but got conversion\ninput:  %q\noutput: %q", tt.description, tt.input, result)
			}
		})
	}
}

func TestLatinToRussianConverter_RealWorldExamples(t *testing.T) {
	converter := NewLatinToRussianConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Technical documentation",
			input:    "Для подключения используйте USB порт и установите драйвер через API",
			expected: "Для подключения используйте ю эс би порт и установите драйвер через эй пи ай",
		},
		{
			name:     "News article",
			input:    "Агентство FBI сообщило, что расследование с использованием AI технологий продолжается",
			expected: "Агентство эф би ай сообщило, что расследование с использованием эй ай технологий продолжается",
		},
		{
			name:     "Aviation text",
			input:    "Самолет DC-10 совершил посадку в аэропорту",
			expected: "Самолет ди си 10 совершил посадку в аэропорту",
		},
		{
			name:     "Medical text",
			input:    "Анализ DNA показал наличие вируса HIV",
			expected: "Анализ ди эн эй показал наличие вируса эйч ай ви",
		},
		{
			name:     "IT article",
			input:    "Сервер поддерживает протоколы HTTP, HTTPS, FTP и SSH",
			expected: "Сервер поддерживает протоколы эйч ти ти пи, эйч ти ти пи эс, эф ти пи и эс эс эйч",
		},
		{
			name:     "Business text",
			input:    "Компания LLC получила сертификат ISO",
			expected: "Компания эл эл си получила сертификат ай эс оу",
		},
		{
			name:     "Mixed technical specs",
			input:    "Процессор CPU работает на частоте 3 GHz, оперативная память RAM составляет 16 GB",
			expected: "Процессор си пи ю работает на частоте 3 джи эйч зет, оперативная память ар эй эм составляет 16 джи би",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertLatinInRussianText(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertLatinInRussianText(%q)\ngot:  %q\nwant: %q", tt.input, result, tt.expected)
			}
		})
	}
}
