package normalize

import (
	"bufio"
	"embed"
	"io"
	"strings"
	"time"
)

//go:embed data/*.csv
var embeddedData embed.FS

// NounRecord represents a noun entry as stored in the database.
type NounRecord struct {
	ID        int64
	Lang      string
	Noun      string
	Gender    string // m, f, n
	Form      string // o (ordinal), c (cardinal)
	Singular  string
	Plurals   string // pipe-separated
	IsCustom  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NounStore defines the interface for noun persistence.
type NounStore interface {
	ListNouns(lang string) ([]*NounRecord, error)
	CreateNoun(noun *NounRecord) error
	UpdateNoun(noun *NounRecord) error
	DeleteNoun(id int64) error
	GetNounByLangAndWord(lang, word string) (*NounRecord, error)
}

// NounDatabase stores grammatical information about nouns for context detection.
type NounDatabase struct {
	nouns map[string]map[string]NounInfo // language -> normalized_noun -> info
	store NounStore                      // optional persistent storage
}

// NewNounDatabase creates a new noun database with built-in entries from embedded CSV files.
func NewNounDatabase() *NounDatabase {
	db := &NounDatabase{
		nouns: make(map[string]map[string]NounInfo),
	}
	db.loadEmbeddedData()
	return db
}

// NewNounDatabaseWithStore creates a noun database that also loads from persistent storage.
// It loads embedded defaults first, then overlays with database entries.
func NewNounDatabaseWithStore(store NounStore) *NounDatabase {
	db := &NounDatabase{
		nouns: make(map[string]map[string]NounInfo),
		store: store,
	}
	db.loadEmbeddedData()
	db.loadFromStore()
	return db
}

// loadFromStore loads nouns from the persistent store and overlays them on defaults.
func (db *NounDatabase) loadFromStore() {
	if db.store == nil {
		return
	}

	nouns, err := db.store.ListNouns("")
	if err != nil {
		return
	}

	for _, n := range nouns {
		info := recordToNounInfo(n)
		db.Add(n.Lang, n.Singular, info)
	}
}

// recordToNounInfo converts a NounRecord to NounInfo.
func recordToNounInfo(r *NounRecord) NounInfo {
	var gender Gender
	switch r.Gender {
	case "m":
		gender = Masculine
	case "f":
		gender = Feminine
	case "n":
		gender = Neuter
	default:
		gender = Masculine
	}

	var form Form
	switch r.Form {
	case "o":
		form = Ordinal
	case "c":
		form = Cardinal
	default:
		form = Cardinal
	}

	plurals := strings.Split(r.Plurals, "|")
	for i := range plurals {
		plurals[i] = strings.TrimSpace(plurals[i])
	}

	return NounInfo{
		Gender:       gender,
		TriggerForm:  form,
		SingularForm: r.Singular,
		PluralForms:  plurals,
	}
}

// nounInfoToRecord converts NounInfo to a NounRecord for storage.
func nounInfoToRecord(lang string, info NounInfo, isCustom bool) *NounRecord {
	var gender string
	switch info.Gender {
	case Masculine:
		gender = "m"
	case Feminine:
		gender = "f"
	case Neuter:
		gender = "n"
	}

	var form string
	switch info.TriggerForm {
	case Ordinal:
		form = "o"
	case Cardinal:
		form = "c"
	}

	return &NounRecord{
		Lang:      lang,
		Noun:      strings.ToLower(info.SingularForm),
		Gender:    gender,
		Form:      form,
		Singular:  info.SingularForm,
		Plurals:   strings.Join(info.PluralForms, "|"),
		IsCustom:  isCustom,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
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

// Add adds a noun to the in-memory database.
func (db *NounDatabase) Add(lang string, noun string, info NounInfo) {
	if db.nouns[lang] == nil {
		db.nouns[lang] = make(map[string]NounInfo)
	}
	db.nouns[lang][strings.ToLower(noun)] = info
}

// AddAndPersist adds a noun to both in-memory database and persistent storage.
func (db *NounDatabase) AddAndPersist(lang string, info NounInfo) error {
	// Add to in-memory
	db.Add(lang, info.SingularForm, info)

	// Persist to store if available
	if db.store != nil {
		record := nounInfoToRecord(lang, info, true)
		return db.store.CreateNoun(record)
	}
	return nil
}

// RemoveAndPersist removes a noun from both in-memory database and persistent storage.
func (db *NounDatabase) RemoveAndPersist(lang, noun string) error {
	// Remove from in-memory
	db.Remove(lang, noun)

	// Remove from store if available
	if db.store != nil {
		existing, err := db.store.GetNounByLangAndWord(lang, strings.ToLower(noun))
		if err != nil {
			return err
		}
		if existing != nil {
			return db.store.DeleteNoun(existing.ID)
		}
	}
	return nil
}

// UpdateAndPersist updates a noun in both in-memory database and persistent storage.
func (db *NounDatabase) UpdateAndPersist(lang string, info NounInfo) error {
	// Update in-memory
	db.Add(lang, info.SingularForm, info)

	// Update in store if available
	if db.store != nil {
		existing, err := db.store.GetNounByLangAndWord(lang, strings.ToLower(info.SingularForm))
		if err != nil {
			return err
		}
		if existing != nil {
			// Update existing record
			existing.Gender = genderToString(info.Gender)
			existing.Form = formToString(info.TriggerForm)
			existing.Singular = info.SingularForm
			existing.Plurals = strings.Join(info.PluralForms, "|")
			existing.UpdatedAt = time.Now()
			return db.store.UpdateNoun(existing)
		} else {
			// Create new record
			record := nounInfoToRecord(lang, info, true)
			return db.store.CreateNoun(record)
		}
	}
	return nil
}

// genderToString converts Gender to string for storage.
func genderToString(g Gender) string {
	switch g {
	case Masculine:
		return "m"
	case Feminine:
		return "f"
	case Neuter:
		return "n"
	default:
		return "m"
	}
}

// formToString converts Form to string for storage.
func formToString(f Form) string {
	switch f {
	case Ordinal:
		return "o"
	case Cardinal:
		return "c"
	default:
		return "c"
	}
}

// Reload reloads nouns from embedded data and persistent storage.
func (db *NounDatabase) Reload() {
	db.nouns = make(map[string]map[string]NounInfo)
	db.loadEmbeddedData()
	db.loadFromStore()
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
