package normalize

import (
	"testing"
)

func TestPartSeparatorDetector_DetectAndMark(t *testing.T) {
	detector := NewPartSeparatorDetector()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "asterisks with spaces",
			input:    "First part.\n* * *\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "triple asterisks",
			input:    "First part.\n***\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "triple asterisks with period",
			input:    "Part one.\n***.\nPart two.",
			expected: "Part one.\n" + PartSeparatorMarker + "\nPart two.",
		},
		{
			name:     "dashes with spaces",
			input:    "First part.\n- - -\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "triple dashes",
			input:    "First part.\n---\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "bullets",
			input:    "First part.\n• • •\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "tildes",
			input:    "First part.\n~ ~ ~\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "hashes",
			input:    "First part.\n# # #\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "dots with spaces",
			input:    "First part.\n. . .\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "underscores",
			input:    "First part.\n___\nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "separator with leading/trailing whitespace",
			input:    "First part.\n   * * *   \nSecond part.",
			expected: "First part.\n" + PartSeparatorMarker + "\nSecond part.",
		},
		{
			name:     "multiple separators",
			input:    "Part 1.\n***\nPart 2.\n---\nPart 3.",
			expected: "Part 1.\n" + PartSeparatorMarker + "\nPart 2.\n" + PartSeparatorMarker + "\nPart 3.",
		},
		{
			name:     "consecutive separators collapsed",
			input:    "Part 1.\n***\n---\nPart 2.",
			expected: "Part 1.\n" + PartSeparatorMarker + "\nPart 2.",
		},
		{
			name:     "no separator",
			input:    "Just regular text.\nWith multiple lines.\nNo separators here.",
			expected: "Just regular text.\nWith multiple lines.\nNo separators here.",
		},
		{
			name:     "asterisks in text not matched",
			input:    "The rating is *** out of *****.",
			expected: "The rating is *** out of *****.",
		},
		{
			name:     "long separator",
			input:    "First.\n*****\nSecond.",
			expected: "First.\n" + PartSeparatorMarker + "\nSecond.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectAndMark(tt.input)
			if result != tt.expected {
				t.Errorf("DetectAndMark() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSplitByMarker(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single marker",
			input:    "Part 1.\n" + PartSeparatorMarker + "\nPart 2.",
			expected: []string{"Part 1.", "Part 2."},
		},
		{
			name:     "multiple markers",
			input:    "Part 1.\n" + PartSeparatorMarker + "\nPart 2.\n" + PartSeparatorMarker + "\nPart 3.",
			expected: []string{"Part 1.", "Part 2.", "Part 3."},
		},
		{
			name:     "no marker",
			input:    "Just text without markers.",
			expected: []string{"Just text without markers."},
		},
		{
			name:     "empty parts filtered",
			input:    PartSeparatorMarker + "\nPart 1.\n" + PartSeparatorMarker + "\n" + PartSeparatorMarker,
			expected: []string{"Part 1."},
		},
		{
			name:     "whitespace trimmed",
			input:    "  Part 1.  \n" + PartSeparatorMarker + "\n  Part 2.  ",
			expected: []string{"Part 1.", "Part 2."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitByMarker(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitByMarker() returned %d parts, want %d", len(result), len(tt.expected))
				return
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("SplitByMarker()[%d] = %q, want %q", i, part, tt.expected[i])
				}
			}
		})
	}
}

func TestHasPartSeparators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "has separator",
			input:    "Text\n" + PartSeparatorMarker + "\nMore text",
			expected: true,
		},
		{
			name:     "no separator",
			input:    "Just regular text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPartSeparators(tt.input)
			if result != tt.expected {
				t.Errorf("HasPartSeparators() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCountPartSeparators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "no separators",
			input:    "Just text",
			expected: 0,
		},
		{
			name:     "one separator",
			input:    "Part 1\n" + PartSeparatorMarker + "\nPart 2",
			expected: 1,
		},
		{
			name:     "three separators",
			input:    "A\n" + PartSeparatorMarker + "\nB\n" + PartSeparatorMarker + "\nC\n" + PartSeparatorMarker + "\nD",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountPartSeparators(tt.input)
			if result != tt.expected {
				t.Errorf("CountPartSeparators() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCustomPatterns(t *testing.T) {
	// Test with custom patterns
	customPatterns := []string{
		`^\s*SCENE\s+BREAK\s*$`,
		`^\s*\[END\]\s*$`,
	}

	detector := NewPartSeparatorDetectorWithPatterns(customPatterns)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "custom scene break",
			input:    "First scene.\nSCENE BREAK\nSecond scene.",
			expected: "First scene.\n" + PartSeparatorMarker + "\nSecond scene.",
		},
		{
			name:     "custom end marker",
			input:    "Part 1.\n[END]\nPart 2.",
			expected: "Part 1.\n" + PartSeparatorMarker + "\nPart 2.",
		},
		{
			name:     "default patterns not matched",
			input:    "First.\n***\nSecond.",
			expected: "First.\n***\nSecond.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectAndMark(tt.input)
			if result != tt.expected {
				t.Errorf("DetectAndMark() = %q, want %q", result, tt.expected)
			}
		})
	}
}
