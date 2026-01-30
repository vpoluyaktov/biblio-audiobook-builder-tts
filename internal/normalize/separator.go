package normalize

import (
	"regexp"
	"strings"
)

// PartSeparatorMarker is inserted in place of detected separators
// This marker is used by the TTS processing to split content and insert silence
const PartSeparatorMarker = "\n{{PART_SEPARATOR}}\n"

// DefaultPartSeparatorPatterns contains common scene/part separator patterns
var DefaultPartSeparatorPatterns = []string{
	`^\s*\*\s*\*\s*\*\s*$`,     // * * *
	`^\s*\*{3,}\s*$`,           // *** or more
	`^\s*-\s*-\s*-\s*$`,        // - - -
	`^\s*-{3,}\s*$`,            // --- or more
	`^\s*•\s*•\s*•\s*$`,        // • • •
	`^\s*~\s*~\s*~\s*$`,        // ~ ~ ~
	`^\s*#\s*#\s*#\s*$`,        // # # #
	`^\s*\.\s*\.\s*\.\s*$`,     // . . .
	`^\s*○\s*○\s*○\s*$`,        // ○ ○ ○
	`^\s*●\s*●\s*●\s*$`,        // ● ● ●
	`^\s*◆\s*◆\s*◆\s*$`,        // ◆ ◆ ◆
	`^\s*◇\s*◇\s*◇\s*$`,        // ◇ ◇ ◇
	`^\s*□\s*□\s*□\s*$`,        // □ □ □
	`^\s*■\s*■\s*■\s*$`,        // ■ ■ ■
	`^\s*☆\s*☆\s*☆\s*$`,        // ☆ ☆ ☆
	`^\s*★\s*★\s*★\s*$`,        // ★ ★ ★
	`^\s*×\s*×\s*×\s*$`,        // × × ×
	`^\s*\+\s*\+\s*\+\s*$`,     // + + +
	`^\s*=\s*=\s*=\s*$`,        // = = =
	`^\s*_{3,}\s*$`,            // ___ or more
}

// PartSeparatorDetector detects and marks part/scene separators in text
type PartSeparatorDetector struct {
	patterns []*regexp.Regexp
}

// NewPartSeparatorDetector creates a new detector with default patterns
func NewPartSeparatorDetector() *PartSeparatorDetector {
	return NewPartSeparatorDetectorWithPatterns(DefaultPartSeparatorPatterns)
}

// NewPartSeparatorDetectorWithPatterns creates a detector with custom patterns
func NewPartSeparatorDetectorWithPatterns(patterns []string) *PartSeparatorDetector {
	d := &PartSeparatorDetector{
		patterns: make([]*regexp.Regexp, 0, len(patterns)),
	}

	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			d.patterns = append(d.patterns, re)
		}
	}

	return d
}

// DetectAndMark scans text for separator patterns and replaces them with markers
// Returns the modified text with separators replaced by PartSeparatorMarker
func (d *PartSeparatorDetector) DetectAndMark(text string) string {
	if len(d.patterns) == 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if d.isSeparatorLine(line) {
			// Replace separator with marker (avoid duplicate markers)
			if len(result) > 0 && result[len(result)-1] == PartSeparatorMarker {
				continue // Skip consecutive separators
			}
			result = append(result, PartSeparatorMarker)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// isSeparatorLine checks if a line matches any separator pattern
func (d *PartSeparatorDetector) isSeparatorLine(line string) bool {
	for _, re := range d.patterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// SplitByMarker splits text at PartSeparatorMarker locations
// Returns a slice of text parts (separators removed)
func SplitByMarker(text string) []string {
	parts := strings.Split(text, PartSeparatorMarker)

	// Clean up parts - trim whitespace and filter empty parts
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// HasPartSeparators checks if text contains any part separator markers
func HasPartSeparators(text string) bool {
	return strings.Contains(text, PartSeparatorMarker)
}

// CountPartSeparators returns the number of part separators in text
func CountPartSeparators(text string) int {
	return strings.Count(text, PartSeparatorMarker)
}
