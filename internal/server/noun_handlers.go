package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"abb_tts/internal/normalize"
	"abb_tts/internal/storage"
)

// NounDB defines the database operations needed for noun management
type NounDB interface {
	ListNouns(lang string) ([]*storage.Noun, error)
	CreateNoun(noun *storage.Noun) error
	UpdateNoun(noun *storage.Noun) error
	DeleteNoun(id int64) error
	GetNoun(id int64) (*storage.Noun, error)
	GetNounByLangAndWord(lang, word string) (*storage.Noun, error)
	GetNounLanguages() ([]string, error)
	CountNouns(lang string) (int, error)
}

// NounRequest represents the request body for creating/updating a noun
type NounRequest struct {
	Lang     string   `json:"lang"`
	Noun     string   `json:"noun"`
	Gender   string   `json:"gender"` // m, f, n
	Form     string   `json:"form"`   // o (ordinal), c (cardinal)
	Singular string   `json:"singular"`
	Plurals  []string `json:"plurals"`
}

// NounResponse represents a noun in API responses
type NounResponse struct {
	ID        int64    `json:"id"`
	Lang      string   `json:"lang"`
	Noun      string   `json:"noun"`
	Gender    string   `json:"gender"`
	Form      string   `json:"form"`
	Singular  string   `json:"singular"`
	Plurals   []string `json:"plurals"`
	IsCustom  bool     `json:"is_custom"`
	IsDefault bool     `json:"is_default"` // true if from embedded defaults
}

// handleNouns handles GET (list) and POST (create) for nouns
func (s *Server) handleNouns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listNouns(w, r)
	case http.MethodPost:
		s.createNoun(w, r)
	case http.MethodOptions:
		s.handleCORS(w)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNoun handles operations on a specific noun
func (s *Server) handleNoun(w http.ResponseWriter, r *http.Request) {
	// Extract noun ID from path: /api/nouns/{id}
	path := r.URL.Path
	prefix := "/api/nouns/"
	if len(path) <= len(prefix) {
		http.Error(w, "Noun ID required", http.StatusBadRequest)
		return
	}

	idStr := path[len(prefix):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid noun ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getNoun(w, r, id)
	case http.MethodPut:
		s.updateNoun(w, r, id)
	case http.MethodDelete:
		s.deleteNoun(w, r, id)
	case http.MethodOptions:
		s.handleCORS(w)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNounLanguages returns available languages for nouns
func (s *Server) handleNounLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get languages from the normalizer's noun database
	if s.worker != nil && s.worker.normalizer != nil {
		langs := s.worker.normalizer.GetNounDatabase().GetLanguages()
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"languages": langs,
		})
		return
	}

	// Fallback to embedded defaults
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"languages": []string{"en", "ru"},
	})
}

// listNouns returns all nouns, optionally filtered by language
func (s *Server) listNouns(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")

	// Get nouns from the normalizer's noun database (includes both defaults and custom)
	if s.worker != nil && s.worker.normalizer != nil {
		nounDB := s.worker.normalizer.GetNounDatabase()

		var responses []NounResponse

		if lang != "" {
			// Get nouns for specific language
			nouns := nounDB.GetNouns(lang)
			for _, info := range nouns {
				responses = append(responses, nounInfoToResponse(lang, info))
			}
		} else {
			// Get nouns for all languages
			for _, l := range nounDB.GetLanguages() {
				nouns := nounDB.GetNouns(l)
				for _, info := range nouns {
					responses = append(responses, nounInfoToResponse(l, info))
				}
			}
		}

		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"nouns": responses,
			"count": len(responses),
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"nouns": []NounResponse{},
		"count": 0,
	})
}

// nounInfoToResponse converts NounInfo to NounResponse
func nounInfoToResponse(lang string, info normalize.NounInfo) NounResponse {
	var gender string
	switch info.Gender {
	case normalize.Masculine:
		gender = "m"
	case normalize.Feminine:
		gender = "f"
	case normalize.Neuter:
		gender = "n"
	}

	var form string
	switch info.TriggerForm {
	case normalize.Ordinal:
		form = "o"
	case normalize.Cardinal:
		form = "c"
	}

	return NounResponse{
		Lang:     lang,
		Noun:     strings.ToLower(info.SingularForm),
		Gender:   gender,
		Form:     form,
		Singular: info.SingularForm,
		Plurals:  info.PluralForms,
	}
}

// getNoun returns a specific noun by ID
func (s *Server) getNoun(w http.ResponseWriter, _ *http.Request, id int64) {
	nounDB, ok := s.db.(NounDB)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "Noun operations not supported")
		return
	}

	noun, err := nounDB.GetNoun(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get noun: %v", err))
		return
	}
	if noun == nil {
		s.jsonError(w, http.StatusNotFound, "Noun not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, storageNounToResponse(noun))
}

// createNoun creates a new noun
func (s *Server) createNoun(w http.ResponseWriter, r *http.Request) {
	var req NounRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Lang == "" || req.Singular == "" {
		s.jsonError(w, http.StatusBadRequest, "Language and singular form are required")
		return
	}

	// Set defaults
	if req.Gender == "" {
		req.Gender = "m"
	}
	if req.Form == "" {
		req.Form = "c"
	}
	if req.Noun == "" {
		req.Noun = strings.ToLower(req.Singular)
	}

	// Convert to NounInfo and add to normalizer
	info := requestToNounInfo(req)

	if s.worker != nil && s.worker.normalizer != nil {
		nounDB := s.worker.normalizer.GetNounDatabase()
		if err := nounDB.AddAndPersist(req.Lang, info); err != nil {
			s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create noun: %v", err))
			return
		}
	}

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"status": "created",
		"noun":   req.Singular,
		"lang":   req.Lang,
	})
}

// updateNoun updates an existing noun
func (s *Server) updateNoun(w http.ResponseWriter, r *http.Request, id int64) {
	nounDB, ok := s.db.(NounDB)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "Noun operations not supported")
		return
	}

	// Get existing noun
	existing, err := nounDB.GetNoun(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get noun: %v", err))
		return
	}
	if existing == nil {
		s.jsonError(w, http.StatusNotFound, "Noun not found")
		return
	}

	var req NounRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields
	if req.Gender != "" {
		existing.Gender = req.Gender
	}
	if req.Form != "" {
		existing.Form = req.Form
	}
	if req.Singular != "" {
		existing.Singular = req.Singular
		existing.Noun = strings.ToLower(req.Singular)
	}
	if len(req.Plurals) > 0 {
		existing.Plurals = strings.Join(req.Plurals, "|")
	}

	if err := nounDB.UpdateNoun(existing); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update noun: %v", err))
		return
	}

	// Reload normalizer's noun database
	if s.worker != nil && s.worker.normalizer != nil {
		s.worker.normalizer.GetNounDatabase().Reload()
	}

	s.jsonResponse(w, http.StatusOK, storageNounToResponse(existing))
}

// deleteNoun deletes a noun
func (s *Server) deleteNoun(w http.ResponseWriter, _ *http.Request, id int64) {
	nounDB, ok := s.db.(NounDB)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "Noun operations not supported")
		return
	}

	// Get noun info before deleting
	existing, err := nounDB.GetNoun(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get noun: %v", err))
		return
	}
	if existing == nil {
		s.jsonError(w, http.StatusNotFound, "Noun not found")
		return
	}

	if err := nounDB.DeleteNoun(id); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete noun: %v", err))
		return
	}

	// Reload normalizer's noun database
	if s.worker != nil && s.worker.normalizer != nil {
		s.worker.normalizer.GetNounDatabase().Reload()
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// requestToNounInfo converts NounRequest to NounInfo
func requestToNounInfo(req NounRequest) normalize.NounInfo {
	var gender normalize.Gender
	switch req.Gender {
	case "m":
		gender = normalize.Masculine
	case "f":
		gender = normalize.Feminine
	case "n":
		gender = normalize.Neuter
	default:
		gender = normalize.Masculine
	}

	var form normalize.Form
	switch req.Form {
	case "o":
		form = normalize.Ordinal
	case "c":
		form = normalize.Cardinal
	default:
		form = normalize.Cardinal
	}

	return normalize.NounInfo{
		Gender:       gender,
		TriggerForm:  form,
		SingularForm: req.Singular,
		PluralForms:  req.Plurals,
	}
}

// storageNounToResponse converts storage.Noun to NounResponse
func storageNounToResponse(n *storage.Noun) NounResponse {
	return NounResponse{
		ID:       n.ID,
		Lang:     n.Lang,
		Noun:     n.Noun,
		Gender:   n.Gender,
		Form:     n.Form,
		Singular: n.Singular,
		Plurals:  strings.Split(n.Plurals, "|"),
		IsCustom: n.IsCustom,
	}
}
