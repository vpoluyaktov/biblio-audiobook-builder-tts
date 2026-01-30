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
	if !strings.Contains(result, "<p>") {
		t.Error("Missing <p> tag")
	}
	if !strings.Contains(result, "<s>Hello world.</s>") {
		t.Errorf("Missing sentence tag, got: %s", result)
	}
}

func TestWrapTextInSSML_MultipleSentences(t *testing.T) {
	result := WrapTextInSSML("First sentence. Second sentence! Third sentence?", true)

	if strings.Count(result, "<s>") != 3 {
		t.Errorf("Expected 3 sentence tags, got %d in: %s", strings.Count(result, "<s>"), result)
	}
	if !strings.Contains(result, "<s>First sentence.</s>") {
		t.Error("Missing first sentence")
	}
	if !strings.Contains(result, "<s>Second sentence!</s>") {
		t.Error("Missing second sentence")
	}
	if !strings.Contains(result, "<s>Third sentence?</s>") {
		t.Error("Missing third sentence")
	}
}

func TestWrapTextInSSML_MultipleParagraphs(t *testing.T) {
	input := "First paragraph.\n\nSecond paragraph."
	result := WrapTextInSSML(input, true)

	if strings.Count(result, "<p>") != 2 {
		t.Errorf("Expected 2 paragraph tags, got %d in: %s", strings.Count(result, "<p>"), result)
	}
}

func TestWrapTextInSSML_SingleNewlineParagraphs(t *testing.T) {
	input := "First paragraph.\nSecond paragraph."
	result := WrapTextInSSML(input, true)

	if strings.Count(result, "<p>") != 2 {
		t.Errorf("Expected 2 paragraph tags for single newline, got %d in: %s", strings.Count(result, "<p>"), result)
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
	// Ellipsis within a sentence should not cause splits
	input := "Wait... what happened next?"
	result := WrapTextInSSML(input, true)

	// Ellipsis followed by lowercase should be one sentence
	if strings.Count(result, "<s>") != 1 {
		t.Errorf("Expected 1 sentence with ellipsis (lowercase continuation), got %d in: %s", strings.Count(result, "<s>"), result)
	}

	// Multiple sentences with ellipsis
	input2 := "First sentence. Wait... Second sentence."
	result2 := WrapTextInSSML(input2, true)
	if strings.Count(result2, "<s>") != 2 {
		t.Errorf("Expected 2 sentences, got %d in: %s", strings.Count(result2, "<s>"), result2)
	}
}

func TestWrapTextInSSML_RussianText(t *testing.T) {
	input := "Привет мир. Как дела? Отлично!"
	result := WrapTextInSSML(input, true)

	if strings.Count(result, "<s>") != 3 {
		t.Errorf("Expected 3 Russian sentences, got %d in: %s", strings.Count(result, "<s>"), result)
	}
}

func TestWrapTextInSSML_ComplexParagraphs(t *testing.T) {
	input := `First paragraph with multiple sentences. This is the second sentence.

Second paragraph here. It also has two sentences.

Third paragraph is short.`

	result := WrapTextInSSML(input, true)

	if strings.Count(result, "<p>") != 3 {
		t.Errorf("Expected 3 paragraphs, got %d", strings.Count(result, "<p>"))
	}
	if strings.Count(result, "<s>") != 5 {
		t.Errorf("Expected 5 sentences, got %d in: %s", strings.Count(result, "<s>"), result)
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
		{"Wait... really?", 1}, // Ellipsis doesn't end sentence, question mark does
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
