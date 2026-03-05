package sanitize

import (
	"testing"
)

func TestYearOfBirthAbbreviation(t *testing.T) {
	sanitizer := NewTextSanitizer()
	sanitizer.LoadDefaultRules()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "year of birth with dots",
			input:    "1950 г.р.",
			expected: "1950 года рождения ",
		},
		{
			name:     "year of birth with spaces",
			input:    "1968 г. р.",
			expected: "1968 года рождения ",
		},
		{
			name:     "year of birth without dots",
			input:    "1985 г р",
			expected: "1985 года рождения ",
		},
		{
			name:     "year of birth in sentence",
			input:    "Родился в 1950 г.р. в Москве",
			expected: "Родился в 1950 года рождения в Москве",
		},
		{
			name:     "weight in grams should still work",
			input:    "Вес 500 г",
			expected: "Вес 500 граммов ",
		},
		{
			name:     "weight in kilograms should still work",
			input:    "Вес 2 кг",
			expected: "Вес 2 килограммов ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeWithOptions(tt.input, "ru", false)
			if result != tt.expected {
				t.Errorf("Year of birth abbreviation test failed\nInput:    %q\nExpected: %q\nGot:      %q",
					tt.input, tt.expected, result)
			}
		})
	}
}
