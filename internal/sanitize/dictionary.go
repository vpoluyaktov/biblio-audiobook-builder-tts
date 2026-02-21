package sanitize

import (
	"bufio"
	"embed"
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
}

// LoadDefaultRulesFromCSV loads pronunciation rules from embedded CSV files
func LoadDefaultRulesFromCSV() ([]DictionaryEntry, error) {
	var allEntries []DictionaryEntry

	// Load English rules
	if data, err := embeddedData.Open("data/en.csv"); err == nil {
		entries, err := loadEntriesFromReader(data)
		data.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to load English rules: %w", err)
		}
		allEntries = append(allEntries, entries...)
	}

	// Load Russian rules
	if data, err := embeddedData.Open("data/ru.csv"); err == nil {
		entries, err := loadEntriesFromReader(data)
		data.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to load Russian rules: %w", err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// loadEntriesFromReader loads pronunciation dictionary entries from a CSV reader
// CSV format: pattern,replacement_plain,replacement_ssml,comment
func loadEntriesFromReader(r io.Reader) ([]DictionaryEntry, error) {
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

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	return entries, nil
}

// parseCSVLine parses a single CSV line into a DictionaryEntry
func parseCSVLine(line string, lineNum int) (DictionaryEntry, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 4 {
		return DictionaryEntry{}, fmt.Errorf("line %d: expected 4 fields (pattern,replacement_plain,replacement_ssml,comment), got %d", lineNum, len(parts))
	}

	pattern := strings.TrimSpace(parts[0])
	replacementPlain := strings.TrimSpace(parts[1])
	replacementSSML := strings.TrimSpace(parts[2])
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
func LoadRulesFromCSVFile(filePath string) ([]DictionaryEntry, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return loadEntriesFromReader(strings.NewReader(string(file)))
}
