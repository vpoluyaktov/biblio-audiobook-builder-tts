package ssml

import (
	"strings"
	"testing"
)

// TestSentenceSeparators tests various sentence-ending punctuation patterns
func TestSentenceSeparators(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedBreaks int // Number of sentence breaks expected
	}{
		{
			name:           "Single period",
			text:           "First sentence. Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Single exclamation",
			text:           "First sentence! Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Single question",
			text:           "First sentence? Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Double exclamation",
			text:           "First sentence!! Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Triple exclamation",
			text:           "First sentence!!! Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Double question",
			text:           "First sentence?? Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Triple question",
			text:           "First sentence??? Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Question exclamation",
			text:           "First sentence?! Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Exclamation question",
			text:           "First sentence!? Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Ellipsis (three dots)",
			text:           "First sentence... Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Multiple ellipsis",
			text:           "First..... Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Mixed punctuation",
			text:           "What?! Really!! Yes... Okay.",
			expectedBreaks: 3,
		},
		{
			name:           "Interrobang",
			text:           "First sentence‽ Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Armenian question mark",
			text:           "First sentence։ Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Arabic question mark",
			text:           "First sentence؟ Second sentence.",
			expectedBreaks: 1,
		},
		{
			name:           "Complex combination",
			text:           "What?! No way!!! Really... Yes. Okay?",
			expectedBreaks: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapTextInSSMLWithOptions(tt.text, SSMLOptions{
				SentenceBreakMs: 500,
			})

			// Count break tags
			breakCount := strings.Count(result, `<break time="500ms"/>`)
			if breakCount != tt.expectedBreaks {
				t.Errorf("Expected %d sentence breaks, got %d\nInput: %s\nOutput: %s",
					tt.expectedBreaks, breakCount, tt.text, result)
			}

			// Verify all original text is preserved (except for XML escaping)
			if !strings.Contains(result, "<speak>") || !strings.Contains(result, "</speak>") {
				t.Errorf("Result should be wrapped in <speak> tags: %s", result)
			}
		})
	}
}

// TestSplitIntoSentences_NewSeparators tests the splitIntoSentences function with new separators
func TestSplitIntoSentences_NewSeparators(t *testing.T) {
	tests := []struct {
		name              string
		text              string
		expectedSentences int
	}{
		{
			name:              "Multiple exclamations",
			text:              "Stop!! Don't move!!! Stay there.",
			expectedSentences: 3,
		},
		{
			name:              "Multiple questions",
			text:              "What?? Why??? How?",
			expectedSentences: 3,
		},
		{
			name:              "Question exclamation combo",
			text:              "What?! Really!? Yes.",
			expectedSentences: 3,
		},
		{
			name:              "Ellipsis variations",
			text:              "Wait... Maybe..... Okay.",
			expectedSentences: 3,
		},
		{
			name:              "Mixed all types",
			text:              "Hello. How are you? I'm fine! Great!! Really??? Yes... Maybe?! Okay.",
			expectedSentences: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentences(tt.text)
			if len(sentences) != tt.expectedSentences {
				t.Errorf("Expected %d sentences, got %d\nInput: %s\nSentences: %v",
					tt.expectedSentences, len(sentences), tt.text, sentences)
			}
		})
	}
}

// TestAddSSMLBreaks_NewSeparators tests AddSSMLBreaks with new sentence separators
func TestAddSSMLBreaks_NewSeparators(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		opts         SSMLOptions
		wantContains []string
	}{
		{
			name: "Multiple exclamations",
			text: "Stop!! Don't do that!!! Okay.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"Stop!!",
				`<break time="500ms"/>`,
				"Don&apos;t do that!!!", // Apostrophe is XML-escaped
				"Okay.",
			},
		},
		{
			name: "Multiple questions",
			text: "What?? Why??? How?",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"What??",
				"Why???",
				"How?",
				`<break time="500ms"/>`,
			},
		},
		{
			name: "Question exclamation",
			text: "What?! Really!? Yes.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"What?!",
				"Really!?",
				"Yes.",
				`<break time="500ms"/>`,
			},
		},
		{
			name: "Ellipsis preserved",
			text: "Wait... Maybe..... Okay.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"Wait...",
				"Maybe.....",
				"Okay.",
				`<break time="500ms"/>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddSSMLBreaks(tt.text, tt.opts)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("AddSSMLBreaks() result missing expected content %q\nGot: %s", want, result)
				}
			}

			// Verify no <speak> tags (AddSSMLBreaks doesn't add them)
			if strings.Contains(result, "<speak>") {
				t.Errorf("AddSSMLBreaks should not add <speak> tags, got: %s", result)
			}
		})
	}
}

// TestSentenceSeparators_EdgeCases tests edge cases with new separators
func TestSentenceSeparators_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Only punctuation",
			text: "!!!",
			want: "!!!",
		},
		{
			name: "Punctuation at start",
			text: "!!! Hello.",
			want: "Hello.",
		},
		{
			name: "Multiple spaces after punctuation",
			text: "Hello!!!    World.",
			want: "Hello!!!",
		},
		{
			name: "No space after punctuation at end",
			text: "Hello world!!!",
			want: "Hello world!!!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddSSMLBreaks(tt.text, SSMLOptions{SentenceBreakMs: 500})
			if !strings.Contains(result, tt.want) {
				t.Errorf("Expected result to contain %q, got: %s", tt.want, result)
			}
		})
	}
}
