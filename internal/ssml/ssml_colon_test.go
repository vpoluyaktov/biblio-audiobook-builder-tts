package ssml

import (
	"strings"
	"testing"
)

// TestColonReplacement tests that colons are replaced with spaces for TTS compatibility
func TestColonReplacement(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantText string
	}{
		{
			name:     "Simple colon replacement",
			text:     "красивую металлическую дощечку с надписью: Академик В.В.",
			wantText: "красивую металлическую дощечку с надписью Академик В.В.",
		},
		{
			name:     "Multiple colons",
			text:     "First: one. Second: two. Third: three.",
			wantText: "First one. Second two. Third three.",
		},
		{
			name:     "Colon at end",
			text:     "The answer is:",
			wantText: "The answer is ",
		},
		{
			name:     "No colons",
			text:     "This text has no colons at all.",
			wantText: "This text has no colons at all.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapTextInSSMLWithOptions(tt.text, SSMLOptions{})

			// Remove SSML tags to check the text content
			textContent := strings.ReplaceAll(result, "<speak>", "")
			textContent = strings.ReplaceAll(textContent, "</speak>", "")

			if !strings.Contains(textContent, tt.wantText) {
				t.Errorf("Expected text to contain %q\nGot: %s", tt.wantText, textContent)
			}

			// Verify colons are removed
			if strings.Contains(textContent, ":") {
				t.Errorf("Text should not contain colons\nGot: %s", textContent)
			}
		})
	}
}

func TestColonBreaks_Configurable(t *testing.T) {
	input := "В нашей системе: это важно: без сомнений"

	withBreaks := AddSSMLBreaks(input, SSMLOptions{
		ColonBreakMs: 250,
	})

	if strings.Count(withBreaks, `<break time="250ms"/>`) != 2 {
		t.Fatalf("expected 2 colon break tags, got %d in %s", strings.Count(withBreaks, `<break time="250ms"/>`), withBreaks)
	}
	if strings.Contains(withBreaks, ":") {
		t.Fatalf("expected no colons when colon breaks enabled, got %s", withBreaks)
	}

	legacy := AddSSMLBreaks(input, SSMLOptions{})
	if strings.Contains(legacy, `<break time="250ms"/>`) {
		t.Fatalf("unexpected colon break tag in legacy mode: %s", legacy)
	}
	if strings.Contains(legacy, ":") {
		t.Fatalf("expected legacy mode to replace colons with spaces, got %s", legacy)
	}
}

// TestColonReplacementWithBreaks tests colon replacement with SSML breaks
func TestColonReplacementWithBreaks(t *testing.T) {
	tests := []struct {
		name             string
		text             string
		opts             SSMLOptions
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Colon replaced with sentence breaks",
			text: "красивую дощечку с надписью: Академик В.В. Смагорин. Эксперт осматривал тело.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			shouldContain: []string{
				"надписью Академик", // Colon replaced with space
				"В.В. Смагорин",
				`<break time="500ms"/>`,
			},
			shouldNotContain: []string{
				"надписью:", // Should not have colon
			},
		},
		{
			name: "Multiple colons with breaks",
			text: "First: one. Second: two.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			shouldContain: []string{
				"First one",
				"Second two",
				`<break time="500ms"/>`,
			},
			shouldNotContain: []string{
				":",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapTextInSSMLWithOptions(tt.text, tt.opts)

			for _, want := range tt.shouldContain {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain %q\nGot: %s", want, result)
				}
			}

			for _, notWant := range tt.shouldNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("Result should NOT contain %q\nGot: %s", notWant, result)
				}
			}
		})
	}
}

// TestAddSSMLBreaks_ColonReplacement tests colon replacement in AddSSMLBreaks
func TestAddSSMLBreaks_ColonReplacement(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		opts     SSMLOptions
		wantText string
	}{
		{
			name:     "Colon replaced in AddSSMLBreaks",
			text:     "красивую дощечку с надписью: Академик В.В. Смагорин.",
			opts:     SSMLOptions{SentenceBreakMs: 500},
			wantText: "надписью Академик", // Colon replaced
		},
		{
			name:     "Multiple colons replaced",
			text:     "First: one. Second: two.",
			opts:     SSMLOptions{SentenceBreakMs: 500},
			wantText: "First one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddSSMLBreaks(tt.text, tt.opts)

			if !strings.Contains(result, tt.wantText) {
				t.Errorf("Expected result to contain %q\nGot: %s", tt.wantText, result)
			}

			// Verify no colons remain
			if strings.Contains(result, ":") {
				t.Errorf("Result should not contain colons\nGot: %s", result)
			}
		})
	}
}
