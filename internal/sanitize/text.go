package sanitize

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// TextForTTS sanitizes text for TTS processing by normalizing problematic
// Unicode characters while preserving all readable text from any language.
// This should be applied during parsing stage so sanitized text is visible
// in output .txt files.
func TextForTTS(text string) string {
	// Replace common problematic Unicode characters with TTS-friendly equivalents
	// These are characters that often cause TTS engines to mispronounce or error
	replacements := map[string]string{
		// Dashes - replace with spoken equivalents
		"\u2014": " - ", // Em dash
		"\u2013": " - ", // En dash
		"\u2015": " - ", // Horizontal bar
		"\u2012": " - ", // Figure dash
		"\u2212": "-",   // Minus sign

		// Quotes - normalize to ASCII quotes
		"\u201C": `"`, // Left double quote
		"\u201D": `"`, // Right double quote
		"\u201E": `"`, // Double low-9 quote
		"\u2018": "'", // Left single quote
		"\u2019": "'", // Right single quote
		"\u201A": "'", // Single low-9 quote
		"\u00AB": `"`, // Left guillemet «
		"\u00BB": `"`, // Right guillemet »
		"\u2039": "'", // Single left guillemet ‹
		"\u203A": "'", // Single right guillemet ›

		// Ellipsis
		"\u2026": "...", // Horizontal ellipsis …

		// Spaces - normalize to regular space
		"\u00A0": " ", // Non-breaking space
		"\u2002": " ", // En space
		"\u2003": " ", // Em space
		"\u2009": " ", // Thin space
		"\u200B": "",  // Zero-width space
		"\u200C": "",  // Zero-width non-joiner
		"\u200D": "",  // Zero-width joiner
		"\uFEFF": "",  // BOM / zero-width no-break space

		// Bullets and markers
		"\u2022": "-", // Bullet •
		"\u2023": "-", // Triangular bullet ‣
		"\u2043": "-", // Hyphen bullet ⁃
		"\u25E6": "-", // White bullet ◦
		"\u00B7": ".", // Middle dot ·

		// Daggers and reference marks
		"\u2020": "", // Dagger †
		"\u2021": "", // Double dagger ‡
		"\u00B6": "", // Pilcrow ¶

		// Legal/trademark symbols - expand to words
		"\u00A9": "(c)",  // Copyright ©
		"\u00AE": "(R)",  // Registered ®
		"\u2122": "(TM)", // Trademark ™

		// Section sign
		"\u00A7": "Section ", // Section sign §

		// Math symbols - expand to words for better TTS
		"\u00B0": " degrees ",               // Degree °
		"\u00B1": " plus or minus ",         // Plus-minus ±
		"\u00D7": " times ",                 // Multiplication ×
		"\u00F7": " divided by ",            // Division ÷
		"\u2248": " approximately ",         // Almost equal ≈
		"\u2260": " not equal to ",          // Not equal ≠
		"\u2264": " less than or equal ",    // Less than or equal ≤
		"\u2265": " greater than or equal ", // Greater than or equal ≥
		"\u221E": " infinity ",              // Infinity ∞

		// Fractions - expand to words
		"\u00BC": " one quarter ",    // ¼
		"\u00BD": " one half ",       // ½
		"\u00BE": " three quarters ", // ¾
		"\u2153": " one third ",      // ⅓
		"\u2154": " two thirds ",     // ⅔
	}

	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Replace multiple consecutive periods with ellipsis-like pause
	multiPeriod := regexp.MustCompile(`\.{4,}`)
	result = multiPeriod.ReplaceAllString(result, "...")

	// Remove only control characters and non-printable characters
	// Keep ALL readable characters from any language
	var cleaned strings.Builder
	for _, r := range result {
		// Skip control characters (except common whitespace)
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			cleaned.WriteRune(' ')
		} else if unicode.Is(unicode.Co, r) { // Private Use Area
			cleaned.WriteRune(' ')
		} else if unicode.Is(unicode.Cs, r) { // Surrogate
			cleaned.WriteRune(' ')
		} else {
			// Keep all other characters (letters, numbers, punctuation from any language)
			cleaned.WriteRune(r)
		}
	}

	// Normalize whitespace
	whitespace := regexp.MustCompile(`\s+`)
	result = whitespace.ReplaceAllString(cleaned.String(), " ")

	return strings.TrimSpace(result)
}

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

// AddRule adds a pronunciation rule
func (d *PronunciationDictionary) AddRule(pattern, replacement string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
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

// TextSanitizer combines TTS sanitization with pronunciation dictionary
type TextSanitizer struct {
	dictionary *PronunciationDictionary
}

// NewTextSanitizer creates a new text sanitizer
func NewTextSanitizer() *TextSanitizer {
	return &TextSanitizer{
		dictionary: NewPronunciationDictionary(),
	}
}

// SetDictionary sets the pronunciation dictionary
func (s *TextSanitizer) SetDictionary(dict *PronunciationDictionary) {
	s.dictionary = dict
}

// GetDictionary returns the pronunciation dictionary
func (s *TextSanitizer) GetDictionary() *PronunciationDictionary {
	return s.dictionary
}

// Sanitize applies both TTS sanitization and pronunciation rules
func (s *TextSanitizer) Sanitize(text string) string {
	// First apply TTS sanitization (Unicode normalization)
	result := TextForTTS(text)

	// Then apply pronunciation dictionary rules
	if s.dictionary != nil && s.dictionary.RuleCount() > 0 {
		result = s.dictionary.Apply(result)
	}

	return result
}

// LoadDefaultRules loads the default pronunciation rules into the dictionary
func (s *TextSanitizer) LoadDefaultRules() {
	for _, rule := range GetDefaultRules() {
		s.dictionary.AddRule(rule.Pattern, rule.Replacement)
	}
}
