package normalize

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Processor handles text normalization, including number-to-words conversion.
type Processor struct {
	nounDB *NounDatabase
}

// NewProcessor creates a new text processor with the default noun database.
func NewProcessor() *Processor {
	return &Processor{
		nounDB: NewNounDatabase(),
	}
}

// NewProcessorWithDB creates a new text processor with a custom noun database.
func NewProcessorWithDB(db *NounDatabase) *Processor {
	return &Processor{
		nounDB: db,
	}
}

// GetNounDatabase returns the noun database used by this processor.
func (p *Processor) GetNounDatabase() *NounDatabase {
	return p.nounDB
}

// numberPattern matches integers (with optional leading minus sign)
var numberPattern = regexp.MustCompile(`-?\d+`)

// Process replaces numbers in text with words based on the language.
// It uses context (surrounding words) to determine cardinal/ordinal form and gender.
func (p *Processor) Process(text, lang string) string {
	converter := GetOrDefault(lang)
	if converter == nil {
		return text
	}

	result := text

	// Apply language-specific preprocessing if available
	if langProc := GetLanguageProcessor(lang); langProc != nil {
		result = langProc.PreProcess(result, converter)
	}

	// Find all remaining numbers and their positions
	matches := numberPattern.FindAllStringIndex(result, -1)
	if len(matches) == 0 {
		return result
	}

	// Process matches from end to start to preserve positions
	for i := len(matches) - 1; i >= 0; i-- {
		start, end := matches[i][0], matches[i][1]
		numStr := result[start:end]
		replaceStart := start
		isHyphenatedModel := false

		// Skip if this looks like a negative number that's actually part of text
		// (e.g., already processed or not a real negative)
		if strings.HasPrefix(numStr, "-") && start > 0 {
			// Get the character before the hyphen (properly handle UTF-8)
			beforeHyphen := result[:start]
			runes := []rune(beforeHyphen)
			if len(runes) > 0 {
				prevRune := runes[len(runes)-1]

				// If previous char is a digit, this hyphen is a suffix marker, skip
				if prevRune >= '0' && prevRune <= '9' {
					continue
				}

				// If previous char is a letter, this is a hyphen separator (e.g., DC-7, Ту-154)
				// Extract just the number part without the hyphen
				if unicode.IsLetter(prevRune) {
					// Skip the hyphen for parsing, but include it in replacement
					start++
					numStr = result[start:end]
					isHyphenatedModel = true
				}
			}
		}

		// Parse the number
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			continue
		}

		// Determine context from surrounding words
		ctx := p.detectContext(result, start, end, lang)

		// Convert number to words
		words := converter.ToWords(n, ctx)

		// Apply language-specific post-processing
		if langProc := GetLanguageProcessor(lang); langProc != nil {
			words = langProc.PostProcessContext(words, ctx)
		}

		// Replace in result
		if isHyphenatedModel {
			// Replace hyphen with space (replaceStart includes the hyphen)
			result = result[:replaceStart] + " " + words + result[end:]
		} else {
			// Normal replacement without adding space
			result = result[:start] + words + result[end:]
		}
	}

	return result
}

// detectContext analyzes surrounding text to determine grammatical context.
func (p *Processor) detectContext(text string, numStart, numEnd int, lang string) Context {
	ctx := DefaultContext()

	// Extract word before the number
	wordBefore := p.extractWordBefore(text, numStart)

	// Extract word after the number
	wordAfter := p.extractWordAfter(text, numEnd)

	// Try language-specific context detection first
	if langProc := GetLanguageProcessor(lang); langProc != nil {
		if langCtx, ok := langProc.DetectContext(wordBefore, wordAfter, p.nounDB); ok {
			return langCtx
		}
	}

	// Generic context detection using noun database
	// Check word before - handles patterns like "Chapter 5"
	if wordBefore != "" {
		if info, ok := p.nounDB.Lookup(lang, wordBefore); ok {
			ctx.Form = info.TriggerForm
			ctx.Gender = info.Gender
			return ctx
		}
	}

	// Check word after - handles patterns like "5 dollars"
	if wordAfter != "" {
		if info, ok := p.nounDB.Lookup(lang, wordAfter); ok {
			ctx.Gender = info.Gender
			ctx.Form = Cardinal
			return ctx
		}
	}

	// Default: cardinal, masculine
	return ctx
}

// extractWordBefore extracts the word immediately before the given position.
// pos is a byte position in the text string.
func (p *Processor) extractWordBefore(text string, pos int) string {
	if pos <= 0 {
		return ""
	}

	// Extract substring before position and convert to runes
	beforeText := text[:pos]
	runes := []rune(beforeText)

	if len(runes) == 0 {
		return ""
	}

	// Skip whitespace backwards
	end := len(runes)
	for end > 0 && unicode.IsSpace(runes[end-1]) {
		end--
	}

	if end <= 0 {
		return ""
	}

	// Find start of word
	start := end
	for start > 0 && isWordChar(runes[start-1]) {
		start--
	}

	if start == end {
		return ""
	}

	return string(runes[start:end])
}

// extractWordAfter extracts the word immediately after the given position.
// pos is a byte position in the text string.
func (p *Processor) extractWordAfter(text string, pos int) string {
	if pos >= len(text) {
		return ""
	}

	// Extract substring after position and convert to runes
	afterText := text[pos:]
	runes := []rune(afterText)

	if len(runes) == 0 {
		return ""
	}

	// Skip whitespace forwards
	start := 0
	for start < len(runes) && unicode.IsSpace(runes[start]) {
		start++
	}

	if start >= len(runes) {
		return ""
	}

	// Find end of word
	end := start
	for end < len(runes) && isWordChar(runes[end]) {
		end++
	}

	if start == end {
		return ""
	}

	return string(runes[start:end])
}

// isWordChar returns true if the rune is a word character (letter or hyphen).
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || r == '-' || r == '\''
}

// ProcessWithContext replaces a specific number with words using explicit context.
// Useful when the caller already knows the grammatical context.
func (p *Processor) ProcessWithContext(text, lang string, ctx Context) string {
	converter := GetOrDefault(lang)
	if converter == nil {
		return text
	}

	matches := numberPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	result := text
	for i := len(matches) - 1; i >= 0; i-- {
		start, end := matches[i][0], matches[i][1]
		numStr := text[start:end]

		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			continue
		}

		words := converter.ToWords(n, ctx)
		result = result[:start] + words + result[end:]
	}

	return result
}

// NormalizeChapter is a convenience function for normalizing chapter titles.
// It uses ordinal form with the appropriate gender for the language.
func (p *Processor) NormalizeChapter(title, lang string) string {
	// Determine gender based on the word "chapter" in the target language
	gender := Masculine
	if langProc := GetLanguageProcessor(lang); langProc != nil {
		gender = langProc.GetChapterGender()
	}

	ctx := Context{
		Form:   Ordinal,
		Gender: gender,
		Case:   Nominative,
	}

	return p.ProcessWithContext(title, lang, ctx)
}

// ExtractNumbers returns all numbers found in the text.
func (p *Processor) ExtractNumbers(text string) []int64 {
	matches := numberPattern.FindAllString(text, -1)
	numbers := make([]int64, 0, len(matches))

	for _, match := range matches {
		n, err := strconv.ParseInt(match, 10, 64)
		if err == nil {
			numbers = append(numbers, n)
		}
	}

	return numbers
}

// HasNumbers returns true if the text contains any numbers.
func (p *Processor) HasNumbers(text string) bool {
	return numberPattern.MatchString(text)
}

// ReplaceNumber replaces a single number with its word equivalent.
func ReplaceNumber(n int64, lang string, ctx Context) string {
	converter := GetOrDefault(lang)
	if converter == nil {
		return strconv.FormatInt(n, 10)
	}
	return converter.ToWords(n, ctx)
}

// NormalizeText is a convenience function that creates a processor and normalizes text.
func NormalizeText(text, lang string) string {
	p := NewProcessor()
	return p.Process(text, lang)
}

// NormalizeTextWithContext is a convenience function with explicit context.
func NormalizeTextWithContext(text, lang string, ctx Context) string {
	p := NewProcessor()
	return p.ProcessWithContext(text, lang, ctx)
}

// SplitIntoWords splits text into words, preserving punctuation as separate tokens.
func SplitIntoWords(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '\'' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			if !unicode.IsSpace(r) {
				words = append(words, string(r))
			}
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}
