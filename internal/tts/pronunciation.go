package tts

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// PronunciationRule represents a single pronunciation replacement rule
type PronunciationRule struct {
	Pattern     *regexp.Regexp
	Replacement string
	Comment     string
}

// PronunciationDictionary manages text replacements for TTS
type PronunciationDictionary struct {
	rules []PronunciationRule
}

// NewPronunciationDictionary creates a new empty dictionary
func NewPronunciationDictionary() *PronunciationDictionary {
	return &PronunciationDictionary{
		rules: make([]PronunciationRule, 0),
	}
}

// LoadFromFile loads pronunciation rules from a file
// Format: pattern -> replacement # optional comment
// Lines starting with # are comments
// Empty lines are ignored
func (d *PronunciationDictionary) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open pronunciation file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule, err := d.parseLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		d.rules = append(d.rules, rule)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// parseLine parses a single rule line
func (d *PronunciationDictionary) parseLine(line string) (PronunciationRule, error) {
	var rule PronunciationRule

	// Extract comment if present
	if idx := strings.Index(line, " # "); idx != -1 {
		rule.Comment = strings.TrimSpace(line[idx+3:])
		line = line[:idx]
	}

	// Split by arrow
	parts := strings.SplitN(line, " -> ", 2)
	if len(parts) != 2 {
		return rule, fmt.Errorf("invalid format, expected 'pattern -> replacement'")
	}

	pattern := strings.TrimSpace(parts[0])
	replacement := strings.TrimSpace(parts[1])

	// Compile regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return rule, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	rule.Pattern = re
	rule.Replacement = replacement

	return rule, nil
}

// AddRule adds a pronunciation rule
func (d *PronunciationDictionary) AddRule(pattern, replacement string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	d.rules = append(d.rules, PronunciationRule{
		Pattern:     re,
		Replacement: replacement,
	})

	return nil
}

// Apply applies all pronunciation rules to the text
func (d *PronunciationDictionary) Apply(text string) string {
	result := text
	for _, rule := range d.rules {
		result = rule.Pattern.ReplaceAllString(result, rule.Replacement)
	}
	return result
}

// RuleCount returns the number of loaded rules
func (d *PronunciationDictionary) RuleCount() int {
	return len(d.rules)
}

// Clear removes all rules
func (d *PronunciationDictionary) Clear() {
	d.rules = make([]PronunciationRule, 0)
}

// GetDefaultRules returns common pronunciation fixes
func GetDefaultRules() []struct {
	Pattern     string
	Replacement string
	Comment     string
} {
	return []struct {
		Pattern     string
		Replacement string
		Comment     string
	}{
		// Common abbreviations
		{`\bMr\.`, "Mister", "Expand Mr."},
		{`\bMrs\.`, "Missus", "Expand Mrs."},
		{`\bDr\.`, "Doctor", "Expand Dr."},
		{`\bSt\.`, "Saint", "Expand St."},
		{`\bvs\.`, "versus", "Expand vs."},
		{`\betc\.`, "etcetera", "Expand etc."},
		{`\be\.g\.`, "for example", "Expand e.g."},
		{`\bi\.e\.`, "that is", "Expand i.e."},

		// Numbers and symbols
		{`\$(\d+)`, "$1 dollars", "Dollar amounts"},
		{`(\d+)%`, "$1 percent", "Percentages"},
		{`&`, " and ", "Ampersand"},

		// Common mispronunciations
		{`(?i)\blinux\b`, "Linux", "Linux pronunciation"},
		{`(?i)\bgithub\b`, "GitHub", "GitHub pronunciation"},

		// Clean up multiple spaces
		{`\s+`, " ", "Normalize whitespace"},
	}
}
