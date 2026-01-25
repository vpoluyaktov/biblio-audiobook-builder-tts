package tts

import (
	"biblio-audiobook-builder-tts/internal/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AzureProvider implements TTS using Azure Cognitive Services Speech API
type AzureProvider struct {
	BaseProvider
	subscriptionKey string
	region          string
	accessToken     string
	tokenExpiry     time.Time
	tokenMu         sync.Mutex
	cachedVoices    []Voice
	lastVoiceFetch  time.Time
	voiceCacheTTL   time.Duration
}

// Azure TTS API constants
const (
	azureTokenValidityMinutes = 9
	azureVoiceCacheTTL        = 1 * time.Hour
	azureMaxRetries           = 3
)

// Azure TTS supported output formats
var azureOutputFormats = []string{
	"audio-16khz-32kbitrate-mono-mp3",
	"audio-16khz-64kbitrate-mono-mp3",
	"audio-16khz-128kbitrate-mono-mp3",
	"audio-24khz-48kbitrate-mono-mp3",
	"audio-24khz-96kbitrate-mono-mp3",
	"audio-24khz-160kbitrate-mono-mp3",
	"audio-48khz-96kbitrate-mono-mp3",
	"audio-48khz-192kbitrate-mono-mp3",
	"riff-16khz-16bit-mono-pcm",
	"riff-24khz-16bit-mono-pcm",
	"riff-48khz-16bit-mono-pcm",
}

// AzureVoicesResponse represents the response from Azure voices list API
type AzureVoicesResponse []AzureVoiceInfo

// AzureVoiceInfo represents a single voice from Azure API
type AzureVoiceInfo struct {
	Name            string   `json:"Name"`
	DisplayName     string   `json:"DisplayName"`
	LocalName       string   `json:"LocalName"`
	ShortName       string   `json:"ShortName"`
	Gender          string   `json:"Gender"`
	Locale          string   `json:"Locale"`
	LocaleName      string   `json:"LocaleName"`
	StyleList       []string `json:"StyleList,omitempty"`
	VoiceType       string   `json:"VoiceType"`
	Status          string   `json:"Status"`
	SampleRateHertz string   `json:"SampleRateHertz,omitempty"`
}

// NewAzureProvider creates a new Azure Cognitive Services TTS provider
func NewAzureProvider(subscriptionKey, region string) Provider {
	return &AzureProvider{
		BaseProvider:    BaseProvider{name: "azure"},
		subscriptionKey: subscriptionKey,
		region:          region,
		voiceCacheTTL:   azureVoiceCacheTTL,
	}
}

// GetName returns the provider name
func (p *AzureProvider) GetName() string {
	return "azure"
}

// getTokenURL returns the token endpoint URL
func (p *AzureProvider) getTokenURL() string {
	return fmt.Sprintf("https://%s.api.cognitive.microsoft.com/sts/v1.0/issuetoken", p.region)
}

// getTTSURL returns the TTS endpoint URL
func (p *AzureProvider) getTTSURL() string {
	return fmt.Sprintf("https://%s.tts.speech.microsoft.com/cognitiveservices/v1", p.region)
}

// getVoicesURL returns the voices list endpoint URL
func (p *AzureProvider) getVoicesURL() string {
	return fmt.Sprintf("https://%s.tts.speech.microsoft.com/cognitiveservices/voices/list", p.region)
}

// isTokenExpired checks if the access token is expired
func (p *AzureProvider) isTokenExpired() bool {
	return p.accessToken == "" || time.Now().After(p.tokenExpiry)
}

// refreshToken gets a new access token from Azure
func (p *AzureProvider) refreshToken() error {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	// Double-check after acquiring lock
	if !p.isTokenExpired() {
		return nil
	}

	req, err := http.NewRequest("POST", p.getTokenURL(), nil)
	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	req.Header.Set("Content-Length", "0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %v", err)
	}

	p.accessToken = string(tokenBytes)
	p.tokenExpiry = time.Now().Add(azureTokenValidityMinutes * time.Minute)
	return nil
}

// getAccessToken returns a valid access token, refreshing if necessary
func (p *AzureProvider) getAccessToken() (string, error) {
	if p.isTokenExpired() {
		if err := p.refreshToken(); err != nil {
			return "", err
		}
	}
	return p.accessToken, nil
}

// GetAvailableVoices returns all available Azure TTS voices
func (p *AzureProvider) GetAvailableVoices() []Voice {
	// Return cached voices if still valid
	if len(p.cachedVoices) > 0 && time.Since(p.lastVoiceFetch) < p.voiceCacheTTL {
		return p.cachedVoices
	}

	voices, err := p.fetchVoicesFromAPI()
	if err != nil {
		// If API call fails and we have cached voices, return them
		if len(p.cachedVoices) > 0 {
			return p.cachedVoices
		}
		// Return empty slice on error with no cache
		return []Voice{}
	}

	p.cachedVoices = voices
	p.lastVoiceFetch = time.Now()
	return voices
}

// fetchVoicesFromAPI fetches voices from Azure TTS API
func (p *AzureProvider) fetchVoicesFromAPI() ([]Voice, error) {
	if p.subscriptionKey == "" || p.region == "" {
		return nil, fmt.Errorf("Azure TTS subscription key or region not configured")
	}

	token, err := p.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	req, err := http.NewRequest("GET", p.getVoicesURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Azure TTS API error: status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse JSON response
	var azureVoices AzureVoicesResponse
	if err := parseJSON(body, &azureVoices); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Convert to Voice structs
	voices := make([]Voice, 0, len(azureVoices))
	for _, av := range azureVoices {
		gender := strings.ToLower(av.Gender)
		voiceType := extractAzureVoiceType(av.VoiceType)

		displayName := fmt.Sprintf("%s (%s)", av.DisplayName, voiceType)

		voices = append(voices, Voice{
			ID:       av.ShortName,
			Name:     displayName,
			Language: av.Locale,
			Gender:   gender,
			Provider: "azure",
		})
	}

	return voices, nil
}

// extractAzureVoiceType extracts a simplified voice type from Azure's VoiceType field
func extractAzureVoiceType(voiceType string) string {
	// Azure voice types: "Standard", "Neural", "NeuralMultilingual", etc.
	if strings.Contains(voiceType, "Neural") {
		return "Neural"
	}
	return "Standard"
}

// GetVoicesByLanguage returns voices filtered by language code
func (p *AzureProvider) GetVoicesByLanguage(languageCode string) []Voice {
	allVoices := p.GetAvailableVoices()
	if languageCode == "" {
		return allVoices
	}

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		if v.Language == languageCode || strings.HasPrefix(v.Language, languageCode+"-") {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetVoicesByModel returns voices filtered by model type (Standard or Neural)
func (p *AzureProvider) GetVoicesByModel(modelType string) []Voice {
	allVoices := p.GetAvailableVoices()
	if modelType == "" {
		return allVoices
	}

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		if strings.Contains(v.Name, modelType) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetVoicesFiltered returns voices filtered by language and model type
func (p *AzureProvider) GetVoicesFiltered(languageCode, modelType string) []Voice {
	allVoices := p.GetAvailableVoices()

	filtered := make([]Voice, 0)
	for _, v := range allVoices {
		matchLang := languageCode == "" || v.Language == languageCode || strings.HasPrefix(v.Language, languageCode+"-")
		matchModel := modelType == "" || strings.Contains(v.Name, modelType)
		if matchLang && matchModel {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetAvailableLanguages returns available languages from voices
func (p *AzureProvider) GetAvailableLanguages() []string {
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

// GetAvailableModels returns available model types
func (p *AzureProvider) GetAvailableModels() []string {
	return []string{"Standard", "Neural"}
}

// ConvertToSpeech converts text to speech using Azure TTS API
func (p *AzureProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	if p.subscriptionKey == "" || p.region == "" {
		return nil, fmt.Errorf("Azure TTS subscription key or region not configured")
	}

	token, err := p.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	// Extract language from voice name (e.g., "en-US-GuyNeural" -> "en-US")
	language := extractAzureLanguage(voice)

	// Build SSML - check if text is already SSML-wrapped
	var ssml string
	trimmedText := strings.TrimSpace(text)
	if strings.HasPrefix(trimmedText, "<speak>") && strings.HasSuffix(trimmedText, "</speak>") {
		// Text is already SSML - extract inner content and wrap with Azure's required attributes
		innerContent := strings.TrimPrefix(trimmedText, "<speak>")
		innerContent = strings.TrimSuffix(innerContent, "</speak>")
		innerContent = strings.TrimSpace(innerContent)
		ssml = fmt.Sprintf(
			`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='%s'><voice name='%s'>%s</voice></speak>`,
			language, voice, innerContent,
		)
	} else {
		// Plain text - escape and wrap
		escapedText := html.EscapeString(text)
		ssml = fmt.Sprintf(
			`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='%s'><voice name='%s'>%s</voice></speak>`,
			language, voice, escapedText,
		)
	}

	logger.Debug("AzureProvider.ConvertToSpeech: voice='%s', ssml_len=%d", voice, len(ssml))
	logger.Debug("AzureProvider SSML text: %s", ssml)

	// Create request
	req, err := http.NewRequest("POST", p.getTTSURL(), bytes.NewBufferString(ssml))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-OutputFormat", "riff-24khz-16bit-mono-pcm") // WAV format
	req.Header.Set("User-Agent", "BiblioHub-Audiobook-Builder")

	// Send request with retries
	var lastErr error
	for retry := 0; retry < azureMaxRetries; retry++ {
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("API request failed: %v", err)
			time.Sleep(time.Duration(1<<retry) * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("Azure TTS API error: status %d: %s", resp.StatusCode, string(body))

			// Refresh token on 401
			if resp.StatusCode == http.StatusUnauthorized {
				p.accessToken = ""
				token, _ = p.getAccessToken()
				req.Header.Set("Authorization", "Bearer "+token)
			}
			time.Sleep(time.Duration(1<<retry) * time.Second)
			continue
		}

		// Read audio data
		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

		return bytes.NewReader(audioData), nil
	}

	return nil, lastErr
}

// extractAzureLanguage extracts the language code from an Azure voice name
// e.g., "en-US-GuyNeural" -> "en-US"
func extractAzureLanguage(voiceName string) string {
	parts := strings.Split(voiceName, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return "en-US" // Default fallback
}

// parseJSON is a simple JSON parser for Azure responses
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// EstimateAzureCost estimates the cost for converting the given number of characters
// Azure TTS pricing: $4/1M chars for Standard, $16/1M chars for Neural
func EstimateAzureCost(charCount int, voiceType string) float64 {
	pricePerMillion := 16.0 // Neural (default)
	if voiceType == "Standard" {
		pricePerMillion = 4.0
	}
	return float64(charCount) / 1000000.0 * pricePerMillion
}

// GetAzureSupportedFormats returns the list of supported output formats
func GetAzureSupportedFormats() []string {
	return azureOutputFormats
}

// RefreshVoices reloads the voice list from the API
func (p *AzureProvider) RefreshVoices() error {
	voices, err := p.fetchVoicesFromAPI()
	if err != nil {
		return err
	}
	p.cachedVoices = voices
	p.lastVoiceFetch = time.Now()
	return nil
}
