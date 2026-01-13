package tts

import (
	"testing"
)

func TestNewAzureProvider(t *testing.T) {
	provider := NewAzureProvider("test-key", "eastus")
	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
	if provider.GetName() != "azure" {
		t.Errorf("Expected provider name 'azure', got '%s'", provider.GetName())
	}
}

func TestAzureProviderGetName(t *testing.T) {
	provider := NewAzureProvider("test-key", "eastus").(*AzureProvider)
	if provider.GetName() != "azure" {
		t.Errorf("Expected 'azure', got '%s'", provider.GetName())
	}
}

func TestAzureProviderURLs(t *testing.T) {
	provider := NewAzureProvider("test-key", "eastus").(*AzureProvider)

	tokenURL := provider.getTokenURL()
	expectedTokenURL := "https://eastus.api.cognitive.microsoft.com/sts/v1.0/issuetoken"
	if tokenURL != expectedTokenURL {
		t.Errorf("Expected token URL '%s', got '%s'", expectedTokenURL, tokenURL)
	}

	ttsURL := provider.getTTSURL()
	expectedTTSURL := "https://eastus.tts.speech.microsoft.com/cognitiveservices/v1"
	if ttsURL != expectedTTSURL {
		t.Errorf("Expected TTS URL '%s', got '%s'", expectedTTSURL, ttsURL)
	}

	voicesURL := provider.getVoicesURL()
	expectedVoicesURL := "https://eastus.tts.speech.microsoft.com/cognitiveservices/voices/list"
	if voicesURL != expectedVoicesURL {
		t.Errorf("Expected voices URL '%s', got '%s'", expectedVoicesURL, voicesURL)
	}
}

func TestAzureProviderURLsWithDifferentRegion(t *testing.T) {
	provider := NewAzureProvider("test-key", "westeurope").(*AzureProvider)

	tokenURL := provider.getTokenURL()
	if tokenURL != "https://westeurope.api.cognitive.microsoft.com/sts/v1.0/issuetoken" {
		t.Errorf("Unexpected token URL: %s", tokenURL)
	}

	ttsURL := provider.getTTSURL()
	if ttsURL != "https://westeurope.tts.speech.microsoft.com/cognitiveservices/v1" {
		t.Errorf("Unexpected TTS URL: %s", ttsURL)
	}
}

func TestExtractAzureVoiceType(t *testing.T) {
	tests := []struct {
		voiceType string
		expected  string
	}{
		{"Neural", "Neural"},
		{"NeuralMultilingual", "Neural"},
		{"Standard", "Standard"},
		{"", "Standard"},
	}

	for _, tt := range tests {
		result := extractAzureVoiceType(tt.voiceType)
		if result != tt.expected {
			t.Errorf("extractAzureVoiceType(%s) = %s, expected %s", tt.voiceType, result, tt.expected)
		}
	}
}

func TestExtractAzureLanguage(t *testing.T) {
	tests := []struct {
		voiceName string
		expected  string
	}{
		{"en-US-GuyNeural", "en-US"},
		{"en-GB-SoniaNeural", "en-GB"},
		{"de-DE-ConradNeural", "de-DE"},
		{"zh-CN-XiaoxiaoNeural", "zh-CN"},
		{"invalid", "en-US"}, // fallback
		{"", "en-US"},        // fallback
	}

	for _, tt := range tests {
		result := extractAzureLanguage(tt.voiceName)
		if result != tt.expected {
			t.Errorf("extractAzureLanguage(%s) = %s, expected %s", tt.voiceName, result, tt.expected)
		}
	}
}

func TestAzureProviderGetAvailableModels(t *testing.T) {
	provider := NewAzureProvider("test-key", "eastus").(*AzureProvider)
	models := provider.GetAvailableModels()

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	hasStandard := false
	hasNeural := false
	for _, m := range models {
		if m == "Standard" {
			hasStandard = true
		}
		if m == "Neural" {
			hasNeural = true
		}
	}

	if !hasStandard {
		t.Error("Expected 'Standard' model")
	}
	if !hasNeural {
		t.Error("Expected 'Neural' model")
	}
}

func TestAzureProviderFiltering(t *testing.T) {
	provider := NewAzureProvider("test-key", "eastus").(*AzureProvider)

	// Inject mock voices for testing
	provider.cachedVoices = []Voice{
		{ID: "en-US-GuyNeural", Name: "Guy (Neural)", Language: "en-US", Gender: "male", Provider: "azure"},
		{ID: "en-US-JennyNeural", Name: "Jenny (Neural)", Language: "en-US", Gender: "female", Provider: "azure"},
		{ID: "en-GB-SoniaNeural", Name: "Sonia (Neural)", Language: "en-GB", Gender: "female", Provider: "azure"},
		{ID: "de-DE-ConradNeural", Name: "Conrad (Neural)", Language: "de-DE", Gender: "male", Provider: "azure"},
	}

	// Test GetVoicesByLanguage
	voices := provider.GetVoicesByLanguage("en-US")
	if len(voices) != 2 {
		t.Errorf("Expected 2 en-US voices, got %d", len(voices))
	}

	voices = provider.GetVoicesByLanguage("en-GB")
	if len(voices) != 1 {
		t.Errorf("Expected 1 en-GB voice, got %d", len(voices))
	}

	// Test GetVoicesFiltered with language only
	voices = provider.GetVoicesFiltered("de-DE", "")
	if len(voices) != 1 {
		t.Errorf("Expected 1 de-DE voice, got %d", len(voices))
	}

	// Test GetVoicesFiltered with model only
	voices = provider.GetVoicesFiltered("", "Neural")
	if len(voices) != 4 {
		t.Errorf("Expected 4 Neural voices, got %d", len(voices))
	}

	// Test GetAvailableLanguages
	languages := provider.GetAvailableLanguages()
	if len(languages) != 3 {
		t.Errorf("Expected 3 languages, got %d", len(languages))
	}
}

func TestAzureProviderConvertToSpeechNoCredentials(t *testing.T) {
	provider := NewAzureProvider("", "").(*AzureProvider)

	_, err := provider.ConvertToSpeech("Hello", "en-US-GuyNeural", nil)
	if err == nil {
		t.Error("Expected error when credentials are not configured")
	}
}

func TestEstimateAzureCost(t *testing.T) {
	tests := []struct {
		charCount int
		voiceType string
		expected  float64
	}{
		{1000000, "Neural", 16.0},
		{1000000, "Standard", 4.0},
		{500000, "Neural", 8.0},
		{500000, "Standard", 2.0},
		{100000, "Neural", 1.6},
		{0, "Neural", 0.0},
	}

	for _, tt := range tests {
		result := EstimateAzureCost(tt.charCount, tt.voiceType)
		if result != tt.expected {
			t.Errorf("EstimateAzureCost(%d, %s) = %.2f, expected %.2f", tt.charCount, tt.voiceType, result, tt.expected)
		}
	}
}

func TestGetAzureSupportedFormats(t *testing.T) {
	formats := GetAzureSupportedFormats()
	if len(formats) == 0 {
		t.Error("Expected at least one supported format")
	}

	// Check for common formats
	hasMP3 := false
	hasPCM := false
	for _, f := range formats {
		if f == "audio-24khz-48kbitrate-mono-mp3" {
			hasMP3 = true
		}
		if f == "riff-24khz-16bit-mono-pcm" {
			hasPCM = true
		}
	}

	if !hasMP3 {
		t.Error("Expected MP3 format to be supported")
	}
	if !hasPCM {
		t.Error("Expected PCM format to be supported")
	}
}

func TestAzureProviderTokenExpiry(t *testing.T) {
	provider := NewAzureProvider("test-key", "eastus").(*AzureProvider)

	// Token should be expired initially (empty)
	if !provider.isTokenExpired() {
		t.Error("Expected token to be expired when empty")
	}
}
