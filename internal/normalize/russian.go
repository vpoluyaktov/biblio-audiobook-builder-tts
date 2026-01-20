package normalize

import (
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

func init() {
	Register(&RussianConverter{})
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
