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
// Performs letter-by-letter transliteration for all Latin letter sequences
// Examples: "FBI" -> "эф би ай", "USB" -> "ю эс би", "A" -> "эй"
func (c *LatinToRussianConverter) ConvertLatinInRussianText(text string) string {
	// Pattern to match Latin letter sequences (1+ letters)
	pattern := regexp.MustCompile(`\b[A-Za-z]+\b`)

	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
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
