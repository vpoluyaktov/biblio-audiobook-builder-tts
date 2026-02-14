// Package normalize provides text normalization for TTS preprocessing,
// including number-to-words conversion for multiple languages.
package normalize

// Gender represents grammatical gender for languages that require it.
type Gender int

const (
	Masculine Gender = iota
	Feminine
	Neuter
)

// Form represents whether a number should be cardinal or ordinal.
type Form int

const (
	Cardinal Form = iota // one, two, three
	Ordinal              // first, second, third
)

// Case represents grammatical case for languages that require it.
type Case int

const (
	Nominative Case = iota // default case
	Genitive
	Dative
	Accusative
	Instrumental
	Prepositional
)

// Context provides grammatical context for number conversion.
type Context struct {
	Form   Form
	Gender Gender
	Case   Case
}

// DefaultContext returns the default context (cardinal, masculine, nominative).
func DefaultContext() Context {
	return Context{
		Form:   Cardinal,
		Gender: Masculine,
		Case:   Nominative,
	}
}

// NumberConverter is the interface for language-specific number-to-words conversion.
type NumberConverter interface {
	// ToWords converts a number to words with the given grammatical context.
	ToWords(n int64, ctx Context) string

	// SupportsContext returns true if the language requires grammatical context.
	SupportsContext() bool

	// LanguageCode returns the ISO 639-1 language code (e.g., "en", "ru").
	LanguageCode() string

	// LanguageName returns the human-readable language name.
	LanguageName() string
}

// NounInfo contains grammatical information about a noun.
type NounInfo struct {
	Gender       Gender
	TriggerForm  Form // whether this noun typically triggers ordinal numbers
	SingularForm string
	PluralForms  []string // plural forms (may have multiple for languages like Russian)
}

// registry holds all registered number converters.
var registry = make(map[string]NumberConverter)

// Register adds a number converter to the registry.
func Register(converter NumberConverter) {
	registry[converter.LanguageCode()] = converter
}

// Get returns the number converter for the given language code.
// Returns nil if no converter is registered for the language.
func Get(langCode string) NumberConverter {
	return registry[langCode]
}

// GetOrDefault returns the number converter for the given language code,
// or the English converter as a fallback.
func GetOrDefault(langCode string) NumberConverter {
	if conv := registry[langCode]; conv != nil {
		return conv
	}
	return registry["en"]
}

// SupportedLanguages returns a list of all registered language codes.
func SupportedLanguages() []string {
	langs := make([]string, 0, len(registry))
	for code := range registry {
		langs = append(langs, code)
	}
	return langs
}

// IsSupported returns true if a converter is registered for the language code.
func IsSupported(langCode string) bool {
	_, ok := registry[langCode]
	return ok
}

// LanguageProcessor is an optional interface for language-specific text processing.
// Languages that need special handling (like Russian ordinal suffixes) can implement this.
type LanguageProcessor interface {
	// PreProcess handles language-specific preprocessing before number replacement.
	// It receives the text and converter, and returns the processed text.
	PreProcess(text string, converter NumberConverter) string

	// DetectContext determines grammatical context from surrounding words.
	// Returns the context and true if language-specific detection was applied.
	DetectContext(wordBefore, wordAfter string, nounDB *NounDatabase) (Context, bool)

	// PostProcessContext applies language-specific post-processing to the context.
	// Called after number conversion to apply case transformations, etc.
	PostProcessContext(words string, ctx Context) string

	// GetChapterGender returns the grammatical gender for "chapter" in this language.
	GetChapterGender() Gender
}

// AbbreviationNormalizer is an optional interface for language-specific
// uppercase abbreviation normalization.
type AbbreviationNormalizer interface {
	// NormalizeAbbreviations expands uppercase abbreviations to spoken letter names.
	NormalizeAbbreviations(text string) string
}

// langProcessorRegistry holds language-specific processors.
var langProcessorRegistry = make(map[string]LanguageProcessor)

// RegisterLanguageProcessor adds a language processor to the registry.
func RegisterLanguageProcessor(langCode string, processor LanguageProcessor) {
	langProcessorRegistry[langCode] = processor
}

// GetLanguageProcessor returns the language processor for the given language code.
// Returns nil if no processor is registered.
func GetLanguageProcessor(langCode string) LanguageProcessor {
	return langProcessorRegistry[langCode]
}
