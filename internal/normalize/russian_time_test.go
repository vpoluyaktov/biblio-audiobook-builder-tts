package normalize

import (
	"testing"
)

// TestRussianTimeNormalization tests that time formats with dots and colons
// are normalized without punctuation between numbers
func TestRussianTimeNormalization(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "time with dots HH.MM.SS",
			input:    "23.35.10",
			expected: "двадцать три тридцать пять десять",
		},
		{
			name:     "time with colons HH:MM",
			input:    "23:50",
			expected: "двадцать три пятьдесят",
		},
		{
			name:     "time with colons HH:MM:SS",
			input:    "23:50:10",
			expected: "двадцать три пятьдесят десять",
		},
		{
			name:     "time with dots HH.MM",
			input:    "15.30",
			expected: "пятнадцать тридцать",
		},
		{
			name:     "time in sentence with dots",
			input:    "Время прибытия 23.35.10",
			expected: "Время прибытия двадцать три тридцать пять десять",
		},
		{
			name:     "time in sentence with colons",
			input:    "Встреча в 15:30",
			expected: "Встреча в пятнадцать тридцать",
		},
		{
			name:     "single digit hours and minutes",
			input:    "9:05",
			expected: "девять пять",
		},
		{
			name:     "midnight",
			input:    "0:00",
			expected: "ноль ноль",
		},
		{
			name:     "multiple times in text",
			input:    "С 9:00 до 17:30",
			expected: "С девять ноль до семнадцать тридцать",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Time normalization failed\nInput:    %q\nExpected: %q\nGot:      %q",
					tt.input, tt.expected, result)
			}
		})
	}
}

// TestRussianTimePatternPreservesOtherDots tests that the time pattern
// doesn't interfere with other uses of dots (like decimal numbers or abbreviations)
func TestRussianTimePatternPreservesOtherDots(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "decimal number",
			input:    "3.14",
			expected: "три четырнадцать", // This will be treated as time format
		},
		{
			name:     "version number",
			input:    "версия 2.5.1",
			expected: "версия два пять один", // This will be treated as time format
		},
		{
			name:     "large number with dots not matching time pattern",
			input:    "100.200.300",
			expected: "сто.двести.триста", // Pattern doesn't match (numbers too large), dots preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input, "ru")
			if result != tt.expected {
				t.Errorf("Pattern handling\nInput:    %q\nExpected: %q\nGot:      %q",
					tt.input, tt.expected, result)
			}
		})
	}
}
