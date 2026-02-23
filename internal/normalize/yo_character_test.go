package normalize

import (
	"testing"
)

// TestYoCharacterNormalization tests that both ё and е spellings work correctly
func TestYoCharacterNormalization(t *testing.T) {
	p := NewProcessor()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "52 ребенка (without ё)",
			input:    "52 ребенка",
			expected: "пятьдесят два ребенка",
		},
		{
			name:     "52 ребёнка (with ё)",
			input:    "52 ребёнка",
			expected: "пятьдесят два ребёнка",
		},
		{
			name:     "Full sentence without ё",
			input:    "Погиб 71 человек, в том числе 52 ребенка",
			expected: "Погиб семьдесят один человек, в том числе пятьдесят два ребенка",
		},
		{
			name:     "Full sentence with ё",
			input:    "Погиб 71 человек, в том числе 52 ребёнка",
			expected: "Погиб семьдесят один человек, в том числе пятьдесят два ребёнка",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Process(%q, ru) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
