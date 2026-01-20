package storage

import (
	"abb_tts/internal/normalize"
	"time"
)

// NounStoreAdapter adapts storage.DB to implement normalize.NounStore interface.
type NounStoreAdapter struct {
	db *DB
}

// NewNounStoreAdapter creates a new adapter for the noun store.
func NewNounStoreAdapter(db *DB) *NounStoreAdapter {
	return &NounStoreAdapter{db: db}
}

// ListNouns retrieves all nouns, optionally filtered by language.
func (a *NounStoreAdapter) ListNouns(lang string) ([]*normalize.NounRecord, error) {
	dbNouns, err := a.db.ListNouns(lang)
	if err != nil {
		return nil, err
	}

	result := make([]*normalize.NounRecord, len(dbNouns))
	for i, n := range dbNouns {
		result[i] = &normalize.NounRecord{
			ID:        n.ID,
			Lang:      n.Lang,
			Noun:      n.Noun,
			Gender:    n.Gender,
			Form:      n.Form,
			Singular:  n.Singular,
			Plurals:   n.Plurals,
			IsCustom:  n.IsCustom,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}
	}
	return result, nil
}

// CreateNoun creates a new noun in the database.
func (a *NounStoreAdapter) CreateNoun(noun *normalize.NounRecord) error {
	dbNoun := &Noun{
		Lang:      noun.Lang,
		Noun:      noun.Noun,
		Gender:    noun.Gender,
		Form:      noun.Form,
		Singular:  noun.Singular,
		Plurals:   noun.Plurals,
		IsCustom:  noun.IsCustom,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := a.db.CreateNoun(dbNoun)
	if err == nil {
		noun.ID = dbNoun.ID
	}
	return err
}

// UpdateNoun updates an existing noun.
func (a *NounStoreAdapter) UpdateNoun(noun *normalize.NounRecord) error {
	dbNoun := &Noun{
		ID:        noun.ID,
		Lang:      noun.Lang,
		Noun:      noun.Noun,
		Gender:    noun.Gender,
		Form:      noun.Form,
		Singular:  noun.Singular,
		Plurals:   noun.Plurals,
		IsCustom:  noun.IsCustom,
		UpdatedAt: time.Now(),
	}
	return a.db.UpdateNoun(dbNoun)
}

// DeleteNoun deletes a noun by ID.
func (a *NounStoreAdapter) DeleteNoun(id int64) error {
	return a.db.DeleteNoun(id)
}

// GetNounByLangAndWord retrieves a noun by language and word.
func (a *NounStoreAdapter) GetNounByLangAndWord(lang, word string) (*normalize.NounRecord, error) {
	dbNoun, err := a.db.GetNounByLangAndWord(lang, word)
	if err != nil {
		return nil, err
	}
	if dbNoun == nil {
		return nil, nil
	}

	return &normalize.NounRecord{
		ID:        dbNoun.ID,
		Lang:      dbNoun.Lang,
		Noun:      dbNoun.Noun,
		Gender:    dbNoun.Gender,
		Form:      dbNoun.Form,
		Singular:  dbNoun.Singular,
		Plurals:   dbNoun.Plurals,
		IsCustom:  dbNoun.IsCustom,
		CreatedAt: dbNoun.CreatedAt,
		UpdatedAt: dbNoun.UpdatedAt,
	}, nil
}
