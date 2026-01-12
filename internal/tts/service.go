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
