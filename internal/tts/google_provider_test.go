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
	// With an invalid API key, GetAvailableVoices should return empty slice
	// (it fetches from API now, not hardcoded)
	provider := NewGoogleProvider("test-api-key")
	voices := provider.GetAvailableVoices()

	// With invalid API key, we expect empty result (API call fails)
	// This is expected behavior - the provider gracefully handles API errors
	if voices == nil {
		t.Fatal("Expected non-nil slice (even if empty)")
	}

	// If we somehow got voices (e.g., in integration test with real key),
	// verify they have correct provider
	for _, voice := range voices {
		if voice.Provider != "google" {
			t.Errorf("Expected provider 'google', got '%s'", voice.Provider)
		}
	}
}

func TestExtractModelType(t *testing.T) {
	tests := []struct {
		voiceName string
		expected  string
	}{
		{"en-US-Wavenet-A", "Wavenet"},
		{"en-US-Standard-B", "Standard"},
		{"en-US-Neural2-C", "Neural2"},
		{"en-US-Studio-M", "Studio"},
		{"en-US-Chirp3-HD-Achernar", "Chirp3-HD"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := extractModelType(tt.voiceName)
		if result != tt.expected {
			t.Errorf("extractModelType(%s) = %s, expected %s", tt.voiceName, result, tt.expected)
		}
	}
}

func TestExtractVoiceLetter(t *testing.T) {
	tests := []struct {
		voiceName string
		expected  string
	}{
		{"en-US-Wavenet-A", "A"},
		{"en-US-Standard-B", "B"},
		{"en-US-Chirp3-HD-Achernar", "Achernar"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := extractVoiceLetter(tt.voiceName)
		if result != tt.expected {
			t.Errorf("extractVoiceLetter(%s) = %s, expected %s", tt.voiceName, result, tt.expected)
		}
	}
}

func TestGoogleProviderFiltering(t *testing.T) {
	provider := NewGoogleProvider("test-api-key").(*GoogleProvider)

	// Inject mock voices for testing
	provider.cachedVoices = []Voice{
		{ID: "en-US-Wavenet-A", Name: "Wavenet A (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Wavenet-B", Name: "Wavenet B (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-A", Name: "Standard A (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-GB-Wavenet-A", Name: "Wavenet A (Female)", Language: "en-GB", Gender: "female", Provider: "google"},
		{ID: "en-GB-Neural2-A", Name: "Neural2 A (Male)", Language: "en-GB", Gender: "male", Provider: "google"},
		{ID: "de-DE-Standard-A", Name: "Standard A (Female)", Language: "de-DE", Gender: "female", Provider: "google"},
	}

	// Test GetVoicesByLanguage
	voices := provider.GetVoicesByLanguage("en-US")
	if len(voices) != 3 {
		t.Errorf("Expected 3 en-US voices, got %d", len(voices))
	}

	voices = provider.GetVoicesByLanguage("en-GB")
	if len(voices) != 2 {
		t.Errorf("Expected 2 en-GB voices, got %d", len(voices))
	}

	// Test GetVoicesByModel
	voices = provider.GetVoicesByModel("Wavenet")
	if len(voices) != 3 {
		t.Errorf("Expected 3 Wavenet voices, got %d", len(voices))
	}

	voices = provider.GetVoicesByModel("Standard")
	if len(voices) != 2 {
		t.Errorf("Expected 2 Standard voices, got %d", len(voices))
	}

	// Test GetVoicesFiltered
	voices = provider.GetVoicesFiltered("en-US", "Wavenet")
	if len(voices) != 2 {
		t.Errorf("Expected 2 en-US Wavenet voices, got %d", len(voices))
	}

	voices = provider.GetVoicesFiltered("en-GB", "Neural2")
	if len(voices) != 1 {
		t.Errorf("Expected 1 en-GB Neural2 voice, got %d", len(voices))
	}

	// Test GetAvailableLanguages
	languages := provider.GetAvailableLanguages()
	if len(languages) != 3 {
		t.Errorf("Expected 3 languages, got %d", len(languages))
	}

	// Test GetAvailableModels
	models := provider.GetAvailableModels()
	if len(models) != 3 {
		t.Errorf("Expected 3 models (Wavenet, Standard, Neural2), got %d", len(models))
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
