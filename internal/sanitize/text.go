package sanitize

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// HasSpeakableContent checks if text contains any speakable content (letters from any language).
// Text with only punctuation, symbols, or whitespace will cause TTS engines to fail.
func HasSpeakableContent(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// HasSpeakableContentForLanguage checks if text contains speakable content for a specific language.
// For Russian (ru), text must contain at least one Cyrillic letter.
// For English (en), text must contain at least one Latin letter.
// For other languages, falls back to HasSpeakableContent.
func HasSpeakableContentForLanguage(text, lang string) bool {
	if !HasSpeakableContent(text) {
		return false
	}

	switch lang {
	case "ru":
		// Russian TTS requires Cyrillic characters
		for _, r := range text {
			if isCyrillic(r) {
				return true
			}
		}
		return false
	case "en":
		// English TTS requires Latin characters
		for _, r := range text {
			if isLatin(r) {
				return true
			}
		}
		return false
	default:
		return HasSpeakableContent(text)
	}
}

// isCyrillic checks if a rune is a Cyrillic letter
func isCyrillic(r rune) bool {
	return (r >= 0x0400 && r <= 0x04FF) || // Cyrillic
		(r >= 0x0500 && r <= 0x052F) // Cyrillic Supplement
}

// isLatin checks if a rune is a Latin letter
func isLatin(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= 0x00C0 && r <= 0x00FF) || // Latin-1 Supplement
		(r >= 0x0100 && r <= 0x017F) // Latin Extended-A
}

// TextForTTS sanitizes text for TTS processing by normalizing problematic
// Unicode characters while preserving all readable text from any language.
// This should be applied during parsing stage so sanitized text is visible
// in output .txt files.
func TextForTTS(text string) string {
	// Replace common problematic Unicode characters with TTS-friendly equivalents
	// These are characters that often cause TTS engines to mispronounce or error
	replacements := map[string]string{
		// ASCII special characters that cause TTS engines to crash
		// These often appear in censored text, broken formatting, or encoding artifacts
		"$":  "",  // Dollar sign
		"%":  "",  // Percent sign
		"#":  "",  // Hash/pound sign
		"^":  "",  // Caret
		"*":  "",  // Asterisk (often used for censoring)
		"@":  "",  // At sign
		"~":  "",  // Tilde
		"|":  "",  // Pipe
		"\\": "",  // Backslash
		"/":  "",  // Forward slash
		"<":  "",  // Less than
		">":  "",  // Greater than
		"{":  "",  // Left brace
		"}":  "",  // Right brace
		"[":  "",  // Left bracket
		"]":  "",  // Right bracket
		"_":  " ", // Underscore (replace with space)

		// Dashes - replace with spoken equivalents
		"\u2014": " - ", // Em dash
		"\u2013": " - ", // En dash
		"\u2015": " - ", // Horizontal bar
		"\u2012": " - ", // Figure dash
		"\u2212": "-",   // Minus sign

		// Quotes - remove to avoid SSML escaping issues with TTS engines
		"\u201C": "", // Left double quote "
		"\u201D": "", // Right double quote "
		"\u201E": "", // Double low-9 quote „
		"\u2018": "", // Left single quote '
		"\u2019": "", // Right single quote '
		"\u201A": "", // Single low-9 quote ‚
		"\u00AB": "", // Left guillemet «
		"\u00BB": "", // Right guillemet »
		"\u2039": "", // Single left guillemet ‹
		"\u203A": "", // Single right guillemet ›
		`"`:      "", // ASCII double quote
		"'":      "", // ASCII single quote
		"`":      "", // Backtick

		// Ellipsis
		"\u2026": "...", // Horizontal ellipsis …

		// Spaces - normalize to regular space
		"\u00A0": " ", // Non-breaking space
		"\u2002": " ", // En space
		"\u2003": " ", // Em space
		"\u2004": " ", // Three-per-em space
		"\u2005": " ", // Four-per-em space (from &#8197;)
		"\u2006": " ", // Six-per-em space
		"\u2007": " ", // Figure space
		"\u2008": " ", // Punctuation space
		"\u2009": " ", // Thin space
		"\u200A": " ", // Hair space
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

		// Superscript numbers
		"\u00B9": "1", // ¹
		"\u00B2": "2", // ²
		"\u00B3": "3", // ³
		"\u2070": "0", // ⁰
		"\u2074": "4", // ⁴
		"\u2075": "5", // ⁵
		"\u2076": "6", // ⁶
		"\u2077": "7", // ⁷
		"\u2078": "8", // ⁸
		"\u2079": "9", // ⁹

		// Subscript numbers
		"\u2080": "0", // ₀
		"\u2081": "1", // ₁
		"\u2082": "2", // ₂
		"\u2083": "3", // ₃
		"\u2084": "4", // ₄
		"\u2085": "5", // ₅
		"\u2086": "6", // ₆
		"\u2087": "7", // ₇
		"\u2088": "8", // ₈
		"\u2089": "9", // ₉

		// Prime marks (feet/inches, minutes/seconds) - remove to avoid SSML issues
		"\u2032": "", // ′ Prime (feet, minutes)
		"\u2033": "", // ″ Double prime (inches, seconds)
		"\u2034": "", // ‴ Triple prime

		// Additional spaces
		"\u202F": " ", // Narrow no-break space
		"\u205F": " ", // Medium mathematical space
		"\u3000": " ", // Ideographic space (CJK)

		// Soft hyphen (invisible, can cause issues)
		"\u00AD": "", // Soft hyphen - remove

		// Ordinal indicators
		"\u00BA": "o", // º Masculine ordinal
		"\u00AA": "a", // ª Feminine ordinal

		// Numero sign
		"\u2116": "No.", // № Numero sign

		// Per mille and per ten thousand
		"\u2030": " per mille ",        // ‰
		"\u2031": " per ten thousand ", // ‱

		// Common arrows - expand to words
		"\u2192": " to ",   // → Right arrow
		"\u2190": " from ", // ← Left arrow
		"\u2194": " to ",   // ↔ Left-right arrow

		// Reference marks
		"\u203B": "*",   // ※ Reference mark
		"\u2042": "***", // ⁂ Asterism

		// Currency (keep symbol but ensure TTS can handle)
		"\u20AC": " euros ",  // €
		"\u00A3": " pounds ", // £
		"\u00A5": " yen ",    // ¥
		"\u00A2": " cents ",  // ¢

		// Other common symbols
		"\u2713": " check ", // ✓ Check mark
		"\u2717": " x ",     // ✗ Ballot X
		"\u2605": " star ",  // ★ Black star
		"\u2606": " star ",  // ☆ White star
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

	// Normalize whitespace while preserving paragraph breaks
	result = cleaned.String()

	// First, normalize line endings to \n
	result = strings.ReplaceAll(result, "\r\n", "\n")
	result = strings.ReplaceAll(result, "\r", "\n")

	// Preserve paragraph breaks (2+ newlines) by replacing with placeholder
	paragraphBreak := regexp.MustCompile(`\n\s*\n`)
	result = paragraphBreak.ReplaceAllString(result, "\n\n")

	// Normalize spaces within lines (but not newlines)
	spaceOnly := regexp.MustCompile(`[ \t]+`)
	result = spaceOnly.ReplaceAllString(result, " ")

	// Clean up: remove spaces at start/end of lines
	lineSpaces := regexp.MustCompile(`(?m)^ +| +$`)
	result = lineSpaces.ReplaceAllString(result, "")

	// Collapse 3+ newlines to 2 (paragraph break)
	multiNewline := regexp.MustCompile(`\n{3,}`)
	result = multiNewline.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}

// PronunciationRule represents a single pronunciation replacement rule
type PronunciationRule struct {
	Pattern          *regexp.Regexp
	ReplacementPlain string
	ReplacementSSML  string
	Language         string
	Comment          string
	Enabled          bool
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

// AddRule adds a pronunciation rule with plain text replacement
func (d *PronunciationDictionary) AddRule(pattern, replacement string) error {
	return d.AddRuleWithSSML(pattern, replacement, replacement, "en", true)
}

// AddRuleWithSSML adds a pronunciation rule with separate plain and SSML replacements
func (d *PronunciationDictionary) AddRuleWithSSML(pattern, replacementPlain, replacementSSML, language string, enabled bool) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	// Default to English if no language specified
	if language == "" {
		language = "en"
	}

	d.rules = append(d.rules, PronunciationRule{
		Pattern:          re,
		ReplacementPlain: replacementPlain,
		ReplacementSSML:  replacementSSML,
		Language:         language,
		Enabled:          enabled,
	})

	return nil
}

// Apply applies all enabled pronunciation rules to the text (uses plain text replacements)
func (d *PronunciationDictionary) Apply(text string) string {
	return d.ApplyWithMode(text, false)
}

// ApplyWithMode applies all enabled pronunciation rules with SSML support option
func (d *PronunciationDictionary) ApplyWithMode(text string, useSSML bool) string {
	return d.ApplyWithLanguage(text, useSSML, "")
}

// applyRuleUnicode applies a single pronunciation rule. It handles \b word
// boundaries correctly for Unicode (Cyrillic, etc.) text: Go's regexp \b only
// recognises ASCII word-character boundaries, so \b adjacent to Cyrillic letters
// never fires. When the pattern contains both \b and non-ASCII characters, this
// function strips the \b anchors and enforces Unicode word boundaries manually.
// For patterns with delimiter suffix (?:[\s\.\,\)]|$), a space is automatically added
// after replacement to handle the consumed delimiter.
func applyRuleUnicode(re *regexp.Regexp, text, replacement string) string {
	patStr := re.String()
	if !strings.Contains(patStr, `\b`) {
		// Check if pattern has delimiter suffix - if so, add space after replacement
		if strings.Contains(patStr, `(?:[\s\.\,\)]|$)`) {
			return re.ReplaceAllStringFunc(text, func(match string) string {
				expanded := re.ReplaceAllString(match, replacement)
				return expanded + " "
			})
		}
		// Normal replacement without adding space
		return re.ReplaceAllString(text, replacement)
	}

	// Only apply the custom path when the pattern contains non-ASCII characters.
	hasNonASCII := false
	for _, r := range patStr {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}
	if !hasNonASCII {
		return re.ReplaceAllString(text, replacement)
	}

	// Strip every \b from the pattern so FindAllStringIndex can find matches,
	// then manually enforce letter/digit boundaries around each match.
	stripped := strings.ReplaceAll(patStr, `\b`, ``)
	strippedRe, err := regexp.Compile(stripped)
	if err != nil {
		return re.ReplaceAllString(text, replacement) // unexpected – fall back
	}

	// Build a byte-offset → rune-index map for O(1) boundary lookups.
	runes := []rune(text)
	byteToRune := make([]int, len(text)+1)
	ri := 0
	for bi := range text {
		byteToRune[bi] = ri
		ri++
	}
	byteToRune[len(text)] = len(runes)

	var result strings.Builder
	lastEnd := 0

	for _, loc := range strippedRe.FindAllStringIndex(text, -1) {
		start, end := loc[0], loc[1]
		startRI := byteToRune[start]
		endRI := byteToRune[end]

		// Unicode word boundary before the match.
		before := start == 0 || func() bool {
			r := runes[startRI-1]
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}()

		// Unicode word boundary after the match.
		after := end == len(text) || func() bool {
			if endRI >= len(runes) {
				return true
			}
			r := runes[endRI]
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}()

		result.WriteString(text[lastEnd:start])
		if before && after {
			result.WriteString(strippedRe.ReplaceAllString(text[start:end], replacement))
		} else {
			result.WriteString(text[start:end])
		}
		lastEnd = end
	}
	result.WriteString(text[lastEnd:])
	return result.String()
}

// ApplyWithLanguage applies enabled pronunciation rules for a specific language
func (d *PronunciationDictionary) ApplyWithLanguage(text string, useSSML bool, language string) string {
	result := text
	for _, rule := range d.rules {
		if !rule.Enabled {
			continue
		}
		// If language is specified, only apply rules for that language
		if language != "" && rule.Language != language {
			continue
		}
		replacement := rule.ReplacementPlain
		if useSSML && rule.ReplacementSSML != "" {
			replacement = rule.ReplacementSSML
		}
		result = applyRuleUnicode(rule.Pattern, result, replacement)
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
	rule.ReplacementPlain = replacement
	rule.ReplacementSSML = replacement
	rule.Enabled = true

	return rule, nil
}

// GetDefaultRules returns common pronunciation fixes loaded from CSV files
// Deprecated: This function is kept for backward compatibility.
// New code should use LoadDefaultRulesFromCSV() directly.
func GetDefaultRules() []struct {
	Pattern          string
	ReplacementPlain string
	ReplacementSSML  string
	Comment          string
	Language         string
} {
	// Load from CSV files
	csvEntries, err := LoadDefaultRulesFromCSV()
	if err != nil {
		// Return empty slice on error
		return []struct {
			Pattern          string
			ReplacementPlain string
			ReplacementSSML  string
			Comment          string
			Language         string
		}{}
	}

	// Convert to old format for compatibility
	result := make([]struct {
		Pattern          string
		ReplacementPlain string
		ReplacementSSML  string
		Comment          string
		Language         string
	}, len(csvEntries))

	for i, entry := range csvEntries {
		result[i] = struct {
			Pattern          string
			ReplacementPlain string
			ReplacementSSML  string
			Comment          string
			Language         string
		}{
			Pattern:          entry.Pattern,
			ReplacementPlain: entry.ReplacementPlain,
			ReplacementSSML:  entry.ReplacementSSML,
			Comment:          entry.Comment,
			Language:         entry.Language,
		}
	}

	return result
}

// TextSanitizer combines TTS sanitization with pronunciation dictionary
type TextSanitizer struct {
	dictionary     *PronunciationDictionary
	latinConverter *LatinToRussianConverter
}

// NewTextSanitizer creates a new text sanitizer
func NewTextSanitizer() *TextSanitizer {
	return &TextSanitizer{
		dictionary:     NewPronunciationDictionary(),
		latinConverter: NewLatinToRussianConverter(),
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
	return s.SanitizeWithLanguage(text, "")
}

// SanitizeWithLanguage applies TTS sanitization and language-specific pronunciation rules
func (s *TextSanitizer) SanitizeWithLanguage(text string, language string) string {
	return s.SanitizeWithOptions(text, language, false)
}

// SanitizeWithOptions applies TTS sanitization and language-specific pronunciation rules
// with optional SSML replacement support
func (s *TextSanitizer) SanitizeWithOptions(text string, language string, useSSML bool) string {
	// First apply TTS sanitization (Unicode normalization)
	result := TextForTTS(text)

	// Then apply pronunciation dictionary rules for the specified language
	if s.dictionary != nil && s.dictionary.RuleCount() > 0 {
		// Log dictionary application for debugging
		if strings.Contains(result, "США") {
			log.Printf("[SANITIZE] Found 'США' in text before dictionary application (lang=%s, useSSML=%v, rules=%d)", language, useSSML, s.dictionary.RuleCount())
		}
		result = s.dictionary.ApplyWithLanguage(result, useSSML, language)
		if strings.Contains(result, "США") {
			log.Printf("[SANITIZE] WARNING: 'США' still present after dictionary application!")
		} else if strings.Contains(result, "сэ шэ") {
			log.Printf("[SANITIZE] SUCCESS: 'США' was replaced with 'сэ шэ а'")
		}
	} else {
		log.Printf("[SANITIZE] Dictionary not applied: dictionary=%v, ruleCount=%d", s.dictionary != nil, s.dictionary.RuleCount())
	}

	return result
}

// LoadDefaultRules loads the default pronunciation rules into the dictionary
func (s *TextSanitizer) LoadDefaultRules() {
	for _, rule := range GetDefaultRules() {
		// Use the language from the CSV entry instead of hardcoding "en"
		lang := rule.Language
		if lang == "" {
			lang = "en" // Fallback to English if not specified
		}
		s.dictionary.AddRuleWithSSML(rule.Pattern, rule.ReplacementPlain, rule.ReplacementSSML, lang, true)
	}
}

// ConvertLatinToRussian applies Latin-to-Russian letter transliteration
func (s *TextSanitizer) ConvertLatinToRussian(text string) string {
	return s.latinConverter.ConvertLatinInRussianText(text)
}
