package sanitize

import (
	"bufio"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

//go:embed data/*.csv
var embeddedData embed.FS

// DictionaryEntry represents a pronunciation dictionary entry from CSV
type DictionaryEntry struct {
	Pattern          string
	ReplacementPlain string
	ReplacementSSML  string
	Comment          string
	Language         string
}

// LoadDefaultRulesFromCSV loads pronunciation rules from embedded CSV files
func LoadDefaultRulesFromCSV() ([]DictionaryEntry, error) {
	var allEntries []DictionaryEntry

	// Load English rules
	if data, err := embeddedData.Open("data/en.csv"); err == nil {
		entries, err := loadEntriesFromReaderWithLanguage(data, "en")
		data.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to load English rules: %w", err)
		}
		allEntries = append(allEntries, entries...)
	}

	// Load Russian rules
	if data, err := embeddedData.Open("data/ru.csv"); err == nil {
		entries, err := loadEntriesFromReaderWithLanguage(data, "ru")
		data.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to load Russian rules: %w", err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// loadEntriesFromReaderWithLanguage loads pronunciation dictionary entries from a CSV reader
// CSV format: pattern,replacement_plain,replacement_ssml,comment
func loadEntriesFromReaderWithLanguage(r io.Reader, language string) ([]DictionaryEntry, error) {
	var entries []DictionaryEntry

	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, err := parseCSVLine(line, lineNum)
		if err != nil {
			// Log warning but continue processing
			continue
		}

		// Set the language for this entry
		entry.Language = language
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	return entries, nil
}

// parseCSVLine parses a single CSV line into a DictionaryEntry
func parseCSVLine(line string, lineNum int) (DictionaryEntry, error) {
	// Use proper CSV parser to handle quoted fields
	r := csv.NewReader(strings.NewReader(line))
	r.FieldsPerRecord = -1 // Allow variable number of fields

	parts, err := r.Read()
	if err != nil {
		return DictionaryEntry{}, fmt.Errorf("line %d: failed to parse CSV: %w", lineNum, err)
	}

	if len(parts) < 4 {
		return DictionaryEntry{}, fmt.Errorf("line %d: expected 4 fields (pattern,replacement_plain,replacement_ssml,comment), got %d", lineNum, len(parts))
	}

	pattern := strings.TrimSpace(parts[0])
	replacementPlain := parts[1]
	replacementSSML := parts[2]
	comment := strings.TrimSpace(parts[3])

	// Validate regex pattern
	if _, err := regexp.Compile(pattern); err != nil {
		return DictionaryEntry{}, fmt.Errorf("line %d: invalid regex pattern '%s': %w", lineNum, pattern, err)
	}

	return DictionaryEntry{
		Pattern:          pattern,
		ReplacementPlain: replacementPlain,
		ReplacementSSML:  replacementSSML,
		Comment:          comment,
	}, nil
}

// LoadRulesFromCSVFile loads pronunciation rules from an external CSV file
// Language is detected from filename if possible (e.g., "ru.csv" -> "ru"), otherwise defaults to "en"
func LoadRulesFromCSVFile(filePath string) ([]DictionaryEntry, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try to detect language from filename
	lang := "en" // default
	if strings.Contains(filePath, "ru.csv") || strings.Contains(filePath, "/ru/") {
		lang = "ru"
	}

	return loadEntriesFromReaderWithLanguage(strings.NewReader(string(file)), lang)
}
