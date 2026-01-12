package tts

import (
	"testing"
)

func TestNewGoogleProvider(t *testing.T) {
	provider := NewGoogleProvider("test-api-key")

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.GetName() != "google" {
		t.Errorf("Expected provider name 'google', got '%s'", provider.GetName())
	}
}

func TestGoogleProviderGetAvailableVoices(t *testing.T) {
	provider := NewGoogleProvider("test-api-key")
	voices := provider.GetAvailableVoices()

	if len(voices) == 0 {
		t.Fatal("Expected at least one voice")
	}

	// Check that all voices have the google provider
	for _, voice := range voices {
		if voice.Provider != "google" {
			t.Errorf("Expected provider 'google', got '%s'", voice.Provider)
		}
	}

	// Check for expected voice types
	hasStandard := false
	hasWavenet := false
	hasNeural2 := false
	hasStudio := false

	for _, voice := range voices {
		if contains(voice.ID, "Standard") {
			hasStandard = true
		}
		if contains(voice.ID, "Wavenet") {
			hasWavenet = true
		}
		if contains(voice.ID, "Neural2") {
			hasNeural2 = true
		}
		if contains(voice.ID, "Studio") {
			hasStudio = true
		}
	}

	if !hasStandard {
		t.Error("Expected Standard voices")
	}
	if !hasWavenet {
		t.Error("Expected WaveNet voices")
	}
	if !hasNeural2 {
		t.Error("Expected Neural2 voices")
	}
	if !hasStudio {
		t.Error("Expected Studio voices")
	}
}

func TestExtractLanguageCode(t *testing.T) {
	tests := []struct {
		voiceName string
		expected  string
	}{
		{"en-US-Wavenet-A", "en-US"},
		{"en-GB-Standard-B", "en-GB"},
		{"en-AU-Neural2-C", "en-AU"},
		{"de-DE-Standard-A", "de-DE"},
		{"invalid", "en-US"}, // fallback
	}

	for _, tt := range tests {
		result := extractLanguageCode(tt.voiceName)
		if result != tt.expected {
			t.Errorf("extractLanguageCode(%s) = %s, expected %s", tt.voiceName, result, tt.expected)
		}
	}
}

func TestGoogleProviderConvertToSpeechNoAPIKey(t *testing.T) {
	provider := NewGoogleProvider("")

	_, err := provider.ConvertToSpeech("Hello", "en-US-Wavenet-A", &ConversionOptions{
		Speed: 1.0,
		Pitch: 1.0,
	})

	if err == nil {
		t.Error("Expected error when API key is not configured")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
