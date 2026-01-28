package sanitize

import (
	"regexp"
	"strings"
)

// FileName sanitizes a string for use as a file or directory name.
// It removes or replaces characters that are:
// - Invalid in file systems (/ \ : * ? " < > |)
// - Problematic for shell/ffmpeg arguments (' " ` $ & ; ( ) [ ] { } ! # ~ ^)
// - Whitespace characters
func FileName(name string) string {
	// Replace characters problematic for file systems and shell/ffmpeg arguments
	replacer := strings.NewReplacer(
		// Problematic substrings (must be before single-character replacements)
		"...", "_", // Triple dot ellipsis (problematic for some audiobook servers)
		"..", "_", // Double dot (parent directory reference)

		// File system reserved characters
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",

		// Shell special characters
		"'", "_", // ASCII apostrophe
		"\u2019", "_", // Right single quote (U+2019)
		"\u2018", "_", // Left single quote (U+2018)
		"`", "_", // Backtick
		"$", "_", // Dollar sign
		"&", "_", // Ampersand
		";", "_", // Semicolon
		"(", "_", // Left paren
		")", "_", // Right paren
		"[", "_", // Left bracket
		"]", "_", // Right bracket
		"{", "_", // Left brace
		"}", "_", // Right brace
		"!", "_", // Exclamation
		"#", "_", // Hash
		"~", "_", // Tilde
		"^", "_", // Caret
		"@", "_", // At sign
		"%", "_", // Percent

		// Whitespace
		" ", "_",
		"\n", "_",
		"\r", "_",
		"\t", "_",
	)
	result := replacer.Replace(name)

	// Collapse multiple underscores into one
	multiUnderscore := regexp.MustCompile(`_+`)
	result = multiUnderscore.ReplaceAllString(result, "_")

	// Trim underscores and dots from ends
	result = strings.Trim(result, "_.")

	// Limit length to 100 runes (not bytes) to avoid cutting UTF-8 characters
	runes := []rune(result)
	if len(runes) > 100 {
		result = string(runes[:100])
		// Trim trailing underscore if we cut in the middle
		result = strings.TrimRight(result, "_")
	}

	if result == "" {
		result = "untitled"
	}

	return result
}

// DirectoryName is an alias for FileName - same rules apply
func DirectoryName(name string) string {
	return FileName(name)
}

// FileNameWithExtension sanitizes a filename while preserving the extension
func FileNameWithExtension(name string, ext string) string {
	// Sanitize the base name
	baseName := FileName(name)

	// Ensure extension starts with a dot
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	return baseName + ext
}

// PathComponent sanitizes a single path component (file or directory name)
// This is stricter than FileName - it also removes dots except for extensions
func PathComponent(name string) string {
	// First apply standard filename sanitization
	result := FileName(name)

	// Remove leading dots (hidden files on Unix)
	result = strings.TrimLeft(result, ".")

	if result == "" {
		result = "untitled"
	}

	return result
}
