package normalize

import (
	"strings"
)

// NounDatabase stores grammatical information about nouns for context detection.
type NounDatabase struct {
	nouns map[string]map[string]NounInfo // language -> normalized_noun -> info
}

// NewNounDatabase creates a new noun database with built-in entries.
func NewNounDatabase() *NounDatabase {
	db := &NounDatabase{
		nouns: make(map[string]map[string]NounInfo),
	}
	db.loadEnglishNouns()
	db.loadRussianNouns()
	return db
}

// Lookup finds noun information by language and noun form.
// It checks the noun and common variations (singular/plural forms).
func (db *NounDatabase) Lookup(lang, noun string) (NounInfo, bool) {
	langNouns, ok := db.nouns[lang]
	if !ok {
		return NounInfo{}, false
	}

	// Normalize: lowercase for English, lowercase for Russian
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

// loadEnglishNouns loads built-in English nouns.
func (db *NounDatabase) loadEnglishNouns() {
	db.nouns["en"] = map[string]NounInfo{
		// Ordinal triggers (chapter, page, etc.)
		"chapter": {
			Gender:       Masculine, // English doesn't use gender, but we need a default
			TriggerForm:  Ordinal,
			SingularForm: "chapter",
			PluralForms:  []string{"chapters"},
		},
		"page": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "page",
			PluralForms:  []string{"pages"},
		},
		"part": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "part",
			PluralForms:  []string{"parts"},
		},
		"section": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "section",
			PluralForms:  []string{"sections"},
		},
		"volume": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "volume",
			PluralForms:  []string{"volumes"},
		},
		"book": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "book",
			PluralForms:  []string{"books"},
		},
		"episode": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "episode",
			PluralForms:  []string{"episodes"},
		},
		"act": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "act",
			PluralForms:  []string{"acts"},
		},
		"scene": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "scene",
			PluralForms:  []string{"scenes"},
		},
		"lesson": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "lesson",
			PluralForms:  []string{"lessons"},
		},
		"step": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "step",
			PluralForms:  []string{"steps"},
		},
		"level": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "level",
			PluralForms:  []string{"levels"},
		},
		"floor": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "floor",
			PluralForms:  []string{"floors"},
		},
		"grade": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "grade",
			PluralForms:  []string{"grades"},
		},

		// Cardinal triggers (quantities)
		"dollar": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "dollar",
			PluralForms:  []string{"dollars"},
		},
		"cent": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "cent",
			PluralForms:  []string{"cents"},
		},
		"pound": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "pound",
			PluralForms:  []string{"pounds"},
		},
		"euro": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "euro",
			PluralForms:  []string{"euros"},
		},
		"year": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "year",
			PluralForms:  []string{"years"},
		},
		"month": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "month",
			PluralForms:  []string{"months"},
		},
		"day": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "day",
			PluralForms:  []string{"days"},
		},
		"hour": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "hour",
			PluralForms:  []string{"hours"},
		},
		"minute": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "minute",
			PluralForms:  []string{"minutes"},
		},
		"second": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "second",
			PluralForms:  []string{"seconds"},
		},
		"percent": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "percent",
			PluralForms:  []string{"percents"},
		},
		"mile": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "mile",
			PluralForms:  []string{"miles"},
		},
		"kilometer": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "kilometer",
			PluralForms:  []string{"kilometers"},
		},
		"meter": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "meter",
			PluralForms:  []string{"meters"},
		},
		"foot": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "foot",
			PluralForms:  []string{"feet"},
		},
		"inch": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "inch",
			PluralForms:  []string{"inches"},
		},
		"person": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "person",
			PluralForms:  []string{"people", "persons"},
		},
		"time": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "time",
			PluralForms:  []string{"times"},
		},
	}
}

// loadRussianNouns loads built-in Russian nouns.
func (db *NounDatabase) loadRussianNouns() {
	db.nouns["ru"] = map[string]NounInfo{
		// Ordinal triggers (feminine)
		"глава": {
			Gender:       Feminine,
			TriggerForm:  Ordinal,
			SingularForm: "глава",
			PluralForms:  []string{"главы", "глав"},
		},
		"страница": {
			Gender:       Feminine,
			TriggerForm:  Ordinal,
			SingularForm: "страница",
			PluralForms:  []string{"страницы", "страниц"},
		},
		"часть": {
			Gender:       Feminine,
			TriggerForm:  Ordinal,
			SingularForm: "часть",
			PluralForms:  []string{"части", "частей"},
		},
		"книга": {
			Gender:       Feminine,
			TriggerForm:  Ordinal,
			SingularForm: "книга",
			PluralForms:  []string{"книги", "книг"},
		},
		"серия": {
			Gender:       Feminine,
			TriggerForm:  Ordinal,
			SingularForm: "серия",
			PluralForms:  []string{"серии", "серий"},
		},
		"сцена": {
			Gender:       Feminine,
			TriggerForm:  Ordinal,
			SingularForm: "сцена",
			PluralForms:  []string{"сцены", "сцен"},
		},

		// Ordinal triggers (masculine)
		"том": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "том",
			PluralForms:  []string{"тома", "томов"},
		},
		"раздел": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "раздел",
			PluralForms:  []string{"раздела", "разделов"},
		},
		"акт": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "акт",
			PluralForms:  []string{"акта", "актов"},
		},
		"эпизод": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "эпизод",
			PluralForms:  []string{"эпизода", "эпизодов"},
		},
		"урок": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "урок",
			PluralForms:  []string{"урока", "уроков"},
		},
		"этаж": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "этаж",
			PluralForms:  []string{"этажа", "этажей"},
		},
		"класс": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "класс",
			PluralForms:  []string{"класса", "классов"},
		},
		"уровень": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "уровень",
			PluralForms:  []string{"уровня", "уровней"},
		},
		"шаг": {
			Gender:       Masculine,
			TriggerForm:  Ordinal,
			SingularForm: "шаг",
			PluralForms:  []string{"шага", "шагов"},
		},

		// Cardinal triggers (masculine)
		"рубль": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "рубль",
			PluralForms:  []string{"рубля", "рублей"},
		},
		"доллар": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "доллар",
			PluralForms:  []string{"доллара", "долларов"},
		},
		"евро": {
			Gender:       Neuter,
			TriggerForm:  Cardinal,
			SingularForm: "евро",
			PluralForms:  []string{"евро"},
		},
		"год": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "год",
			PluralForms:  []string{"года", "лет"},
		},
		"месяц": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "месяц",
			PluralForms:  []string{"месяца", "месяцев"},
		},
		"день": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "день",
			PluralForms:  []string{"дня", "дней"},
		},
		"час": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "час",
			PluralForms:  []string{"часа", "часов"},
		},
		"человек": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "человек",
			PluralForms:  []string{"человека", "человек"},
		},
		"раз": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "раз",
			PluralForms:  []string{"раза", "раз"},
		},
		"процент": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "процент",
			PluralForms:  []string{"процента", "процентов"},
		},
		"километр": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "километр",
			PluralForms:  []string{"километра", "километров"},
		},
		"метр": {
			Gender:       Masculine,
			TriggerForm:  Cardinal,
			SingularForm: "метр",
			PluralForms:  []string{"метра", "метров"},
		},

		// Cardinal triggers (feminine)
		"копейка": {
			Gender:       Feminine,
			TriggerForm:  Cardinal,
			SingularForm: "копейка",
			PluralForms:  []string{"копейки", "копеек"},
		},
		"минута": {
			Gender:       Feminine,
			TriggerForm:  Cardinal,
			SingularForm: "минута",
			PluralForms:  []string{"минуты", "минут"},
		},
		"секунда": {
			Gender:       Feminine,
			TriggerForm:  Cardinal,
			SingularForm: "секунда",
			PluralForms:  []string{"секунды", "секунд"},
		},
		"неделя": {
			Gender:       Feminine,
			TriggerForm:  Cardinal,
			SingularForm: "неделя",
			PluralForms:  []string{"недели", "недель"},
		},
		"штука": {
			Gender:       Feminine,
			TriggerForm:  Cardinal,
			SingularForm: "штука",
			PluralForms:  []string{"штуки", "штук"},
		},
		"миля": {
			Gender:       Feminine,
			TriggerForm:  Cardinal,
			SingularForm: "миля",
			PluralForms:  []string{"мили", "миль"},
		},
	}
}
