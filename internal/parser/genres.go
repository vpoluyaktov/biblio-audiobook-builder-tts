package parser

import "strings"

// fb2GenresEN maps FB2 genre codes to English human-readable names.
var fb2GenresEN = map[string]string{
	// Science Fiction & Fantasy
	"sf":                 "Science Fiction",
	"sf_action":          "Action SF",
	"sf_epic":            "Epic SF",
	"sf_heroic":          "Heroic SF",
	"sf_detective":       "Detective SF",
	"sf_cyberpunk":       "Cyberpunk",
	"sf_space":           "Space SF",
	"sf_social":          "Social SF",
	"sf_horror":          "Horror SF",
	"sf_humor":           "Humorous SF",
	"sf_fantasy":         "Fantasy",
	"sf_history":         "Historical SF",
	"sf_etc":             "Other SF",
	"sf_stimpank":        "Steampunk",
	"sf_technofantas":    "Technofantasy",
	"sf_litrpg":          "LitRPG",
	"sf_postapocalyptic": "Post-Apocalyptic",
	"sf_mystic":          "Mystic SF",
	"sf_camping":         "Camping SF",
	"sf_realrpg":         "RealRPG",

	// Detective & Thriller
	"det_classic":    "Classic Detective",
	"det_police":     "Police Detective",
	"det_action":     "Action",
	"det_irony":      "Ironic Detective",
	"det_history":    "Historical Detective",
	"det_espionage":  "Espionage",
	"det_crime":      "Crime",
	"det_political":  "Political Detective",
	"det_maniac":     "Maniac",
	"det_hard":       "Hard-Boiled",
	"det_cozy":       "Cozy Mystery",
	"det_su":         "Soviet Detective",
	"detective":      "Detective",

	// Prose
	"prose_classic":       "Classic Prose",
	"prose_history":       "Historical Prose",
	"prose_contemporary":  "Contemporary Prose",
	"prose_counter":       "Counterculture",
	"prose_rus_classic":   "Russian Classic",
	"prose_su_classics":   "Soviet Classic",
	"prose_military":      "Military Prose",
	"prose":               "Prose",
	"prose_magic":         "Magical Realism",
	"prose_abs":           "Absurdist Prose",
	"prose_neformatic":    "Nonconformist Prose",

	// Romance & Love
	"love_contemporary": "Contemporary Romance",
	"love_history":      "Historical Romance",
	"love_detective":    "Romantic Detective",
	"love_short":        "Short Romance",
	"love_erotica":      "Erotica",
	"love_sf":           "Romantic SF",
	"love":              "Romance",

	// Adventure
	"adv_western":  "Western",
	"adv_history":  "Historical Adventure",
	"adv_indian":   "Indian Adventure",
	"adv_maritime": "Maritime Adventure",
	"adv_geo":      "Travel & Geography",
	"adv_animal":   "Animal Adventure",
	"adventure":    "Adventure",

	// Children
	"child_tale":      "Fairy Tale",
	"child_verse":     "Children's Verse",
	"child_prose":     "Children's Prose",
	"child_sf":        "Children's SF",
	"child_det":       "Children's Detective",
	"child_adv":       "Children's Adventure",
	"child_education": "Children's Education",
	"children":        "Children's Literature",

	// Poetry & Drama
	"poetry":      "Poetry",
	"dramaturgy":  "Dramaturgy",

	// Antique Literature
	"antique_ant":      "Antique Literature",
	"antique_european": "European Antique",
	"antique_russian":  "Russian Antique",
	"antique_east":     "Eastern Antique",
	"antique_myths":    "Myths & Legends",
	"antique":          "Antique",

	// Science & Education
	"sci_history":    "History",
	"sci_psychology": "Psychology",
	"sci_culture":    "Cultural Studies",
	"sci_religion":   "Religious Studies",
	"sci_philosophy": "Philosophy",
	"sci_politics":   "Political Science",
	"sci_business":   "Business",
	"sci_juris":      "Law",
	"sci_linguistic": "Linguistics",
	"sci_medicine":   "Medicine",
	"sci_phys":       "Physics",
	"sci_math":       "Mathematics",
	"sci_chem":       "Chemistry",
	"sci_biology":    "Biology",
	"sci_tech":       "Technology",
	"sci_ecology":    "Ecology",
	"sci_geo":        "Geography",
	"science":        "Science",
	"sci_cosmos":     "Astronomy",
	"sci_pedagogy":   "Pedagogy",
	"sci_social":     "Sociology",
	"sci_economy":    "Economics",
	"sci_state":      "State Science",
	"sci_zoo":        "Zoology",
	"sci_botany":     "Botany",
	"sci_philology":  "Philology",
	"sci_orgchem":    "Organic Chemistry",
	"sci_anachem":    "Analytical Chemistry",

	// Computers
	"comp_www":         "Internet",
	"comp_programming": "Programming",
	"comp_hard":        "Hardware",
	"comp_soft":        "Software",
	"comp_db":          "Databases",
	"comp_osnet":       "OS & Networking",
	"computers":        "Computers",

	// Reference
	"ref_encyc":  "Encyclopedia",
	"ref_dict":   "Dictionary",
	"ref_ref":    "Reference",
	"ref_guide":  "Guide",
	"reference":  "Reference",

	// Nonfiction
	"nonf_biography":  "Biography",
	"nonf_publicism":  "Publicism",
	"nonf_criticism":  "Criticism",
	"design":          "Art & Design",
	"nonfiction":      "Nonfiction",
	"geo_guides":      "Travel Guide",

	// Religion & Esoteric
	"religion_rel":          "Religion",
	"religion_esoterics":    "Esoterics",
	"religion_self":         "Self-Help",
	"religion":              "Religion & Spirituality",
	"religion_christianity": "Christianity",
	"religion_orthodoxy":    "Orthodox Christianity",
	"religion_catholicism":  "Catholicism",
	"religion_islam":        "Islam",
	"religion_buddhism":     "Buddhism",
	"religion_hinduism":     "Hinduism",
	"religion_paganism":     "Paganism",

	// Humor
	"humor_prose":    "Humorous Prose",
	"humor_verse":    "Humorous Verse",
	"humor_anecdote": "Anecdotes",
	"humor":          "Humor",
	"humor_satire":   "Satire",

	// Home & Family
	"home_cooking":   "Cooking",
	"home_pets":      "Pets",
	"home_crafts":    "Crafts",
	"home_entertain": "Entertainment",
	"home_health":    "Health",
	"home_garden":    "Garden",
	"home_diy":       "DIY",
	"home_sport":     "Sport",
	"home_sex":       "Erotica & Sex",
	"home":           "Home & Family",

	// Thriller
	"thriller": "Thriller",

	// Other / Misc
	"periodic":      "Periodicals",
	"comics":        "Comics",
	"tale_chivalry": "Chivalric Romance",
	"fanfiction":    "Fan Fiction",
	"unrecognised":  "Other",
}

// fb2GenresRU maps FB2 genre codes to Russian human-readable names.
var fb2GenresRU = map[string]string{
	// Science Fiction & Fantasy
	"sf":                 "Научная фантастика",
	"sf_action":          "Боевая фантастика",
	"sf_epic":            "Эпическая фантастика",
	"sf_heroic":          "Героическая фантастика",
	"sf_detective":       "Детективная фантастика",
	"sf_cyberpunk":       "Киберпанк",
	"sf_space":           "Космическая фантастика",
	"sf_social":          "Социальная фантастика",
	"sf_horror":          "Ужасы и мистика",
	"sf_humor":           "Юмористическая фантастика",
	"sf_fantasy":         "Фэнтези",
	"sf_history":         "Альтернативная история",
	"sf_etc":             "Прочая фантастика",
	"sf_stimpank":        "Стимпанк",
	"sf_technofantas":    "Технофэнтези",
	"sf_litrpg":          "ЛитРПГ",
	"sf_postapocalyptic": "Постапокалипсис",
	"sf_mystic":          "Мистика",
	"sf_camping":         "Попаданцы",
	"sf_realrpg":         "РеалРПГ",

	// Detective & Thriller
	"det_classic":   "Классический детектив",
	"det_police":    "Полицейский детектив",
	"det_action":    "Боевик",
	"det_irony":     "Иронический детектив",
	"det_history":   "Исторический детектив",
	"det_espionage": "Шпионский детектив",
	"det_crime":     "Криминальный детектив",
	"det_political": "Политический детектив",
	"det_maniac":    "Маньяки",
	"det_hard":      "Крутой детектив",
	"det_cozy":      "Уютный детектив",
	"det_su":        "Советский детектив",
	"detective":     "Детектив",

	// Prose
	"prose_classic":      "Классическая проза",
	"prose_history":      "Историческая проза",
	"prose_contemporary": "Современная проза",
	"prose_counter":      "Контркультура",
	"prose_rus_classic":  "Русская классическая проза",
	"prose_su_classics":  "Советская классическая проза",
	"prose_military":     "Военная проза",
	"prose":              "Проза",
	"prose_magic":        "Магический реализм",
	"prose_abs":          "Абсурдистская проза",
	"prose_neformatic":   "Неформатная проза",

	// Romance & Love
	"love_contemporary": "Современные любовные романы",
	"love_history":      "Исторические любовные романы",
	"love_detective":    "Остросюжетные любовные романы",
	"love_short":        "Короткие любовные романы",
	"love_erotica":      "Эротика",
	"love_sf":           "Любовная фантастика",
	"love":              "Любовный роман",

	// Adventure
	"adv_western":  "Вестерн",
	"adv_history":  "Исторические приключения",
	"adv_indian":   "Приключения про индейцев",
	"adv_maritime": "Морские приключения",
	"adv_geo":      "Путешествия и география",
	"adv_animal":   "Природа и животные",
	"adventure":    "Приключения",

	// Children
	"child_tale":      "Сказка",
	"child_verse":     "Детские стихи",
	"child_prose":     "Детская проза",
	"child_sf":        "Детская фантастика",
	"child_det":       "Детский детектив",
	"child_adv":       "Детские приключения",
	"child_education": "Детская образовательная литература",
	"children":        "Детская литература",

	// Poetry & Drama
	"poetry":     "Поэзия",
	"dramaturgy": "Драматургия",

	// Antique Literature
	"antique_ant":      "Античная литература",
	"antique_european": "Европейская старинная литература",
	"antique_russian":  "Древнерусская литература",
	"antique_east":     "Древневосточная литература",
	"antique_myths":    "Мифы. Легенды. Эпос",
	"antique":          "Старинная литература",

	// Science & Education
	"sci_history":    "История",
	"sci_psychology": "Психология",
	"sci_culture":    "Культурология",
	"sci_religion":   "Религиоведение",
	"sci_philosophy": "Философия",
	"sci_politics":   "Политика",
	"sci_business":   "Деловая литература",
	"sci_juris":      "Юриспруденция",
	"sci_linguistic": "Языкознание",
	"sci_medicine":   "Медицина",
	"sci_phys":       "Физика",
	"sci_math":       "Математика",
	"sci_chem":       "Химия",
	"sci_biology":    "Биология",
	"sci_tech":       "Технические науки",
	"sci_ecology":    "Экология",
	"sci_geo":        "География",
	"science":        "Наука",
	"sci_cosmos":     "Астрономия",
	"sci_pedagogy":   "Педагогика",
	"sci_social":     "Социология",
	"sci_economy":    "Экономика",
	"sci_state":      "Государство и право",
	"sci_zoo":        "Зоология",
	"sci_botany":     "Ботаника",
	"sci_philology":  "Филология",
	"sci_orgchem":    "Органическая химия",
	"sci_anachem":    "Аналитическая химия",

	// Computers
	"comp_www":         "Интернет",
	"comp_programming": "Программирование",
	"comp_hard":        "Компьютерное железо",
	"comp_soft":        "Программы",
	"comp_db":          "Базы данных",
	"comp_osnet":       "ОС и сети",
	"computers":        "Компьютеры",

	// Reference
	"ref_encyc":  "Энциклопедия",
	"ref_dict":   "Словарь",
	"ref_ref":    "Справочник",
	"ref_guide":  "Руководство",
	"reference":  "Справочная литература",

	// Nonfiction
	"nonf_biography": "Биография",
	"nonf_publicism": "Публицистика",
	"nonf_criticism": "Критика",
	"design":         "Искусство и дизайн",
	"nonfiction":     "Документальная литература",
	"geo_guides":     "Путеводитель",

	// Religion & Esoteric
	"religion_rel":          "Религия",
	"religion_esoterics":    "Эзотерика",
	"religion_self":         "Самосовершенствование",
	"religion":              "Религия и духовность",
	"religion_christianity": "Христианство",
	"religion_orthodoxy":    "Православие",
	"religion_catholicism":  "Католицизм",
	"religion_islam":        "Ислам",
	"religion_buddhism":     "Буддизм",
	"religion_hinduism":     "Индуизм",
	"religion_paganism":     "Язычество",

	// Humor
	"humor_prose":    "Юмористическая проза",
	"humor_verse":    "Юмористические стихи",
	"humor_anecdote": "Анекдоты",
	"humor":          "Юмор",
	"humor_satire":   "Сатира",

	// Home & Family
	"home_cooking":   "Кулинария",
	"home_pets":      "Домашние животные",
	"home_crafts":    "Хобби и ремёсла",
	"home_entertain": "Развлечения",
	"home_health":    "Здоровье",
	"home_garden":    "Сад и огород",
	"home_diy":       "Сделай сам",
	"home_sport":     "Спорт",
	"home_sex":       "Эротика, секс",
	"home":           "Дом и семья",

	// Thriller
	"thriller": "Триллер",

	// Other / Misc
	"periodic":      "Периодика",
	"comics":        "Комиксы",
	"tale_chivalry": "Рыцарский роман",
	"fanfiction":    "Фанфик",
	"unrecognised":  "Прочее",
}

// isRussian returns true if the language string represents Russian.
// It handles ISO codes (ru, ru-RU, ru_RU, RU), full name forms
// (Russian, русский), and is case-insensitive.
func isRussian(language string) bool {
	s := strings.TrimSpace(language)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if lower == "russian" || lower == "русский" {
		return true
	}
	// Must start with "ru" followed by separator or end-of-string
	if !strings.HasPrefix(lower, "ru") {
		return false
	}
	rest := lower[2:]
	return rest == "" || strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, "_")
}

// MapGenreName translates a single FB2 genre code into a human-readable name
// for the given language. If the code is not found in the mapping table, the
// original code is returned unchanged. English is the default for any language
// that is not recognized as Russian.
func MapGenreName(code, language string) string {
	var table map[string]string
	if isRussian(language) {
		table = fb2GenresRU
	} else {
		table = fb2GenresEN
	}
	if name, ok := table[code]; ok {
		return name
	}
	return code
}
