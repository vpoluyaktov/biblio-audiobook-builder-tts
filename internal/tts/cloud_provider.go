package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CloudProvider uses cloud-based TTS services
type CloudProvider struct {
	BaseProvider
	service     string
	apiKey      string
	apiEndpoint string
}

// NewCloudProvider creates a new cloud TTS provider
func NewCloudProvider(service, apiKey, apiEndpoint string) Provider {
	return &CloudProvider{
		BaseProvider: BaseProvider{name: "cloud"},
		service:      service,
		apiKey:       apiKey,
		apiEndpoint:  apiEndpoint,
	}
}

// GetAvailableVoices returns a list of available voices for the cloud service
func (p *CloudProvider) GetAvailableVoices() []Voice {
	// This would typically make an API call to get available voices
	// For now, return a static list based on the service
	voices := []Voice{}

	switch p.service {
	case "google":
		voices = append(voices, []Voice{
			{ID: "en-US-Standard-A", Name: "US English Female", Language: "en-US", Gender: "female", Provider: "google"},
			{ID: "en-US-Standard-B", Name: "US English Male", Language: "en-US", Gender: "male", Provider: "google"},
		}...)
	case "azure":
		voices = append(voices, []Voice{
			{ID: "en-US-JennyNeural", Name: "Jenny", Language: "en-US", Gender: "female", Provider: "azure"},
			{ID: "en-US-GuyNeural", Name: "Guy", Language: "en-US", Gender: "male", Provider: "azure"},
		}...)
	}

	return voices
}

// ConvertToSpeech converts text to speech using the cloud service
func (p *CloudProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	var requestBody []byte
	var err error

	switch p.service {
	case "google":
		requestBody, err = json.Marshal(map[string]interface{}{
			"input": map[string]string{
				"text": text,
			},
			"voice": map[string]string{
				"languageCode": "en-US",
				"name":         voice,
			},
			"audioConfig": map[string]interface{}{
				"audioEncoding": "MP3",
				"speakingRate":  options.Speed,
				"pitch":         options.Pitch,
			},
		})
	case "azure":
		requestBody, err = json.Marshal(map[string]interface{}{
			"text":  text,
			"voice": voice,
			"options": map[string]interface{}{
				"rate":  options.Speed,
				"pitch": options.Pitch,
			},
		})
	default:
		return nil, fmt.Errorf("unsupported cloud service: %s", p.service)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req, err := http.NewRequest("POST", p.apiEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	// Read the audio content
	audioContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return bytes.NewReader(audioContent), nil
}

// GetAvailableLanguages returns available language codes
func (p *CloudProvider) GetAvailableLanguages() []string {
	langMap := make(map[string]bool)
	for _, v := range p.GetAvailableVoices() {
		langMap[v.Language] = true
	}
	languages := make([]string, 0, len(langMap))
	for lang := range langMap {
		languages = append(languages, lang)
	}
	return languages
}

// GetModelsForLanguage returns available models (cloud provider has no models)
func (p *CloudProvider) GetModelsForLanguage(language string) []string {
	return []string{}
}

// GetVoicesFiltered returns voices filtered by language and model
func (p *CloudProvider) GetVoicesFiltered(language, model string) []Voice {
	if language == "" {
		return p.GetAvailableVoices()
	}
	var filtered []Voice
	for _, v := range p.GetAvailableVoices() {
		if v.Language == language {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// RefreshVoices reloads the voice list (no-op for cloud provider)
func (p *CloudProvider) RefreshVoices() error {
	return nil
}
