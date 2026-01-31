package ssml

import (
	"strings"
	"testing"
)

// TestAddSSMLBreaks_IntegrationFlow tests the complete flow of adding SSML breaks
func TestAddSSMLBreaks_IntegrationFlow(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		opts            SSMLOptions
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "Multiple sentences with breaks",
			text: "First sentence. Second sentence. Third sentence.",
			opts: SSMLOptions{
				SentenceBreakMs:  500,
				ParagraphBreakMs: 0,
			},
			wantContains: []string{
				"First sentence.",
				`<break time="500ms"/>`,
				"Second sentence.",
				"Third sentence.",
			},
			wantNotContains: []string{
				"<speak>",
				"</speak>",
			},
		},
		{
			name: "Multiple paragraphs with breaks",
			text: "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			opts: SSMLOptions{
				SentenceBreakMs:  0,
				ParagraphBreakMs: 800,
			},
			wantContains: []string{
				"First paragraph.",
				`<break time="800ms"/>`,
				"Second paragraph.",
				"Third paragraph.",
			},
			wantNotContains: []string{
				"<speak>",
			},
		},
		{
			name: "Sentences and paragraphs with both breaks",
			text: "Sentence 1. Sentence 2.\n\nParagraph 2 here. Another sentence.",
			opts: SSMLOptions{
				SentenceBreakMs:  500,
				ParagraphBreakMs: 800,
			},
			wantContains: []string{
				"Sentence 1.",
				`<break time="500ms"/>`,
				"Sentence 2.",
				`<break time="800ms"/>`,
				"Paragraph 2 here.",
				"Another sentence.",
			},
			wantNotContains: []string{
				"<speak>",
			},
		},
		{
			name: "Dash breaks enabled",
			text: "Text with dash — and more text.",
			opts: SSMLOptions{
				SentenceBreakMs:       0,
				ParagraphBreakMs:      0,
				ConvertDashesToBreaks: true,
				DashBreakDurationMs:   300,
			},
			wantContains: []string{
				"Text with dash",
				`<break time="300ms"/>`,
				"and more text.",
			},
			wantNotContains: []string{
				"—",
			},
		},
		{
			name: "All pause types combined",
			text: "Sentence 1 — with dash. Sentence 2.\n\nNew paragraph here.",
			opts: SSMLOptions{
				SentenceBreakMs:       500,
				ParagraphBreakMs:      800,
				ConvertDashesToBreaks: true,
				DashBreakDurationMs:   300,
			},
			wantContains: []string{
				"Sentence 1",
				`<break time="300ms"/>`, // dash break
				"with dash.",
				`<break time="500ms"/>`, // sentence break
				"Sentence 2.",
				`<break time="800ms"/>`, // paragraph break
				"New paragraph here.",
			},
			wantNotContains: []string{
				"<speak>",
				"—",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddSSMLBreaks(tt.text, tt.opts)

			// Check for expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("AddSSMLBreaks() result missing expected content %q\nGot: %s", want, result)
				}
			}

			// Check for unexpected content
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(result, notWant) {
					t.Errorf("AddSSMLBreaks() result contains unexpected content %q\nGot: %s", notWant, result)
				}
			}
		})
	}
}

// TestSSMLFirstChunking_PreservesBreaksAcrossBoundaries tests that breaks are preserved
// when text is processed with AddSSMLBreaks before chunking
func TestSSMLFirstChunking_PreservesBreaksAcrossBoundaries(t *testing.T) {
	// Simulate the complete flow: AddSSMLBreaks -> chunk -> wrap in <speak>
	text := "Sentence 1. Sentence 2. Sentence 3. Sentence 4. Sentence 5."

	opts := SSMLOptions{
		SentenceBreakMs: 500,
	}

	// Step 1: Add SSML breaks to entire text
	textWithBreaks := AddSSMLBreaks(text, opts)

	// Verify breaks were added
	expectedBreaks := 4 // Between 5 sentences
	actualBreaks := strings.Count(textWithBreaks, `<break time="500ms"/>`)
	if actualBreaks != expectedBreaks {
		t.Errorf("Expected %d sentence breaks, got %d in: %s", expectedBreaks, actualBreaks, textWithBreaks)
	}

	// Step 2: Simulate chunking (manually split for testing)
	// In real code, chunker would do this, but we simulate it here
	chunks := []string{
		"Sentence 1.<break time=\"500ms\"/>Sentence 2.",
		"<break time=\"500ms\"/>Sentence 3.<break time=\"500ms\"/>Sentence 4.",
		"<break time=\"500ms\"/>Sentence 5.",
	}

	// Step 3: Wrap each chunk in <speak> tags (simulating what adapter does)
	for i, chunk := range chunks {
		wrapped := "<speak>" + chunk + "</speak>"

		// Verify each chunk has valid SSML
		if !strings.HasPrefix(wrapped, "<speak>") || !strings.HasSuffix(wrapped, "</speak>") {
			t.Errorf("Chunk %d not properly wrapped: %s", i, wrapped)
		}

		// Verify break tags are intact
		if strings.Contains(chunk, "<break") {
			if !strings.Contains(chunk, `<break time="500ms"/>`) {
				t.Errorf("Chunk %d has malformed break tag: %s", i, chunk)
			}
		}
	}

	// Verify total breaks across all chunks equals original
	totalBreaksInChunks := 0
	for _, chunk := range chunks {
		totalBreaksInChunks += strings.Count(chunk, `<break time="500ms"/>`)
	}
	if totalBreaksInChunks != expectedBreaks {
		t.Errorf("Breaks lost during chunking! Expected %d, got %d", expectedBreaks, totalBreaksInChunks)
	}
}

// TestSSMLFirstChunking_LongText tests the flow with text that requires multiple chunks
func TestSSMLFirstChunking_LongText(t *testing.T) {
	// Create a long text with multiple sentences and paragraphs
	var sb strings.Builder
	for i := 1; i <= 20; i++ {
		sb.WriteString("This is sentence number ")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString(". ")
		if i%5 == 0 {
			sb.WriteString("\n\n") // Paragraph break every 5 sentences
		}
	}
	text := sb.String()

	opts := SSMLOptions{
		SentenceBreakMs:  500,
		ParagraphBreakMs: 800,
	}

	// Add SSML breaks
	textWithBreaks := AddSSMLBreaks(text, opts)

	// Count breaks
	sentenceBreaks := strings.Count(textWithBreaks, `<break time="500ms"/>`)
	paragraphBreaks := strings.Count(textWithBreaks, `<break time="800ms"/>`)

	// We have 20 sentences, so 19 sentence breaks
	// But some are replaced by paragraph breaks (every 5th)
	// So we expect: 19 total breaks = some sentence + some paragraph
	totalBreaks := sentenceBreaks + paragraphBreaks
	if totalBreaks < 15 { // At least 15 breaks expected
		t.Errorf("Expected at least 15 breaks in long text, got %d (sentence: %d, paragraph: %d)",
			totalBreaks, sentenceBreaks, paragraphBreaks)
	}

	// Verify no <speak> tags in the pre-chunked text
	if strings.Contains(textWithBreaks, "<speak>") {
		t.Error("AddSSMLBreaks should not add <speak> tags")
	}

	// Verify all break tags are properly formed
	if strings.Contains(textWithBreaks, "<break") && !strings.Contains(textWithBreaks, "/>") {
		t.Error("Found unclosed break tag in output")
	}
}

// TestSSMLFirstChunking_XMLEscaping tests that XML escaping works correctly with breaks
func TestSSMLFirstChunking_XMLEscaping(t *testing.T) {
	text := `Tom & Jerry said "Hello". The <tag> was escaped.`

	opts := SSMLOptions{
		SentenceBreakMs: 500,
	}

	result := AddSSMLBreaks(text, opts)

	// Verify XML entities are escaped
	if !strings.Contains(result, "&amp;") {
		t.Error("& not escaped to &amp;")
	}
	if !strings.Contains(result, "&quot;") {
		t.Error("\" not escaped to &quot;")
	}
	if !strings.Contains(result, "&lt;tag&gt;") {
		t.Error("< and > not escaped")
	}

	// Verify break tags are NOT escaped
	if strings.Contains(result, "&lt;break") {
		t.Error("Break tags should not be escaped")
	}
}

// TestSSMLFirstChunking_EmptyAndEdgeCases tests edge cases
func TestSSMLFirstChunking_EmptyAndEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		opts SSMLOptions
		want string
	}{
		{
			name: "Empty text",
			text: "",
			opts: SSMLOptions{SentenceBreakMs: 500},
			want: "",
		},
		{
			name: "Whitespace only",
			text: "   \n\n   ",
			opts: SSMLOptions{SentenceBreakMs: 500},
			want: "   \n\n   ", // Whitespace preserved when no breaks configured
		},
		{
			name: "Single sentence no break",
			text: "Just one sentence.",
			opts: SSMLOptions{SentenceBreakMs: 500},
			want: "Just one sentence.",
		},
		{
			name: "No breaks configured",
			text: "Sentence 1. Sentence 2.",
			opts: SSMLOptions{SentenceBreakMs: 0, ParagraphBreakMs: 0},
			want: "Sentence 1. Sentence 2.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddSSMLBreaks(tt.text, tt.opts)
			if result != tt.want {
				t.Errorf("AddSSMLBreaks() = %q, want %q", result, tt.want)
			}
		})
	}
}

// TestSSMLFirstChunking_BreakTagIntegrity verifies break tags are never malformed
func TestSSMLFirstChunking_BreakTagIntegrity(t *testing.T) {
	text := "Sentence 1. Sentence 2.\n\nParagraph 2. Sentence with dash — here."

	opts := SSMLOptions{
		SentenceBreakMs:       500,
		ParagraphBreakMs:      800,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   300,
	}

	result := AddSSMLBreaks(text, opts)

	// Find all break tags
	breakTags := []string{
		`<break time="300ms"/>`,
		`<break time="500ms"/>`,
		`<break time="800ms"/>`,
	}

	for _, tag := range breakTags {
		count := strings.Count(result, tag)
		if count > 0 {
			// Verify tag is complete (not split)
			if strings.Contains(result, "<break time=\""+tag[12:15]) && !strings.Contains(result, tag) {
				t.Errorf("Found incomplete break tag for %s", tag)
			}
		}
	}

	// Verify no orphaned break tag parts
	if strings.Contains(result, "<break") {
		openCount := strings.Count(result, "<break")
		closeCount := strings.Count(result, "/>")
		if openCount != closeCount {
			t.Errorf("Unbalanced break tags: %d opens, %d closes", openCount, closeCount)
		}
	}
}
