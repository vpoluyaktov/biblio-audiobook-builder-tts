package ssml

import (
	"strings"
	"testing"
)

// TestRussianInitialsProtection tests Russian-specific abbreviated initials patterns
func TestRussianInitialsProtection(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedBreaks int
		shouldContain  []string
	}{
		{
			name:           "Russian initials В.В.",
			text:           "Академик В.В. Смагорин работал здесь.",
			expectedBreaks: 0,
			shouldContain:  []string{"В.В. Смагорин"},
		},
		{
			name:           "Russian initials with sentence after",
			text:           "Академик В.В. Смагорин. Судебно-медицинский эксперт.",
			expectedBreaks: 1,
			shouldContain:  []string{"В.В. Смагорин"},
		},
		{
			name:           "Initials with spaces В. В.",
			text:           "Академик В. В. Смагорин работал здесь.",
			expectedBreaks: 0,
			shouldContain:  []string{"В.В. Смагорин"},
		},
		{
			name:           "Multiple people with initials",
			text:           "А.А. Иванов и Б.Б. Петров встретились.",
			expectedBreaks: 0,
			shouldContain:  []string{"А.А. Иванов", "Б.Б. Петров"},
		},
		{
			name:           "Complex example from book",
			text:           "Академик В.В. Смагорин. Судебно-медицинский эксперт осматривал тело.",
			expectedBreaks: 1,
			shouldContain:  []string{"В.В. Смагорин"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapTextInSSMLWithOptions(tt.text, SSMLOptions{
				SentenceBreakMs: 500,
			})

			breakCount := strings.Count(result, `<break time="500ms"/>`)
			if breakCount != tt.expectedBreaks {
				t.Errorf("Expected %d sentence breaks, got %d\nInput: %s\nOutput: %s",
					tt.expectedBreaks, breakCount, tt.text, result)
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q\nInput: %s\nOutput: %s",
						expected, tt.text, result)
				}
			}
		})
	}
}

// TestRussianSplitIntoSentences_Initials tests Russian sentence splitting with initials
func TestRussianSplitIntoSentences_Initials(t *testing.T) {
	tests := []struct {
		name              string
		text              string
		expectedSentences int
		shouldContain     []string
	}{
		{
			name:              "Russian initials not split",
			text:              "Академик В.В. Смагорин работал.",
			expectedSentences: 1,
			shouldContain:     []string{"В.В. Смагорин"},
		},
		{
			name:              "Two sentences with Russian initials",
			text:              "Академик В.В. Смагорин. Он работал здесь.",
			expectedSentences: 2,
			shouldContain:     []string{"В.В. Смагорин", "Он работал"},
		},
		{
			name:              "Multiple Russian initials in one sentence",
			text:              "А.А. Иванов и Б.Б. Петров встретились.",
			expectedSentences: 1,
			shouldContain:     []string{"А.А. Иванов", "Б.Б. Петров"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentences(tt.text)
			if len(sentences) != tt.expectedSentences {
				t.Errorf("Expected %d sentences, got %d\nInput: %s\nSentences: %v",
					tt.expectedSentences, len(sentences), tt.text, sentences)
			}

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

// TestRussianAddSSMLBreaks_Initials tests Russian initials with AddSSMLBreaks
func TestRussianAddSSMLBreaks_Initials(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		opts           SSMLOptions
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Russian initials preserved",
			text: "Академик В.В. Смагорин. Эксперт осматривал тело.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"В.В. Смагорин",
				`<break time="500ms"/>`,
				"Эксперт осматривал",
			},
			wantNotContain: []string{
				"В. <break",
			},
		},
		{
			name: "Russian initials with spaces normalized",
			text: "Профессор И. И. Петров. Он работал.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			wantContains: []string{
				"И.И. Петров",
				`<break time="500ms"/>`,
			},
			wantNotContain: []string{
				"И. <break",
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
