// Package ssml provides SSML (Speech Synthesis Markup Language) text wrapping
// for TTS providers that support SSML markup.
package ssml

import (
	"fmt"
	"regexp"
	"strings"
)

// sentenceEndPattern matches sentence-ending punctuation
// Includes various combinations found in ebooks:
// - Single punctuation: . ! ?
// - Multiple punctuation: !! !!! ?? ??? ?! !?
// - Ellipsis: ... .... .....
// - Unicode variants: Armenian ։, Arabic ؟, Interrobang ‽
var sentenceEndPattern = regexp.MustCompile(`([.!?։؟‽]+)(\s+|$)`)

// ellipsisPattern matches ellipsis (2 or more dots)
var ellipsisPattern = regexp.MustCompile(`\.{2,}`)

// initialsPattern matches abbreviated initials like "В.В." or "A.B.C."
// Matches one or more uppercase letters each followed by a dot, with optional spaces
// Examples: В.В., A.B., И.И.И., V. V., A. B. C.
var initialsPattern = regexp.MustCompile(`(?:\p{Lu}\.(?:\s*\p{Lu}\.)+)`)

// SSMLOptions contains options for SSML text wrapping
type SSMLOptions struct {
	SentenceBreakMs       int  // Duration of break between sentences in ms (0 = no breaks)
	ParagraphBreakMs      int  // Duration of break between paragraphs in ms (0 = no breaks)
	ConvertDashesToBreaks bool // Convert inline dashes to SSML break tags
	DashBreakDurationMs   int  // Duration of break for dashes (default 300ms)

	// Deprecated: Use SentenceBreakMs and ParagraphBreakMs instead
	UseSentencePauses bool // Legacy: Add paragraph/sentence tags for natural pauses
}

// WrapTextInSSML wraps text in SSML speak tags with optional sentence/paragraph pauses
func WrapTextInSSML(text string, useSentencePauses bool) string {
	return WrapTextInSSMLWithOptions(text, SSMLOptions{
		UseSentencePauses: useSentencePauses,
	})
}

// AddSSMLBreaks adds SSML break tags to text without wrapping in <speak> tags.
// This is used to pre-process text before chunking, so breaks are preserved across chunk boundaries.
// The text should later be wrapped in <speak> tags per chunk.
func AddSSMLBreaks(text string, opts SSMLOptions) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	// Handle legacy UseSentencePauses flag
	if opts.UseSentencePauses && opts.SentenceBreakMs == 0 && opts.ParagraphBreakMs == 0 {
		opts.SentenceBreakMs = 500
		opts.ParagraphBreakMs = 800
	}

	// Choose the appropriate escape function based on options
	escapeFunc := escapeXML
	if opts.ConvertDashesToBreaks {
		breakMs := opts.DashBreakDurationMs
		if breakMs <= 0 {
			breakMs = 300
		}
		escapeFunc = func(t string) string {
			return escapeXMLWithPauseBreaks(t, breakMs)
		}
	}

	// If no sentence/paragraph breaks needed, just escape and return
	if opts.SentenceBreakMs == 0 && opts.ParagraphBreakMs == 0 {
		return escapeFunc(text)
	}

	// Add break tags between sentences and paragraphs
	paragraphs := splitIntoParagraphs(text)

	var result strings.Builder

	for paraIdx, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		sentences := splitIntoSentences(para)
		for sentIdx, sent := range sentences {
			sent = strings.TrimSpace(sent)
			if sent == "" {
				continue
			}

			result.WriteString(escapeFunc(sent))

			// Add sentence break after each sentence (except last in paragraph)
			if opts.SentenceBreakMs > 0 && sentIdx < len(sentences)-1 {
				result.WriteString(fmt.Sprintf(` <break time="%dms"/> `, opts.SentenceBreakMs))
			}
		}

		// Add paragraph break after each paragraph (except last)
		if opts.ParagraphBreakMs > 0 && paraIdx < len(paragraphs)-1 {
			result.WriteString(fmt.Sprintf("\n\n<break time=\"%dms\"/>\n\n", opts.ParagraphBreakMs))
		}
	}

	return result.String()
}

// WrapTextInSSMLWithOptions converts plain text to SSML format with full options control.
func WrapTextInSSMLWithOptions(text string, opts SSMLOptions) string {
	if strings.TrimSpace(text) == "" {
		return "<speak></speak>"
	}

	// Handle legacy UseSentencePauses flag
	if opts.UseSentencePauses && opts.SentenceBreakMs == 0 && opts.ParagraphBreakMs == 0 {
		opts.SentenceBreakMs = 500
		opts.ParagraphBreakMs = 800
	}

	// Choose the appropriate escape function based on options
	escapeFunc := escapeXML
	if opts.ConvertDashesToBreaks {
		breakMs := opts.DashBreakDurationMs
		if breakMs <= 0 {
			breakMs = 300
		}
		escapeFunc = func(t string) string {
			return escapeXMLWithPauseBreaks(t, breakMs)
		}
	}

	// Simple wrapper without paragraph/sentence breaks
	if opts.SentenceBreakMs == 0 && opts.ParagraphBreakMs == 0 {
		return "<speak>" + escapeFunc(text) + "</speak>"
	}

	// SSML with break tags between sentences and paragraphs
	paragraphs := splitIntoParagraphs(text)

	var result strings.Builder
	result.WriteString("<speak>")

	for paraIdx, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		sentences := splitIntoSentences(para)
		for sentIdx, sent := range sentences {
			sent = strings.TrimSpace(sent)
			if sent == "" {
				continue
			}

			result.WriteString(escapeFunc(sent))

			// Add sentence break after each sentence (except last in paragraph)
			if opts.SentenceBreakMs > 0 && sentIdx < len(sentences)-1 {
				result.WriteString(fmt.Sprintf(` <break time="%dms"/> `, opts.SentenceBreakMs))
			}
		}

		// Add paragraph break after each paragraph (except last)
		if opts.ParagraphBreakMs > 0 && paraIdx < len(paragraphs)-1 {
			result.WriteString(fmt.Sprintf("\n\n<break time=\"%dms\"/>\n\n", opts.ParagraphBreakMs))
		}
	}

	result.WriteString("</speak>")
	return result.String()
}

// splitIntoParagraphs splits text into paragraphs by newlines.
// Handles both single and double newlines as paragraph separators.
func splitIntoParagraphs(text string) []string {
	// First try splitting by double newlines (explicit paragraph breaks)
	// Then fall back to single newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Split by one or more newlines
	parts := regexp.MustCompile(`\n+`).Split(text, -1)

	var paragraphs []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			paragraphs = append(paragraphs, part)
		}
	}

	return paragraphs
}

// splitIntoSentences splits a paragraph into sentences.
// Handles common sentence-ending punctuation including ellipsis
// Protects abbreviated initials (like В.В. or A.B.) from being split
func splitIntoSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// Protect abbreviated initials from being split (e.g., В.В., A.B.C.)
	// Replace them with placeholders
	type initialReplacement struct {
		placeholder string
		original    string
	}
	var replacements []initialReplacement

	text = initialsPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Remove spaces from initials for consistency (В. В. -> В.В.)
		normalized := strings.ReplaceAll(match, " ", "")
		placeholder := fmt.Sprintf("\x00INIT%d\x00", len(replacements))
		replacements = append(replacements, initialReplacement{
			placeholder: placeholder,
			original:    normalized,
		})
		return placeholder
	})

	// Find all sentence boundaries
	var sentences []string
	lastEnd := 0

	matches := sentenceEndPattern.FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		if match[0] == -1 {
			continue
		}

		// Get the position after the punctuation mark
		sentenceEnd := match[3] // End of punctuation + whitespace

		sentence := text[lastEnd:sentenceEnd]
		sentence = strings.TrimSpace(sentence)

		if sentence != "" {
			sentences = append(sentences, sentence)
		}

		lastEnd = sentenceEnd
	}

	// Add remaining text as the last sentence
	if lastEnd < len(text) {
		remaining := strings.TrimSpace(text[lastEnd:])
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}

	// If no sentences were found, return the whole text as one sentence
	if len(sentences) == 0 {
		text = strings.TrimSpace(text)
		// Restore initials before returning
		for _, repl := range replacements {
			text = strings.ReplaceAll(text, repl.placeholder, repl.original)
		}
		return []string{text}
	}

	// Restore initials in all sentences
	for i, sent := range sentences {
		for _, repl := range replacements {
			sent = strings.ReplaceAll(sent, repl.placeholder, repl.original)
		}
		sentences[i] = sent
	}

	return sentences
}

// escapeXML escapes special XML characters in text.
func escapeXML(text string) string {
	// Order matters: & must be escaped first
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&apos;")
	return text
}

// PausePatternDef defines a pattern that should be converted to an SSML break
// Pattern is a regex string, Replacement uses %s as placeholder for break tag
type PausePatternDef struct {
	Pattern     string
	Replacement string
}

// DefaultPausePatternDefs contains patterns that should trigger pauses in TTS
// These are applied in order, so more specific patterns should come first
var DefaultPausePatternDefs = []PausePatternDef{
	// Ellipsis: "..." or "…" - adds pause after ellipsis
	{`\.{3,}`, `...%s`},
	{`…`, `…%s`},
	// Inline dashes: " - ", " — ", " – " (with surrounding spaces)
	{`\s+[-—–]\s+`, ` %s `},
}

// compiledPausePatterns holds the compiled regex patterns (initialized once)
var compiledPausePatterns []struct {
	Pattern     *regexp.Regexp
	Replacement string
}

func init() {
	compiledPausePatterns = make([]struct {
		Pattern     *regexp.Regexp
		Replacement string
	}, len(DefaultPausePatternDefs))
	for i, def := range DefaultPausePatternDefs {
		compiledPausePatterns[i].Pattern = regexp.MustCompile(def.Pattern)
		compiledPausePatterns[i].Replacement = def.Replacement
	}
}

// breakPlaceholder is used to protect break tags from XML escaping
const breakPlaceholder = "\x01BREAK\x01"

// ConvertPausePatterns replaces pause patterns with SSML break tags
// This helps TTS engines that don't naturally pause on certain punctuation
func ConvertPausePatterns(text string, breakDurationMs int) string {
	if breakDurationMs <= 0 {
		breakDurationMs = 300 // Default 300ms pause
	}

	for _, p := range compiledPausePatterns {
		replacement := strings.Replace(p.Replacement, "%s", breakPlaceholder, 1)
		text = p.Pattern.ReplaceAllString(text, replacement)
	}

	return text
}

// ConvertDashesToBreaks is kept for backward compatibility
// Deprecated: Use ConvertPausePatterns instead
func ConvertDashesToBreaks(text string, breakDurationMs int) string {
	return ConvertPausePatterns(text, breakDurationMs)
}

// ConvertDashesToBreaksDefault uses the default 300ms break duration
func ConvertDashesToBreaksDefault(text string) string {
	return ConvertDashesToBreaks(text, 300)
}

// escapeXMLWithPauseBreaks escapes XML but preserves pause pattern-to-break conversions
// It first replaces pause patterns with placeholders, escapes XML, then restores break tags
func escapeXMLWithPauseBreaks(text string, breakDurationMs int) string {
	if breakDurationMs <= 0 {
		breakDurationMs = 300
	}

	// Replace pause patterns with placeholder
	text = ConvertPausePatterns(text, breakDurationMs)

	// Escape XML characters
	text = escapeXML(text)

	// Restore break tags with spaces (placeholders are not affected by XML escaping)
	breakTag := fmt.Sprintf(` <break time="%dms"/> `, breakDurationMs)
	text = strings.ReplaceAll(text, breakPlaceholder, breakTag)

	return text
}
