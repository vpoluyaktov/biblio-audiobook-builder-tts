package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider implements TTS using OpenAI's Text-to-Speech API
type OpenAIProvider struct {
	BaseProvider
	apiKey       string
	cachedVoices []Voice
	lastFetch    time.Time
	cacheTTL     time.Duration
}

// OpenAI TTS API endpoint
const (
	openAITTSEndpoint    = "https://api.openai.com/v1/audio/speech"
	defaultOpenAICacheTTL = 1 * time.Hour
)

// OpenAI TTS supported models
var openAIModels = []string{"tts-1", "tts-1-hd"}

// OpenAI TTS supported voices
var openAIVoices = []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}

// OpenAI TTS supported output formats
var openAIFormats = []string{"mp3", "opus", "aac", "flac", "wav", "pcm"}

// OpenAITTSRequest represents the request body for OpenAI TTS API
type OpenAITTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

// OpenAIErrorResponse represents an error response from OpenAI API
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIProvider creates a new OpenAI TTS provider
func NewOpenAIProvider(apiKey string) Provider {
	return &OpenAIProvider{
		BaseProvider: BaseProvider{name: "openai"},
		apiKey:       apiKey,
		cacheTTL:     defaultOpenAICacheTTL,
	}
}

// GetName returns the provider name
func (p *OpenAIProvider) GetName() string {
	return "openai"
}

// GetAvailableVoices returns all available OpenAI TTS voices
// OpenAI has a fixed set of voices, so we return them statically
func (p *OpenAIProvider) GetAvailableVoices() []Voice {
	// Return cached voices if still valid
	if len(p.cachedVoices) > 0 && time.Since(p.lastFetch) < p.cacheTTL {
		return p.cachedVoices
	}

	voices := make([]Voice, 0, len(openAIVoices)*len(openAIModels))

	// Create voice entries for each voice and model combination
	for _, model := range openAIModels {
		for _, voiceName := range openAIVoices {
			voiceID := fmt.Sprintf("%s:%s", model, voiceName)
			displayName := fmt.Sprintf("%s (%s)", capitalizeFirst(voiceName), model)

			voices = append(voices, Voice{
				ID:       voiceID,
				Name:     displayName,
				Language: "en", // OpenAI auto-detects language, but we mark as English by default
				Gender:   getOpenAIVoiceGender(voiceName),
				Provider: "openai",
			})
		}
	}

	p.cachedVoices = voices
	p.lastFetch = time.Now()
	return voices
}

// getOpenAIVoiceGender returns the gender for an OpenAI voice
func getOpenAIVoiceGender(voiceName string) string {
	// Based on OpenAI documentation and common perception
	switch voiceName {
	case "alloy":
		return "neutral"
	case "echo":
		return "male"
	case "fable":
		return "neutral"
	case "onyx":
		return "male"
	case "nova":
		return "female"
	case "shimmer":
		return "female"
	default:
		return "neutral"
	}
}

// GetVoicesByModel returns voices filtered by model type
func (p *OpenAIProvider) GetVoicesByModel(modelType string) []Voice {
	allVoices := p.GetAvailableVoices()
	if modelType == "" {
		return allVoices
	}

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		model, _ := parseOpenAIVoiceID(v.ID)
		if model == modelType {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetVoicesFiltered returns voices filtered by language and model type
func (p *OpenAIProvider) GetVoicesFiltered(languageCode, modelType string) []Voice {
	allVoices := p.GetAvailableVoices()

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		model, _ := parseOpenAIVoiceID(v.ID)
		matchLang := languageCode == "" || v.Language == languageCode
		matchModel := modelType == "" || model == modelType
		if matchLang && matchModel {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetAvailableLanguages returns available languages
// OpenAI TTS supports many languages automatically, but we return a common set
func (p *OpenAIProvider) GetAvailableLanguages() []string {
	return []string{"en"} // OpenAI auto-detects, so we just return English as default
}

// GetAvailableModels returns available model types
func (p *OpenAIProvider) GetAvailableModels() []string {
	return openAIModels
}

// parseOpenAIVoiceID parses a voice ID in format "model:voice" and returns model and voice
func parseOpenAIVoiceID(voiceID string) (model, voice string) {
	// Default to tts-1 model if no model specified
	model = "tts-1"
	voice = voiceID

	// Check if voice ID contains model prefix
	for _, m := range openAIModels {
		prefix := m + ":"
		if len(voiceID) > len(prefix) && voiceID[:len(prefix)] == prefix {
			model = m
			voice = voiceID[len(prefix):]
			break
		}
	}

	// Validate voice name, default to "alloy" if invalid
	validVoice := false
	for _, v := range openAIVoices {
		if voice == v {
			validVoice = true
			break
		}
	}
	if !validVoice {
		voice = "alloy"
	}

	return model, voice
}

// ConvertToSpeech converts text to speech using OpenAI TTS API
func (p *OpenAIProvider) ConvertToSpeech(text string, voiceID string, options *ConversionOptions) (io.Reader, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	// Parse voice ID to get model and voice
	model, voice := parseOpenAIVoiceID(voiceID)

	// Build request
	request := OpenAITTSRequest{
		Model:          model,
		Input:          text,
		Voice:          voice,
		ResponseFormat: "wav", // Use WAV for consistent quality with other providers
	}

	// Apply speed (OpenAI supports 0.25 to 4.0, default 1.0)
	if options != nil && options.Speed > 0 {
		speed := options.Speed
		// Clamp to OpenAI's supported range
		if speed < 0.25 {
			speed = 0.25
		}
		if speed > 4.0 {
			speed = 4.0
		}
		request.Speed = speed
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", openAITTSEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request with timeout
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp OpenAIErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI TTS API error: %s (type: %s)", errorResp.Error.Message, errorResp.Error.Type)
		}
		return nil, fmt.Errorf("OpenAI TTS API error: status %d", resp.StatusCode)
	}

	// Return audio data directly (OpenAI returns raw audio, not base64)
	return bytes.NewReader(body), nil
}

// GetSupportedVoices returns the list of supported voice names
func GetOpenAISupportedVoices() []string {
	return openAIVoices
}

// GetSupportedModels returns the list of supported model names
func GetOpenAISupportedModels() []string {
	return openAIModels
}

// GetSupportedFormats returns the list of supported output formats
func GetOpenAISupportedFormats() []string {
	return openAIFormats
}

// EstimateCost estimates the cost for converting the given number of characters
// OpenAI TTS pricing: $15.00 per 1M characters for tts-1, $30.00 per 1M characters for tts-1-hd
func EstimateOpenAICost(charCount int, model string) float64 {
	pricePerMillion := 15.0 // tts-1 standard
	if model == "tts-1-hd" {
		pricePerMillion = 30.0
	}
	return float64(charCount) / 1000000.0 * pricePerMillion
}
