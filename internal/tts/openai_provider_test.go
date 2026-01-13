package tts

import (
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key")

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.GetName() != "openai" {
		t.Errorf("Expected provider name 'openai', got '%s'", provider.GetName())
	}
}

func TestOpenAIProviderGetAvailableVoices(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key")
	voices := provider.GetAvailableVoices()

	if voices == nil {
		t.Fatal("Expected non-nil slice")
	}

	// OpenAI has 6 voices * 2 models = 12 voice combinations
	expectedCount := len(openAIVoices) * len(openAIModels)
	if len(voices) != expectedCount {
		t.Errorf("Expected %d voices, got %d", expectedCount, len(voices))
	}

	// Verify all voices have correct provider
	for _, voice := range voices {
		if voice.Provider != "openai" {
			t.Errorf("Expected provider 'openai', got '%s'", voice.Provider)
		}
	}
}

func TestOpenAIVoiceGender(t *testing.T) {
	tests := []struct {
		voice    string
		expected string
	}{
		{"alloy", "neutral"},
		{"echo", "male"},
		{"fable", "neutral"},
		{"onyx", "male"},
		{"nova", "female"},
		{"shimmer", "female"},
		{"unknown", "neutral"},
	}

	for _, tt := range tests {
		result := getOpenAIVoiceGender(tt.voice)
		if result != tt.expected {
			t.Errorf("getOpenAIVoiceGender(%s) = %s, expected %s", tt.voice, result, tt.expected)
		}
	}
}

func TestParseOpenAIVoiceID(t *testing.T) {
	tests := []struct {
		voiceID       string
		expectedModel string
		expectedVoice string
	}{
		{"tts-1:alloy", "tts-1", "alloy"},
		{"tts-1-hd:nova", "tts-1-hd", "nova"},
		{"alloy", "tts-1", "alloy"},         // No model prefix, defaults to tts-1
		{"echo", "tts-1", "echo"},           // No model prefix
		{"tts-1:invalid", "tts-1", "alloy"}, // Invalid voice defaults to alloy
		{"invalid", "tts-1", "alloy"},       // Invalid voice defaults to alloy
		{"tts-1-hd:shimmer", "tts-1-hd", "shimmer"},
	}

	for _, tt := range tests {
		model, voice := parseOpenAIVoiceID(tt.voiceID)
		if model != tt.expectedModel {
			t.Errorf("parseOpenAIVoiceID(%s) model = %s, expected %s", tt.voiceID, model, tt.expectedModel)
		}
		if voice != tt.expectedVoice {
			t.Errorf("parseOpenAIVoiceID(%s) voice = %s, expected %s", tt.voiceID, voice, tt.expectedVoice)
		}
	}
}

func TestOpenAIProviderFiltering(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key").(*OpenAIProvider)

	// Test GetVoicesByModel
	voices := provider.GetVoicesByModel("tts-1")
	if len(voices) != len(openAIVoices) {
		t.Errorf("Expected %d tts-1 voices, got %d", len(openAIVoices), len(voices))
	}

	voices = provider.GetVoicesByModel("tts-1-hd")
	if len(voices) != len(openAIVoices) {
		t.Errorf("Expected %d tts-1-hd voices, got %d", len(openAIVoices), len(voices))
	}

	// Test GetVoicesFiltered with model only
	voices = provider.GetVoicesFiltered("", "tts-1")
	if len(voices) != len(openAIVoices) {
		t.Errorf("Expected %d voices for tts-1 model, got %d", len(openAIVoices), len(voices))
	}

	// Test GetAvailableLanguages
	languages := provider.GetAvailableLanguages()
	if len(languages) != 1 || languages[0] != "en" {
		t.Errorf("Expected ['en'], got %v", languages)
	}

	// Test GetAvailableModels
	models := provider.GetAvailableModels()
	if len(models) != 2 {
		t.Errorf("Expected 2 models (tts-1, tts-1-hd), got %d", len(models))
	}
}

func TestOpenAIProviderConvertToSpeechNoAPIKey(t *testing.T) {
	provider := NewOpenAIProvider("")

	_, err := provider.ConvertToSpeech("Hello", "tts-1:alloy", &ConversionOptions{
		Speed: 1.0,
	})

	if err == nil {
		t.Error("Expected error when API key is not configured")
	}
}

func TestEstimateOpenAICost(t *testing.T) {
	tests := []struct {
		charCount int
		model     string
		expected  float64
	}{
		{1000000, "tts-1", 15.0},
		{1000000, "tts-1-hd", 30.0},
		{500000, "tts-1", 7.5},
		{500000, "tts-1-hd", 15.0},
		{100000, "tts-1", 1.5},
		{100000, "tts-1-hd", 3.0},
	}

	for _, tt := range tests {
		result := EstimateOpenAICost(tt.charCount, tt.model)
		if result != tt.expected {
			t.Errorf("EstimateOpenAICost(%d, %s) = %f, expected %f", tt.charCount, tt.model, result, tt.expected)
		}
	}
}

func TestGetOpenAISupportedVoices(t *testing.T) {
	voices := GetOpenAISupportedVoices()
	if len(voices) != 6 {
		t.Errorf("Expected 6 supported voices, got %d", len(voices))
	}

	expectedVoices := []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}
	for i, v := range expectedVoices {
		if voices[i] != v {
			t.Errorf("Expected voice %s at index %d, got %s", v, i, voices[i])
		}
	}
}

func TestGetOpenAISupportedModels(t *testing.T) {
	models := GetOpenAISupportedModels()
	if len(models) != 2 {
		t.Errorf("Expected 2 supported models, got %d", len(models))
	}

	if models[0] != "tts-1" || models[1] != "tts-1-hd" {
		t.Errorf("Expected [tts-1, tts-1-hd], got %v", models)
	}
}

func TestGetOpenAISupportedFormats(t *testing.T) {
	formats := GetOpenAISupportedFormats()
	if len(formats) != 6 {
		t.Errorf("Expected 6 supported formats, got %d", len(formats))
	}

	expectedFormats := []string{"mp3", "opus", "aac", "flac", "wav", "pcm"}
	for i, f := range expectedFormats {
		if formats[i] != f {
			t.Errorf("Expected format %s at index %d, got %s", f, i, formats[i])
		}
	}
}
