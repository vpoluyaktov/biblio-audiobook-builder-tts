package sanitize

import (
	"regexp"
	"strings"
)

// LatinToRussianConverter handles letter-by-letter conversion of Latin to Russian pronunciation
type LatinToRussianConverter struct {
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

// ConvertLatinInRussianText converts Latin letters in Russian text to Russian pronunciation
// Only performs letter-by-letter transliteration for uppercase Latin sequences in Russian context
// Examples: "FBI" -> "эф би ай", "USB" -> "ю эс би"
func (c *LatinToRussianConverter) ConvertLatinInRussianText(text string) string {
	// Pattern to match uppercase Latin letter sequences (2+ letters)
	pattern := regexp.MustCompile(`\b[A-Z]{2,}\b`)

	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Check if this is in Russian context
		if !c.isInRussianContext(text, match) {
			return match
		}

		// Convert each letter to Russian pronunciation
		var converted []string
		for _, r := range match {
			if pronunciation, ok := c.letterMap[r]; ok {
				converted = append(converted, pronunciation)
			} else {
				// If we can't convert a letter, return original
				return match
			}
		}

		return strings.Join(converted, " ")
	})

	return result
}

// isInRussianContext checks if a Latin sequence appears in Russian text context
func (c *LatinToRussianConverter) isInRussianContext(fullText, match string) bool {
	index := strings.Index(fullText, match)
	if index == -1 {
		return false
	}

	// Check characters before and after the match (within a window)
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
	totalLetters := cyrillicCount + latinCount
	if totalLetters > 0 && float64(cyrillicCount)/float64(totalLetters) >= 0.2 {
		return true
	}

	return false
}
