package tts

import (
	"fmt"
	"io"

	"github.com/vpoluyaktov/abb_tts/internal/config"
)

// Service interface defines methods for text-to-speech conversion
type Service interface {
	ConvertToSpeech(text string, options *ConversionOptions) (io.Reader, error)
	GetAvailableVoices() []Voice
	GetAvailableProviders() []string
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

	// Initialize cloud providers if configured
	if cfg.CloudAPIKey != "" {
		if cfg.GoogleTTSEndpoint != "" {
			s.providers["google"] = NewCloudProvider("google", cfg.CloudAPIKey, cfg.GoogleTTSEndpoint)
		}
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
func (s *service) ConvertToSpeech(text string, options *ConversionOptions) (io.Reader, error) {
	provider, exists := s.providers[options.Provider]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", options.Provider)
	}

	return provider.ConvertToSpeech(text, options.Voice, options)
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
