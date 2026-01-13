package tts

import (
	"fmt"
	"io"

	"abb_tts/internal/config"
)

// Service interface defines methods for text-to-speech conversion
type Service interface {
	ConvertToSpeech(text string, options *ConversionOptions) (io.Reader, error)
	ConvertToSpeechWithProgress(text string, options *ConversionOptions, progressCb ProgressCallback) (io.Reader, error)
	GetAvailableVoices() []Voice
	GetVoicesFiltered(provider, language, model string) []Voice
	GetAvailableLanguages(provider string) []string
	GetAvailableModels(provider string) []string
	GetAvailableProviders() []string
	GetAdapter(providerName string) (*Adapter, error)
	ReloadProviders()
}

// ConversionOptions contains settings for TTS conversion
type ConversionOptions struct {
	Voice    string
	Provider string
	Speed    float64
	Pitch    float64
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
	cfg       *config.Config
	providers map[string]Provider
}

// NewService creates a new TTS service instance
func NewService(cfg *config.Config) Service {
	s := &service{
		cfg:       cfg,
		providers: make(map[string]Provider),
	}

	// Initialize local providers
	s.providers["espeak"] = NewLocalProvider("espeak")

	// Initialize Google Cloud TTS if API key is configured
	if cfg.GoogleAPIKey != "" {
		s.providers["google"] = NewGoogleProvider(cfg.GoogleAPIKey)
	}

	// Initialize other cloud providers if configured (legacy support)
	if cfg.CloudAPIKey != "" {
		if cfg.AzureTTSEndpoint != "" {
			s.providers["azure"] = NewCloudProvider("azure", cfg.CloudAPIKey, cfg.AzureTTSEndpoint)
		}
	}

	// If no providers are available, add espeak as default
	if len(s.providers) == 0 {
		s.providers["espeak"] = NewLocalProvider("espeak")
	}

	return s
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

	return NewAdapter(provider, DefaultChunkerConfig()), nil
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

		// Check if provider supports filtering (GoogleProvider does)
		if gp, ok := provider.(*GoogleProvider); ok {
			return gp.GetVoicesFiltered(language, model)
		}

		// For other providers, get all voices and filter manually
		allVoices := provider.GetAvailableVoices()
		for _, v := range allVoices {
			matchLang := language == "" || v.Language == language
			matchModel := model == "" || extractModelType(v.ID) == model
			if matchLang && matchModel {
				voices = append(voices, v)
			}
		}
		return voices
	}

	// Get voices from all providers and filter
	for _, provider := range s.providers {
		if gp, ok := provider.(*GoogleProvider); ok {
			voices = append(voices, gp.GetVoicesFiltered(language, model)...)
		} else {
			allVoices := provider.GetAvailableVoices()
			for _, v := range allVoices {
				matchLang := language == "" || v.Language == language
				matchModel := model == "" || extractModelType(v.ID) == model
				if matchLang && matchModel {
					voices = append(voices, v)
				}
			}
		}
	}
	return voices
}

// GetAvailableLanguages returns available languages for a provider
func (s *service) GetAvailableLanguages(providerName string) []string {
	langMap := make(map[string]bool)

	if providerName != "" {
		provider, exists := s.providers[providerName]
		if !exists {
			return []string{}
		}

		// Check if provider has GetAvailableLanguages method
		if gp, ok := provider.(*GoogleProvider); ok {
			return gp.GetAvailableLanguages()
		}

		// For other providers, extract from voices
		for _, v := range provider.GetAvailableVoices() {
			langMap[v.Language] = true
		}
	} else {
		// Get from all providers
		for _, provider := range s.providers {
			for _, v := range provider.GetAvailableVoices() {
				langMap[v.Language] = true
			}
		}
	}

	languages := make([]string, 0, len(langMap))
	for lang := range langMap {
		languages = append(languages, lang)
	}
	return languages
}

// GetAvailableModels returns available model types for a provider
func (s *service) GetAvailableModels(providerName string) []string {
	modelMap := make(map[string]bool)

	if providerName != "" {
		provider, exists := s.providers[providerName]
		if !exists {
			return []string{}
		}

		// Check if provider has GetAvailableModels method
		if gp, ok := provider.(*GoogleProvider); ok {
			return gp.GetAvailableModels()
		}

		// For other providers, extract from voices
		for _, v := range provider.GetAvailableVoices() {
			model := extractModelType(v.ID)
			if model != "" {
				modelMap[model] = true
			}
		}
	} else {
		// Get from all providers
		for _, provider := range s.providers {
			for _, v := range provider.GetAvailableVoices() {
				model := extractModelType(v.ID)
				if model != "" {
					modelMap[model] = true
				}
			}
		}
	}

	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}
	return models
}

// ReloadProviders reinitializes providers based on current config
func (s *service) ReloadProviders() {
	s.providers = make(map[string]Provider)

	// Initialize local providers
	s.providers["espeak"] = NewLocalProvider("espeak")

	// Initialize Google Cloud TTS if API key is configured
	if s.cfg.GoogleAPIKey != "" {
		s.providers["google"] = NewGoogleProvider(s.cfg.GoogleAPIKey)
	}

	// Initialize other cloud providers if configured (legacy support)
	if s.cfg.CloudAPIKey != "" {
		if s.cfg.AzureTTSEndpoint != "" {
			s.providers["azure"] = NewCloudProvider("azure", s.cfg.CloudAPIKey, s.cfg.AzureTTSEndpoint)
		}
	}
}
