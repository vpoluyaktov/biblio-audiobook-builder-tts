package normalize

import (
	"strings"
)

// EnglishConverter converts numbers to English words.
type EnglishConverter struct{}

// Ensure EnglishConverter implements NumberConverter.
var _ NumberConverter = (*EnglishConverter)(nil)

var (
	onesEN = [...]string{
		"zero", "one", "two", "three", "four",
		"five", "six", "seven", "eight", "nine",
	}

	teensEN = [...]string{
		"ten", "eleven", "twelve", "thirteen", "fourteen",
		"fifteen", "sixteen", "seventeen", "eighteen", "nineteen",
	}

	tensEN = [...]string{
		"", "", "twenty", "thirty", "forty",
		"fifty", "sixty", "seventy", "eighty", "ninety",
	}

	onesOrdinalEN = [...]string{
		"zeroth", "first", "second", "third", "fourth",
		"fifth", "sixth", "seventh", "eighth", "ninth",
	}

	teensOrdinalEN = [...]string{
		"tenth", "eleventh", "twelfth", "thirteenth", "fourteenth",
		"fifteenth", "sixteenth", "seventeenth", "eighteenth", "nineteenth",
	}

	tensOrdinalEN = [...]string{
		"", "", "twentieth", "thirtieth", "fortieth",
		"fiftieth", "sixtieth", "seventieth", "eightieth", "ninetieth",
	}

	englishOrdinalSuffixes = []string{
		"first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth",
		"eleventh", "twelfth", "thirteenth", "fourteenth", "fifteenth", "sixteenth", "seventeenth", "eighteenth", "nineteenth", "twentieth",
		"twenty-first", "twenty-second", "twenty-third", "twenty-fourth", "twenty-fifth", "twenty-sixth", "twenty-seventh", "twenty-eighth", "twenty-ninth", "thirtieth",
		"thirty-first", "fortieth",
		"fiftieth", "sixtieth", "seventieth", "eightieth", "ninetieth",
	}
)

// EnglishProcessor implements LanguageProcessor for English-specific text processing.
type EnglishProcessor struct {
	nounDB *NounDatabase
}

// Ensure EnglishProcessor implements LanguageProcessor.
var _ LanguageProcessor = (*EnglishProcessor)(nil)

func init() {
	Register(&EnglishConverter{})
	RegisterLanguageProcessor("en", &EnglishProcessor{nounDB: NewNounDatabase()})
}

// LanguageCode returns the ISO 639-1 language code.
func (e *EnglishConverter) LanguageCode() string {
	return "en"
}

// LanguageName returns the human-readable language name.
func (e *EnglishConverter) LanguageName() string {
	return "English"
}

// SupportsContext returns false as English doesn't require gender/case context.
func (e *EnglishConverter) SupportsContext() bool {
	return false
}

// ToWords converts a number to English words.
func (e *EnglishConverter) ToWords(n int64, ctx Context) string {
	if ctx.Form == Ordinal {
		return e.toOrdinal(n)
	}
	return e.toCardinal(n)
}

// toCardinal converts a number to cardinal words (one, two, three).
func (e *EnglishConverter) toCardinal(n int64) string {
	if n < 0 {
		return "minus " + e.toCardinal(-n)
	}
	if n == 0 {
		return onesEN[0]
	}
	return strings.TrimSpace(e.cardinalRecursive(n))
}

// cardinalRecursive handles the recursive conversion for cardinals.
func (e *EnglishConverter) cardinalRecursive(n int64) string {
	switch {
	case n < 10:
		return onesEN[n]
	case n < 20:
		return teensEN[n-10]
	case n < 100:
		if n%10 == 0 {
			return tensEN[n/10]
		}
		return tensEN[n/10] + "-" + onesEN[n%10]
	case n < 1000:
		remainder := n % 100
		if remainder == 0 {
			return onesEN[n/100] + " hundred"
		}
		return onesEN[n/100] + " hundred " + e.cardinalRecursive(remainder)
	case n < 1000000:
		remainder := n % 1000
		thousands := e.cardinalRecursive(n / 1000)
		if remainder == 0 {
			return thousands + " thousand"
		}
		return thousands + " thousand " + e.cardinalRecursive(remainder)
	case n < 1000000000:
		remainder := n % 1000000
		millions := e.cardinalRecursive(n / 1000000)
		if remainder == 0 {
			return millions + " million"
		}
		return millions + " million " + e.cardinalRecursive(remainder)
	case n < 1000000000000:
		remainder := n % 1000000000
		billions := e.cardinalRecursive(n / 1000000000)
		if remainder == 0 {
			return billions + " billion"
		}
		return billions + " billion " + e.cardinalRecursive(remainder)
	default:
		remainder := n % 1000000000000
		trillions := e.cardinalRecursive(n / 1000000000000)
		if remainder == 0 {
			return trillions + " trillion"
		}
		return trillions + " trillion " + e.cardinalRecursive(remainder)
	}
}

// PreProcess handles English-specific preprocessing before number replacement.
// It converts Roman numerals to Arabic numbers based on context.
func (p *EnglishProcessor) PreProcess(text string, converter NumberConverter) string {
	return ProcessRomanNumerals(text, "en", converter, p.nounDB)
}

// DetectContext determines grammatical context from surrounding words for English.
// English doesn't have grammatical gender, so this mainly handles ordinal triggers.
func (p *EnglishProcessor) DetectContext(wordBefore, wordAfter string, nounDB *NounDatabase) (Context, bool) {
	ctx := DefaultContext()

	// Check word before - handles patterns like "Chapter 5"
	if wordBefore != "" {
		if info, ok := nounDB.Lookup("en", wordBefore); ok {
			ctx.Form = info.TriggerForm
			return ctx, true
		}
	}

	// Check word after - handles patterns like "5 dollars"
	if wordAfter != "" {
		if _, ok := nounDB.Lookup("en", wordAfter); ok {
			ctx.Form = Cardinal
			return ctx, true
		}
	}

	return ctx, false
}

// PostProcessContext applies English-specific post-processing.
// English doesn't need case transformations, so this is a no-op.
func (p *EnglishProcessor) PostProcessContext(words string, ctx Context) string {
	return words
}

// GetChapterGender returns Masculine as English doesn't have grammatical gender.
func (p *EnglishProcessor) GetChapterGender() Gender {
	return Masculine
}


// toOrdinal converts a number to ordinal words (first, second, third).
func (e *EnglishConverter) toOrdinal(n int64) string {
	if n < 0 {
		return "minus " + e.toOrdinal(-n)
	}
	if n == 0 {
		return onesOrdinalEN[0]
	}
	return strings.TrimSpace(e.ordinalRecursive(n, true))
}

// ordinalRecursive handles the recursive conversion for ordinals.
// isLast indicates if this is the last (rightmost) part that should be ordinal.
func (e *EnglishConverter) ordinalRecursive(n int64, isLast bool) string {
	switch {
	case n < 10:
		if isLast {
			return onesOrdinalEN[n]
		}
		return onesEN[n]
	case n < 20:
		if isLast {
			return teensOrdinalEN[n-10]
		}
		return teensEN[n-10]
	case n < 100:
		if n%10 == 0 {
			if isLast {
				return tensOrdinalEN[n/10]
			}
			return tensEN[n/10]
		}
		return tensEN[n/10] + "-" + e.ordinalRecursive(n%10, isLast)
	case n < 1000:
		remainder := n % 100
		if remainder == 0 {
			if isLast {
				return onesEN[n/100] + " hundredth"
			}
			return onesEN[n/100] + " hundred"
		}
		return onesEN[n/100] + " hundred " + e.ordinalRecursive(remainder, isLast)
	case n < 1000000:
		remainder := n % 1000
		thousands := e.cardinalRecursive(n / 1000)
		if remainder == 0 {
			if isLast {
				return thousands + " thousandth"
			}
			return thousands + " thousand"
		}
		return thousands + " thousand " + e.ordinalRecursive(remainder, isLast)
	case n < 1000000000:
		remainder := n % 1000000
		millions := e.cardinalRecursive(n / 1000000)
		if remainder == 0 {
			if isLast {
				return millions + " millionth"
			}
			return millions + " million"
		}
		return millions + " million " + e.ordinalRecursive(remainder, isLast)
	case n < 1000000000000:
		remainder := n % 1000000000
		billions := e.cardinalRecursive(n / 1000000000)
		if remainder == 0 {
			if isLast {
				return billions + " billionth"
			}
			return billions + " billion"
		}
		return billions + " billion " + e.ordinalRecursive(remainder, isLast)
	default:
		remainder := n % 1000000000000
		trillions := e.cardinalRecursive(n / 1000000000000)
		if remainder == 0 {
			if isLast {
				return trillions + " trillionth"
			}
			return trillions + " trillion"
		}
		return trillions + " trillion " + e.ordinalRecursive(remainder, isLast)
	}
}
