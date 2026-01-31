package ssml

import (
	"strings"
	"testing"
)

// TestInitialsProtection tests that abbreviated initials are not split into separate sentences
// Language-agnostic tests for the initials protection feature
func TestInitialsProtection(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedBreaks int // Number of sentence breaks expected
		shouldContain  []string
	}{
		{
			name:           "English initials A.B.",
			text:           "Professor A.B. Smith published the paper.",
			expectedBreaks: 0,
			shouldContain:  []string{"A.B. Smith"},
		},
		{
			name:           "English initials with sentence after",
			text:           "Professor J.K. Rowling wrote books. She was famous.",
			expectedBreaks: 1,
			shouldContain:  []string{"J.K. Rowling"},
		},
		{
			name:           "Initials with spaces normalized",
			text:           "Professor A. B. Smith published the paper.",
			expectedBreaks: 0,
			shouldContain:  []string{"A.B. Smith"}, // Spaces should be removed
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

			// Check that expected content is present
			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q\nInput: %s\nOutput: %s",
						expected, tt.text, result)
				}
			}
		})
	}
}

// TestSplitIntoSentences_Initials tests the sentence splitting with initials
func TestSplitIntoSentences_Initials(t *testing.T) {
	tests := []struct {
		name              string
		text              string
		expectedSentences int
		shouldContain     []string
	}{
		{
			name:              "Initials not split",
			text:              "Professor A.B. Smith worked here.",
			expectedSentences: 1,
			shouldContain:     []string{"A.B. Smith"},
		},
		{
			name:              "Two sentences with initials",
			text:              "Professor A.B. Smith. He worked here.",
			expectedSentences: 2,
			shouldContain:     []string{"A.B. Smith", "He worked"},
		},
		{
			name:              "Initials with spaces normalized",
			text:              "Professor A. B. Smith.",
			expectedSentences: 1,
			shouldContain:     []string{"A.B. Smith"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentences(tt.text)
			if len(sentences) != tt.expectedSentences {
				t.Errorf("Expected %d sentences, got %d\nInput: %s\nSentences: %v",
					tt.expectedSentences, len(sentences), tt.text, sentences)
			}

			// Check that expected content is present in the sentences
			allText := strings.Join(sentences, " ")
			for _, expected := range tt.shouldContain {
				if !strings.Contains(allText, expected) {
					t.Errorf("Expected sentences to contain %q\nInput: %s\nSentences: %v",
						expected, tt.text, sentences)
				}
			}
		})
	}
}

// TestAddSSMLBreaks_Initials tests AddSSMLBreaks with initials
func TestAddSSMLBreaks_Initials(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		opts           SSMLOptions
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "English initials preserved",
			text: "Professor A.B. Smith published papers. He was a scientist.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"A.B. Smith",
				`<break time="500ms"/>`,
			},
			wantNotContain: []string{
				"A. <break", // Should NOT have break between initials
			},
		},
		{
			name: "Initials with spaces normalized",
			text: "Professor J. K. Rowling. She wrote books.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"J.K. Rowling", // Spaces removed
				`<break time="500ms"/>`,
			},
			wantNotContain: []string{
				"J. <break", // Should NOT have break between initials
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

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("AddSSMLBreaks() result should NOT contain %q\nGot: %s", notWant, result)
				}
			}
		})
	}
}
