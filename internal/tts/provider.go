package tts

import (
	"io"
)

// Provider defines the interface for TTS providers
type Provider interface {
	// GetName returns the provider's name
	GetName() string

	// GetAvailableVoices returns a list of available voices
	GetAvailableVoices() []Voice

	// ConvertToSpeech converts text to speech
	ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error)

	// RefreshVoices reloads the voice list from the provider
	// Returns error if refresh fails
	RefreshVoices() error
}

// BaseProvider implements common functionality for TTS providers
type BaseProvider struct {
	name string
}

func (p *BaseProvider) GetName() string {
	return p.name
}
