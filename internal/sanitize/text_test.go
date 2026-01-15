package sanitize

import (
	"strings"
	"testing"
)

func TestTextForTTS_PreservesAllLanguages(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"English text", "Hello, world! This is a test.", "Hello, world! This is a test."},
		{"Russian Cyrillic", "Привет, мир! Это тест.", "Привет, мир! Это тест."},
		{"Ukrainian Cyrillic", "Привіт, світ! Це тест.", "Привіт, світ! Це тест."},
		{"Chinese characters", "你好世界", "你好世界"},
		{"Japanese", "こんにちは世界", "こんにちは世界"},
		{"Korean Hangul", "안녕하세요", "안녕하세요"},
		{"Arabic", "مرحبا بالعالم", "مرحبا بالعالم"},
		{"Hebrew", "שלום עולם", "שלום עולם"},
		{"Greek", "Γειά σου κόσμε", "Γειά σου κόσμε"},
		{"French accents", "café très délicieux", "café très délicieux"},
		{"German umlauts", "Größe Übung Äpfel ß", "Größe Übung Äpfel ß"},
		{"Spanish", "Año señor niño", "Año señor niño"},
		{"Polish", "ąęćłńóśźż", "ąęćłńóśźż"},
		{"Mixed languages", "Hello Привет 你好", "Hello Привет 你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextForTTS(tt.input)
			if result != tt.expected {
				t.Errorf("TextForTTS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTextForTTS_ReplacesProblematicCharacters(t *testing.T) {
	// Em dash U+2014
	result := TextForTTS("Hello\u2014world")
	if !strings.Contains(result, "Hello - world") {
		t.Errorf("Em dash not replaced: %q", result)
	}

	// En dash U+2013
	result = TextForTTS("pages 1\u20132010")
	if !strings.Contains(result, "1 - 2010") {
		t.Errorf("En dash not replaced: %q", result)
	}

	// Curly double quotes U+201C U+201D
	result = TextForTTS("\u201cHello\u201d")
	if result != "\"Hello\"" {
		t.Errorf("Curly quotes not replaced: %q", result)
	}

	// Curly single quotes U+2018 U+2019
	result = TextForTTS("\u2018Hello\u2019")
	if result != "'Hello'" {
		t.Errorf("Curly single quotes not replaced: %q", result)
	}

	// Guillemets U+00AB U+00BB
	result = TextForTTS("\u00abHello\u00bb")
	if result != "\"Hello\"" {
		t.Errorf("Guillemets not replaced: %q", result)
	}

	// Ellipsis U+2026
	result = TextForTTS("Hello\u2026world")
	if result != "Hello...world" {
		t.Errorf("Ellipsis not replaced: %q", result)
	}

	// Non-breaking space U+00A0
	result = TextForTTS("Hello\u00a0world")
	if result != "Hello world" {
		t.Errorf("Non-breaking space not replaced: %q", result)
	}

	// Zero-width space U+200B
	result = TextForTTS("Hello\u200bworld")
	if result != "Helloworld" {
		t.Errorf("Zero-width space not removed: %q", result)
	}

	// Bullet U+2022
	result = TextForTTS("\u2022 Item one")
	if result != "- Item one" {
		t.Errorf("Bullet not replaced: %q", result)
	}

	// Copyright U+00A9
	result = TextForTTS("\u00a9 2024")
	if result != "(c) 2024" {
		t.Errorf("Copyright not expanded: %q", result)
	}

	// Trademark U+2122
	result = TextForTTS("Brand\u2122")
	if result != "Brand(TM)" {
		t.Errorf("Trademark not expanded: %q", result)
	}

	// Degree U+00B0
	result = TextForTTS("90\u00b0")
	if result != "90 degrees" {
		t.Errorf("Degree not expanded: %q", result)
	}

	// Multiplication U+00D7
	result = TextForTTS("2\u00d73")
	if result != "2 times 3" {
		t.Errorf("Multiplication not expanded: %q", result)
	}

	// Division U+00F7
	result = TextForTTS("6\u00f72")
	if result != "6 divided by 2" {
		t.Errorf("Division not expanded: %q", result)
	}

	// Fraction one half U+00BD
	result = TextForTTS("\u00bd cup")
	if result != "one half cup" {
		t.Errorf("Fraction not expanded: %q", result)
	}

	// Right single quotation mark U+2019 (from HTML entity &#8217;)
	result = TextForTTS("Hello\u2019s World")
	if result != "Hello's World" {
		t.Errorf("Right single quote (U+2019) not converted to ASCII apostrophe: %q", result)
	}

	// Double low-9 quotation mark U+201E (German/Polish opening quote)
	result = TextForTTS("\u201eHello\u201c")
	if result != "\"Hello\"" {
		t.Errorf("Double low-9 quote not replaced: %q", result)
	}

	// Single low-9 quotation mark U+201A
	result = TextForTTS("\u201aHello\u2018")
	if result != "'Hello'" {
		t.Errorf("Single low-9 quote not replaced: %q", result)
	}

	// Single guillemets U+2039 U+203A
	result = TextForTTS("\u2039Hello\u203a")
	if result != "'Hello'" {
		t.Errorf("Single guillemets not replaced: %q", result)
	}

	// Four-per-em space U+2005 (from HTML entity &#8197;)
	result = TextForTTS("Text\u2005here")
	if result != "Text here" {
		t.Errorf("Four-per-em space (U+2005) not converted to regular space: %q", result)
	}

	// En space U+2002
	result = TextForTTS("Text\u2002here")
	if result != "Text here" {
		t.Errorf("En space (U+2002) not converted to regular space: %q", result)
	}

	// Em space U+2003
	result = TextForTTS("Text\u2003here")
	if result != "Text here" {
		t.Errorf("Em space (U+2003) not converted to regular space: %q", result)
	}

	// Thin space U+2009
	result = TextForTTS("Text\u2009here")
	if result != "Text here" {
		t.Errorf("Thin space (U+2009) not converted to regular space: %q", result)
	}
}

func TestTextForTTS_RemovesControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Preserves newlines as space", "Line 1\nLine 2", "Line 1 Line 2"},
		{"Preserves tabs as space", "Word1\tWord2", "Word1 Word2"},
		{"Removes null character", "Hello\x00World", "Hello World"},
		{"Removes bell character", "Hello\x07World", "Hello World"},
		{"Removes backspace", "Hello\x08World", "Hello World"},
		{"Removes escape", "Hello\x1bWorld", "Hello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextForTTS(tt.input)
			if result != tt.expected {
				t.Errorf("TextForTTS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTextForTTS_NormalizesWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Multiple spaces", "Hello    world", "Hello world"},
		{"Mixed whitespace", "Hello \t \n world", "Hello world"},
		{"Leading and trailing", "   Hello world   ", "Hello world"},
		{"Multiple periods", "Hello.....world", "Hello...world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextForTTS(tt.input)
			if result != tt.expected {
				t.Errorf("TextForTTS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTextForTTS_RealWorldExamples(t *testing.T) {
	// Russian book text with em dashes
	result := TextForTTS("\u2014 Привет! \u2014 сказал он.")
	if result != "- Привет! - сказал он." {
		t.Errorf("Russian em dashes: %q", result)
	}

	// Russian text with guillemets
	result = TextForTTS("Он сказал: \u00abПривет!\u00bb")
	if result != "Он сказал: \"Привет!\"" {
		t.Errorf("Russian guillemets: %q", result)
	}

	// English with smart quotes
	result = TextForTTS("\u201cHello,\u201d she said")
	if result != "\"Hello,\" she said" {
		t.Errorf("Smart quotes: %q", result)
	}

	// Technical text with symbols
	result = TextForTTS("25\u00b0C \u00b1 2\u00b0C")
	if result != "25 degrees C plus or minus 2 degrees C" {
		t.Errorf("Technical symbols: %q", result)
	}

	// Recipe with fractions
	result = TextForTTS("\u00bd cup and \u00bc teaspoon")
	if result != "one half cup and one quarter teaspoon" {
		t.Errorf("Fractions: %q", result)
	}

	// Legal text
	result = TextForTTS("\u00a9 2024 Company\u2122")
	if result != "(c) 2024 Company(TM)" {
		t.Errorf("Legal symbols: %q", result)
	}
}

func TestPronunciationDictionary(t *testing.T) {
	t.Run("AddRule and Apply", func(t *testing.T) {
		dict := NewPronunciationDictionary()
		err := dict.AddRule(`\bMr\.`, "Mister")
		if err != nil {
			t.Fatalf("AddRule failed: %v", err)
		}

		result := dict.Apply("Mr. Smith went to the store.")
		expected := "Mister Smith went to the store."
		if result != expected {
			t.Errorf("Apply() = %q, want %q", result, expected)
		}
	})

	t.Run("Invalid regex pattern", func(t *testing.T) {
		dict := NewPronunciationDictionary()
		err := dict.AddRule(`[invalid`, "replacement")
		if err == nil {
			t.Error("AddRule should fail for invalid regex")
		}
	})

	t.Run("Multiple rules applied in order", func(t *testing.T) {
		dict := NewPronunciationDictionary()
		dict.AddRule(`\bDr\.`, "Doctor")
		dict.AddRule(`\bMr\.`, "Mister")
		dict.AddRule(`\bMrs\.`, "Missus")

		result := dict.Apply("Dr. Smith and Mr. Jones met Mrs. Brown.")
		expected := "Doctor Smith and Mister Jones met Missus Brown."
		if result != expected {
			t.Errorf("Apply() = %q, want %q", result, expected)
		}
	})

	t.Run("RuleCount", func(t *testing.T) {
		dict := NewPronunciationDictionary()
		if dict.RuleCount() != 0 {
			t.Error("New dictionary should have 0 rules")
		}

		dict.AddRule(`test`, "TEST")
		if dict.RuleCount() != 1 {
			t.Error("Dictionary should have 1 rule after AddRule")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		dict := NewPronunciationDictionary()
		dict.AddRule(`test`, "TEST")
		dict.Clear()
		if dict.RuleCount() != 0 {
			t.Error("Dictionary should have 0 rules after Clear")
		}
	})
}

func TestGetDefaultRules(t *testing.T) {
	rules := GetDefaultRules()
	if len(rules) == 0 {
		t.Error("GetDefaultRules should return non-empty slice")
	}

	for _, rule := range rules {
		dict := NewPronunciationDictionary()
		err := dict.AddRule(rule.Pattern, rule.Replacement)
		if err != nil {
			t.Errorf("Default rule pattern %q failed to compile: %v", rule.Pattern, err)
		}
	}
}

func TestTextSanitizer(t *testing.T) {
	t.Run("Sanitize combines TTS and dictionary", func(t *testing.T) {
		sanitizer := NewTextSanitizer()
		sanitizer.LoadDefaultRules()

		input := "Mr. Smith said: \u00abHello\u2014world!\u00bb"
		result := sanitizer.Sanitize(input)

		if strings.Contains(result, "Mr.") {
			t.Error("Should have expanded Mr. to Mister")
		}
		if strings.Contains(result, "\u00ab") || strings.Contains(result, "\u00bb") {
			t.Error("Should have replaced guillemets")
		}
		if strings.Contains(result, "\u2014") {
			t.Error("Should have replaced em dash")
		}
	})

	t.Run("Sanitize without dictionary", func(t *testing.T) {
		sanitizer := NewTextSanitizer()

		input := "Mr. Smith said: \u00abHello\u00bb"
		result := sanitizer.Sanitize(input)

		if strings.Contains(result, "\u00ab") {
			t.Error("Should have replaced guillemets")
		}
		if !strings.Contains(result, "Mr.") {
			t.Error("Should NOT have expanded Mr. without dictionary rules")
		}
	})

	t.Run("SetDictionary", func(t *testing.T) {
		sanitizer := NewTextSanitizer()
		dict := NewPronunciationDictionary()
		dict.AddRule(`test`, "TEST")

		sanitizer.SetDictionary(dict)

		result := sanitizer.Sanitize("This is a test.")
		if !strings.Contains(result, "TEST") {
			t.Error("Custom dictionary should be applied")
		}
	})

	t.Run("GetDictionary", func(t *testing.T) {
		sanitizer := NewTextSanitizer()
		dict := sanitizer.GetDictionary()
		if dict == nil {
			t.Error("GetDictionary should return non-nil dictionary")
		}
	})
}

func TestTextForTTS_EmptyAndEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Only whitespace", "   \t\n   ", ""},
		{"Single character", "A", "A"},
		{"Single Cyrillic", "Я", "Я"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextForTTS(tt.input)
			if result != tt.expected {
				t.Errorf("TextForTTS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}

	t.Run("Very long string preserved", func(t *testing.T) {
		input := strings.Repeat("Тест ", 1000)
		expected := strings.TrimSpace(input)
		result := TextForTTS(input)
		if len(result) != len(expected) {
			t.Errorf("Long string length = %d, want %d", len(result), len(expected))
		}
	})
}
