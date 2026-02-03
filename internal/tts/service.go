package tts

import (
	"fmt"
	"io"

	"biblio-audiobook-builder-tts/internal/config"
	"biblio-audiobook-builder-tts/internal/logger"
	"biblio-audiobook-builder-tts/internal/storage"
)

// Service interface defines methods for text-to-speech conversion
type Service interface {
	ConvertToSpeech(text string, options *ConversionOptions) (io.Reader, error)
	ConvertToSpeechWithProgress(text string, options *ConversionOptions, progressCb ProgressCallback) (io.Reader, error)
	GetAvailableVoices() []Voice
	GetVoicesFiltered(provider, language, model string) []Voice
	GetAvailableLanguages(provider string) []string
	GetAvailableModels(provider, language string) []string
	GetAvailableProviders() []string
	GetAdapter(providerName string) (*Adapter, error)
	ReloadProviders()
	RefreshProvider(providerID string) error
	GetProviderInfo(providerID string) *ProviderInfo
}

// ProviderInfo contains runtime information about a provider
type ProviderInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Enabled          bool   `json:"enabled"`
	Available        bool   `json:"available"`
	TTSWorkers       int    `json:"tts_workers"`
	NormalizeNumbers bool   `json:"normalize_numbers"`
	SSMLSupport      bool   `json:"ssml_support"`
	StressEnabled    bool   `json:"stress_enabled"`
	IsDefault        bool   `json:"is_default"`
	VoiceCount       int    `json:"voice_count"`
	Error            string `json:"error,omitempty"`
}

// ConversionOptions contains settings for TTS conversion
type ConversionOptions struct {
	Voice                 string
	Provider              string
	Speed                 float64
	Pitch                 float64
	Language              string // ISO 639-1 language code (e.g., "en", "ru")
	SSMLSupport           bool   // Whether to wrap chunks in SSML tags
	SentenceBreakMs       int    // Duration of break between sentences in ms (0 = no breaks)
	ParagraphBreakMs      int    // Duration of break between paragraphs in ms (0 = no breaks)
	ConvertDashesToBreaks bool   // Convert inline dashes to SSML break tags (requires SSMLSupport)
	DashBreakDurationMs   int    // Duration of break for dashes in milliseconds (default 300)
	TitleBreakMs          int    // Duration of break after titles in ms (0 = no breaks)
}

// Voice represents a TTS voice
type Voice struct {
	ID       string
	Name     string
	Language string
	Gender   string
	Provider string
}

type service struct {
	cfg           *config.Config
	db            *storage.DB
	providers     map[string]Provider
	providerInfos map[string]*storage.TTSProvider // Cached provider info from DB
}

// NewService creates a new TTS service instance
func NewService(cfg *config.Config) Service {
	return NewServiceWithDB(cfg, nil)
}

// NewServiceWithDB creates a new TTS service instance with database support
func NewServiceWithDB(cfg *config.Config, db *storage.DB) Service {
	s := &service{
		cfg:           cfg,
		db:            db,
		providers:     make(map[string]Provider),
		providerInfos: make(map[string]*storage.TTSProvider),
	}

	s.loadProviders()
	return s
}

// loadProviders loads and initializes providers from the database or config
func (s *service) loadProviders() {
	s.providers = make(map[string]Provider)
	s.providerInfos = make(map[string]*storage.TTSProvider)

	// Load from database
	if s.db != nil {
		dbProviders, err := s.db.ListProviders(true) // Only enabled providers
		if err == nil && len(dbProviders) > 0 {
			for _, dbProv := range dbProviders {
				s.providerInfos[dbProv.ID] = dbProv
				s.initializeProviderFromDB(dbProv)
			}
			logger.Debug("Loaded %d providers from database", len(s.providers))
			return
		}
	}

	// Fallback: initialize espeak as default if no database
	s.providers["espeak"] = NewLocalProvider("espeak")
	logger.Debug("No database available, initialized espeak as default provider")
}

// initializeProviderFromDB initializes a TTS provider from database config
func (s *service) initializeProviderFromDB(dbProv *storage.TTSProvider) {
	switch dbProv.ID {
	case "espeak":
		s.providers["espeak"] = NewLocalProvider("espeak")
		logger.Debug("Initialized espeak provider")

	case "google":
		if dbProv.APIKey != "" {
			s.providers["google"] = NewGoogleProvider(dbProv.APIKey)
			logger.Debug("Initialized Google TTS provider")
		}

	case "openai":
		if dbProv.APIKey != "" {
			s.providers["openai"] = NewOpenAIProvider(dbProv.APIKey)
			logger.Debug("Initialized OpenAI TTS provider")
		}

	case "azure":
		if dbProv.APIKey != "" && dbProv.Region != "" {
			s.providers["azure"] = NewAzureProvider(dbProv.APIKey, dbProv.Region)
			logger.Debug("Initialized Azure TTS provider with region: %s", dbProv.Region)
		}

	case "opentts":
		if dbProv.URL != "" {
			s.providers["opentts"] = NewOpenTTSProvider(dbProv.URL)
			logger.Debug("Initialized OpenTTS provider with URL: %s", dbProv.URL)
		}

	case "rhvoice":
		if dbProv.URL != "" {
			s.providers["rhvoice"] = NewRHVoiceProvider(dbProv.URL)
			logger.Debug("Initialized RHVoice provider with URL: %s", dbProv.URL)
		}

	case "silero":
		if dbProv.URL != "" {
			s.providers["silero"] = NewSileroProvider(dbProv.URL)
			logger.Debug("Initialized Silero TTS provider with URL: %s", dbProv.URL)
		}

	case "openvoice":
		if dbProv.URL != "" {
			s.providers["openvoice"] = NewOpenVoiceProvider(dbProv.URL)
			logger.Debug("Initialized OpenVoice TTS provider with URL: %s", dbProv.URL)
		}
	}
}

// ConvertToSpeech converts text to speech using specified options
// This uses the adapter for automatic chunking
func (s *service) ConvertToSpeech(text string, options *ConversionOptions) (io.Reader, error) {
	adapter, err := s.GetAdapter(options.Provider)
	if err != nil {
		return nil, err
	}

	return adapter.ConvertToSpeech(text, options.Voice, options, nil)
}

// ConvertToSpeechWithProgress converts text to speech with progress callback
func (s *service) ConvertToSpeechWithProgress(text string, options *ConversionOptions, progressCb ProgressCallback) (io.Reader, error) {
	adapter, err := s.GetAdapter(options.Provider)
	if err != nil {
		return nil, err
	}

	return adapter.ConvertToSpeech(text, options.Voice, options, progressCb)
}

// GetAdapter returns a TTS adapter for the specified provider
func (s *service) GetAdapter(providerName string) (*Adapter, error) {
	provider, exists := s.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// Get provider-specific chunk size from database config
	chunkerConfig := DefaultChunkerConfig()
	if dbProvider, exists := s.providerInfos[providerName]; exists && dbProvider.MaxChunkSize > 0 {
		chunkerConfig.MaxChunkSize = dbProvider.MaxChunkSize
		logger.Debug("Using provider-specific MaxChunkSize=%d for %s", dbProvider.MaxChunkSize, providerName)
	}

	return NewAdapter(provider, chunkerConfig), nil
}

// GetAvailableVoices returns a list of available TTS voices
func (s *service) GetAvailableVoices() []Voice {
	var voices []Voice
	for _, provider := range s.providers {
		voices = append(voices, provider.GetAvailableVoices()...)
	}
	return voices
}

// GetAvailableProviders returns a list of available TTS providers
func (s *service) GetAvailableProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetVoicesFiltered returns voices filtered by provider, language, and model
func (s *service) GetVoicesFiltered(providerName, language, model string) []Voice {
	var voices []Voice

	// If provider is specified, only get voices from that provider
	if providerName != "" {
		provider, exists := s.providers[providerName]
		if !exists {
			return voices
		}
		return provider.GetVoicesFiltered(language, model)
	}

	// Get voices from all providers and filter
	for _, provider := range s.providers {
		voices = append(voices, provider.GetVoicesFiltered(language, model)...)
	}
	return voices
}

// GetAvailableLanguages returns available languages for a provider
func (s *service) GetAvailableLanguages(providerName string) []string {
	if providerName != "" {
		provider, exists := s.providers[providerName]
		if !exists {
			return []string{}
		}
		return provider.GetAvailableLanguages()
	}

	// Get from all providers
	langMap := make(map[string]bool)
	for _, provider := range s.providers {
		for _, lang := range provider.GetAvailableLanguages() {
			langMap[lang] = true
		}
	}

	languages := make([]string, 0, len(langMap))
	for lang := range langMap {
		languages = append(languages, lang)
	}
	return languages
}

// GetAvailableModels returns available model types for a provider
func (s *service) GetAvailableModels(providerName, language string) []string {
	if providerName != "" {
		provider, exists := s.providers[providerName]
		if !exists {
			return []string{}
		}
		return provider.GetModelsForLanguage(language)
	}

	// Get from all providers
	modelMap := make(map[string]bool)
	for _, provider := range s.providers {
		for _, model := range provider.GetModelsForLanguage(language) {
			modelMap[model] = true
		}
	}

	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}
	return models
}

// ReloadProviders reinitializes providers based on current config or database
func (s *service) ReloadProviders() {
	s.loadProviders()
}

// RefreshProvider refreshes the voice list for a specific provider
func (s *service) RefreshProvider(providerID string) error {
	provider, exists := s.providers[providerID]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerID)
	}
	return provider.RefreshVoices()
}

// GetProviderInfo returns runtime information about a specific provider
func (s *service) GetProviderInfo(providerID string) *ProviderInfo {
	info := &ProviderInfo{
		ID:        providerID,
		Available: false,
	}

	// Check if provider is loaded from database
	if dbProv, ok := s.providerInfos[providerID]; ok {
		info.Name = dbProv.Name
		info.Type = dbProv.Type
		info.Enabled = dbProv.Enabled
		info.TTSWorkers = dbProv.TTSWorkers
		info.NormalizeNumbers = dbProv.NormalizeNumbers
		info.SSMLSupport = dbProv.SSMLSupport
		info.StressEnabled = dbProv.StressEnabled
		info.IsDefault = dbProv.IsDefault
	}

	// Check if provider is actually available (initialized)
	if provider, ok := s.providers[providerID]; ok {
		info.Available = true
		info.VoiceCount = len(provider.GetAvailableVoices())
		if info.Name == "" {
			info.Name = provider.GetName()
		}
	}

	return info
}

// GetAllProviderInfos returns runtime information about all providers
func (s *service) GetAllProviderInfos() []*ProviderInfo {
	var infos []*ProviderInfo

	// If we have database providers, use them as the source of truth
	if s.db != nil {
		dbProviders, err := s.db.ListProviders(false) // All providers
		if err == nil {
			for _, dbProv := range dbProviders {
				info := &ProviderInfo{
					ID:               dbProv.ID,
					Name:             dbProv.Name,
					Type:             dbProv.Type,
					Enabled:          dbProv.Enabled,
					TTSWorkers:       dbProv.TTSWorkers,
					NormalizeNumbers: dbProv.NormalizeNumbers,
					SSMLSupport:      dbProv.SSMLSupport,
					StressEnabled:    dbProv.StressEnabled,
					IsDefault:        dbProv.IsDefault,
					Available:        false,
				}

				// Check if provider is actually available
				if provider, ok := s.providers[dbProv.ID]; ok {
					info.Available = true
					info.VoiceCount = len(provider.GetAvailableVoices())
				}

				infos = append(infos, info)
			}
			return infos
		}
	}

	// Fallback: return info for loaded providers only
	for id, provider := range s.providers {
		info := &ProviderInfo{
			ID:         id,
			Name:       provider.GetName(),
			Available:  true,
			VoiceCount: len(provider.GetAvailableVoices()),
		}
		infos = append(infos, info)
	}

	return infos
}
