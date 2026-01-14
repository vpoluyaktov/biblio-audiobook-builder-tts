package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"abb_tts/internal/logger"
	"abb_tts/internal/opds"
	"abb_tts/internal/parser"
	"abb_tts/internal/storage"

	"github.com/google/uuid"
)

// OPDSDB defines the database operations needed for OPDS
type OPDSDB interface {
	ListOPDSSources(enabledOnly bool) ([]*storage.OPDSSource, error)
	GetOPDSSource(id string) (*storage.OPDSSource, error)
	CreateOPDSSource(source *storage.OPDSSource) error
	UpdateOPDSSource(source *storage.OPDSSource) error
	DeleteOPDSSource(id string) error
	InitializeDefaultOPDSSources() error
}

// opdsClient is the shared OPDS client
var opdsClient = opds.NewClient()

// handleOPDSSources handles OPDS source CRUD operations
func (s *Server) handleOPDSSources(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	db, ok := s.db.(OPDSDB)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "Database not configured for OPDS")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listOPDSSources(w, r, db)
	case http.MethodPost:
		s.createOPDSSource(w, r, db)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOPDSSource handles operations on a specific OPDS source
func (s *Server) handleOPDSSource(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	db, ok := s.db.(OPDSDB)
	if !ok {
		s.jsonError(w, http.StatusInternalServerError, "Database not configured for OPDS")
		return
	}

	// Extract source ID from path: /api/opds/sources/{id}
	path := r.URL.Path
	prefix := "/api/opds/sources/"
	if len(path) <= len(prefix) {
		http.Error(w, "Source ID required", http.StatusBadRequest)
		return
	}

	sourceID := path[len(prefix):]
	// Remove any trailing path segments
	if idx := strings.Index(sourceID, "/"); idx != -1 {
		sourceID = sourceID[:idx]
	}

	switch r.Method {
	case http.MethodGet:
		s.getOPDSSource(w, r, db, sourceID)
	case http.MethodPut:
		s.updateOPDSSource(w, r, db, sourceID)
	case http.MethodDelete:
		s.deleteOPDSSource(w, r, db, sourceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listOPDSSources returns all OPDS sources
func (s *Server) listOPDSSources(w http.ResponseWriter, r *http.Request, db OPDSDB) {
	enabledOnly := r.URL.Query().Get("enabled") == "true"
	sources, err := db.ListOPDSSources(enabledOnly)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list sources: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"sources": sources,
	})
}

// createOPDSSource creates a new OPDS source
func (s *Server) createOPDSSource(w http.ResponseWriter, r *http.Request, db OPDSDB) {
	var req struct {
		Name        string `json:"name"`
		URL         string `json:"url"`
		Description string `json:"description"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		Enabled     bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.URL == "" {
		s.jsonError(w, http.StatusBadRequest, "Name and URL are required")
		return
	}

	// Generate ID from name
	id := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	id = strings.ReplaceAll(id, "_", "-")

	source := &storage.OPDSSource{
		ID:          id,
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		Username:    req.Username,
		Password:    req.Password,
		IsDefault:   false,
		Enabled:     req.Enabled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.CreateOPDSSource(source); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create source: %v", err))
		return
	}

	logger.Info("Created OPDS source: %s (%s)", source.Name, source.URL)
	s.jsonResponse(w, http.StatusCreated, source)
}

// getOPDSSource returns a specific OPDS source
func (s *Server) getOPDSSource(w http.ResponseWriter, _ *http.Request, db OPDSDB, id string) {
	source, err := db.GetOPDSSource(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get source: %v", err))
		return
	}
	if source == nil {
		s.jsonError(w, http.StatusNotFound, "Source not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, source)
}

// updateOPDSSource updates an existing OPDS source
func (s *Server) updateOPDSSource(w http.ResponseWriter, r *http.Request, db OPDSDB, id string) {
	source, err := db.GetOPDSSource(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get source: %v", err))
		return
	}
	if source == nil {
		s.jsonError(w, http.StatusNotFound, "Source not found")
		return
	}

	var req struct {
		Name        string  `json:"name"`
		URL         string  `json:"url"`
		Description string  `json:"description"`
		Username    *string `json:"username"`
		Password    *string `json:"password"`
		Enabled     *bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name != "" {
		source.Name = req.Name
	}
	if req.URL != "" {
		source.URL = req.URL
	}
	if req.Description != "" {
		source.Description = req.Description
	}
	if req.Username != nil {
		source.Username = *req.Username
	}
	if req.Password != nil {
		source.Password = *req.Password
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}
	source.UpdatedAt = time.Now()

	if err := db.UpdateOPDSSource(source); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update source: %v", err))
		return
	}

	logger.Info("Updated OPDS source: %s", source.Name)
	s.jsonResponse(w, http.StatusOK, source)
}

// deleteOPDSSource deletes an OPDS source
func (s *Server) deleteOPDSSource(w http.ResponseWriter, _ *http.Request, db OPDSDB, id string) {
	source, err := db.GetOPDSSource(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get source: %v", err))
		return
	}
	if source == nil {
		s.jsonError(w, http.StatusNotFound, "Source not found")
		return
	}

	if err := db.DeleteOPDSSource(id); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete source: %v", err))
		return
	}

	logger.Info("Deleted OPDS source: %s", source.Name)
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleOPDSBrowse handles browsing an OPDS catalog
func (s *Server) handleOPDSBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get catalog URL from query params
	catalogURL := r.URL.Query().Get("url")
	if catalogURL == "" {
		s.jsonError(w, http.StatusBadRequest, "URL parameter required")
		return
	}

	// Get optional source ID for authentication
	sourceID := r.URL.Query().Get("source_id")

	// Create client with auth if source has credentials
	client := opds.NewClient()
	if sourceID != "" {
		if db, ok := s.db.(OPDSDB); ok {
			if source, err := db.GetOPDSSource(sourceID); err == nil && source != nil {
				if source.Username != "" && source.Password != "" {
					client.SetAuth(source.Username, source.Password)
				}
			}
		}
	}

	catalog, err := client.FetchCatalog(catalogURL)
	if err != nil {
		s.jsonError(w, http.StatusBadGateway, fmt.Sprintf("Failed to fetch catalog: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, catalog)
}

// handleOPDSSearch handles searching an OPDS catalog
func (s *Server) handleOPDSSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	searchURL := r.URL.Query().Get("url")
	query := r.URL.Query().Get("q")
	sourceID := r.URL.Query().Get("source_id")

	if searchURL == "" || query == "" {
		s.jsonError(w, http.StatusBadRequest, "URL and q parameters required")
		return
	}

	// Create client with auth if source has credentials
	client := opds.NewClient()
	if sourceID != "" {
		if db, ok := s.db.(OPDSDB); ok {
			if source, err := db.GetOPDSSource(sourceID); err == nil && source != nil {
				if source.Username != "" && source.Password != "" {
					client.SetAuth(source.Username, source.Password)
				}
			}
		}
	}

	catalog, err := client.Search(searchURL, query)
	if err != nil {
		s.jsonError(w, http.StatusBadGateway, fmt.Sprintf("Search failed: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, catalog)
}

// handleOPDSDownload handles downloading a book from OPDS and creating a preview
func (s *Server) handleOPDSDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL      string `json:"url"`
		Title    string `json:"title"`
		Format   string `json:"format"`
		Author   string `json:"author"`
		SourceID string `json:"source_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		s.jsonError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Create client with auth if source has credentials
	client := opds.NewClient()
	if req.SourceID != "" {
		if db, ok := s.db.(OPDSDB); ok {
			if source, err := db.GetOPDSSource(req.SourceID); err == nil && source != nil {
				if source.Username != "" && source.Password != "" {
					client.SetAuth(source.Username, source.Password)
				}
			}
		}
	}

	// Download the book
	data, contentType, err := client.DownloadBook(req.URL)
	if err != nil {
		s.jsonError(w, http.StatusBadGateway, fmt.Sprintf("Failed to download book: %v", err))
		return
	}

	// Determine file extension
	ext := ".epub"
	if req.Format == "fb2" || strings.Contains(contentType, "fb2") || strings.Contains(contentType, "fictionbook") {
		ext = ".fb2"
	}

	// Generate filename
	filename := req.Title
	if filename == "" {
		filename = "book"
	}
	// Sanitize filename
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	filename += ext

	// Save to temp directory
	tempDir := s.cfg.TempDir
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to create temp directory")
		return
	}

	tempPath := filepath.Join(tempDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename))
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to save book")
		return
	}

	// Parse the book to create a preview
	var book *parser.Book
	if ext == ".epub" {
		p := parser.NewEpubParser()
		book, err = p.ParseEpub(bytes.NewReader(data))
	} else {
		p := parser.NewFB2Parser()
		book, err = p.ParseFB2(bytes.NewReader(data))
	}

	if err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse book: %v", err))
		return
	}

	// Create preview
	preview := s.previewStore.CreatePreview(book, filename)

	// Store the temp path in the preview for later use
	preview.FilePath = tempPath

	logger.Info("Downloaded and parsed OPDS book: %s (%d chapters)", preview.BookTitle, preview.TotalChapters)

	s.jsonResponse(w, http.StatusOK, preview)
}

// handleOPDSConvert handles converting a downloaded OPDS book
func (s *Server) handleOPDSConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PreviewID string  `json:"preview_id"`
		Provider  string  `json:"provider"`
		Voice     string  `json:"voice"`
		Speed     float64 `json:"speed"`
		Pitch     float64 `json:"pitch"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PreviewID == "" {
		s.jsonError(w, http.StatusBadRequest, "Preview ID is required")
		return
	}

	// Get the preview
	preview, exists := s.previewStore.GetPreview(req.PreviewID)
	if !exists {
		s.jsonError(w, http.StatusNotFound, "Preview not found")
		return
	}

	if preview.FilePath == "" {
		s.jsonError(w, http.StatusBadRequest, "Preview has no associated file")
		return
	}

	// Use defaults if not specified
	provider := req.Provider
	if provider == "" {
		provider = s.cfg.DefaultProvider
	}
	voice := req.Voice
	if voice == "" {
		voice = s.cfg.DefaultVoice
	}
	speed := req.Speed
	if speed == 0 {
		speed = s.cfg.DefaultSpeed
	}
	pitch := req.Pitch
	if pitch == 0 {
		pitch = s.cfg.DefaultPitch
	}

	// Create job
	job := NewJob(preview.FileName, preview.FilePath, provider, voice, speed, pitch)
	s.store.Add(job)

	// Broadcast job creation
	s.hub.Broadcast(WSMessage{
		Type:    WSTypeJobCreated,
		Payload: job.Clone(),
	})

	logger.Info("Created job %s from OPDS book %s", job.ID, preview.BookTitle)

	s.jsonResponse(w, http.StatusCreated, job.Clone())
}

// handleOPDSProxy proxies requests to OPDS catalogs (for CORS)
func (s *Server) handleOPDSProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		s.jsonError(w, http.StatusBadRequest, "URL parameter required")
		return
	}

	// Create request
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Invalid URL: %v", err))
		return
	}

	req.Header.Set("Accept", "application/atom+xml, application/xml, text/xml, image/*")
	req.Header.Set("User-Agent", "abb_tts OPDS Client/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.jsonError(w, http.StatusBadGateway, fmt.Sprintf("Failed to fetch: %v", err))
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

// OPDSPreviewWithPath extends Preview with file path
type OPDSPreviewWithPath struct {
	*Preview
	FilePath string `json:"file_path,omitempty"`
}

// Ensure Preview has FilePath field - we'll add it to the preview store
func init() {
	// Register a unique ID generator for OPDS sources
	_ = uuid.New()
}
