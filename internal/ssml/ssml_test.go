package ssml

import (
	"strings"
	"testing"
)

func TestWrapTextInSSML_EmptyText(t *testing.T) {
	result := WrapTextInSSML("")
	expected := "<speak></speak>"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	result = WrapTextInSSML("   ")
	if result != expected {
		t.Errorf("Expected %q for whitespace, got %q", expected, result)
	}
}

func TestWrapTextInSSML_SingleSentence(t *testing.T) {
	result := WrapTextInSSML("Hello world.")
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
	result := WrapTextInSSML("First sentence. Second sentence! Third sentence?")

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
	result := WrapTextInSSML(input)

	if strings.Count(result, "<p>") != 2 {
		t.Errorf("Expected 2 paragraph tags, got %d in: %s", strings.Count(result, "<p>"), result)
	}
}

func TestWrapTextInSSML_SingleNewlineParagraphs(t *testing.T) {
	input := "First paragraph.\nSecond paragraph."
	result := WrapTextInSSML(input)

	if strings.Count(result, "<p>") != 2 {
		t.Errorf("Expected 2 paragraph tags for single newline, got %d in: %s", strings.Count(result, "<p>"), result)
	}
}

func TestWrapTextInSSML_XMLEscaping(t *testing.T) {
	input := "Tom & Jerry said \"Hello\" and <waved>."
	result := WrapTextInSSML(input)

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
	result := WrapTextInSSML(input)

	// Ellipsis followed by lowercase should be one sentence
	if strings.Count(result, "<s>") != 1 {
		t.Errorf("Expected 1 sentence with ellipsis (lowercase continuation), got %d in: %s", strings.Count(result, "<s>"), result)
	}

	// Multiple sentences with ellipsis
	input2 := "First sentence. Wait... Second sentence."
	result2 := WrapTextInSSML(input2)
	if strings.Count(result2, "<s>") != 2 {
		t.Errorf("Expected 2 sentences, got %d in: %s", strings.Count(result2, "<s>"), result2)
	}
}

func TestWrapTextInSSML_RussianText(t *testing.T) {
	input := "Привет мир. Как дела? Отлично!"
	result := WrapTextInSSML(input)

	if strings.Count(result, "<s>") != 3 {
		t.Errorf("Expected 3 Russian sentences, got %d in: %s", strings.Count(result, "<s>"), result)
	}
}

func TestWrapTextInSSML_ComplexParagraphs(t *testing.T) {
	input := `First paragraph with multiple sentences. This is the second sentence.

Second paragraph here. It also has two sentences.

Third paragraph is short.`

	result := WrapTextInSSML(input)

	if strings.Count(result, "<p>") != 3 {
		t.Errorf("Expected 3 paragraphs, got %d", strings.Count(result, "<p>"))
	}
	if strings.Count(result, "<s>") != 5 {
		t.Errorf("Expected 5 sentences, got %d in: %s", strings.Count(result, "<s>"), result)
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
