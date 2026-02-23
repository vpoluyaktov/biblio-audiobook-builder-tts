package sanitize

import (
	"regexp"
	"strings"
)

// LatinToRussianConverter handles conversion of Latin to Russian pronunciation
// - Uppercase letters (e.g., "FBI", "TSAC") are converted letter-by-letter
// - Lowercase/mixed-case words (e.g., "desent", "iPhone") are phonetically transliterated
type LatinToRussianConverter struct {
	letterMap   map[rune]string
	phoneticMap map[rune]string
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
		phoneticMap: map[rune]string{
			'a': "а", 'A': "а",
			'b': "б", 'B': "б",
			'c': "к", 'C': "к", // Default to 'к', context-sensitive handling in transliterateWord
			'd': "д", 'D': "д",
			'e': "е", 'E': "е",
			'f': "ф", 'F': "ф",
			'g': "г", 'G': "г",
			'h': "х", 'H': "х",
			'i': "и", 'I': "и",
			'j': "дж", 'J': "дж",
			'k': "к", 'K': "к",
			'l': "л", 'L': "л",
			'm': "м", 'M': "м",
			'n': "н", 'N': "н",
			'o': "о", 'O': "о",
			'p': "п", 'P': "п",
			'q': "к", 'Q': "к",
			'r': "р", 'R': "р",
			's': "с", 'S': "с",
			't': "т", 'T': "т",
			'u': "у", 'U': "у",
			'v': "в", 'V': "в",
			'w': "в", 'W': "в",
			'x': "кс", 'X': "кс",
			'y': "й", 'Y': "й",
			'z': "з", 'Z': "з",
		},
	}
}

// ConvertLatinInRussianText converts Latin letters in Russian text to Russian pronunciation
// - Uppercase-only words (e.g., "FBI", "TSAC") are converted letter-by-letter: "FBI" -> "эф би ай"
// - Lowercase or mixed-case words (e.g., "desent", "iPhone") are phonetically transliterated: "desent" -> "дисент"
func (c *LatinToRussianConverter) ConvertLatinInRussianText(text string) string {
	// Pattern to match Latin letter sequences (1+ letters)
	pattern := regexp.MustCompile(`\b[A-Za-z]+\b`)

	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Check if the word is all uppercase
		isAllUppercase := true
		for _, r := range match {
			if r >= 'a' && r <= 'z' {
				isAllUppercase = false
				break
			}
		}

		if isAllUppercase {
			// Letter-by-letter conversion for uppercase abbreviations
			return c.convertLetterByLetter(match)
		}

		// Phonetic transliteration for lowercase/mixed-case words
		return c.transliterateWord(match)
	})

	return result
}

// convertLetterByLetter converts each letter to its Russian pronunciation
// Example: "FBI" -> "эф би ай"
func (c *LatinToRussianConverter) convertLetterByLetter(word string) string {
	var converted []string
	for _, r := range word {
		if pronunciation, ok := c.letterMap[r]; ok {
			converted = append(converted, pronunciation)
		} else {
			// If we can't convert a letter, return original
			return word
		}
	}
	return strings.Join(converted, " ")
}

// transliterateWord phonetically transliterates a word to Russian
// Example: "desent" -> "дисент", "iPhone" -> "айфон"
func (c *LatinToRussianConverter) transliterateWord(word string) string {
	var result strings.Builder
	runes := []rune(strings.ToLower(word))

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Handle special combinations
		if i < len(runes)-1 {
			next := runes[i+1]

			// "ch" -> "ч"
			if r == 'c' && next == 'h' {
				result.WriteString("ч")
				i++ // Skip next character
				continue
			}

			// "sh" -> "ш"
			if r == 's' && next == 'h' {
				result.WriteString("ш")
				i++ // Skip next character
				continue
			}

			// "th" -> "т" (simplified)
			if r == 't' && next == 'h' {
				result.WriteString("т")
				i++ // Skip next character
				continue
			}

			// "ph" -> "ф"
			if r == 'p' && next == 'h' {
				result.WriteString("ф")
				i++ // Skip next character
				continue
			}

			// "ck" -> "к"
			if r == 'c' && next == 'k' {
				result.WriteString("к")
				i++ // Skip next character
				continue
			}
		}

		// Single character transliteration
		if transliterated, ok := c.phoneticMap[r]; ok {
			result.WriteString(transliterated)
		} else {
			// If we can't transliterate, keep original
			result.WriteRune(r)
		}
	}

	return result.String()
}
