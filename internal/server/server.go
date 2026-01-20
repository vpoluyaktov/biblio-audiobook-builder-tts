package server

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"abb_tts/internal/config"
	"abb_tts/internal/logger"
	"abb_tts/internal/parser"
	"abb_tts/internal/storage"
	"abb_tts/internal/tts"
)

//go:embed assets/*
var assetsFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

// Server represents the HTTP server for abb_tts
type Server struct {
	addr         string
	cfg          *config.Config
	db           ConfigDB
	previewStore *PreviewStore
	hub          *Hub
	worker       *Worker
	ttsService   tts.Service
	httpServer   *http.Server
}

// ConfigDB defines the database operations needed for config and job persistence
type ConfigDB interface {
	SetConfig(key, value string) error
	DeleteJob(id string) error
	GetJob(id string) (*storage.Job, error)
	GetPendingJob() (*storage.Job, error)
	ListJobs(status string, limit int) ([]*storage.Job, error)
	CreateJob(job *storage.Job) error
	UpdateJob(job *storage.Job) error
}

// New creates a new server instance
func New(addr string, cfg *config.Config, ttsService tts.Service) *Server {
	previewStore := NewPreviewStore(30 * time.Minute) // 30 min TTL for previews
	hub := NewHub()

	s := &Server{
		addr:         addr,
		cfg:          cfg,
		previewStore: previewStore,
		hub:          hub,
		ttsService:   ttsService,
	}

	// Worker will be initialized after database is set
	return s
}

// SetDB sets the database for config persistence and initializes the worker
func (s *Server) SetDB(db ConfigDB) {
	s.db = db
	// Initialize worker with database access
	s.worker = NewWorker(db, s.hub, s.ttsService, s.cfg)
}

// jobToStorageJob converts a server.Job to storage.Job for database persistence
func jobToStorageJob(job *Job) *storage.Job {
	// Convert worker progress
	var workerProgress []storage.WorkerProgress
	job.mu.RLock()
	if len(job.WorkerProgress) > 0 {
		workerProgress = make([]storage.WorkerProgress, len(job.WorkerProgress))
		for i, wp := range job.WorkerProgress {
			workerProgress[i] = storage.WorkerProgress{
				WorkerID:       wp.WorkerID,
				ChapterIndex:   wp.ChapterIndex,
				ChapterTitle:   wp.ChapterTitle,
				Progress:       wp.Progress,
				ChunksTotal:    wp.ChunksTotal,
				ChunksComplete: wp.ChunksComplete,
				Active:         wp.Active,
			}
		}
	}
	numWorkers := job.NumWorkers
	job.mu.RUnlock()

	return &storage.Job{
		ID:                 job.ID,
		Status:             string(job.Status),
		FileName:           job.FileName,
		FilePath:           job.FilePath,
		Provider:           job.Provider,
		Voice:              job.Voice,
		Speed:              job.Speed,
		Pitch:              job.Pitch,
		BookTitle:          job.BookTitle,
		BookAuthor:         job.BookAuthor,
		OutputPath:         job.OutputPath,
		M4BFile:            job.M4BFile,
		M4BFiles:           job.M4BFiles,
		ConversionProgress: job.ConversionProgress,
		BuildProgress:      job.BuildProgress,
		CurrentChapter:     job.CurrentChapter,
		TotalChapters:      job.TotalChapters,
		CurrentChapterNum:  job.CurrentChapterNum,
		WorkerProgress:     workerProgress,
		NumWorkers:         numWorkers,
		Error:              job.Error,
		CreatedAt:          job.CreatedAt,
		StartedAt:          job.StartedAt,
		CompletedAt:        job.CompletedAt,
	}
}

// storageJobToJob converts a storage.Job to server.Job
func storageJobToJob(dbJob *storage.Job) *Job {
	return &Job{
		ID:                 dbJob.ID,
		FileName:           dbJob.FileName,
		FilePath:           dbJob.FilePath,
		Status:             JobStatus(dbJob.Status),
		ConversionProgress: dbJob.ConversionProgress,
		BuildProgress:      dbJob.BuildProgress,
		CurrentChapter:     dbJob.CurrentChapter,
		TotalChapters:      dbJob.TotalChapters,
		CurrentChapterNum:  dbJob.CurrentChapterNum,
		Provider:           dbJob.Provider,
		Voice:              dbJob.Voice,
		Speed:              dbJob.Speed,
		Pitch:              dbJob.Pitch,
		BookTitle:          dbJob.BookTitle,
		BookAuthor:         dbJob.BookAuthor,
		OutputPath:         dbJob.OutputPath,
		M4BFile:            dbJob.M4BFile,
		M4BFiles:           dbJob.M4BFiles,
		CreatedAt:          dbJob.CreatedAt,
		StartedAt:          dbJob.StartedAt,
		CompletedAt:        dbJob.CompletedAt,
		Error:              dbJob.Error,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.hub.Run()

	// Start job worker
	s.worker.Start()

	// Setup routes
	mux := http.NewServeMux()

	// Static assets
	mux.Handle("/assets/", http.FileServer(http.FS(assetsFS)))

	// API endpoints
	mux.HandleFunc("/api/jobs", s.handleJobs)
	mux.HandleFunc("/api/jobs/", s.handleJob)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/preview", s.handlePreview)
	mux.HandleFunc("/api/preview/", s.handlePreviewByID)
	mux.HandleFunc("/api/providers", s.handleProviders)
	mux.HandleFunc("/api/voices", s.handleVoices)
	mux.HandleFunc("/api/languages", s.handleLanguages)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/pricing", s.handlePricing)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/settings/test-audiobookshelf", s.handleTestAudiobookshelf)
	mux.HandleFunc("/api/settings/test-opentts", s.handleTestOpenTTS)
	mux.HandleFunc("/api/settings/test-rhvoice", s.handleTestRHVoice)
	mux.HandleFunc("/api/settings/test-silero", s.handleTestSilero)
	mux.HandleFunc("/api/test-voice", s.handleTestVoice)
	mux.HandleFunc("/api/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(s.hub, w, r)
	})

	// Noun endpoints (for text normalization)
	mux.HandleFunc("/api/nouns", s.handleNouns)
	mux.HandleFunc("/api/nouns/", s.handleNoun)
	mux.HandleFunc("/api/nouns/languages", s.handleNounLanguages)

	// OPDS endpoints
	mux.HandleFunc("/api/opds/sources", s.handleOPDSSources)
	mux.HandleFunc("/api/opds/sources/", s.handleOPDSSource)
	mux.HandleFunc("/api/opds/browse", s.handleOPDSBrowse)
	mux.HandleFunc("/api/opds/search", s.handleOPDSSearch)
	mux.HandleFunc("/api/opds/download", s.handleOPDSDownload)
	mux.HandleFunc("/api/opds/convert", s.handleOPDSConvert)
	mux.HandleFunc("/api/opds/proxy", s.handleOPDSProxy)

	// Serve main page
	mux.HandleFunc("/", s.handleIndex)

	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	logger.Info("Server starting on %s", s.addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.worker.Stop()
	return s.httpServer.Shutdown(ctx)
}

// handleIndex serves the main page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := templatesFS.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading page", http.StatusInternalServerError)
		logger.Error("Template error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleJobs handles GET (list) and POST (create via JSON) for jobs
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listJobs(w, r)
	case http.MethodOptions:
		s.handleCORS(w)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleJob handles operations on a specific job
func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path: /api/jobs/{id} or /api/jobs/{id}/download
	path := r.URL.Path
	prefix := "/api/jobs/"
	if len(path) <= len(prefix) {
		http.Error(w, "Job ID required", http.StatusBadRequest)
		return
	}

	remaining := path[len(prefix):]
	var jobID string
	var action string

	if idx := indexOf(remaining, "/"); idx != -1 {
		jobID = remaining[:idx]
		action = remaining[idx+1:]
	} else {
		jobID = remaining
	}

	switch r.Method {
	case http.MethodGet:
		switch action {
		case "download":
			s.downloadJobZip(w, r, jobID)
		case "files":
			s.listJobFiles(w, r, jobID)
		default:
			s.getJob(w, r, jobID)
		}
	case http.MethodDelete:
		s.deleteJob(w, r, jobID)
	case http.MethodOptions:
		s.handleCORS(w)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listJobs returns all jobs from database
func (s *Server) listJobs(w http.ResponseWriter, _ *http.Request) {
	if s.db == nil {
		s.jsonResponse(w, http.StatusOK, []JobDTO{})
		return
	}

	dbJobs, err := s.db.ListJobs("", 0)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list jobs: %v", err))
		return
	}

	jobs := make([]JobDTO, 0, len(dbJobs))
	for _, dbJob := range dbJobs {
		job := storageJobToJob(dbJob)
		jobs = append(jobs, job.Clone())
	}
	s.jsonResponse(w, http.StatusOK, jobs)
}

// getJob returns a specific job from database
func (s *Server) getJob(w http.ResponseWriter, _ *http.Request, id string) {
	if s.db == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	dbJob, err := s.db.GetJob(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get job: %v", err))
		return
	}
	if dbJob == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	job := storageJobToJob(dbJob)
	s.jsonResponse(w, http.StatusOK, job.Clone())
}

// deleteJob cancels/deletes a job and cleans up associated files
func (s *Server) deleteJob(w http.ResponseWriter, _ *http.Request, id string) {
	if s.db == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	// Get job info from database
	dbJob, err := s.db.GetJob(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get job: %v", err))
		return
	}
	if dbJob == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	filePath := dbJob.FilePath
	outputPath := dbJob.OutputPath

	// Delete from database
	if err := s.db.DeleteJob(id); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete job: %v", err))
		return
	}

	// Clean up files
	s.cleanupJobFiles(filePath, outputPath)

	s.hub.Broadcast(WSMessage{
		Type:    WSTypeJobDeleted,
		Payload: map[string]string{"id": id},
	})

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// cleanupJobFiles removes the uploaded file and output directory for a job
func (s *Server) cleanupJobFiles(filePath, outputPath string) {
	// Delete the uploaded book file
	if filePath != "" {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to delete uploaded file %s: %v", filePath, err)
		} else if err == nil {
			logger.Debug("Deleted uploaded file: %s", filePath)
		}
	}

	// Delete the output directory (contains chapter files and M4B)
	if outputPath != "" {
		if err := os.RemoveAll(outputPath); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to delete output directory %s: %v", outputPath, err)
		} else if err == nil {
			logger.Debug("Deleted output directory: %s", outputPath)
		}
	}
}

// downloadJobZip serves the M4B audiobook file for download
func (s *Server) downloadJobZip(w http.ResponseWriter, r *http.Request, id string) {
	if s.db == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	dbJob, err := s.db.GetJob(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get job: %v", err))
		return
	}
	if dbJob == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	if dbJob.Status != string(JobStatusCompleted) {
		s.jsonError(w, http.StatusBadRequest, "Job not completed")
		return
	}

	if dbJob.OutputPath == "" {
		s.jsonError(w, http.StatusNotFound, "Output not available")
		return
	}

	// Find the M4B file in the output directory
	files, err := os.ReadDir(dbJob.OutputPath)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to read output directory")
		return
	}

	var m4bFile string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".m4b") {
			m4bFile = filepath.Join(dbJob.OutputPath, f.Name())
			break
		}
	}

	if m4bFile == "" {
		s.jsonError(w, http.StatusNotFound, "M4B file not found")
		return
	}

	// Serve the file for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(m4bFile)))
	w.Header().Set("Content-Type", "audio/mp4")
	http.ServeFile(w, r, m4bFile)
}

// listJobFiles returns a list of files in the job output directory
func (s *Server) listJobFiles(w http.ResponseWriter, _ *http.Request, id string) {
	if s.db == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	dbJob, err := s.db.GetJob(id)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get job: %v", err))
		return
	}
	if dbJob == nil {
		s.jsonError(w, http.StatusNotFound, "Job not found")
		return
	}

	if dbJob.Status != string(JobStatusCompleted) {
		s.jsonError(w, http.StatusBadRequest, "Job not completed")
		return
	}

	if dbJob.OutputPath == "" {
		s.jsonError(w, http.StatusNotFound, "Output not available")
		return
	}

	files, err := os.ReadDir(dbJob.OutputPath)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to read output directory")
		return
	}

	fileList := make([]map[string]interface{}, 0)
	for _, f := range files {
		if !f.IsDir() {
			info, _ := f.Info()
			fileList = append(fileList, map[string]interface{}{
				"name": f.Name(),
				"size": info.Size(),
			})
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"job_id":      id,
		"output_path": dbJob.OutputPath,
		"files":       fileList,
	})
}

// handleUpload handles file upload and job creation
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	// Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".epub" && ext != ".fb2" {
		s.jsonError(w, http.StatusBadRequest, "Unsupported file format. Only .epub and .fb2 are supported")
		return
	}

	// Get TTS settings from form
	provider := r.FormValue("provider")
	if provider == "" {
		provider = s.cfg.DefaultProvider
	}

	voice := r.FormValue("voice")
	if voice == "" {
		voice = s.cfg.DefaultVoice
	}

	speed := parseFloat(r.FormValue("speed"), s.cfg.DefaultSpeed)
	pitch := parseFloat(r.FormValue("pitch"), s.cfg.DefaultPitch)

	// Save file to temp directory
	tempDir := s.cfg.TempDir
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to create temp directory")
		return
	}

	tempPath := filepath.Join(tempDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename))
	tempFile, err := os.Create(tempPath)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to save uploaded file")
		return
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to save uploaded file")
		return
	}

	// Create job
	job := NewJob(header.Filename, tempPath, provider, voice, speed, pitch)

	// Save to database
	if s.db != nil {
		dbJob := jobToStorageJob(job)
		if err := s.db.CreateJob(dbJob); err != nil {
			s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create job: %v", err))
			return
		}
	}

	// Broadcast job creation
	s.hub.Broadcast(WSMessage{
		Type:    WSTypeJobCreated,
		Payload: job.Clone(),
	})

	logger.Info("Created job %s for file %s", job.ID, header.Filename)

	s.jsonResponse(w, http.StatusCreated, job.Clone())
}

// handleProviders returns available TTS providers
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := s.ttsService.GetAvailableProviders()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
		"default":   s.cfg.DefaultProvider,
	})
}

// handleVoices returns available voices for a provider
// Supports query params: provider, language, model
func (s *Server) handleVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	language := r.URL.Query().Get("language")
	model := r.URL.Query().Get("model")

	// Use filtered method if any filter is specified
	var voices []tts.Voice
	if provider != "" || language != "" || model != "" {
		voices = s.ttsService.GetVoicesFiltered(provider, language, model)
	} else {
		voices = s.ttsService.GetAvailableVoices()
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"voices":  voices,
		"default": s.cfg.DefaultVoice,
	})
}

// handleLanguages returns available languages for a provider
func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	languages := s.ttsService.GetAvailableLanguages(provider)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"languages": languages,
	})
}

// handleModels returns available model types for a provider
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	language := r.URL.Query().Get("language")
	models := s.ttsService.GetAvailableModels(provider, language)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"models": models,
	})
}

// handlePricing returns pricing for all models of a provider
// Query params: provider, chars (character count)
func (s *Server) handlePricing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	charsStr := r.URL.Query().Get("chars")

	chars := 0
	if charsStr != "" {
		fmt.Sscanf(charsStr, "%d", &chars)
	}

	// Get all model pricing for this provider
	models := GetAllModelPricing(provider, chars)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"provider":   provider,
		"characters": chars,
		"currency":   "USD",
		"models":     models,
	})
}

// handleConfig returns current configuration (non-sensitive)
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"default_provider": s.cfg.DefaultProvider,
		"default_voice":    s.cfg.DefaultVoice,
		"default_speed":    s.cfg.DefaultSpeed,
		"default_pitch":    s.cfg.DefaultPitch,
		"bit_rate_kbs":     s.cfg.BitRateKbs,
		"sample_rate_hz":   s.cfg.SampleRateHz,
	})
}

// Helper functions

func (s *Server) handleCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

func indexOf(s string, substr string) int {
	for i := 0; i < len(s); i++ {
		if s[i:i+1] == substr {
			return i
		}
	}
	return -1
}

func parseFloat(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return defaultVal
	}
	return f
}

// handlePreview handles book preview creation (POST /api/preview)
func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	// Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".epub" && ext != ".fb2" {
		s.jsonError(w, http.StatusBadRequest, "Unsupported file format. Only .epub and .fb2 are supported")
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Parse the book
	var book *parser.Book
	if ext == ".epub" {
		p := parser.NewEpubParser()
		book, err = p.ParseEpub(bytes.NewReader(content))
	} else {
		p := parser.NewFB2Parser()
		book, err = p.ParseFB2(bytes.NewReader(content))
	}

	if err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse book: %v", err))
		return
	}

	// Create preview
	preview := s.previewStore.CreatePreview(book, header.Filename)

	logger.Info("Created preview %s for %s (%d chapters, %d words)",
		preview.ID, preview.BookTitle, preview.TotalChapters, preview.TotalWords)

	s.jsonResponse(w, http.StatusOK, preview)
}

// handlePreviewByID handles preview retrieval and cover image (GET /api/preview/{id}, GET /api/preview/{id}/cover)
func (s *Server) handlePreviewByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/preview/{id} or /api/preview/{id}/cover
	path := r.URL.Path
	path = path[len("/api/preview/"):]

	// Check if this is a cover request
	if len(path) > 6 && path[len(path)-6:] == "/cover" {
		id := path[:len(path)-6]
		s.servePreviewCover(w, r, id)
		return
	}

	// Get preview by ID
	id := path
	preview, exists := s.previewStore.GetPreview(id)
	if !exists {
		s.jsonError(w, http.StatusNotFound, "Preview not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, preview)
}

// servePreviewCover serves the cover image for a preview
func (s *Server) servePreviewCover(w http.ResponseWriter, _ *http.Request, id string) {
	cover, exists := s.previewStore.GetCover(id)
	if !exists {
		http.Error(w, "Cover not found", http.StatusNotFound)
		return
	}

	// Detect content type
	contentType := "image/jpeg"
	if len(cover) > 8 && cover[0] == 0x89 && cover[1] == 0x50 {
		contentType = "image/png"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Write(cover)
}

// GetHub returns the WebSocket hub
func (s *Server) GetHub() *Hub {
	return s.hub
}

// GetDB returns the database
func (s *Server) GetDB() ConfigDB {
	return s.db
}

// GetPreviewStore returns the preview store
func (s *Server) GetPreviewStore() *PreviewStore {
	return s.previewStore
}

// GetAddr returns the server address
func (s *Server) GetAddr() string {
	return s.addr
}

// TestVoiceRequest represents the request body for voice testing
type TestVoiceRequest struct {
	Provider string  `json:"provider"`
	Voice    string  `json:"voice"`
	Text     string  `json:"text"`
	Speed    float64 `json:"speed"`
	Pitch    float64 `json:"pitch"`
}

// handleTestVoice generates a short audio sample for voice testing
func (s *Server) handleTestVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestVoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate and set defaults
	if req.Provider == "" {
		req.Provider = s.cfg.DefaultProvider
	}
	if req.Voice == "" {
		req.Voice = s.cfg.DefaultVoice
	}
	if req.Text == "" {
		req.Text = "Hello, this is a test of the text to speech voice."
	}
	if req.Speed == 0 {
		req.Speed = 1.0
	}
	if req.Pitch == 0 {
		req.Pitch = 1.0
	}

	// Limit text length to 500 characters (using rune count for proper Unicode support)
	if utf8.RuneCountInString(req.Text) > 500 {
		runes := []rune(req.Text)
		req.Text = string(runes[:500])
	}

	// Get the TTS adapter for the provider
	adapter, err := s.ttsService.GetAdapter(req.Provider)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Provider not available: %v", err))
		return
	}

	// Convert text to speech
	options := &tts.ConversionOptions{
		Voice:    req.Voice,
		Provider: req.Provider,
		Speed:    req.Speed,
		Pitch:    req.Pitch,
	}

	audioReader, err := adapter.ConvertToSpeech(req.Text, req.Voice, options, nil)
	if err != nil {
		logger.Error("TTS conversion failed: %v", err)
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("TTS conversion failed: %v", err))
		return
	}

	// Read audio data
	audioData, err := io.ReadAll(audioReader)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to read audio data")
		return
	}

	// Determine content type based on provider
	contentType := "audio/wav"
	if req.Provider == "google" || req.Provider == "openai" || req.Provider == "azure" {
		contentType = "audio/mpeg"
	}

	// Set headers and write audio data
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(audioData)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(audioData)
}
