package ssml

import (
	"strings"
	"testing"
)

func TestWrapTextInSSML_EmptyText(t *testing.T) {
	result := WrapTextInSSML("", true)
	expected := "<speak></speak>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	result = WrapTextInSSML("   ", true)
	if result != expected {
		t.Errorf("Expected %q for whitespace, got %q", expected, result)
	}
}

func TestWrapTextInSSML_SingleSentence(t *testing.T) {
	result := WrapTextInSSML("Hello world.", true)

	if !strings.Contains(result, "<speak>") {
		t.Error("Missing <speak> tag")
	}
	if !strings.Contains(result, "</speak>") {
		t.Error("Missing </speak> tag")
	}
	// With new break tag implementation, single sentence has no breaks
	if !strings.Contains(result, "Hello world.") {
		t.Errorf("Missing text content, got: %s", result)
	}
	// Should use break tags with default 500ms/800ms durations
	expected := "<speak>Hello world.</speak>"
	if result != expected {
		t.Errorf("Expected: %s, got: %s", expected, result)
	}
}

func TestWrapTextInSSML_MultipleSentences(t *testing.T) {
	result := WrapTextInSSML("First sentence. Second sentence! Third sentence?", true)
	// With break tags, we expect 2 breaks between 3 sentences (500ms default)
	if strings.Count(result, `<break time="500ms"/>`) != 2 {
		t.Errorf("Expected 2 sentence break tags, got %d in: %s", strings.Count(result, `<break time="500ms"/>`), result)
	}
	// All sentences should be present
	if !strings.Contains(result, "First sentence.") {
		t.Error("Missing first sentence")
	}
	if !strings.Contains(result, "Second sentence!") {
		t.Error("Missing second sentence")
	}
	if !strings.Contains(result, "Third sentence?") {
		t.Error("Missing third sentence")
	}
}

func TestWrapTextInSSML_MultipleParagraphs(t *testing.T) {
	input := "First paragraph.\n\nSecond paragraph."
	result := WrapTextInSSML(input, true)

	// With break tags, we expect 1 paragraph break between 2 paragraphs (800ms default)
	if strings.Count(result, `<break time="800ms"/>`) != 1 {
		t.Errorf("Expected 1 paragraph break tag, got %d in: %s", strings.Count(result, `<break time="800ms"/>`), result)
	}
}

func TestWrapTextInSSML_SingleNewlineParagraphs(t *testing.T) {
	input := "First paragraph.\nSecond paragraph."
	result := WrapTextInSSML(input, true)

	// Single newline also creates paragraph break
	if strings.Count(result, `<break time="800ms"/>`) != 1 {
		t.Errorf("Expected 1 paragraph break tag for single newline, got %d in: %s", strings.Count(result, `<break time="800ms"/>`), result)
	}
}

func TestWrapTextInSSML_XMLEscaping(t *testing.T) {
	input := "Tom & Jerry said \"Hello\" and <waved>."
	result := WrapTextInSSML(input, true)

	if !strings.Contains(result, "&amp;") {
		t.Error("& not escaped")
	}
	if !strings.Contains(result, "&lt;") {
		t.Error("< not escaped")
	}
	if !strings.Contains(result, "&gt;") {
		t.Error("> not escaped")
	}
	if !strings.Contains(result, "&quot;") {
		t.Error("\" not escaped")
	}
}

func TestWrapTextInSSML_Ellipsis(t *testing.T) {
	// Test that ellipsis is treated as a sentence boundary
	input := "Wait... what happened next?"
	result := WrapTextInSSMLWithOptions(input, SSMLOptions{
		SentenceBreakMs: 500,
	})
	// Ellipsis should create a sentence break
	if strings.Count(result, "<break") != 1 {
		t.Errorf("Expected 1 break after ellipsis, got: %s", result)
	}

	// Test ellipsis in multiple sentences
	input2 := "First sentence. Wait... Second sentence."
	result2 := WrapTextInSSMLWithOptions(input2, SSMLOptions{
		SentenceBreakMs: 500,
	})
	// Should have 2 breaks: after period and after ellipsis
	if strings.Count(result2, "<break") != 2 {
		t.Errorf("Expected 2 sentence breaks, got %d in: %s", strings.Count(result2, "<break"), result2)
	}
}

func TestWrapTextInSSML_RussianText(t *testing.T) {
	input := "Привет мир. Как дела? Отлично!"
	result := WrapTextInSSML(input, true)

	// 3 sentences = 2 sentence breaks
	if strings.Count(result, `<break time="500ms"/>`) != 2 {
		t.Errorf("Expected 2 sentence breaks for 3 Russian sentences, got %d in: %s", strings.Count(result, `<break time="500ms"/>`), result)
	}
}

func TestWrapTextInSSML_ComplexParagraphs(t *testing.T) {
	input := `First paragraph with multiple sentences. This is the second sentence.

Second paragraph here. It also has two sentences.

Third paragraph is short.`

	result := WrapTextInSSML(input, true)

	// 3 paragraphs = 2 paragraph breaks
	if strings.Count(result, `<break time="800ms"/>`) != 2 {
		t.Errorf("Expected 2 paragraph breaks, got %d", strings.Count(result, `<break time="800ms"/>`))
	}
	// 5 sentences total, but some are at paragraph ends, so we have sentence breaks within paragraphs
	// Para 1: 2 sentences (1 break), Para 2: 2 sentences (1 break), Para 3: 1 sentence (0 breaks)
	// Total: 2 sentence breaks + 2 paragraph breaks = 4 breaks total
	totalBreaks := strings.Count(result, "<break")
	if totalBreaks != 4 {
		t.Errorf("Expected 4 total breaks (2 sentence + 2 paragraph), got %d in: %s", totalBreaks, result)
	}
}

func TestWrapTextInSSML_WithoutSentencePauses(t *testing.T) {
	// When useSentencePauses is false, should only wrap in <speak> tags
	input := "First sentence. Second sentence!\n\nNew paragraph."
	result := WrapTextInSSML(input, false)

	expected := "<speak>First sentence. Second sentence!\n\nNew paragraph.</speak>"
	// The text should be XML-escaped but not have <p> or <s> tags
	if strings.Contains(result, "<p>") {
		t.Errorf("Should not contain <p> tags when useSentencePauses=false, got: %s", result)
	}
	if strings.Contains(result, "<s>") {
		t.Errorf("Should not contain <s> tags when useSentencePauses=false, got: %s", result)
	}
	if !strings.HasPrefix(result, "<speak>") {
		t.Errorf("Should start with <speak>, got: %s", result)
	}
	if !strings.HasSuffix(result, "</speak>") {
		t.Errorf("Should end with </speak>, got: %s", result)
	}

	// Test XML escaping still works
	inputWithSpecialChars := "Tom & Jerry"
	resultWithSpecialChars := WrapTextInSSML(inputWithSpecialChars, false)
	if !strings.Contains(resultWithSpecialChars, "&amp;") {
		t.Errorf("Should still escape XML characters, got: %s", resultWithSpecialChars)
	}
	_ = expected // silence unused variable warning
}

func TestWrapTextInSSMLWithOptions_DashesToBreaks(t *testing.T) {
	// Test dash-to-break conversion
	input := "цель - дыра"
	result := WrapTextInSSMLWithOptions(input, SSMLOptions{
		UseSentencePauses:     false,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   300,
	})

	if !strings.Contains(result, `<break time="300ms"/>`) {
		t.Errorf("Expected break tag in output, got: %s", result)
	}
	if strings.Contains(result, " - ") {
		t.Errorf("Dash should be replaced with break tag, got: %s", result)
	}

	// Test with custom duration
	result2 := WrapTextInSSMLWithOptions(input, SSMLOptions{
		UseSentencePauses:     false,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   500,
	})
	if !strings.Contains(result2, `<break time="500ms"/>`) {
		t.Errorf("Expected 500ms break tag, got: %s", result2)
	}

	// Test with em-dash
	inputEmDash := "цель — дыра"
	resultEmDash := WrapTextInSSMLWithOptions(inputEmDash, SSMLOptions{
		UseSentencePauses:     false,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   300,
	})
	if !strings.Contains(resultEmDash, `<break time="300ms"/>`) {
		t.Errorf("Expected break tag for em-dash, got: %s", resultEmDash)
	}

	// Test disabled conversion
	resultDisabled := WrapTextInSSMLWithOptions(input, SSMLOptions{
		UseSentencePauses:     false,
		ConvertDashesToBreaks: false,
	})
	if strings.Contains(resultDisabled, "<break") {
		t.Errorf("Break tag should not be present when disabled, got: %s", resultDisabled)
	}

	// Test ellipsis conversion
	inputEllipsis := "Если бы там... были просто люди"
	resultEllipsis := WrapTextInSSMLWithOptions(inputEllipsis, SSMLOptions{
		UseSentencePauses:     false,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   300,
	})
	if !strings.Contains(resultEllipsis, `<break time="300ms"/>`) {
		t.Errorf("Expected break tag after ellipsis, got: %s", resultEllipsis)
	}
	if !strings.Contains(resultEllipsis, "...") {
		t.Errorf("Ellipsis should be preserved, got: %s", resultEllipsis)
	}

	// Test unicode ellipsis
	inputUnicodeEllipsis := "Мягкие… податливые тела"
	resultUnicodeEllipsis := WrapTextInSSMLWithOptions(inputUnicodeEllipsis, SSMLOptions{
		UseSentencePauses:     false,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   300,
	})
	if !strings.Contains(resultUnicodeEllipsis, `<break time="300ms"/>`) {
		t.Errorf("Expected break tag after unicode ellipsis, got: %s", resultUnicodeEllipsis)
	}
}

func TestSplitIntoParagraphs(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"Single paragraph", 1},
		{"Para one\n\nPara two", 2},
		{"Para one\nPara two", 2},
		{"Para one\n\n\nPara two", 2},
		{"  \n\n  ", 0},
		{"", 0},
	}

	for _, tt := range tests {
		result := splitIntoParagraphs(tt.input)
		if len(result) != tt.expected {
			t.Errorf("splitIntoParagraphs(%q): expected %d paragraphs, got %d: %v",
				tt.input, tt.expected, len(result), result)
		}
	}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"One sentence.", 1},
		{"First. Second.", 2},
		{"Question? Answer!", 2},
		{"Wait... really?", 2}, // Ellipsis now ends sentence, plus question mark
		{"No punctuation", 1},
		{"", 0},
	}

	for _, tt := range tests {
		result := splitIntoSentences(tt.input)
		if len(result) != tt.expected {
			t.Errorf("splitIntoSentences(%q): expected %d sentences, got %d: %v",
				tt.input, tt.expected, len(result), result)
		}
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
		{"\"quoted\"", "&quot;quoted&quot;"},
		{"it's", "it&apos;s"},
		{"a & b < c > d", "a &amp; b &lt; c &gt; d"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}
