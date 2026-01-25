package tts

import (
	"biblio-audiobook-builder-tts/internal/logger"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GoogleProvider implements TTS using Google Cloud Text-to-Speech API
type GoogleProvider struct {
	BaseProvider
	apiKey       string
	cachedVoices []Voice
	lastFetch    time.Time
	cacheTTL     time.Duration
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

// GoogleVoicesListResponse represents the response from voices.list API
type GoogleVoicesListResponse struct {
	Voices []GoogleVoiceInfo `json:"voices"`
}

// GoogleVoiceInfo represents a single voice from the API
type GoogleVoiceInfo struct {
	LanguageCodes          []string `json:"languageCodes"`
	Name                   string   `json:"name"`
	SsmlGender             string   `json:"ssmlGender"`
	NaturalSampleRateHertz int      `json:"naturalSampleRateHertz"`
}

// Google Cloud TTS API endpoints
const (
	googleTTSEndpoint    = "https://texttospeech.googleapis.com/v1/text:synthesize"
	googleVoicesEndpoint = "https://texttospeech.googleapis.com/v1/voices"
	defaultVoiceCacheTTL = 1 * time.Hour
)

// NewGoogleProvider creates a new Google Cloud TTS provider
func NewGoogleProvider(apiKey string) Provider {
	return &GoogleProvider{
		BaseProvider: BaseProvider{name: "google"},
		apiKey:       apiKey,
		cacheTTL:     defaultVoiceCacheTTL,
	}
}

// GetName returns the provider name
func (p *GoogleProvider) GetName() string {
	return "google"
}

// GetAvailableVoices returns all available Google Cloud TTS voices from the API
// Results are cached for cacheTTL duration to avoid excessive API calls
func (p *GoogleProvider) GetAvailableVoices() []Voice {
	// Return cached voices if still valid
	if len(p.cachedVoices) > 0 && time.Since(p.lastFetch) < p.cacheTTL {
		return p.cachedVoices
	}

	// Fetch from API
	voices, err := p.fetchVoicesFromAPI("")
	if err != nil {
		// If API call fails and we have cached voices, return them
		if len(p.cachedVoices) > 0 {
			return p.cachedVoices
		}
		// Return empty slice on error with no cache
		return []Voice{}
	}

	p.cachedVoices = voices
	p.lastFetch = time.Now()
	return voices
}

// GetVoicesByLanguage returns voices filtered by language code
func (p *GoogleProvider) GetVoicesByLanguage(languageCode string) []Voice {
	allVoices := p.GetAvailableVoices()
	if languageCode == "" {
		return allVoices
	}

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		if v.Language == languageCode {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetVoicesByModel returns voices filtered by model type (e.g., "Standard", "Wavenet", "Neural2", "Studio", "Chirp3-HD")
func (p *GoogleProvider) GetVoicesByModel(modelType string) []Voice {
	allVoices := p.GetAvailableVoices()
	if modelType == "" {
		return allVoices
	}

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		if extractModelType(v.ID) == modelType {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetVoicesFiltered returns voices filtered by both language and model type
func (p *GoogleProvider) GetVoicesFiltered(languageCode, modelType string) []Voice {
	allVoices := p.GetAvailableVoices()

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		matchLang := languageCode == "" || v.Language == languageCode
		matchModel := modelType == "" || extractModelType(v.ID) == modelType
		if matchLang && matchModel {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetAvailableLanguages returns a list of unique language codes from available voices
func (p *GoogleProvider) GetAvailableLanguages() []string {
	allVoices := p.GetAvailableVoices()
	langMap := make(map[string]bool)
	for _, v := range allVoices {
		langMap[v.Language] = true
	}

	languages := make([]string, 0, len(langMap))
	for lang := range langMap {
		languages = append(languages, lang)
	}
	return languages
}

// GetAvailableModels returns a list of unique model types from available voices
func (p *GoogleProvider) GetAvailableModels() []string {
	return p.GetModelsForLanguage("")
}

// GetModelsForLanguage returns models filtered by language
func (p *GoogleProvider) GetModelsForLanguage(language string) []string {
	allVoices := p.GetAvailableVoices()
	modelMap := make(map[string]bool)
	for _, v := range allVoices {
		if language == "" || v.Language == language {
			model := extractModelType(v.ID)
			if model != "" {
				modelMap[model] = true
			}
		}
	}

	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}
	return models
}

// fetchVoicesFromAPI fetches voices from Google Cloud TTS API
// If languageCode is provided, only voices for that language are returned
func (p *GoogleProvider) fetchVoicesFromAPI(languageCode string) ([]Voice, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Google Cloud TTS API key not configured")
	}

	// Build URL with optional language filter
	url := fmt.Sprintf("%s?key=%s", googleVoicesEndpoint, p.apiKey)
	if languageCode != "" {
		url += "&languageCode=" + languageCode
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp GoogleTTSErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("Google TTS API error: %s (code: %d)", errorResp.Error.Message, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("Google TTS API error: status %d", resp.StatusCode)
	}

	var voicesResp GoogleVoicesListResponse
	if err := json.Unmarshal(body, &voicesResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Convert to Voice structs
	voices := make([]Voice, 0, len(voicesResp.Voices))
	for _, gv := range voicesResp.Voices {
		// Each voice can support multiple languages, create an entry for each
		for _, lang := range gv.LanguageCodes {
			gender := strings.ToLower(gv.SsmlGender)
			if gender == "ssml_voice_gender_unspecified" {
				gender = "neutral"
			}

			modelType := extractModelType(gv.Name)
			voiceLetter := extractVoiceLetter(gv.Name)
			displayName := fmt.Sprintf("%s %s (%s)", modelType, voiceLetter, capitalizeFirst(gender))

			voices = append(voices, Voice{
				ID:       gv.Name,
				Name:     displayName,
				Language: lang,
				Gender:   gender,
				Provider: "google",
			})
		}
	}

	return voices, nil
}

// extractModelType extracts the model type from a voice name
// e.g., "en-US-Wavenet-A" -> "Wavenet", "en-US-Chirp3-HD-Achernar" -> "Chirp3-HD"
func extractModelType(voiceName string) string {
	parts := strings.Split(voiceName, "-")
	if len(parts) < 3 {
		return ""
	}

	// Handle multi-part model names like "Chirp3-HD"
	if len(parts) >= 4 && parts[2] == "Chirp3" && parts[3] == "HD" {
		return "Chirp3-HD"
	}

	return parts[2]
}

// extractVoiceLetter extracts the voice identifier from a voice name
// e.g., "en-US-Wavenet-A" -> "A", "en-US-Chirp3-HD-Achernar" -> "Achernar"
func extractVoiceLetter(voiceName string) string {
	parts := strings.Split(voiceName, "-")
	if len(parts) < 4 {
		return ""
	}

	// Handle multi-part model names like "Chirp3-HD"
	if len(parts) >= 5 && parts[2] == "Chirp3" && parts[3] == "HD" {
		return parts[4]
	}

	return parts[len(parts)-1]
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ConvertToSpeech converts text to speech using Google Cloud TTS API
func (p *GoogleProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Google Cloud TTS API key not configured")
	}

	// Extract language code from voice name (e.g., "en-US-Wavenet-A" -> "en-US")
	languageCode := extractLanguageCode(voice)

	// Build request - check if text is already SSML-wrapped
	var input GoogleTTSInput
	trimmedText := strings.TrimSpace(text)
	if strings.HasPrefix(trimmedText, "<speak>") && strings.HasSuffix(trimmedText, "</speak>") {
		// Text is already SSML - use SSML input
		input = GoogleTTSInput{SSML: trimmedText}
	} else {
		// Plain text
		input = GoogleTTSInput{Text: text}
	}

	request := GoogleTTSRequest{
		Input: input,
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

	// Log the text being sent
	if request.Input.SSML != "" {
		logger.Debug("GoogleProvider.ConvertToSpeech: voice='%s', ssml_len=%d", voice, len(request.Input.SSML))
		logger.Debug("GoogleProvider SSML text: %s", request.Input.SSML)
	} else {
		logger.Debug("GoogleProvider.ConvertToSpeech: voice='%s', text_len=%d", voice, len(request.Input.Text))
		logger.Debug("GoogleProvider text: %s", request.Input.Text)
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

// RefreshVoices reloads the voice list from the API
func (p *GoogleProvider) RefreshVoices() error {
	voices, err := p.fetchVoicesFromAPI("")
	if err != nil {
		return err
	}
	p.cachedVoices = voices
	p.lastFetch = time.Now()
	return nil
}
