package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"biblio-audiobook-builder-tts/internal/storage"
)

// PronunciationRuleRequest represents the request body for creating/updating a pronunciation rule
type PronunciationRuleRequest struct {
	Pattern          string `json:"pattern"`
	ReplacementPlain string `json:"replacement_plain"`
	ReplacementSSML  string `json:"replacement_ssml"`
	Comment          string `json:"comment"`
	Enabled          bool   `json:"enabled"`
}

// handlePronunciationRules handles GET (list) and POST (create) for pronunciation rules
func (s *Server) handlePronunciationRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPronunciationRules(w, r)
	case http.MethodPost:
		s.createPronunciationRule(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePronunciationRule handles GET, PUT, and DELETE for a single pronunciation rule
func (s *Server) handlePronunciationRule(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := r.URL.Path
	prefix := "/api/pronunciation/"
	idx := strings.Index(path, prefix)
	if idx == -1 || len(path) <= idx+len(prefix) {
		http.Error(w, "Rule ID required", http.StatusBadRequest)
		return
	}

	idStr := path[idx+len(prefix):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getPronunciationRule(w, r, id)
	case http.MethodPut:
		s.updatePronunciationRule(w, r, id)
	case http.MethodDelete:
		s.deletePronunciationRule(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listPronunciationRules returns all pronunciation rules
func (s *Server) listPronunciationRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.db.GetAllPronunciationRules()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve pronunciation rules: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// getPronunciationRule returns a single pronunciation rule by ID
func (s *Server) getPronunciationRule(w http.ResponseWriter, r *http.Request, id int64) {
	rule, err := s.db.GetPronunciationRule(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get pronunciation rule: %v", err))
		return
	}

	if rule == nil {
		s.jsonError(w, http.StatusNotFound, "Pronunciation rule not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// createPronunciationRule creates a new pronunciation rule
func (s *Server) createPronunciationRule(w http.ResponseWriter, r *http.Request) {
	var req PronunciationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}
	if req.ReplacementPlain == "" {
		http.Error(w, "Replacement (plain) is required", http.StatusBadRequest)
		return
	}

	// If SSML replacement is empty, use plain replacement
	if req.ReplacementSSML == "" {
		req.ReplacementSSML = req.ReplacementPlain
	}

	rule := &storage.PronunciationRule{
		Pattern:          req.Pattern,
		ReplacementPlain: req.ReplacementPlain,
		ReplacementSSML:  req.ReplacementSSML,
		Comment:          req.Comment,
		Enabled:          req.Enabled,
	}

	if err := s.db.CreatePronunciationRule(rule); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create pronunciation rule: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// updatePronunciationRule updates an existing pronunciation rule
func (s *Server) updatePronunciationRule(w http.ResponseWriter, r *http.Request, id int64) {
	var req PronunciationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}
	if req.ReplacementPlain == "" {
		http.Error(w, "Replacement (plain) is required", http.StatusBadRequest)
		return
	}

	// If SSML replacement is empty, use plain replacement
	if req.ReplacementSSML == "" {
		req.ReplacementSSML = req.ReplacementPlain
	}

	rule := &storage.PronunciationRule{
		ID:               id,
		Pattern:          req.Pattern,
		ReplacementPlain: req.ReplacementPlain,
		ReplacementSSML:  req.ReplacementSSML,
		Comment:          req.Comment,
		Enabled:          req.Enabled,
	}

	if err := s.db.UpdatePronunciationRule(rule); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update pronunciation rule: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// deletePronunciationRule deletes a pronunciation rule
func (s *Server) deletePronunciationRule(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.db.DeletePronunciationRule(id); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete pronunciation rule: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
