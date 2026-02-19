package normalize

import (
	"regexp"
	"strings"
	"unicode"
)

// Roman numeral values
var romanValues = map[rune]int{
	'I': 1,
	'V': 5,
	'X': 10,
	'L': 50,
	'C': 100,
	'D': 500,
	'M': 1000,
}

// RomanNumeralPattern matches standalone Roman numerals (I, II, III, IV, V, VI, VII, VIII, IX, X, etc.)
// Must be surrounded by word boundaries or whitespace/punctuation
var RomanNumeralPattern = regexp.MustCompile(`\b([IVXLCDM]+)\b`)

// IsValidRoman checks if a string is a valid Roman numeral.
// It validates the structure and ensures it's not just a word like "I" in English.
func IsValidRoman(s string) bool {
	if len(s) == 0 {
		return false
	}

	s = strings.ToUpper(s)

	// Check all characters are valid Roman numeral characters
	for _, r := range s {
		if _, ok := romanValues[r]; !ok {
			return false
		}
	}

	// Validate structure: no more than 3 consecutive same characters (except M)
	// and proper subtractive notation
	count := 1
	prev := rune(0)
	for _, r := range s {
		if r == prev {
			count++
			if count > 3 && r != 'M' {
				return false
			}
		} else {
			count = 1
		}
		prev = r
	}

	// Ensure it parses to a valid number
	val := ParseRoman(s)
	return val > 0
}

// ParseRoman converts a Roman numeral string to an integer.
// Returns 0 if the string is not a valid Roman numeral.
func ParseRoman(s string) int64 {
	if len(s) == 0 {
		return 0
	}

	s = strings.ToUpper(s)
	result := int64(0)
	prev := int64(0)

	// Process from right to left
	for i := len(s) - 1; i >= 0; i-- {
		r := rune(s[i])
		val, ok := romanValues[r]
		if !ok {
			return 0
		}

		v := int64(val)
		if v < prev {
			result -= v
		} else {
			result += v
		}
		prev = v
	}

	return result
}

// RomanMatch represents a matched Roman numeral with its context
type RomanMatch struct {
	Start      int    // byte position in text
	End        int    // byte position in text
	Roman      string // the Roman numeral string
	Value      int64  // numeric value
	WordBefore string // word before the Roman numeral (if any)
	WordAfter  string // word after the Roman numeral (if any)
}

// FindRomanNumerals finds all Roman numerals in text with their surrounding context.
// Note: This returns all potential Roman numerals. Use FilterRomanNumerals to filter
// based on noun database context.
func FindRomanNumerals(text string) []RomanMatch {
	matches := RomanNumeralPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []RomanMatch
	runes := []rune(text)

	for _, match := range matches {
		start, end := match[2], match[3] // submatch positions
		roman := text[start:end]

		if !IsValidRoman(roman) {
			continue
		}

		value := ParseRoman(roman)
		if value == 0 {
			continue
		}

		// Extract word before
		wordBefore := extractWordBeforePos(runes, byteToRunePos(text, start))

		// Extract word after
		wordAfter := extractWordAfterPos(runes, byteToRunePos(text, end))

		results = append(results, RomanMatch{
			Start:      start,
			End:        end,
			Roman:      roman,
			Value:      value,
			WordBefore: wordBefore,
			WordAfter:  wordAfter,
		})
	}

	return results
}

// FilterRomanNumerals filters Roman numeral matches to only include those with
// recognized context nouns. This prevents false positives like "I am" being
// interpreted as "one am".
func FilterRomanNumerals(matches []RomanMatch, lang string, nounDB *NounDatabase) []RomanMatch {
	if len(matches) == 0 {
		return nil
	}

	var filtered []RomanMatch
	for _, match := range matches {
		// Check if either surrounding word is a known noun
		hasContext := false

		if match.WordBefore != "" {
			if _, ok := nounDB.Lookup(lang, match.WordBefore); ok {
				hasContext = true
			}
		}

		if !hasContext && match.WordAfter != "" {
			if _, ok := nounDB.Lookup(lang, match.WordAfter); ok {
				hasContext = true
			}
		}

		// Only include Roman numerals that have a recognized context noun
		// This prevents "I am here" from being converted to "one am here"
		if hasContext {
			filtered = append(filtered, match)
		}
	}

	return filtered
}

// byteToRunePos converts a byte position to a rune position
func byteToRunePos(text string, bytePos int) int {
	return len([]rune(text[:bytePos]))
}

// extractWordBeforePos extracts the word immediately before the given rune position
func extractWordBeforePos(runes []rune, pos int) string {
	if pos <= 0 {
		return ""
	}

	// Skip whitespace backwards
	end := pos
	for end > 0 && unicode.IsSpace(runes[end-1]) {
		end--
	}

	if end <= 0 {
		return ""
	}

	// Find start of word
	start := end
	for start > 0 && isWordCharRune(runes[start-1]) {
		start--
	}

	if start == end {
		return ""
	}

	return string(runes[start:end])
}

// extractWordAfterPos extracts the word immediately after the given rune position
func extractWordAfterPos(runes []rune, pos int) string {
	if pos >= len(runes) {
		return ""
	}

	// Skip whitespace forwards
	start := pos
	for start < len(runes) && unicode.IsSpace(runes[start]) {
		start++
	}

	if start >= len(runes) {
		return ""
	}

	// Find end of word
	end := start
	for end < len(runes) && isWordCharRune(runes[end]) {
		end++
	}

	if start == end {
		return ""
	}

	return string(runes[start:end])
}

// isWordCharRune returns true if the rune is a word character
func isWordCharRune(r rune) bool {
	return unicode.IsLetter(r) || r == '-' || r == '\''
}

// RomanPosition indicates where the Roman numeral appears relative to a context word
type RomanPosition int

const (
	RomanAlone      RomanPosition = iota // No context word found
	RomanAfterNoun                       // "Part I", "Глава III" - Roman comes after noun
	RomanBeforeNoun                      // "I Глава", "II Часть" - Roman comes before noun
)

// DetermineRomanPosition determines the position of a Roman numeral relative to context nouns.
// It uses the noun database to identify trigger words.
func DetermineRomanPosition(match RomanMatch, lang string, nounDB *NounDatabase) (RomanPosition, *NounInfo) {
	// Check word before - if it's a known noun, Roman is after noun
	if match.WordBefore != "" {
		if info, ok := nounDB.Lookup(lang, match.WordBefore); ok {
			return RomanAfterNoun, &info
		}
	}

	// Check word after - if it's a known noun, Roman is before noun
	if match.WordAfter != "" {
		if info, ok := nounDB.Lookup(lang, match.WordAfter); ok {
			return RomanBeforeNoun, &info
		}
	}

	return RomanAlone, nil
}

// ProcessRomanNumerals converts Roman numerals in text to words.
// It uses the noun database to determine context and form.
// This is a shared utility that language processors can use.
func ProcessRomanNumerals(text, lang string, converter NumberConverter, nounDB *NounDatabase) string {
	matches := FindRomanNumerals(text)
	if len(matches) == 0 {
		return text
	}

	// Filter to only include Roman numerals with recognized context nouns
	matches = FilterRomanNumerals(matches, lang, nounDB)
	if len(matches) == 0 {
		return text
	}

	result := text

	// Process from end to start to preserve positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]

		position, nounInfo := DetermineRomanPosition(match, lang, nounDB)

		var ctx Context
		switch position {
		case RomanAfterNoun:
			// English: "Part I" → "Part one" (cardinal), "Chapter VII" → "Chapter seven" (cardinal)
			// Russian: "Глава III" → "Глава третья" (ordinal), "Часть I" → "Часть первая" (ordinal)
			ctx = Context{
				Form:   Ordinal,
				Gender: Masculine,
				Case:   Nominative,
			}
			if nounInfo != nil {
				// English Roman numerals after nouns always use cardinal form
				// Other languages use the noun's trigger form
				if lang == "en" {
					ctx.Form = Cardinal
				} else {
					ctx.Form = nounInfo.TriggerForm
				}
				ctx.Gender = nounInfo.Gender
			}

		case RomanBeforeNoun:
			// "I Глава", "II Часть" → ordinal with noun's gender
			// "First Chapter", "Первая Глава"
			ctx = Context{
				Form:   Ordinal,
				Gender: Masculine,
				Case:   Nominative,
			}
			if nounInfo != nil {
				ctx.Gender = nounInfo.Gender
			}

		case RomanAlone:
			// No context - default to cardinal masculine
			ctx = Context{
				Form:   Cardinal,
				Gender: Masculine,
				Case:   Nominative,
			}
		}

		// Convert to words
		words := converter.ToWords(match.Value, ctx)

		// Replace in result
		result = result[:match.Start] + words + result[match.End:]
	}

	return result
}
