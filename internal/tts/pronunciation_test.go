package tts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPronunciationDictionary(t *testing.T) {
	d := NewPronunciationDictionary()
	if d == nil {
		t.Error("NewPronunciationDictionary should not return nil")
	}
	if d.RuleCount() != 0 {
		t.Errorf("New dictionary should have 0 rules, got %d", d.RuleCount())
	}
}

func TestAddRule(t *testing.T) {
	d := NewPronunciationDictionary()

	err := d.AddRule(`\bMr\.`, "Mister")
	if err != nil {
		t.Errorf("AddRule failed: %v", err)
	}

	if d.RuleCount() != 1 {
		t.Errorf("Expected 1 rule, got %d", d.RuleCount())
	}
}

func TestAddRuleInvalidRegex(t *testing.T) {
	d := NewPronunciationDictionary()

	err := d.AddRule(`[invalid`, "replacement")
	if err == nil {
		t.Error("AddRule should fail with invalid regex")
	}
}

func TestApply(t *testing.T) {
	d := NewPronunciationDictionary()
	d.AddRule(`\bMr\.`, "Mister")
	d.AddRule(`\bDr\.`, "Doctor")

	tests := []struct {
		input    string
		expected string
	}{
		{"Mr. Smith", "Mister Smith"},
		{"Dr. Jones", "Doctor Jones"},
		{"Mr. and Dr. Brown", "Mister and Doctor Brown"},
		{"No replacements here", "No replacements here"},
	}

	for _, tt := range tests {
		result := d.Apply(tt.input)
		if result != tt.expected {
			t.Errorf("Apply(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestApplyWithCaptures(t *testing.T) {
	d := NewPronunciationDictionary()
	d.AddRule(`\$(\d+)`, "$1 dollars")
	d.AddRule(`(\d+)%`, "$1 percent")

	tests := []struct {
		input    string
		expected string
	}{
		{"$100", "100 dollars"},
		{"50%", "50 percent"},
		{"$25 and 10%", "25 dollars and 10 percent"},
	}

	for _, tt := range tests {
		result := d.Apply(tt.input)
		if result != tt.expected {
			t.Errorf("Apply(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "pronunciation.txt")

	content := `# This is a comment
\bMr\. -> Mister # Expand Mr.
\bDr\. -> Doctor

# Another comment
\$(\d+) -> $1 dollars
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	d := NewPronunciationDictionary()
	err = d.LoadFromFile(filePath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if d.RuleCount() != 3 {
		t.Errorf("Expected 3 rules, got %d", d.RuleCount())
	}

	// Test application
	result := d.Apply("Mr. Smith paid $50")
	expected := "Mister Smith paid 50 dollars"
	if result != expected {
		t.Errorf("Apply() = %q, want %q", result, expected)
	}
}

func TestLoadFromFileInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.txt")

	content := `invalid line without arrow`
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	d := NewPronunciationDictionary()
	err = d.LoadFromFile(filePath)
	if err == nil {
		t.Error("LoadFromFile should fail with invalid format")
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	d := NewPronunciationDictionary()
	err := d.LoadFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("LoadFromFile should fail with nonexistent file")
	}
}

func TestClear(t *testing.T) {
	d := NewPronunciationDictionary()
	d.AddRule(`test`, "replacement")

	if d.RuleCount() != 1 {
		t.Error("Should have 1 rule before clear")
	}

	d.Clear()

	if d.RuleCount() != 0 {
		t.Error("Should have 0 rules after clear")
	}
}

func TestGetDefaultRules(t *testing.T) {
	rules := GetDefaultRules()
	if len(rules) == 0 {
		t.Error("GetDefaultRules should return some rules")
	}

	// Verify all default rules have valid regex
	for _, rule := range rules {
		d := NewPronunciationDictionary()
		err := d.AddRule(rule.Pattern, rule.Replacement)
		if err != nil {
			t.Errorf("Default rule has invalid regex: %s - %v", rule.Pattern, err)
		}
	}
}
