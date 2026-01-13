package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"abb_tts/internal/audiobookshelf"
)

// SettingsRequest represents the settings form data
type SettingsRequest struct {
	// General
	ServerHost  string `json:"server_host"`
	ServerPort  string `json:"server_port"`
	OpenBrowser bool   `json:"open_browser"`
	OutputDir   string `json:"output_dir"`
	TempDir     string `json:"temp_dir"`
	LogFile     string `json:"log_file"`

	// TTS
	DefaultProvider         string  `json:"default_provider"`
	DefaultVoice            string  `json:"default_voice"`
	DefaultSpeed            float64 `json:"default_speed"`
	DefaultPitch            float64 `json:"default_pitch"`
	UseDefaultPronunciation bool    `json:"use_default_pronunciation"`
	PronunciationDictFile   string  `json:"pronunciation_dict_file"`

	// Output
	BitRateKbs        int `json:"bit_rate_kbs"`
	SampleRateHz      int `json:"sample_rate_hz"`
	ChapterGapSeconds int `json:"chapter_gap_seconds"`
	MaxFileSizeMB     int `json:"max_file_size_mb"`

	// Cloud TTS
	GoogleAPIKey string `json:"google_api_key"`

	// Audiobookshelf
	AudiobookshelfURL      string `json:"audiobookshelf_url"`
	AudiobookshelfUser     string `json:"audiobookshelf_user"`
	AudiobookshelfPassword string `json:"audiobookshelf_password"`
	AudiobookshelfLibrary  string `json:"audiobookshelf_library"`
}

// handleSettings handles GET (load) and POST (save) for settings
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPost:
		s.saveSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSettings returns all configuration settings
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	settings := SettingsRequest{
		// General
		ServerHost:  s.cfg.ServerHost,
		ServerPort:  s.cfg.ServerPort,
		OpenBrowser: s.cfg.OpenBrowser,
		OutputDir:   s.cfg.OutputDir,
		TempDir:     s.cfg.TempDir,
		LogFile:     s.cfg.LogFile,

		// TTS
		DefaultProvider:         s.cfg.DefaultProvider,
		DefaultVoice:            s.cfg.DefaultVoice,
		DefaultSpeed:            s.cfg.DefaultSpeed,
		DefaultPitch:            s.cfg.DefaultPitch,
		UseDefaultPronunciation: s.cfg.UseDefaultPronunciation,
		PronunciationDictFile:   s.cfg.PronunciationDictFile,

		// Output
		BitRateKbs:        s.cfg.BitRateKbs,
		SampleRateHz:      s.cfg.SampleRateHz,
		ChapterGapSeconds: s.cfg.ChapterGapSeconds,
		MaxFileSizeMB:     s.cfg.MaxFileSizeMB,

		// Cloud TTS
		GoogleAPIKey: s.cfg.GoogleAPIKey,

		// Audiobookshelf
		AudiobookshelfURL:      s.cfg.AudiobookshelfURL,
		AudiobookshelfUser:     s.cfg.AudiobookshelfUser,
		AudiobookshelfPassword: s.cfg.AudiobookshelfPassword,
		AudiobookshelfLibrary:  s.cfg.AudiobookshelfLibrary,
	}

	s.jsonResponse(w, http.StatusOK, settings)
}

// saveSettings saves configuration settings
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) {
	var req SettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update config in memory
	// General
	s.cfg.ServerHost = req.ServerHost
	s.cfg.ServerPort = req.ServerPort
	s.cfg.OpenBrowser = req.OpenBrowser
	s.cfg.OutputDir = req.OutputDir
	s.cfg.TempDir = req.TempDir
	s.cfg.LogFile = req.LogFile

	// TTS
	s.cfg.DefaultProvider = req.DefaultProvider
	s.cfg.DefaultVoice = req.DefaultVoice
	s.cfg.DefaultSpeed = req.DefaultSpeed
	s.cfg.DefaultPitch = req.DefaultPitch
	s.cfg.UseDefaultPronunciation = req.UseDefaultPronunciation
	s.cfg.PronunciationDictFile = req.PronunciationDictFile

	// Output
	s.cfg.BitRateKbs = req.BitRateKbs
	s.cfg.SampleRateHz = req.SampleRateHz
	s.cfg.ChapterGapSeconds = req.ChapterGapSeconds
	s.cfg.MaxFileSizeMB = req.MaxFileSizeMB

	// Cloud TTS
	s.cfg.GoogleAPIKey = req.GoogleAPIKey

	// Audiobookshelf
	s.cfg.AudiobookshelfURL = req.AudiobookshelfURL
	s.cfg.AudiobookshelfUser = req.AudiobookshelfUser
	s.cfg.AudiobookshelfPassword = req.AudiobookshelfPassword
	s.cfg.AudiobookshelfLibrary = req.AudiobookshelfLibrary

	// Persist to database if available
	if s.db != nil {
		configs := map[string]string{
			"log_file":                  req.LogFile,
			"output_dir":                req.OutputDir,
			"temp_dir":                  req.TempDir,
			"default_voice":             req.DefaultVoice,
			"default_provider":          req.DefaultProvider,
			"server_port":               req.ServerPort,
			"server_host":               req.ServerHost,
			"open_browser":              fmt.Sprintf("%t", req.OpenBrowser),
			"bit_rate_kbs":              fmt.Sprintf("%d", req.BitRateKbs),
			"sample_rate_hz":            fmt.Sprintf("%d", req.SampleRateHz),
			"default_speed":             fmt.Sprintf("%.2f", req.DefaultSpeed),
			"default_pitch":             fmt.Sprintf("%.2f", req.DefaultPitch),
			"chapter_gap_seconds":       fmt.Sprintf("%d", req.ChapterGapSeconds),
			"pronunciation_dict_file":   req.PronunciationDictFile,
			"use_default_pronunciation": fmt.Sprintf("%t", req.UseDefaultPronunciation),
			"max_file_size_mb":          fmt.Sprintf("%d", req.MaxFileSizeMB),
			"audiobookshelf_url":        req.AudiobookshelfURL,
			"audiobookshelf_user":       req.AudiobookshelfUser,
			"audiobookshelf_password":   req.AudiobookshelfPassword,
			"audiobookshelf_library":    req.AudiobookshelfLibrary,
			"google_api_key":            req.GoogleAPIKey,
		}

		for key, value := range configs {
			if err := s.db.SetConfig(key, value); err != nil {
				log.Printf("Warning: Failed to save config %s: %v", key, err)
			}
		}
	}

	// Reload TTS providers to pick up any API key changes
	s.ttsService.ReloadProviders()

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "saved"})
}

// TestAudiobookshelfRequest represents the test connection request
type TestAudiobookshelfRequest struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// handleTestAudiobookshelf tests connection to Audiobookshelf server
func (s *Server) handleTestAudiobookshelf(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestAudiobookshelfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		s.jsonError(w, http.StatusBadRequest, "Server URL is required")
		return
	}

	// Test connection
	client := audiobookshelf.NewClient(req.URL)
	if err := client.Login(req.User, req.Password); err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Try to get libraries to verify full access
	libraries, err := client.GetLibraries()
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   "Login successful but failed to get libraries: " + err.Error(),
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"libraries": len(libraries),
	})
}
