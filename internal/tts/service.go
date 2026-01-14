package tts

import (
	"fmt"
	"io"

	"abb_tts/internal/config"
	"abb_tts/internal/logger"
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

	// Initialize OpenTTS if URL is configured
	if cfg.OpenTTSURL != "" {
		logger.Debug("Initializing OpenTTS provider with URL: %s", cfg.OpenTTSURL)
		s.providers["opentts"] = NewOpenTTSProvider(cfg.OpenTTSURL)
	} else {
		logger.Debug("OpenTTS URL not configured, skipping OpenTTS provider")
	}

	// Initialize RHVoice if URL is configured
	if cfg.RHVoiceURL != "" {
		logger.Debug("Initializing RHVoice provider with URL: %s", cfg.RHVoiceURL)
		s.providers["rhvoice"] = NewRHVoiceProvider(cfg.RHVoiceURL)
	} else {
		logger.Debug("RHVoice URL not configured, skipping RHVoice provider")
	}

	// Initialize OpenAI TTS if API key is configured
	if cfg.OpenAIAPIKey != "" {
		logger.Debug("Initializing OpenAI TTS provider")
		s.providers["openai"] = NewOpenAIProvider(cfg.OpenAIAPIKey)
	} else {
		logger.Debug("OpenAI API key not configured, skipping OpenAI provider")
	}

	// Initialize Azure TTS if subscription key and region are configured
	if cfg.AzureTTSKey != "" && cfg.AzureTTSRegion != "" {
		logger.Debug("Initializing Azure TTS provider with region: %s", cfg.AzureTTSRegion)
		s.providers["azure"] = NewAzureProvider(cfg.AzureTTSKey, cfg.AzureTTSRegion)
	} else {
		logger.Debug("Azure TTS key or region not configured, skipping Azure provider")
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

		// Check if provider supports filtering (OpenTTSProvider does)
		if op, ok := provider.(*OpenTTSProvider); ok {
			return op.GetVoicesFiltered(language, model)
		}

		// Check if provider supports filtering (OpenAIProvider does)
		if oap, ok := provider.(*OpenAIProvider); ok {
			return oap.GetVoicesFiltered(language, model)
		}

		// Check if provider supports filtering (AzureProvider does)
		if ap, ok := provider.(*AzureProvider); ok {
			return ap.GetVoicesFiltered(language, model)
		}

		// Check if provider supports filtering (RHVoiceProvider does)
		if rp, ok := provider.(*RHVoiceProvider); ok {
			return rp.GetVoicesFiltered(language, model)
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

		// Check if provider has GetAvailableLanguages method (OpenTTS)
		if op, ok := provider.(*OpenTTSProvider); ok {
			return op.GetAvailableLanguages()
		}

		// Check if provider has GetAvailableLanguages method (OpenAI)
		if oap, ok := provider.(*OpenAIProvider); ok {
			return oap.GetAvailableLanguages()
		}

		// Check if provider has GetAvailableLanguages method (Azure)
		if ap, ok := provider.(*AzureProvider); ok {
			return ap.GetAvailableLanguages()
		}

		// Check if provider has GetAvailableLanguages method (RHVoice)
		if rp, ok := provider.(*RHVoiceProvider); ok {
			return rp.GetAvailableLanguages()
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

		// Check if provider is OpenTTS - use engines as models
		if op, ok := provider.(*OpenTTSProvider); ok {
			return op.GetAvailableEngines()
		}

		// Check if provider is OpenAI - use models
		if oap, ok := provider.(*OpenAIProvider); ok {
			return oap.GetAvailableModels()
		}

		// Check if provider is Azure - use models
		if ap, ok := provider.(*AzureProvider); ok {
			return ap.GetAvailableModels()
		}

		// Check if provider is RHVoice - use models
		if rp, ok := provider.(*RHVoiceProvider); ok {
			return rp.GetAvailableModels()
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

	// Initialize OpenTTS if URL is configured
	if s.cfg.OpenTTSURL != "" {
		s.providers["opentts"] = NewOpenTTSProvider(s.cfg.OpenTTSURL)
	}

	// Initialize RHVoice if URL is configured
	if s.cfg.RHVoiceURL != "" {
		s.providers["rhvoice"] = NewRHVoiceProvider(s.cfg.RHVoiceURL)
	}

	// Initialize OpenAI TTS if API key is configured
	if s.cfg.OpenAIAPIKey != "" {
		s.providers["openai"] = NewOpenAIProvider(s.cfg.OpenAIAPIKey)
	}

	// Initialize Azure TTS if subscription key and region are configured
	if s.cfg.AzureTTSKey != "" && s.cfg.AzureTTSRegion != "" {
		s.providers["azure"] = NewAzureProvider(s.cfg.AzureTTSKey, s.cfg.AzureTTSRegion)
	}
}
