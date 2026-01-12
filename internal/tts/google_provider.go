package tts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GoogleProvider implements TTS using Google Cloud Text-to-Speech API
type GoogleProvider struct {
	BaseProvider
	apiKey string
}

// GoogleTTSRequest represents the request body for Google Cloud TTS API
type GoogleTTSRequest struct {
	Input       GoogleTTSInput       `json:"input"`
	Voice       GoogleTTSVoice       `json:"voice"`
	AudioConfig GoogleTTSAudioConfig `json:"audioConfig"`
}

// GoogleTTSInput represents the input text
type GoogleTTSInput struct {
	Text string `json:"text,omitempty"`
	SSML string `json:"ssml,omitempty"`
}

// GoogleTTSVoice represents voice selection parameters
type GoogleTTSVoice struct {
	LanguageCode string `json:"languageCode"`
	Name         string `json:"name,omitempty"`
	SsmlGender   string `json:"ssmlGender,omitempty"`
}

// GoogleTTSAudioConfig represents audio configuration
type GoogleTTSAudioConfig struct {
	AudioEncoding   string  `json:"audioEncoding"`
	SpeakingRate    float64 `json:"speakingRate,omitempty"`
	Pitch           float64 `json:"pitch,omitempty"`
	SampleRateHertz int     `json:"sampleRateHertz,omitempty"`
}

// GoogleTTSResponse represents the API response
type GoogleTTSResponse struct {
	AudioContent string `json:"audioContent"`
}

// GoogleTTSErrorResponse represents an error response
type GoogleTTSErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Google Cloud TTS API endpoint
const googleTTSEndpoint = "https://texttospeech.googleapis.com/v1/text:synthesize"

// NewGoogleProvider creates a new Google Cloud TTS provider
func NewGoogleProvider(apiKey string) Provider {
	return &GoogleProvider{
		BaseProvider: BaseProvider{name: "google"},
		apiKey:       apiKey,
	}
}

// GetName returns the provider name
func (p *GoogleProvider) GetName() string {
	return "google"
}

// GetAvailableVoices returns available Google Cloud TTS voices
// This is a curated list of popular voices - the full list has 400+ voices
func (p *GoogleProvider) GetAvailableVoices() []Voice {
	return []Voice{
		// English (US) - Standard voices (cheaper)
		{ID: "en-US-Standard-A", Name: "Standard A (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-B", Name: "Standard B (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Standard-C", Name: "Standard C (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-D", Name: "Standard D (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Standard-E", Name: "Standard E (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-F", Name: "Standard F (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-G", Name: "Standard G (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-H", Name: "Standard H (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Standard-I", Name: "Standard I (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Standard-J", Name: "Standard J (Male)", Language: "en-US", Gender: "male", Provider: "google"},

		// English (US) - WaveNet voices (higher quality)
		{ID: "en-US-Wavenet-A", Name: "WaveNet A (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Wavenet-B", Name: "WaveNet B (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Wavenet-C", Name: "WaveNet C (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Wavenet-D", Name: "WaveNet D (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Wavenet-E", Name: "WaveNet E (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Wavenet-F", Name: "WaveNet F (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Wavenet-G", Name: "WaveNet G (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Wavenet-H", Name: "WaveNet H (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Wavenet-I", Name: "WaveNet I (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Wavenet-J", Name: "WaveNet J (Male)", Language: "en-US", Gender: "male", Provider: "google"},

		// English (US) - Neural2 voices (best quality)
		{ID: "en-US-Neural2-A", Name: "Neural2 A (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Neural2-C", Name: "Neural2 C (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Neural2-D", Name: "Neural2 D (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Neural2-E", Name: "Neural2 E (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Neural2-F", Name: "Neural2 F (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Neural2-G", Name: "Neural2 G (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Neural2-H", Name: "Neural2 H (Female)", Language: "en-US", Gender: "female", Provider: "google"},
		{ID: "en-US-Neural2-I", Name: "Neural2 I (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Neural2-J", Name: "Neural2 J (Male)", Language: "en-US", Gender: "male", Provider: "google"},

		// English (US) - Studio voices (premium, most natural)
		{ID: "en-US-Studio-M", Name: "Studio M (Male)", Language: "en-US", Gender: "male", Provider: "google"},
		{ID: "en-US-Studio-O", Name: "Studio O (Female)", Language: "en-US", Gender: "female", Provider: "google"},

		// English (GB) - British voices
		{ID: "en-GB-Standard-A", Name: "GB Standard A (Female)", Language: "en-GB", Gender: "female", Provider: "google"},
		{ID: "en-GB-Standard-B", Name: "GB Standard B (Male)", Language: "en-GB", Gender: "male", Provider: "google"},
		{ID: "en-GB-Standard-C", Name: "GB Standard C (Female)", Language: "en-GB", Gender: "female", Provider: "google"},
		{ID: "en-GB-Standard-D", Name: "GB Standard D (Male)", Language: "en-GB", Gender: "male", Provider: "google"},
		{ID: "en-GB-Wavenet-A", Name: "GB WaveNet A (Female)", Language: "en-GB", Gender: "female", Provider: "google"},
		{ID: "en-GB-Wavenet-B", Name: "GB WaveNet B (Male)", Language: "en-GB", Gender: "male", Provider: "google"},
		{ID: "en-GB-Neural2-A", Name: "GB Neural2 A (Female)", Language: "en-GB", Gender: "female", Provider: "google"},
		{ID: "en-GB-Neural2-B", Name: "GB Neural2 B (Male)", Language: "en-GB", Gender: "male", Provider: "google"},

		// English (AU) - Australian voices
		{ID: "en-AU-Standard-A", Name: "AU Standard A (Female)", Language: "en-AU", Gender: "female", Provider: "google"},
		{ID: "en-AU-Standard-B", Name: "AU Standard B (Male)", Language: "en-AU", Gender: "male", Provider: "google"},
		{ID: "en-AU-Wavenet-A", Name: "AU WaveNet A (Female)", Language: "en-AU", Gender: "female", Provider: "google"},
		{ID: "en-AU-Wavenet-B", Name: "AU WaveNet B (Male)", Language: "en-AU", Gender: "male", Provider: "google"},
		{ID: "en-AU-Neural2-A", Name: "AU Neural2 A (Female)", Language: "en-AU", Gender: "female", Provider: "google"},
		{ID: "en-AU-Neural2-B", Name: "AU Neural2 B (Male)", Language: "en-AU", Gender: "male", Provider: "google"},
	}
}

// ConvertToSpeech converts text to speech using Google Cloud TTS API
func (p *GoogleProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Google Cloud TTS API key not configured")
	}

	// Extract language code from voice name (e.g., "en-US-Wavenet-A" -> "en-US")
	languageCode := extractLanguageCode(voice)

	// Build request
	request := GoogleTTSRequest{
		Input: GoogleTTSInput{
			Text: text,
		},
		Voice: GoogleTTSVoice{
			LanguageCode: languageCode,
			Name:         voice,
		},
		AudioConfig: GoogleTTSAudioConfig{
			AudioEncoding:   "LINEAR16", // WAV format for better quality
			SampleRateHertz: 44100,
		},
	}

	// Apply speed (Google uses 0.25 to 4.0, default 1.0)
	if options != nil && options.Speed > 0 {
		request.AudioConfig.SpeakingRate = options.Speed
	}

	// Apply pitch (Google uses -20.0 to 20.0 semitones, default 0)
	// Convert from our 0.5-2.0 scale to Google's -20 to 20 scale
	if options != nil && options.Pitch > 0 {
		// Map 0.5-2.0 to -10 to 10 (reasonable range)
		request.AudioConfig.Pitch = (options.Pitch - 1.0) * 10.0
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s?key=%s", googleTTSEndpoint, p.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
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
		var errorResp GoogleTTSErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("Google TTS API error: %s (code: %d)", errorResp.Error.Message, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("Google TTS API error: status %d", resp.StatusCode)
	}

	// Parse response
	var ttsResponse GoogleTTSResponse
	if err := json.Unmarshal(body, &ttsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Decode base64 audio content
	audioData, err := base64.StdEncoding.DecodeString(ttsResponse.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio content: %v", err)
	}

	return bytes.NewReader(audioData), nil
}

// extractLanguageCode extracts the language code from a voice name
// e.g., "en-US-Wavenet-A" -> "en-US"
func extractLanguageCode(voiceName string) string {
	parts := strings.Split(voiceName, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return "en-US" // Default fallback
}
