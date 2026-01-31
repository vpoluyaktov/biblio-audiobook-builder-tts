package tts

import (
	"strings"
	"testing"
)

func TestChunker_SSMLAwareness(t *testing.T) {
	config := &ChunkerConfig{
		Mode:          ChunkBySize,
		MaxChunkSize:  50, // Small size to force splitting
		PreserveWords: true,
	}
	chunker := NewChunker(config)

	// Test case 1: Ensure SSML tags are not split
	text := `Sentence 1.<break time="500ms"/>Sentence 2.<break time="500ms"/>Sentence 3.`
	chunks := chunker.Chunk(text)

	for i, chunk := range chunks {
		// Verify no chunk starts with a closing tag or ends with an opening tag
		if strings.HasPrefix(strings.TrimSpace(chunk), "/>") {
			t.Errorf("Chunk %d starts with closing tag: %q", i, chunk)
		}
		if strings.HasSuffix(strings.TrimSpace(chunk), "<break") {
			t.Errorf("Chunk %d ends with incomplete opening tag: %q", i, chunk)
		}
		if strings.Contains(chunk, "<") && !strings.Contains(chunk, ">") {
			t.Errorf("Chunk %d has unclosed tag: %q", i, chunk)
		}
		if strings.Contains(chunk, ">") && !strings.Contains(chunk, "<") {
			t.Errorf("Chunk %d has unopened tag: %q", i, chunk)
		}
	}

	// Test case 2: Verify complete SSML tags stay together
	text2 := `Short.<break time="500ms"/>This is a longer sentence that might need splitting.`
	chunks2 := chunker.Chunk(text2)

	for i, chunk := range chunks2 {
		// Count opening and closing brackets
		openCount := strings.Count(chunk, "<break")
		closeCount := strings.Count(chunk, "/>")
		
		// Each chunk should have balanced tags (or none)
		if openCount != closeCount {
			t.Errorf("Chunk %d has unbalanced SSML tags (open: %d, close: %d): %q", 
				i, openCount, closeCount, chunk)
		}
	}
}

func TestChunker_isInsideSSMLTag(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	tests := []struct {
		text     string
		pos      int
		expected bool
		desc     string
	}{
		{
			text:     `Hello<break time="500ms"/>World`,
			pos:      5,
			expected: false,
			desc:     "Position before tag",
		},
		{
			text:     `Hello<break time="500ms"/>World`,
			pos:      6,
			expected: true,
			desc:     "Position at tag start",
		},
		{
			text:     `Hello<break time="500ms"/>World`,
			pos:      15,
			expected: true,
			desc:     "Position inside tag",
		},
		{
			text:     `Hello<break time="500ms"/>World`,
			pos:      26,
			expected: false,
			desc:     "Position after tag",
		},
		{
			text:     `No tags here`,
			pos:      5,
			expected: false,
			desc:     "No tags in text",
		},
		{
			text:     `<break time="500ms"/>`,
			pos:      10,
			expected: true,
			desc:     "Inside tag at start of text",
		},
		{
			text:     `Text<break time="500ms"/>`,
			pos:      25,
			expected: false,
			desc:     "After tag at end of text",
		},
	}

	for _, tt := range tests {
		result := chunker.isInsideSSMLTag(tt.text, tt.pos)
		if result != tt.expected {
			t.Errorf("%s: isInsideSSMLTag(%q, %d) = %v, want %v",
				tt.desc, tt.text, tt.pos, result, tt.expected)
		}
	}
}

func TestChunker_findBreakPoint_WithSSML(t *testing.T) {
	config := &ChunkerConfig{
		Mode:          ChunkBySize,
		MaxChunkSize:  50,
		PreserveWords: true,
	}
	chunker := NewChunker(config)

	// Text with SSML tag right after a sentence
	text := `Sentence 1.<break time="500ms"/>Sentence 2 is longer.`
	
	// Try to find break point around position 40
	breakPoint := chunker.findBreakPoint(text, 40)
	
	// The break point should NOT be inside the SSML tag
	if chunker.isInsideSSMLTag(text, breakPoint) {
		t.Errorf("findBreakPoint returned position %d which is inside an SSML tag in: %q",
			breakPoint, text)
	}
	
	// Verify the resulting chunks are valid
	chunk1 := text[:breakPoint]
	chunk2 := text[breakPoint:]
	
	// Neither chunk should have broken SSML tags
	if strings.HasSuffix(chunk1, "<break") || strings.HasSuffix(chunk1, "<") {
		t.Errorf("Chunk 1 ends with incomplete tag: %q", chunk1)
	}
	if strings.HasPrefix(chunk2, "/>") || strings.HasPrefix(chunk2, ">") {
		t.Errorf("Chunk 2 starts with tag fragment: %q", chunk2)
	}
}
