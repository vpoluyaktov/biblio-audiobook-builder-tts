package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"abb_tts/internal/audiobookshelf"
	"abb_tts/internal/logger"
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

	// Performance
	ConcurrentTTSWorkers int            `json:"concurrent_tts_workers"` // Legacy, kept for backward compatibility
	ProviderTTSWorkers   map[string]int `json:"provider_tts_workers"`   // Per-provider TTS worker counts
	ConcurrentEncoders   int            `json:"concurrent_encoders"`

	// Cloud TTS
	OpenAIAPIKey   string `json:"openai_api_key"`
	GoogleAPIKey   string `json:"google_api_key"`
	AzureTTSKey    string `json:"azure_tts_key"`
	AzureTTSRegion string `json:"azure_tts_region"`

	// OpenTTS
	OpenTTSURL string `json:"opentts_url"`

	// RHVoice
	RHVoiceURL string `json:"rhvoice_url"`

	// Silero
	SileroURL string `json:"silero_url"`

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
func (s *Server) getSettings(w http.ResponseWriter, _ *http.Request) {
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

		// Performance
		ConcurrentTTSWorkers: s.cfg.ConcurrentTTSWorkers,
		ProviderTTSWorkers:   s.cfg.ProviderTTSWorkers,
		ConcurrentEncoders:   s.cfg.ConcurrentEncoders,

		// Cloud TTS
		OpenAIAPIKey:   s.cfg.OpenAIAPIKey,
		GoogleAPIKey:   s.cfg.GoogleAPIKey,
		AzureTTSKey:    s.cfg.AzureTTSKey,
		AzureTTSRegion: s.cfg.AzureTTSRegion,

		// OpenTTS
		OpenTTSURL: s.cfg.OpenTTSURL,

		// RHVoice
		RHVoiceURL: s.cfg.RHVoiceURL,

		// Silero
		SileroURL: s.cfg.SileroURL,

		// Audiobookshelf
		AudiobookshelfURL:      s.cfg.AudiobookshelfURL,
		AudiobookshelfUser:     s.cfg.AudiobookshelfUser,
		AudiobookshelfPassword: s.cfg.AudiobookshelfPassword,
		AudiobookshelfLibrary:  s.cfg.AudiobookshelfLibrary,
	}

	s.jsonResponse(w, http.StatusOK, settings)
}

// saveSettings saves configuration settings
// Note: This uses a raw map to detect which fields were actually provided,
// preventing accidental overwrites of unset fields.
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) {
	// First decode into a raw map to see which fields were provided
	var rawMap map[string]interface{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Now decode into the struct for type safety
	var req SettingsRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Helper to check if a field was provided in the request
	wasProvided := func(field string) bool {
		_, ok := rawMap[field]
		return ok
	}

	// Update config in memory - only update fields that were provided
	// General
	if wasProvided("server_host") {
		s.cfg.ServerHost = req.ServerHost
	}
	if wasProvided("server_port") {
		s.cfg.ServerPort = req.ServerPort
	}
	if wasProvided("open_browser") {
		s.cfg.OpenBrowser = req.OpenBrowser
	}
	if wasProvided("output_dir") {
		s.cfg.OutputDir = req.OutputDir
	}
	if wasProvided("temp_dir") {
		s.cfg.TempDir = req.TempDir
	}
	if wasProvided("log_file") {
		s.cfg.LogFile = req.LogFile
	}

	// TTS
	if wasProvided("default_provider") {
		s.cfg.DefaultProvider = req.DefaultProvider
	}
	if wasProvided("default_voice") {
		s.cfg.DefaultVoice = req.DefaultVoice
	}
	if wasProvided("default_speed") {
		s.cfg.DefaultSpeed = req.DefaultSpeed
	}
	if wasProvided("default_pitch") {
		s.cfg.DefaultPitch = req.DefaultPitch
	}
	if wasProvided("use_default_pronunciation") {
		s.cfg.UseDefaultPronunciation = req.UseDefaultPronunciation
	}
	if wasProvided("pronunciation_dict_file") {
		s.cfg.PronunciationDictFile = req.PronunciationDictFile
	}

	// Output
	if wasProvided("bit_rate_kbs") {
		s.cfg.BitRateKbs = req.BitRateKbs
	}
	if wasProvided("sample_rate_hz") {
		s.cfg.SampleRateHz = req.SampleRateHz
	}
	if wasProvided("chapter_gap_seconds") {
		s.cfg.ChapterGapSeconds = req.ChapterGapSeconds
	}
	if wasProvided("max_file_size_mb") {
		s.cfg.MaxFileSizeMB = req.MaxFileSizeMB
	}

	// Performance
	if wasProvided("concurrent_tts_workers") {
		s.cfg.ConcurrentTTSWorkers = req.ConcurrentTTSWorkers
	}
	if wasProvided("provider_tts_workers") {
		if s.cfg.ProviderTTSWorkers == nil {
			s.cfg.ProviderTTSWorkers = make(map[string]int)
		}
		for provider, workers := range req.ProviderTTSWorkers {
			s.cfg.ProviderTTSWorkers[provider] = workers
		}
	}
	if wasProvided("concurrent_encoders") {
		s.cfg.ConcurrentEncoders = req.ConcurrentEncoders
	}

	// Cloud TTS
	if wasProvided("openai_api_key") {
		s.cfg.OpenAIAPIKey = req.OpenAIAPIKey
	}
	if wasProvided("google_api_key") {
		s.cfg.GoogleAPIKey = req.GoogleAPIKey
	}
	if wasProvided("azure_tts_key") {
		s.cfg.AzureTTSKey = req.AzureTTSKey
	}
	if wasProvided("azure_tts_region") {
		s.cfg.AzureTTSRegion = req.AzureTTSRegion
	}

	// OpenTTS
	if wasProvided("opentts_url") {
		s.cfg.OpenTTSURL = req.OpenTTSURL
	}

	// RHVoice
	if wasProvided("rhvoice_url") {
		s.cfg.RHVoiceURL = req.RHVoiceURL
	}

	// Silero
	if wasProvided("silero_url") {
		s.cfg.SileroURL = req.SileroURL
	}

	// Audiobookshelf
	if wasProvided("audiobookshelf_url") {
		s.cfg.AudiobookshelfURL = req.AudiobookshelfURL
	}
	if wasProvided("audiobookshelf_user") {
		s.cfg.AudiobookshelfUser = req.AudiobookshelfUser
	}
	if wasProvided("audiobookshelf_password") {
		s.cfg.AudiobookshelfPassword = req.AudiobookshelfPassword
	}
	if wasProvided("audiobookshelf_library") {
		s.cfg.AudiobookshelfLibrary = req.AudiobookshelfLibrary
	}

	// Persist to database if available - only save fields that were provided
	if s.db != nil {
		configs := map[string]struct {
			value    string
			provided bool
		}{
			"log_file":                  {req.LogFile, wasProvided("log_file")},
			"output_dir":                {req.OutputDir, wasProvided("output_dir")},
			"temp_dir":                  {req.TempDir, wasProvided("temp_dir")},
			"default_voice":             {req.DefaultVoice, wasProvided("default_voice")},
			"default_provider":          {req.DefaultProvider, wasProvided("default_provider")},
			"server_port":               {req.ServerPort, wasProvided("server_port")},
			"server_host":               {req.ServerHost, wasProvided("server_host")},
			"open_browser":              {fmt.Sprintf("%t", req.OpenBrowser), wasProvided("open_browser")},
			"bit_rate_kbs":              {fmt.Sprintf("%d", req.BitRateKbs), wasProvided("bit_rate_kbs")},
			"sample_rate_hz":            {fmt.Sprintf("%d", req.SampleRateHz), wasProvided("sample_rate_hz")},
			"default_speed":             {fmt.Sprintf("%.2f", req.DefaultSpeed), wasProvided("default_speed")},
			"default_pitch":             {fmt.Sprintf("%.2f", req.DefaultPitch), wasProvided("default_pitch")},
			"chapter_gap_seconds":       {fmt.Sprintf("%d", req.ChapterGapSeconds), wasProvided("chapter_gap_seconds")},
			"pronunciation_dict_file":   {req.PronunciationDictFile, wasProvided("pronunciation_dict_file")},
			"use_default_pronunciation": {fmt.Sprintf("%t", req.UseDefaultPronunciation), wasProvided("use_default_pronunciation")},
			"max_file_size_mb":          {fmt.Sprintf("%d", req.MaxFileSizeMB), wasProvided("max_file_size_mb")},
			"concurrent_tts_workers":    {fmt.Sprintf("%d", req.ConcurrentTTSWorkers), wasProvided("concurrent_tts_workers")},
			"concurrent_encoders":       {fmt.Sprintf("%d", req.ConcurrentEncoders), wasProvided("concurrent_encoders")},
			"audiobookshelf_url":        {req.AudiobookshelfURL, wasProvided("audiobookshelf_url")},
			"audiobookshelf_user":       {req.AudiobookshelfUser, wasProvided("audiobookshelf_user")},
			"audiobookshelf_password":   {req.AudiobookshelfPassword, wasProvided("audiobookshelf_password")},
			"audiobookshelf_library":    {req.AudiobookshelfLibrary, wasProvided("audiobookshelf_library")},
			"openai_api_key":            {req.OpenAIAPIKey, wasProvided("openai_api_key")},
			"google_api_key":            {req.GoogleAPIKey, wasProvided("google_api_key")},
			"azure_tts_key":             {req.AzureTTSKey, wasProvided("azure_tts_key")},
			"azure_tts_region":          {req.AzureTTSRegion, wasProvided("azure_tts_region")},
			"opentts_url":               {req.OpenTTSURL, wasProvided("opentts_url")},
			"rhvoice_url":               {req.RHVoiceURL, wasProvided("rhvoice_url")},
			"silero_url":                {req.SileroURL, wasProvided("silero_url")},
		}

		for key, cfg := range configs {
			if cfg.provided {
				if err := s.db.SetConfig(key, cfg.value); err != nil {
					logger.Warn("Failed to save config %s: %v", key, err)
				}
			}
		}

		// Save per-provider TTS workers
		if wasProvided("provider_tts_workers") && req.ProviderTTSWorkers != nil {
			for provider, workers := range req.ProviderTTSWorkers {
				key := "tts_workers_" + provider
				if err := s.db.SetConfig(key, fmt.Sprintf("%d", workers)); err != nil {
					logger.Warn("Failed to save config %s: %v", key, err)
				}
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

// TestOpenTTSRequest represents the test connection request
type TestOpenTTSRequest struct {
	URL string `json:"url"`
}

// handleTestOpenTTS tests connection to OpenTTS server
func (s *Server) handleTestOpenTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestOpenTTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		s.jsonError(w, http.StatusBadRequest, "Server URL is required")
		return
	}

	// Test connection by fetching languages
	client := &http.Client{}
	resp, err := client.Get(req.URL + "/api/languages")
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Server returned status %d", resp.StatusCode),
		})
		return
	}

	// Get voice count
	voiceResp, err := client.Get(req.URL + "/api/voices")
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"voice_count": 0,
		})
		return
	}
	defer voiceResp.Body.Close()

	var voices map[string]interface{}
	if err := json.NewDecoder(voiceResp.Body).Decode(&voices); err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"voice_count": 0,
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"voice_count": len(voices),
	})
}

// TestRHVoiceRequest represents the test connection request
type TestRHVoiceRequest struct {
	URL string `json:"url"`
}

// handleTestRHVoice tests connection to RHVoice server
func (s *Server) handleTestRHVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestRHVoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		s.jsonError(w, http.StatusBadRequest, "Server URL is required")
		return
	}

	// Test connection by fetching server info
	client := &http.Client{}
	resp, err := client.Get(req.URL + "/info")
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Server returned status %d", resp.StatusCode),
		})
		return
	}

	// Parse server info to get voice count
	var serverInfo struct {
		SupportVoices []string `json:"SUPPORT_VOICES"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&serverInfo); err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"voice_count": 0,
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"voice_count": len(serverInfo.SupportVoices),
	})
}

// TestSileroRequest represents the test connection request for Silero
type TestSileroRequest struct {
	URL string `json:"url"`
}

// handleTestSilero tests connection to Silero TTS server
func (s *Server) handleTestSilero(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.handleCORS(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestSileroRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		s.jsonError(w, http.StatusBadRequest, "Server URL is required")
		return
	}

	// Test connection by fetching health endpoint
	client := &http.Client{}
	resp, err := client.Get(req.URL + "/health")
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Server returned status %d", resp.StatusCode),
		})
		return
	}

	// Fetch voices to get count
	voicesResp, err := client.Get(req.URL + "/api/voices")
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"voice_count": 0,
		})
		return
	}
	defer voicesResp.Body.Close()

	// Parse voices map to get count
	var voicesMap map[string]interface{}
	if err := json.NewDecoder(voicesResp.Body).Decode(&voicesMap); err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"voice_count": 0,
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"voice_count": len(voicesMap),
	})
}
