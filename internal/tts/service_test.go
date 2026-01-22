package tts

import (
	"testing"

	"biblio-audiobook-builder-tts/internal/config"
)

func TestNewService_DefaultProvider(t *testing.T) {
	cfg := &config.Config{}

	svc := NewService(cfg)

	// Verify espeak provider is registered as default
	providers := svc.GetAvailableProviders()
	found := false
	for _, p := range providers {
		if p == "espeak" {
			found = true
			break
		}
	}
	if !found {
		t.Error("espeak provider should be registered as default")
	}
}

func TestService_GetAvailableProviders(t *testing.T) {
	cfg := &config.Config{}

	svc := NewService(cfg)
	providers := svc.GetAvailableProviders()

	// Should have at least espeak
	if len(providers) == 0 {
		t.Error("expected at least one provider")
	}
}
