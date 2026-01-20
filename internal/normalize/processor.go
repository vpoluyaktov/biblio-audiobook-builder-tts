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

// russianOrdinalSuffixPattern matches Russian ordinal suffixes after numbers
// e.g., "1996-м", "1996-го", "1996-й", "1996-я", "1996-е", "1996-ом", "1996-ым"
var russianOrdinalSuffixPattern = regexp.MustCompile(`(\d+)-([мгйяеыо][оаяу|мй]?|ого|ему|ым|ом|ой|ую|ая|ое|ые|ых|ым|ыми)`)

// Process replaces numbers in text with words based on the language.
// It uses context (surrounding words) to determine cardinal/ordinal form and gender.
func (p *Processor) Process(text, lang string) string {
	converter := GetOrDefault(lang)
	if converter == nil {
		return text
	}

	result := text

	// For Russian, first handle numbers with ordinal suffixes (e.g., "1996-м году")
	if lang == "ru" {
		result = p.processRussianOrdinalSuffixes(result, converter)
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

		// Skip if this looks like a negative number that's actually part of text
		// (e.g., already processed or not a real negative)
		if strings.HasPrefix(numStr, "-") && start > 0 {
			prevChar := result[start-1]
			// If previous char is a digit, this hyphen is a suffix marker, skip
			if prevChar >= '0' && prevChar <= '9' {
				continue
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

		// Replace in result
		result = result[:start] + words + result[end:]
	}

	return result
}

// processRussianOrdinalSuffixes handles Russian numbers with ordinal suffixes like "1996-м"
func (p *Processor) processRussianOrdinalSuffixes(text string, converter NumberConverter) string {
	// Find all matches from end to start
	matches := russianOrdinalSuffixPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	result := text
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullStart, fullEnd := match[0], match[1]
		numStart, numEnd := match[2], match[3]
		suffixStart, suffixEnd := match[4], match[5]

		numStr := text[numStart:numEnd]
		suffix := text[suffixStart:suffixEnd]

		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			continue
		}

		// Determine gender from suffix
		gender := p.genderFromRussianSuffix(suffix)

		ctx := Context{
			Form:   Ordinal,
			Gender: gender,
			Case:   Nominative,
		}

		// Convert number to ordinal words (nominative case)
		words := converter.ToWords(n, ctx)

		// Apply case ending transformation based on suffix
		words = p.applyRussianCaseEnding(words, suffix, gender)

		// Replace the entire match (number + hyphen + suffix) with the ordinal word
		result = result[:fullStart] + words + result[fullEnd:]
	}

	return result
}

// genderFromRussianSuffix determines grammatical gender from Russian ordinal suffix
func (p *Processor) genderFromRussianSuffix(suffix string) Gender {
	suffix = strings.ToLower(suffix)
	switch suffix {
	case "й", "го", "ого", "ему", "ым", "ом", "м":
		return Masculine
	case "я", "ую", "ая", "ей", "ою":
		return Feminine
	case "е", "ое":
		return Neuter
	default:
		return Masculine
	}
}

// applyRussianCaseEnding transforms nominative ordinal ending to the appropriate case
func (p *Processor) applyRussianCaseEnding(words, suffix string, gender Gender) string {
	suffix = strings.ToLower(suffix)

	// Map suffix to case ending transformation
	// The converter outputs nominative case, we need to transform the last word's ending
	wordList := strings.Split(words, " ")
	if len(wordList) == 0 {
		return words
	}

	lastWord := wordList[len(wordList)-1]
	var newEnding string

	switch suffix {
	// Genitive case (родительный падеж)
	case "го", "ого":
		newEnding = p.transformToGenitive(lastWord, gender)
	// Dative case (дательный падеж)
	case "ему", "ому":
		newEnding = p.transformToDative(lastWord, gender)
	// Instrumental case (творительный падеж)
	case "ым", "им":
		newEnding = p.transformToInstrumental(lastWord, gender)
	// Prepositional case (предложный падеж)
	case "м", "ом":
		newEnding = p.transformToPrepositional(lastWord, gender)
	// Accusative feminine (винительный падеж)
	case "ую":
		newEnding = p.transformToAccusativeFem(lastWord)
	default:
		// Nominative - no change needed
		return words
	}

	if newEnding != "" {
		wordList[len(wordList)-1] = newEnding
		return strings.Join(wordList, " ")
	}
	return words
}

// transformToGenitive transforms nominative ordinal to genitive case
func (p *Processor) transformToGenitive(word string, gender Gender) string {
	// Masculine/Neuter: -ый/-ий/-ой -> -ого, -ий -> -ьего (for третий)
	// Feminine: -ая/-яя -> -ой/-ей
	if gender == Feminine {
		if strings.HasSuffix(word, "ая") {
			return strings.TrimSuffix(word, "ая") + "ой"
		}
		if strings.HasSuffix(word, "яя") {
			return strings.TrimSuffix(word, "яя") + "ей"
		}
		if strings.HasSuffix(word, "ья") {
			return strings.TrimSuffix(word, "ья") + "ьей"
		}
	} else {
		if strings.HasSuffix(word, "ий") {
			// Special case for третий -> третьего
			if word == "третий" {
				return "третьего"
			}
			return strings.TrimSuffix(word, "ий") + "ьего"
		}
		if strings.HasSuffix(word, "ый") {
			return strings.TrimSuffix(word, "ый") + "ого"
		}
		if strings.HasSuffix(word, "ой") {
			return strings.TrimSuffix(word, "ой") + "ого"
		}
	}
	return word
}

// transformToDative transforms nominative ordinal to dative case
func (p *Processor) transformToDative(word string, gender Gender) string {
	if gender == Feminine {
		if strings.HasSuffix(word, "ая") {
			return strings.TrimSuffix(word, "ая") + "ой"
		}
		if strings.HasSuffix(word, "яя") {
			return strings.TrimSuffix(word, "яя") + "ей"
		}
	} else {
		if strings.HasSuffix(word, "ий") {
			if word == "третий" {
				return "третьему"
			}
			return strings.TrimSuffix(word, "ий") + "ьему"
		}
		if strings.HasSuffix(word, "ый") {
			return strings.TrimSuffix(word, "ый") + "ому"
		}
		if strings.HasSuffix(word, "ой") {
			return strings.TrimSuffix(word, "ой") + "ому"
		}
	}
	return word
}

// transformToInstrumental transforms nominative ordinal to instrumental case
func (p *Processor) transformToInstrumental(word string, gender Gender) string {
	if gender == Feminine {
		if strings.HasSuffix(word, "ая") {
			return strings.TrimSuffix(word, "ая") + "ой"
		}
		if strings.HasSuffix(word, "яя") {
			return strings.TrimSuffix(word, "яя") + "ей"
		}
	} else {
		if strings.HasSuffix(word, "ий") {
			if word == "третий" {
				return "третьим"
			}
			return strings.TrimSuffix(word, "ий") + "ьим"
		}
		if strings.HasSuffix(word, "ый") {
			return strings.TrimSuffix(word, "ый") + "ым"
		}
		if strings.HasSuffix(word, "ой") {
			return strings.TrimSuffix(word, "ой") + "ым"
		}
	}
	return word
}

// transformToPrepositional transforms nominative ordinal to prepositional case
func (p *Processor) transformToPrepositional(word string, gender Gender) string {
	if gender == Feminine {
		if strings.HasSuffix(word, "ая") {
			return strings.TrimSuffix(word, "ая") + "ой"
		}
		if strings.HasSuffix(word, "яя") {
			return strings.TrimSuffix(word, "яя") + "ей"
		}
	} else {
		if strings.HasSuffix(word, "ий") {
			if word == "третий" {
				return "третьем"
			}
			return strings.TrimSuffix(word, "ий") + "ьем"
		}
		if strings.HasSuffix(word, "ый") {
			return strings.TrimSuffix(word, "ый") + "ом"
		}
		if strings.HasSuffix(word, "ой") {
			return strings.TrimSuffix(word, "ой") + "ом"
		}
	}
	return word
}

// transformToAccusativeFem transforms nominative feminine ordinal to accusative case
func (p *Processor) transformToAccusativeFem(word string) string {
	if strings.HasSuffix(word, "ая") {
		return strings.TrimSuffix(word, "ая") + "ую"
	}
	if strings.HasSuffix(word, "яя") {
		return strings.TrimSuffix(word, "яя") + "юю"
	}
	return word
}

// detectContext analyzes surrounding text to determine grammatical context.
func (p *Processor) detectContext(text string, numStart, numEnd int, lang string) Context {
	ctx := DefaultContext()

	// Extract word before the number
	wordBefore := p.extractWordBefore(text, numStart)

	// Extract word after the number
	wordAfter := p.extractWordAfter(text, numEnd)

	// Check if word before is a known noun (e.g., "Chapter 5", "Глава 5")
	// In this pattern, the noun comes first, so we use its trigger form
	if wordBefore != "" {
		if info, ok := p.nounDB.Lookup(lang, wordBefore); ok {
			ctx.Form = info.TriggerForm
			ctx.Gender = info.Gender
			return ctx
		}
	}

	// Check if word after is a known noun (e.g., "5 dollars", "5 рублей")
	// In this pattern, the number comes first, so we always use cardinal form
	// regardless of the noun's trigger form (e.g., "15 страниц" = "fifteen pages", not "fifteenth pages")
	if wordAfter != "" {
		if info, ok := p.nounDB.Lookup(lang, wordAfter); ok {
			ctx.Form = Cardinal // Always cardinal when number precedes noun
			ctx.Gender = info.Gender
			return ctx
		}
	}

	// Default: cardinal, masculine
	return ctx
}

// extractWordBefore extracts the word immediately before the given position.
func (p *Processor) extractWordBefore(text string, pos int) string {
	if pos <= 0 {
		return ""
	}

	// Convert to runes for proper UTF-8 handling
	runes := []rune(text)

	// Find the rune position corresponding to byte position
	bytePos := 0
	runePos := 0
	for runePos < len(runes) && bytePos < pos {
		bytePos += len(string(runes[runePos]))
		runePos++
	}

	if runePos <= 0 {
		return ""
	}

	// Skip whitespace backwards
	end := runePos
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
func (p *Processor) extractWordAfter(text string, pos int) string {
	if pos >= len(text) {
		return ""
	}

	// Convert to runes for proper UTF-8 handling
	runes := []rune(text)

	// Find the rune position corresponding to byte position
	bytePos := 0
	runePos := 0
	for runePos < len(runes) && bytePos < pos {
		bytePos += len(string(runes[runePos]))
		runePos++
	}

	if runePos >= len(runes) {
		return ""
	}

	// Skip whitespace forwards
	start := runePos
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
	if lang == "ru" {
		// "глава" is feminine in Russian
		gender = Feminine
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
