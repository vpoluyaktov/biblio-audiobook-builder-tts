package normalize

import (
	"bufio"
	"embed"
	"io"
	"strings"
)

//go:embed data/*.csv
var embeddedData embed.FS

// NounDatabase stores grammatical information about nouns for context detection.
type NounDatabase struct {
	nouns map[string]map[string]NounInfo // language -> normalized_noun -> info
}

// NewNounDatabase creates a new noun database with built-in entries from embedded CSV files.
func NewNounDatabase() *NounDatabase {
	db := &NounDatabase{
		nouns: make(map[string]map[string]NounInfo),
	}
	db.loadEmbeddedData()
	return db
}

// loadEmbeddedData loads noun data from embedded CSV files.
func (db *NounDatabase) loadEmbeddedData() {
	// Load English nouns
	if data, err := embeddedData.Open("data/en.csv"); err == nil {
		db.LoadFromReader("en", data)
		data.Close()
	}

	// Load Russian nouns
	if data, err := embeddedData.Open("data/ru.csv"); err == nil {
		db.LoadFromReader("ru", data)
		data.Close()
	}
}

// LoadFromReader loads noun data from a CSV reader.
// CSV format: noun,gender,form,singular,plurals (pipe-separated)
// gender: m=masculine, f=feminine, n=neuter
// form: o=ordinal, c=cardinal
func (db *NounDatabase) LoadFromReader(lang string, r io.Reader) error {
	if db.nouns[lang] == nil {
		db.nouns[lang] = make(map[string]NounInfo)
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		info, err := parseCSVLine(line)
		if err != nil {
			continue // Skip malformed lines
		}

		db.nouns[lang][strings.ToLower(info.SingularForm)] = info
	}

	return scanner.Err()
}

// parseCSVLine parses a single CSV line into NounInfo.
func parseCSVLine(line string) (NounInfo, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 5 {
		return NounInfo{}, io.EOF // Not enough fields
	}

	// Parse gender
	var gender Gender
	switch strings.ToLower(strings.TrimSpace(parts[1])) {
	case "m":
		gender = Masculine
	case "f":
		gender = Feminine
	case "n":
		gender = Neuter
	default:
		gender = Masculine
	}

	// Parse form
	var form Form
	switch strings.ToLower(strings.TrimSpace(parts[2])) {
	case "o":
		form = Ordinal
	case "c":
		form = Cardinal
	default:
		form = Cardinal
	}

	// Parse plural forms (pipe-separated)
	plurals := strings.Split(parts[4], "|")
	for i := range plurals {
		plurals[i] = strings.TrimSpace(plurals[i])
	}

	return NounInfo{
		Gender:       gender,
		TriggerForm:  form,
		SingularForm: strings.TrimSpace(parts[3]),
		PluralForms:  plurals,
	}, nil
}

// Lookup finds noun information by language and noun form.
// It checks the noun and common variations (singular/plural forms).
func (db *NounDatabase) Lookup(lang, noun string) (NounInfo, bool) {
	langNouns, ok := db.nouns[lang]
	if !ok {
		return NounInfo{}, false
	}

	// Normalize: lowercase
	normalized := strings.ToLower(noun)

	// Direct lookup
	if info, ok := langNouns[normalized]; ok {
		return info, true
	}

	// Check if this noun is a plural form of a known noun
	for _, info := range langNouns {
		for _, plural := range info.PluralForms {
			if strings.ToLower(plural) == normalized {
				return info, true
			}
		}
	}

	return NounInfo{}, false
}

// Add adds a noun to the database.
func (db *NounDatabase) Add(lang string, noun string, info NounInfo) {
	if db.nouns[lang] == nil {
		db.nouns[lang] = make(map[string]NounInfo)
	}
	db.nouns[lang][strings.ToLower(noun)] = info
}

// GetNouns returns all nouns for a language.
func (db *NounDatabase) GetNouns(lang string) map[string]NounInfo {
	if langNouns, ok := db.nouns[lang]; ok {
		// Return a copy to prevent modification
		result := make(map[string]NounInfo, len(langNouns))
		for k, v := range langNouns {
			result[k] = v
		}
		return result
	}
	return nil
}

// GetLanguages returns all languages in the database.
func (db *NounDatabase) GetLanguages() []string {
	langs := make([]string, 0, len(db.nouns))
	for lang := range db.nouns {
		langs = append(langs, lang)
	}
	return langs
}

// Remove removes a noun from the database.
func (db *NounDatabase) Remove(lang, noun string) {
	if langNouns, ok := db.nouns[lang]; ok {
		delete(langNouns, strings.ToLower(noun))
	}
}

// Count returns the number of nouns for a language.
func (db *NounDatabase) Count(lang string) int {
	if langNouns, ok := db.nouns[lang]; ok {
		return len(langNouns)
	}
	return 0
}
