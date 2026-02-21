package sanitize

import (
	"regexp"
	"strings"
	"unicode"
)

// LatinToRussianConverter handles conversion of Latin letters to Russian pronunciation
type LatinToRussianConverter struct {
	// Map of individual Latin letters to Russian pronunciation
	letterMap map[rune]string
}

// NewLatinToRussianConverter creates a new converter
func NewLatinToRussianConverter() *LatinToRussianConverter {
	return &LatinToRussianConverter{
		letterMap: map[rune]string{
			'A': "эй", 'a': "эй",
			'B': "би", 'b': "би",
			'C': "си", 'c': "си",
			'D': "ди", 'd': "ди",
			'E': "и", 'e': "и",
			'F': "эф", 'f': "эф",
			'G': "джи", 'g': "джи",
			'H': "эйч", 'h': "эйч",
			'I': "ай", 'i': "ай",
			'J': "джей", 'j': "джей",
			'K': "кей", 'k': "кей",
			'L': "эл", 'l': "эл",
			'M': "эм", 'm': "эм",
			'N': "эн", 'n': "эн",
			'O': "оу", 'o': "оу",
			'P': "пи", 'p': "пи",
			'Q': "кью", 'q': "кью",
			'R': "ар", 'r': "ар",
			'S': "эс", 's': "эс",
			'T': "ти", 't': "ти",
			'U': "ю", 'u': "ю",
			'V': "ви", 'v': "ви",
			'W': "дабл ю", 'w': "дабл ю",
			'X': "экс", 'x': "экс",
			'Y': "уай", 'y': "уай",
			'Z': "зет", 'z': "зет",
		},
	}
}

// ConvertLatinInRussianText converts Latin letters/abbreviations in Russian text to Russian pronunciation
// This handles cases like:
// - "DC" -> "ди си"
// - "NSA" -> "эн эс эй"
// - "DC-19" -> "ди си 19" (hyphen with number)
// - "L-3" -> "эл 3" (single letter with number)
func (c *LatinToRussianConverter) ConvertLatinInRussianText(text string) string {
	// Pattern to match Latin letter sequences (abbreviations)
	// Matches word-boundary Latin letters, optionally followed by hyphen and digits
	// Also matches digit+letter patterns like 5G, 4G
	// Examples: DC, NSA, FBI, DC-19, L-3, GHz, 5G, 4G
	pattern := regexp.MustCompile(`\b(\d+[A-Z]|[A-Z][A-Za-z]*)(-[\d\-A-Z]+)?\b`)

	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Check if this is surrounded by Cyrillic text (indicating Russian context)
		// If not in Russian context, leave as is
		if !c.isInRussianContext(text, match) {
			return match
		}

		// Split into letter part and optional suffix part (number or letter)
		parts := strings.SplitN(match, "-", 2)
		letterPart := parts[0]

		// Check if this is a digit+letter pattern (like 5G, 4G)
		digitLetterPattern := regexp.MustCompile(`^(\d+)([A-Z]+)$`)
		if digitLetterPattern.MatchString(letterPart) {
			matches := digitLetterPattern.FindStringSubmatch(letterPart)
			if len(matches) == 3 {
				digitPart := matches[1]
				lettersPart := matches[2]

				// Convert the letter part
				var converted []string
				for _, r := range lettersPart {
					if pronunciation, ok := c.letterMap[r]; ok {
						converted = append(converted, pronunciation)
					}
				}

				return digitPart + " " + strings.Join(converted, " ")
			}
		}

		// Only convert if it's all uppercase or a known mixed-case abbreviation
		if !c.isConvertibleAbbreviation(letterPart) {
			return match
		}

		// Convert letters to Russian pronunciation
		var converted []string
		for _, r := range letterPart {
			if pronunciation, ok := c.letterMap[r]; ok {
				converted = append(converted, pronunciation)
			} else {
				// If we can't convert, return original
				return match
			}
		}

		result := strings.Join(converted, " ")

		// If there's a suffix part (e.g., "-19" in "DC-19" or "-19-A" in "DC-19-A")
		if len(parts) > 1 {
			suffix := parts[1]
			// Check if suffix contains more letters that need conversion
			if strings.ContainsAny(suffix, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
				// Complex suffix like "19-A", recursively convert the letter part
				subParts := strings.Split(suffix, "-")
				for i, subPart := range subParts {
					if regexp.MustCompile(`^[A-Z]+$`).MatchString(subPart) {
						// Convert this letter part
						var subConverted []string
						for _, r := range subPart {
							if pronunciation, ok := c.letterMap[r]; ok {
								subConverted = append(subConverted, pronunciation)
							}
						}
						subParts[i] = strings.Join(subConverted, " ")
					}
				}
				result += " " + strings.Join(subParts, "-")
			} else {
				// Simple numeric suffix
				result += " " + suffix
			}
		}

		return result
	})

	return result
}

// isConvertibleAbbreviation checks if a string should be converted
// Returns true for all-uppercase abbreviations or known mixed-case ones
func (c *LatinToRussianConverter) isConvertibleAbbreviation(s string) bool {
	// Check if all uppercase
	allUpper := true
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			allUpper = false
			break
		}
	}

	if allUpper {
		return true
	}

	// Known mixed-case abbreviations
	mixedCase := map[string]bool{
		"GHz":  true,
		"MHz":  true,
		"KHz":  true,
		"kHz":  true,
		"WiFi": true,
		"PhD":  true,
		"HTML": true,
		"CSS":  true,
	}

	return mixedCase[s]
}

// isInRussianContext checks if a Latin sequence appears in Russian text context
// by looking for Cyrillic characters nearby
func (c *LatinToRussianConverter) isInRussianContext(fullText, match string) bool {
	// Find the position of the match in the full text
	index := strings.Index(fullText, match)
	if index == -1 {
		return false
	}

	// Check characters before and after the match (within a reasonable window)
	windowSize := 50
	start := index - windowSize
	if start < 0 {
		start = 0
	}
	end := index + len(match) + windowSize
	if end > len(fullText) {
		end = len(fullText)
	}

	contextWindow := fullText[start:end]

	// Count Cyrillic vs Latin characters in the context
	cyrillicCount := 0
	latinCount := 0

	for _, r := range contextWindow {
		if isCyrillic(r) {
			cyrillicCount++
		} else if isLatin(r) {
			latinCount++
		}
	}

	// Require at least some Cyrillic presence
	if cyrillicCount == 0 {
		return false
	}

	// If there are more Cyrillic than Latin, definitely Russian context
	if cyrillicCount > latinCount {
		return true
	}

	// If Cyrillic is at least 20% of total letters, consider it Russian context
	// This handles cases like "Код HTML и CSS" where abbreviations dominate
	totalLetters := cyrillicCount + latinCount
	if totalLetters > 0 && float64(cyrillicCount)/float64(totalLetters) >= 0.2 {
		return true
	}

	return false
}

// hasOnlyCyrillic checks if a string contains only Cyrillic characters (and spaces/punctuation)
func hasOnlyCyrillic(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !isCyrillic(r) {
			return false
		}
	}
	return true
}

// hasAnyCyrillic checks if a string contains any Cyrillic characters
func hasAnyCyrillic(s string) bool {
	for _, r := range s {
		if isCyrillic(r) {
			return true
		}
	}
	return false
}
