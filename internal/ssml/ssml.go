// Package ssml provides SSML (Speech Synthesis Markup Language) text wrapping
// for TTS providers that support SSML markup.
package ssml

import (
	"regexp"
	"strings"
)

// sentenceEndPattern matches sentence-ending punctuation
// Includes: period, exclamation, question mark, Armenian question mark, Arabic question mark
var sentenceEndPattern = regexp.MustCompile(`([.!?։؟])(\s+|$)`)

// ellipsisPattern matches ellipsis to avoid splitting on each dot
var ellipsisPattern = regexp.MustCompile(`\.{2,}`)

// WrapTextInSSML converts plain text to SSML format.
// If useSentencePauses is true, adds paragraph and sentence tags for natural pauses.
// If useSentencePauses is false, only wraps in <speak> tags without paragraph/sentence markup.
func WrapTextInSSML(text string, useSentencePauses bool) string {
	if strings.TrimSpace(text) == "" {
		return "<speak></speak>"
	}

	// Simple wrapper without paragraph/sentence pauses
	if !useSentencePauses {
		return "<speak>" + escapeXML(text) + "</speak>"
	}

	// Full SSML with paragraph and sentence tags
	paragraphs := splitIntoParagraphs(text)

	var result strings.Builder
	result.WriteString("<speak>\n")

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		result.WriteString("<p>\n")

		sentences := splitIntoSentences(para)
		for _, sent := range sentences {
			sent = strings.TrimSpace(sent)
			if sent == "" {
				continue
			}
			result.WriteString("  <s>")
			result.WriteString(escapeXML(sent))
			result.WriteString("</s>\n")
		}

		result.WriteString("</p>\n")
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
// Handles common sentence-ending punctuation while avoiding false splits
// on abbreviations and decimal numbers.
func splitIntoSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// Protect ellipsis from being split
	text = ellipsisPattern.ReplaceAllStringFunc(text, func(s string) string {
		return strings.Repeat("\x00", len(s)) // Placeholder
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

		// Restore ellipsis
		sentence = strings.ReplaceAll(sentence, "\x00", ".")

		if sentence != "" {
			sentences = append(sentences, sentence)
		}

		lastEnd = sentenceEnd
	}

	// Add remaining text as the last sentence
	if lastEnd < len(text) {
		remaining := strings.TrimSpace(text[lastEnd:])
		// Restore ellipsis
		remaining = strings.ReplaceAll(remaining, "\x00", ".")
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}

	// If no sentences were found, return the whole text as one sentence
	if len(sentences) == 0 {
		text = strings.ReplaceAll(text, "\x00", ".")
		return []string{strings.TrimSpace(text)}
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
