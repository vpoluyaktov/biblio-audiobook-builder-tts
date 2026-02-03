package normalize

import (
	"regexp"
	"strconv"
	"strings"
)

// RussianConverter converts numbers to Russian words with gender and case support.
type RussianConverter struct{}

// Ensure RussianConverter implements NumberConverter.
var _ NumberConverter = (*RussianConverter)(nil)

// Cardinal numbers 0-9 by gender
var onesCardinalRU = map[Gender][]string{
	Masculine: {"ноль", "один", "два", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять"},
	Feminine:  {"ноль", "одна", "две", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять"},
	Neuter:    {"ноль", "одно", "два", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять"},
}

// Ordinal numbers 0-9 by gender (nominative case)
var onesOrdinalRU = map[Gender][]string{
	Masculine: {"нулевой", "первый", "второй", "третий", "четвёртый", "пятый", "шестой", "седьмой", "восьмой", "девятый"},
	Feminine:  {"нулевая", "первая", "вторая", "третья", "четвёртая", "пятая", "шестая", "седьмая", "восьмая", "девятая"},
	Neuter:    {"нулевое", "первое", "второе", "третье", "четвёртое", "пятое", "шестое", "седьмое", "восьмое", "девятое"},
}

// Teens 10-19 (same for all genders in cardinal)
var teensCardinalRU = []string{
	"десять", "одиннадцать", "двенадцать", "тринадцать", "четырнадцать",
	"пятнадцать", "шестнадцать", "семнадцать", "восемнадцать", "девятнадцать",
}

// Teens ordinal 10-19 by gender
var teensOrdinalRU = map[Gender][]string{
	Masculine: {"десятый", "одиннадцатый", "двенадцатый", "тринадцатый", "четырнадцатый", "пятнадцатый", "шестнадцатый", "семнадцатый", "восемнадцатый", "девятнадцатый"},
	Feminine:  {"десятая", "одиннадцатая", "двенадцатая", "тринадцатая", "четырнадцатая", "пятнадцатая", "шестнадцатая", "семнадцатая", "восемнадцатая", "девятнадцатая"},
	Neuter:    {"десятое", "одиннадцатое", "двенадцатое", "тринадцатое", "четырнадцатое", "пятнадцатое", "шестнадцатое", "семнадцатое", "восемнадцатое", "девятнадцатое"},
}

// Tens 20-90 cardinal
var tensCardinalRU = []string{
	"", "", "двадцать", "тридцать", "сорок",
	"пятьдесят", "шестьдесят", "семьдесят", "восемьдесят", "девяносто",
}

// Tens ordinal 20-90 by gender
var tensOrdinalRU = map[Gender][]string{
	Masculine: {"", "", "двадцатый", "тридцатый", "сороковой", "пятидесятый", "шестидесятый", "семидесятый", "восьмидесятый", "девяностый"},
	Feminine:  {"", "", "двадцатая", "тридцатая", "сороковая", "пятидесятая", "шестидесятая", "семидесятая", "восьмидесятая", "девяностая"},
	Neuter:    {"", "", "двадцатое", "тридцатое", "сороковое", "пятидесятое", "шестидесятое", "семидесятое", "восьмидесятое", "девяностое"},
}

// Hundreds 100-900 cardinal
var hundredsCardinalRU = []string{
	"", "сто", "двести", "триста", "четыреста",
	"пятьсот", "шестьсот", "семьсот", "восемьсот", "девятьсот",
}

// Hundreds ordinal 100-900 by gender
var hundredsOrdinalRU = map[Gender][]string{
	Masculine: {"", "сотый", "двухсотый", "трёхсотый", "четырёхсотый", "пятисотый", "шестисотый", "семисотый", "восьмисотый", "девятисотый"},
	Feminine:  {"", "сотая", "двухсотая", "трёхсотая", "четырёхсотая", "пятисотая", "шестисотая", "семисотая", "восьмисотая", "девятисотая"},
	Neuter:    {"", "сотое", "двухсотое", "трёхсотое", "четырёхсотое", "пятисотое", "шестисотое", "семисотое", "восьмисотое", "девятисотое"},
}

// Scale words for thousands, millions, billions
type scaleWord struct {
	one  string // 1 тысяча
	few  string // 2-4 тысячи
	many string // 5+ тысяч
}

var scalesRU = []scaleWord{
	{"", "", ""}, // ones
	{"тысяча", "тысячи", "тысяч"},           // thousands (feminine)
	{"миллион", "миллиона", "миллионов"},    // millions (masculine)
	{"миллиард", "миллиарда", "миллиардов"}, // billions (masculine)
	{"триллион", "триллиона", "триллионов"}, // trillions (masculine)
}

// Scale ordinal suffixes by gender
var scaleOrdinalRU = map[Gender][]string{
	Masculine: {"", "тысячный", "миллионный", "миллиардный", "триллионный"},
	Feminine:  {"", "тысячная", "миллионная", "миллиардная", "триллионная"},
	Neuter:    {"", "тысячное", "миллионное", "миллиардное", "триллионное"},
}

// RussianProcessor implements LanguageProcessor for Russian-specific text processing.
type RussianProcessor struct {
	nounDB *NounDatabase
}

// Ensure RussianProcessor implements LanguageProcessor.
var _ LanguageProcessor = (*RussianProcessor)(nil)

func init() {
	Register(&RussianConverter{})
	RegisterLanguageProcessor("ru", &RussianProcessor{nounDB: NewNounDatabase()})
}

// RussianOrdinalSuffixPattern matches Russian ordinal suffixes after numbers
// e.g., "1996-м", "1996-го", "1996-й", "1996-я", "1996-е", "1996-ом", "1996-ым", "90-х"
var RussianOrdinalSuffixPattern = regexp.MustCompile(`(\d+)-([мгйяеыо][оаяу|мй]?|ого|ему|ым|ом|ой|ую|ая|ое|ые|ых|ым|ыми|х)`)

// RussianYearAbbrevPattern matches "гг." abbreviation for "годов" (years)
var RussianYearAbbrevPattern = regexp.MustCompile(`гг\.`)

// RussianYearOfBirthPattern matches "<number> г.р." or "<number> г. р." abbreviation for "года рождения" (year of birth)
// The year should be converted to genitive ordinal case
var RussianYearOfBirthPattern = regexp.MustCompile(`(\d+)\s*г\.\s*р\.`)

// RussianDateRangePattern matches date ranges like "6-16 августа" or "1-5 марта"
// Both numbers should be converted to ordinal neuter (for dates)
// Example: "6-16 августа" → "шестое, тире, шестнадцатое августа"
var RussianDateRangePattern = regexp.MustCompile(`(\d+)-(\d+)\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)`)

// GenderFromSuffix determines grammatical gender from Russian ordinal suffix
func GenderFromSuffix(suffix string) Gender {
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

// DetectCase detects grammatical case from Russian word ending
func DetectCase(word string) Case {
	word = strings.ToLower(word)
	// Genitive case for year ordinals: "года" (but not "лет" which is for quantities)
	if word == "года" {
		return Genitive
	}
	return Nominative
}

// IsRussianMonth checks if the word is a Russian month name (in genitive form)
func IsRussianMonth(word string) bool {
	word = strings.ToLower(word)
	months := map[string]bool{
		"января": true, "февраля": true, "марта": true, "апреля": true,
		"мая": true, "июня": true, "июля": true, "августа": true,
		"сентября": true, "октября": true, "ноября": true, "декабря": true,
	}
	return months[word]
}

// IsGenitiveTrigger checks if a word triggers genitive case for following dates.
// These are typically verbs or prepositions that require genitive case.
func IsGenitiveTrigger(word string) bool {
	word = strings.ToLower(word)
	triggers := map[string]bool{
		// Past tense verbs that trigger genitive for dates
		"случилось": true, "произошло": true, "было": true, "состоялось": true,
		"началось": true, "закончилось": true, "завершилось": true,
		"родился": true, "родилась": true, "родились": true,
		"умер": true, "умерла": true, "умерли": true,
		"женился": true, "вышла": true,
		// Prepositions that trigger genitive
		"до": true, "после": true, "с": true, "от": true, "около": true,
		"начиная": true, "кроме": true,
	}
	return triggers[word]
}

// GenderFromNounEnding determines grammatical gender from Russian noun ending.
// This is a fallback heuristic for nouns not in the database.
// It first attempts to normalize plural forms to singular, then applies
// standard Russian grammar rules for nominative case:
// - Consonant or -й → Masculine
// - -а or -я → Feminine
// - -о or -е → Neuter
// - -ь (soft sign) → Ambiguous, defaults to Masculine
func GenderFromNounEnding(word string) Gender {
	word = strings.ToLower(word)
	runes := []rune(word)
	if len(runes) == 0 {
		return Masculine
	}

	// Try to detect gender from plural endings first
	if gender, ok := genderFromPluralEnding(word, runes); ok {
		return gender
	}

	// Fall back to singular ending detection
	lastRune := runes[len(runes)-1]

	switch lastRune {
	case 'а', 'я':
		return Feminine
	case 'о', 'е', 'ё':
		return Neuter
	case 'ь':
		// Soft sign is ambiguous - could be masculine or feminine
		// Default to masculine as it's slightly more common
		return Masculine
	default:
		// Consonants and -й are masculine
		return Masculine
	}
}

// genderFromPluralEnding attempts to detect gender from Russian plural noun endings.
// Returns the detected gender and true if a plural pattern was recognized.
// Russian nouns after numbers typically appear in genitive case:
// - After 2-4: genitive singular (собаки, окна, стола)
// - After 5+: genitive plural (собак, окон, столов)
func genderFromPluralEnding(word string, runes []rune) (Gender, bool) {
	if len(runes) < 2 {
		return Masculine, false
	}

	lastRune := runes[len(runes)-1]
	lastTwo := string(runes[len(runes)-2:])

	// Genitive plural endings (after 5, 6, 7, 8, 9, 10, 11-19, 20, etc.)
	switch {
	// -ов, -ев, -ёв → Masculine (столов, музеев)
	case strings.HasSuffix(word, "ов") || strings.HasSuffix(word, "ев") || strings.HasSuffix(word, "ёв"):
		return Masculine, true

	// -ей → Could be Masculine (врачей) or Neuter (морей) or Feminine (ночей)
	// Most commonly masculine, but ambiguous
	case strings.HasSuffix(word, "ей"):
		return Masculine, true

	// Genitive singular endings (after 2, 3, 4, 22, 23, 24, etc.)
	// -и → Feminine genitive singular (собаки, книги, земли)
	case lastRune == 'и':
		return Feminine, true

	// -ы → Feminine genitive singular for hard stems (воды, горы)
	// But also nominative plural for masculine (столы) - context dependent
	// After numbers 2-4, it's more likely feminine genitive singular
	case lastRune == 'ы':
		return Feminine, true

	// -а after consonant → Could be:
	// - Neuter genitive singular (окна from окно)
	// - Feminine nominative singular (кошка, Москва)
	// - Masculine genitive singular (стола from стол)
	//
	// Neuter gen.sg pattern: short words (3-4 chars) like окна, яйца
	// Feminine nom.sg: longer words ending in -ка, -ва, -на, etc.
	// Only match neuter for very short words that look like gen.sg of neuter nouns
	case lastRune == 'а' && len(runes) >= 3:
		// Only match as neuter plural if word is very short (3-4 chars)
		// and has consonant cluster before -а (like "окна" from "окно")
		if len(runes) <= 4 {
			prevRune := runes[len(runes)-2]
			if !isRussianVowel(prevRune) && len(runes) >= 4 {
				thirdLast := runes[len(runes)-3]
				// Pattern like "окна" - consonant + consonant + а
				if !isRussianVowel(thirdLast) {
					return Neuter, true
				}
			}
		}
		// For longer words or other patterns, don't match - let singular detection handle
		return Feminine, false

	// -я after consonant → Could be Neuter genitive singular (моря, поля)
	// But also Feminine nominative singular (земля, семья)
	// Only treat as neuter if it looks like a plural form (short word after number 2-4)
	// For safety, don't match here - let singular detection handle -я as feminine
	case lastRune == 'я' && len(runes) >= 3:
		prevRune := runes[len(runes)-2]
		// Only match neuter if previous is 'р' (моря, поля pattern) and word is short
		if !isRussianVowel(prevRune) && (prevRune == 'р' || prevRune == 'л') && len(runes) <= 4 {
			return Neuter, true
		}
		return Feminine, false // Let singular detection handle it

	// Zero ending (consonant) after removing vowel → check common patterns
	// Words like "собак", "книг", "окон" - genitive plural
	case lastTwo == "ок" || lastTwo == "он" || lastTwo == "ен":
		// Could be neuter (окон) - but hard to tell without more context
		return Neuter, false
	}

	return Masculine, false
}

// isRussianVowel checks if a rune is a Russian vowel
func isRussianVowel(r rune) bool {
	vowels := map[rune]bool{
		'а': true, 'е': true, 'ё': true, 'и': true, 'о': true,
		'у': true, 'ы': true, 'э': true, 'ю': true, 'я': true,
	}
	return vowels[r]
}

// TransformOrdinalCase transforms nominative ordinal to target case
func TransformOrdinalCase(words string, targetCase Case, gender Gender) string {
	wordList := strings.Split(words, " ")
	if len(wordList) == 0 {
		return words
	}

	lastWord := wordList[len(wordList)-1]
	var newWord string

	switch targetCase {
	case Genitive:
		newWord = transformToGenitive(lastWord, gender)
	case Dative:
		newWord = transformToDative(lastWord, gender)
	case Instrumental:
		newWord = transformToInstrumental(lastWord, gender)
	case Prepositional:
		newWord = transformToPrepositional(lastWord, gender)
	case Accusative:
		if gender == Feminine {
			newWord = transformToAccusativeFem(lastWord)
		} else {
			return words
		}
	default:
		return words
	}

	if newWord != "" && newWord != lastWord {
		wordList[len(wordList)-1] = newWord
		return strings.Join(wordList, " ")
	}
	return words
}

// TransformBySuffix transforms nominative ordinal based on explicit suffix
func TransformBySuffix(words, suffix string, gender Gender) string {
	suffix = strings.ToLower(suffix)
	wordList := strings.Split(words, " ")
	if len(wordList) == 0 {
		return words
	}

	lastWord := wordList[len(wordList)-1]
	var newWord string

	switch suffix {
	case "го", "ого":
		newWord = transformToGenitive(lastWord, gender)
	case "ему", "ому":
		newWord = transformToDative(lastWord, gender)
	case "ым", "им":
		newWord = transformToInstrumental(lastWord, gender)
	case "м", "ом":
		newWord = transformToPrepositional(lastWord, gender)
	case "ую":
		newWord = transformToAccusativeFem(lastWord)
	case "х", "ых":
		newWord = transformToGenitivePlural(lastWord)
	default:
		return words
	}

	if newWord != "" && newWord != lastWord {
		wordList[len(wordList)-1] = newWord
		return strings.Join(wordList, " ")
	}
	return words
}

func transformToGenitive(word string, gender Gender) string {
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
		if strings.HasSuffix(word, "ое") {
			return strings.TrimSuffix(word, "ое") + "ого"
		}
	}
	return word
}

func transformToDative(word string, gender Gender) string {
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

func transformToInstrumental(word string, gender Gender) string {
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

func transformToPrepositional(word string, gender Gender) string {
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

func transformToAccusativeFem(word string) string {
	if strings.HasSuffix(word, "ая") {
		return strings.TrimSuffix(word, "ая") + "ую"
	}
	if strings.HasSuffix(word, "яя") {
		return strings.TrimSuffix(word, "яя") + "юю"
	}
	return word
}

// transformToGenitivePlural transforms ordinal to genitive plural form
// Used for decades like "90-х" → "девяностых"
func transformToGenitivePlural(word string) string {
	// Handle ordinal endings
	if strings.HasSuffix(word, "ый") {
		return strings.TrimSuffix(word, "ый") + "ых"
	}
	if strings.HasSuffix(word, "ий") {
		return strings.TrimSuffix(word, "ий") + "их"
	}
	if strings.HasSuffix(word, "ой") {
		return strings.TrimSuffix(word, "ой") + "ых"
	}
	// Handle cardinal tens (девяносто → девяностых)
	if strings.HasSuffix(word, "о") {
		return word + "х"
	}
	if strings.HasSuffix(word, "ь") {
		return strings.TrimSuffix(word, "ь") + "ых"
	}
	return word + "х"
}

// decadeSpecialCases maps round numbers to their decade forms in genitive plural
// These are compound forms that don't follow regular ordinal rules
var decadeSpecialCases = map[int64]string{
	1000:  "тысячных",
	2000:  "двухтысячных",
	3000:  "трёхтысячных",
	4000:  "четырёхтысячных",
	5000:  "пятитысячных",
	6000:  "шеститысячных",
	7000:  "семитысячных",
	8000:  "восьмитысячных",
	9000:  "девятитысячных",
	10000: "десятитысячных",
}

// LanguageCode returns the ISO 639-1 language code.
func (r *RussianConverter) LanguageCode() string {
	return "ru"
}

// LanguageName returns the human-readable language name.
func (r *RussianConverter) LanguageName() string {
	return "Russian"
}

// SupportsContext returns true as Russian requires gender/case context.
func (r *RussianConverter) SupportsContext() bool {
	return true
}

// ToWords converts a number to Russian words with the given context.
func (r *RussianConverter) ToWords(n int64, ctx Context) string {
	if ctx.Form == Ordinal {
		return r.toOrdinal(n, ctx.Gender)
	}
	return r.toCardinal(n, ctx.Gender)
}

// toCardinal converts a number to cardinal Russian words.
func (r *RussianConverter) toCardinal(n int64, gender Gender) string {
	if n < 0 {
		return "минус " + r.toCardinal(-n, gender)
	}
	if n == 0 {
		return onesCardinalRU[gender][0]
	}
	return strings.TrimSpace(r.cardinalRecursive(n, gender, 0))
}

// cardinalRecursive handles recursive conversion for cardinals.
// scaleLevel: 0=ones, 1=thousands, 2=millions, etc.
func (r *RussianConverter) cardinalRecursive(n int64, gender Gender, scaleLevel int) string {
	if n == 0 {
		return ""
	}

	// Determine the gender for this scale level
	// Thousands are feminine, millions/billions/trillions are masculine
	currentGender := gender
	if scaleLevel == 1 {
		currentGender = Feminine
	} else if scaleLevel > 1 {
		currentGender = Masculine
	}

	switch {
	case n < 10:
		return onesCardinalRU[currentGender][n]
	case n < 20:
		return teensCardinalRU[n-10]
	case n < 100:
		tens := tensCardinalRU[n/10]
		ones := n % 10
		if ones == 0 {
			return tens
		}
		return tens + " " + onesCardinalRU[currentGender][ones]
	case n < 1000:
		hundreds := hundredsCardinalRU[n/100]
		remainder := n % 100
		if remainder == 0 {
			return hundreds
		}
		return hundreds + " " + r.cardinalRecursive(remainder, gender, scaleLevel)
	default:
		return r.cardinalWithScale(n, gender)
	}
}

// cardinalWithScale handles numbers >= 1000.
func (r *RussianConverter) cardinalWithScale(n int64, gender Gender) string {
	if n == 0 {
		return ""
	}

	var parts []string
	scaleLevel := 0
	remaining := n

	for remaining > 0 && scaleLevel < len(scalesRU) {
		chunk := remaining % 1000
		remaining = remaining / 1000

		if chunk > 0 {
			// Determine gender for this chunk
			chunkGender := gender
			if scaleLevel == 1 {
				chunkGender = Feminine // thousands are feminine
			} else if scaleLevel > 1 {
				chunkGender = Masculine // millions, billions are masculine
			}

			chunkWords := r.cardinalChunk(chunk, chunkGender)
			if scaleLevel > 0 {
				scaleWord := r.getScaleWord(chunk, scaleLevel)
				chunkWords = chunkWords + " " + scaleWord
			}
			parts = append([]string{chunkWords}, parts...)
		}
		scaleLevel++
	}

	return strings.Join(parts, " ")
}

// cardinalChunk converts a number 0-999 to words.
func (r *RussianConverter) cardinalChunk(n int64, gender Gender) string {
	if n == 0 {
		return ""
	}
	if n < 10 {
		return onesCardinalRU[gender][n]
	}
	if n < 20 {
		return teensCardinalRU[n-10]
	}
	if n < 100 {
		tens := tensCardinalRU[n/10]
		ones := n % 10
		if ones == 0 {
			return tens
		}
		return tens + " " + onesCardinalRU[gender][ones]
	}

	hundreds := hundredsCardinalRU[n/100]
	remainder := n % 100
	if remainder == 0 {
		return hundreds
	}
	return hundreds + " " + r.cardinalChunk(remainder, gender)
}

// getScaleWord returns the appropriate scale word (тысяча/тысячи/тысяч, etc.)
func (r *RussianConverter) getScaleWord(n int64, scaleLevel int) string {
	if scaleLevel == 0 || scaleLevel >= len(scalesRU) {
		return ""
	}

	scale := scalesRU[scaleLevel]
	lastTwo := n % 100
	lastOne := n % 10

	// Special cases for 11-14
	if lastTwo >= 11 && lastTwo <= 14 {
		return scale.many
	}

	switch lastOne {
	case 1:
		return scale.one
	case 2, 3, 4:
		return scale.few
	default:
		return scale.many
	}
}

// toOrdinal converts a number to ordinal Russian words.
func (r *RussianConverter) toOrdinal(n int64, gender Gender) string {
	if n < 0 {
		return "минус " + r.toOrdinal(-n, gender)
	}
	if n == 0 {
		return onesOrdinalRU[gender][0]
	}
	return strings.TrimSpace(r.ordinalRecursive(n, gender, true))
}

// ordinalRecursive handles recursive conversion for ordinals.
// isLast indicates if this is the last (rightmost) part that should be ordinal.
func (r *RussianConverter) ordinalRecursive(n int64, gender Gender, isLast bool) string {
	switch {
	case n < 10:
		if isLast {
			return onesOrdinalRU[gender][n]
		}
		return onesCardinalRU[Masculine][n]
	case n < 20:
		if isLast {
			return teensOrdinalRU[gender][n-10]
		}
		return teensCardinalRU[n-10]
	case n < 100:
		tens := n / 10
		ones := n % 10
		if ones == 0 {
			if isLast {
				return tensOrdinalRU[gender][tens]
			}
			return tensCardinalRU[tens]
		}
		return tensCardinalRU[tens] + " " + r.ordinalRecursive(ones, gender, isLast)
	case n < 1000:
		hundreds := n / 100
		remainder := n % 100
		if remainder == 0 {
			if isLast {
				return hundredsOrdinalRU[gender][hundreds]
			}
			return hundredsCardinalRU[hundreds]
		}
		return hundredsCardinalRU[hundreds] + " " + r.ordinalRecursive(remainder, gender, isLast)
	default:
		return r.ordinalWithScale(n, gender)
	}
}

// ordinalWithScale handles ordinal numbers >= 1000.
func (r *RussianConverter) ordinalWithScale(n int64, gender Gender) string {
	// Find the highest scale
	scaleLevel := 0
	temp := n
	for temp >= 1000 && scaleLevel < len(scalesRU)-1 {
		temp /= 1000
		scaleLevel++
	}

	// Get the divisor for this scale
	divisor := int64(1)
	for i := 0; i < scaleLevel; i++ {
		divisor *= 1000
	}

	highPart := n / divisor
	remainder := n % divisor

	if remainder == 0 {
		// The whole number is at this scale, use ordinal form
		if highPart == 1 {
			return scaleOrdinalRU[gender][scaleLevel]
		}
		// For numbers like 2000, 3000, etc.
		chunkGender := Masculine
		if scaleLevel == 1 {
			chunkGender = Feminine
		}
		return r.cardinalChunk(highPart, chunkGender) + " " + scaleOrdinalRU[gender][scaleLevel]
	}

	// There's a remainder, so the scale word is cardinal
	var parts []string

	// Add the high part with cardinal scale word
	chunkGender := Masculine
	if scaleLevel == 1 {
		chunkGender = Feminine
	}
	highWords := r.cardinalChunk(highPart, chunkGender)
	scaleWord := r.getScaleWord(highPart, scaleLevel)
	parts = append(parts, highWords+" "+scaleWord)

	// Add the remainder as ordinal
	parts = append(parts, r.ordinalRecursive(remainder, gender, true))

	return strings.Join(parts, " ")
}

// processDateRanges handles date range patterns like "6-16 августа".
// Both numbers are converted to ordinal neuter (for dates) with comma and "тире" between them.
// Example: "6-16 августа" → "шестое, тире, шестнадцатое августа"
func (p *RussianProcessor) processDateRanges(text string, converter NumberConverter) string {
	matches := RussianDateRangePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	result := text
	// Process from end to start to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullStart, fullEnd := match[0], match[1]
		num1Start, num1End := match[2], match[3]
		num2Start, num2End := match[4], match[5]
		monthStart, monthEnd := match[6], match[7]

		num1Str := result[num1Start:num1End]
		num2Str := result[num2Start:num2End]
		month := result[monthStart:monthEnd]

		n1, err1 := strconv.ParseInt(num1Str, 10, 64)
		n2, err2 := strconv.ParseInt(num2Str, 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		// Convert both numbers to ordinal neuter (for dates like "шестое", "шестнадцатое")
		ctx := Context{
			Form:   Ordinal,
			Gender: Neuter,
			Case:   Nominative,
		}

		words1 := converter.ToWords(n1, ctx)
		words2 := converter.ToWords(n2, ctx)

		// Build replacement: "шестое, тире, шестнадцатое августа"
		replacement := words1 + ", тире, " + words2 + " " + month
		result = result[:fullStart] + replacement + result[fullEnd:]
	}

	return result
}

// processYearOfBirth handles the "г.р." (года рождения - year of birth) abbreviation.
// It converts the year to genitive ordinal case and expands "г.р." to "года рождения".
// Example: "1968 г. р." → "одна тысяча девятьсот шестьдесят восьмого года рождения"
func (p *RussianProcessor) processYearOfBirth(text string, converter NumberConverter) string {
	matches := RussianYearOfBirthPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	result := text
	// Process from end to start to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullStart, fullEnd := match[0], match[1]
		numStart, numEnd := match[2], match[3]

		numStr := result[numStart:numEnd]
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			continue
		}

		// Convert year to ordinal masculine (for "год")
		ctx := Context{
			Form:   Ordinal,
			Gender: Masculine,
			Case:   Nominative,
		}
		words := converter.ToWords(n, ctx)

		// Transform to genitive case (восьмой → восьмого)
		words = TransformOrdinalCase(words, Genitive, Masculine)

		// Replace the entire match with ordinal year + "года рождения"
		replacement := words + " года рождения"
		result = result[:fullStart] + replacement + result[fullEnd:]
	}

	return result
}

// PreProcess handles Russian-specific preprocessing before number replacement.
// It processes Roman numerals, ordinal suffixes like "1996-м году" and abbreviations like "гг." before general number processing.
func (p *RussianProcessor) PreProcess(text string, converter NumberConverter) string {
	result := text

	// First, process Roman numerals (e.g., "Глава I" → "Глава одна", "I Глава" → "Первая Глава")
	result = ProcessRomanNumerals(result, "ru", converter, p.nounDB)

	// Expand "гг." abbreviation to "годов"
	result = RussianYearAbbrevPattern.ReplaceAllString(result, "годов")

	// Process "<number> г.р." pattern (year of birth) - convert year to genitive ordinal + "года рождения"
	result = p.processYearOfBirth(result, converter)

	// Process date ranges like "6-16 августа" → "шестое, тире, шестнадцатое августа"
	result = p.processDateRanges(result, converter)

	// Find all ordinal suffix matches from end to start
	matches := RussianOrdinalSuffixPattern.FindAllStringSubmatchIndex(result, -1)
	if len(matches) == 0 {
		return result
	}

	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullStart, fullEnd := match[0], match[1]
		numStart, numEnd := match[2], match[3]
		suffixStart, suffixEnd := match[4], match[5]

		numStr := result[numStart:numEnd]
		suffix := result[suffixStart:suffixEnd]

		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			continue
		}

		// Determine gender from suffix
		gender := GenderFromSuffix(suffix)

		var words string

		// Check for special decade cases (e.g., "2000-х" → "двухтысячных")
		if (suffix == "х" || suffix == "ых") && decadeSpecialCases[n] != "" {
			words = decadeSpecialCases[n]
		} else {
			ctx := Context{
				Form:   Ordinal,
				Gender: gender,
				Case:   Nominative,
			}

			// Convert number to ordinal words (nominative case)
			words = converter.ToWords(n, ctx)

			// Apply case ending transformation based on suffix
			words = TransformBySuffix(words, suffix, gender)
		}

		// Replace the entire match (number + hyphen + suffix) with the ordinal word
		result = result[:fullStart] + words + result[fullEnd:]
	}

	return result
}

// DetectContext determines grammatical context from surrounding words for Russian.
// Returns the context and true if Russian-specific detection was applied.
func (p *RussianProcessor) DetectContext(wordBefore, wordAfter string, nounDB *NounDatabase) (Context, bool) {
	ctx := DefaultContext()

	// Check if word before triggers genitive case (for dates)
	genitiveTriggered := wordBefore != "" && IsGenitiveTrigger(wordBefore)

	// Check word before - skip if it's a Russian month (doesn't affect following number)
	if wordBefore != "" && !genitiveTriggered {
		if IsRussianMonth(strings.ToLower(wordBefore)) {
			// Month before number doesn't affect it (e.g., "марта 1996")
			// Fall through to check word after
		} else if info, ok := nounDB.Lookup("ru", wordBefore); ok {
			ctx.Form = info.TriggerForm
			ctx.Gender = info.Gender
			return ctx, true
		}
	}

	// Check word after - handles patterns like "5 рублей", "25 марта"
	if wordAfter != "" {
		if info, ok := nounDB.Lookup("ru", wordAfter); ok {
			ctx.Gender = info.Gender

			lowerWord := strings.ToLower(wordAfter)
			if IsRussianMonth(lowerWord) {
				// Dates: "25 марта" → ordinal neuter
				// Use genitive case if triggered by preceding word
				ctx.Form = Ordinal
				if genitiveTriggered {
					ctx.Case = Genitive
				}
				return ctx, true
			} else if lowerWord == "год" {
				// Year nominative: "1996 год" → ordinal masculine
				ctx.Form = Ordinal
				ctx.Gender = Masculine
				return ctx, true
			} else if lowerWord == "года" {
				// Year genitive: "1996 года" → ordinal masculine genitive
				ctx.Form = Ordinal
				ctx.Gender = Masculine
				ctx.Case = Genitive
				return ctx, true
			}

			// Default for quantities: cardinal
			ctx.Form = Cardinal
			return ctx, true
		}

		// Fallback: noun not in database - use ending-based gender detection
		// Default to cardinal form since we can't determine ordinal triggers
		ctx.Gender = GenderFromNounEnding(wordAfter)
		ctx.Form = Cardinal
		return ctx, true
	}

	return ctx, false
}

// PostProcessContext applies Russian-specific post-processing to ordinal words.
// It handles case transformations for ordinals.
func (p *RussianProcessor) PostProcessContext(words string, ctx Context) string {
	if ctx.Form == Ordinal && ctx.Case != Nominative {
		return TransformOrdinalCase(words, ctx.Case, ctx.Gender)
	}
	return words
}

// GetChapterGender returns Feminine because "глава" (chapter) is feminine in Russian.
func (p *RussianProcessor) GetChapterGender() Gender {
	return Feminine
}
