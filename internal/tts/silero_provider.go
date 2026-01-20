package tts

import (
	"abb_tts/internal/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// SileroProvider uses Silero TTS server for text-to-speech conversion
type SileroProvider struct {
	BaseProvider
	serverURL  string
	httpClient *http.Client
	voices     []Voice
	voicesMap  map[string]sileroVoice
}

// sileroVoice represents a voice from Silero TTS API
type sileroVoice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
	Locale   string `json:"locale"`
	TTSName  string `json:"tts_name"`
	ModelID  string `json:"model_id"`
}

// NewSileroProvider creates a new Silero TTS provider
func NewSileroProvider(serverURL string) *SileroProvider {
	// Ensure URL doesn't have trailing slash
	serverURL = strings.TrimSuffix(serverURL, "/")

	p := &SileroProvider{
		BaseProvider: BaseProvider{name: "silero"},
		serverURL:    serverURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // TTS can take a while for long text
		},
		voicesMap: make(map[string]sileroVoice),
	}

	// Load voices on initialization
	if err := p.loadVoices(); err != nil {
		logger.Warn("Failed to load Silero voices: %v", err)
	} else {
		logger.Debug("Loaded %d voices from Silero TTS server at %s", len(p.voices), serverURL)
	}

	return p
}

// loadVoices fetches available voices from the Silero TTS server
func (p *SileroProvider) loadVoices() error {
	resp, err := p.httpClient.Get(p.serverURL + "/api/voices")
	if err != nil {
		return fmt.Errorf("failed to fetch voices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch voices: status %d", resp.StatusCode)
	}

	// Silero returns voices as a map (OpenTTS-compatible format)
	var voicesMap map[string]sileroVoice
	if err := json.NewDecoder(resp.Body).Decode(&voicesMap); err != nil {
		return fmt.Errorf("failed to decode voices: %w", err)
	}

	p.voices = make([]Voice, 0, len(voicesMap))
	p.voicesMap = make(map[string]sileroVoice)

	for fullID, v := range voicesMap {
		p.voicesMap[fullID] = v
		voice := Voice{
			ID:       fullID, // e.g., "silero:v3_en#en_0"
			Name:     formatSileroVoiceName(v.Name, v.ModelID),
			Language: v.Language,
			Gender:   v.Gender,
			Provider: "silero",
		}
		p.voices = append(p.voices, voice)
	}

	// Sort voices by language, then by name
	sort.Slice(p.voices, func(i, j int) bool {
		if p.voices[i].Language != p.voices[j].Language {
			return p.voices[i].Language < p.voices[j].Language
		}
		return p.voices[i].Name < p.voices[j].Name
	})

	return nil
}

// formatSileroVoiceName creates a readable voice name
func formatSileroVoiceName(name, modelID string) string {
	// Clean up name
	name = strings.ReplaceAll(name, "_", " ")
	if modelID != "" {
		return fmt.Sprintf("%s (%s)", name, modelID)
	}
	return name
}

// GetAvailableVoices returns a list of available voices
func (p *SileroProvider) GetAvailableVoices() []Voice {
	return p.voices
}

// GetVoicesFiltered returns voices filtered by language and model
func (p *SileroProvider) GetVoicesFiltered(language, model string) []Voice {
	if language == "" && model == "" {
		return p.voices
	}

	var filtered []Voice
	for fullID, v := range p.voicesMap {
		matchLang := language == "" || v.Language == language || strings.HasPrefix(v.Language, language)
		matchModel := model == "" || v.ModelID == model
		if matchLang && matchModel {
			voice := Voice{
				ID:       fullID,
				Name:     formatSileroVoiceName(v.Name, v.ModelID),
				Language: v.Language,
				Gender:   v.Gender,
				Provider: "silero",
			}
			filtered = append(filtered, voice)
		}
	}

	// Sort by name
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	return filtered
}

// GetAvailableLanguages returns available languages
func (p *SileroProvider) GetAvailableLanguages() []string {
	langMap := make(map[string]bool)
	for _, v := range p.voices {
		langMap[v.Language] = true
	}

	languages := make([]string, 0, len(langMap))
	for lang := range langMap {
		languages = append(languages, lang)
	}
	sort.Strings(languages)
	return languages
}

// GetAvailableModels returns available Silero models
func (p *SileroProvider) GetAvailableModels() []string {
	return p.GetModelsForLanguage("")
}

// GetModelsForLanguage returns models filtered by language
func (p *SileroProvider) GetModelsForLanguage(language string) []string {
	modelMap := make(map[string]bool)
	for _, v := range p.voicesMap {
		if v.ModelID != "" {
			// Filter by language if specified
			if language == "" || v.Language == language {
				modelMap[v.ModelID] = true
			}
		}
	}

	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}
	sort.Strings(models)
	return models
}

// normalizeVoiceID normalizes voice ID to lowercase speaker name
func normalizeVoiceID(voice string) string {
	parts := strings.Split(voice, "#")
	if len(parts) != 2 {
		return voice
	}
	return fmt.Sprintf("%s#%s", parts[0], strings.ToLower(parts[1]))
}

// ConvertToSpeech converts text to speech using Silero TTS
func (p *SileroProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	// Normalize voice ID: Silero expects lowercase speaker names
	// Voice format: silero:model_id#speaker
	normalizedVoice := normalizeVoiceID(voice)
	logger.Debug("SileroProvider.ConvertToSpeech: voice='%s' -> '%s', text_len=%d", voice, normalizedVoice, len(text))

	// Build the TTS URL
	params := url.Values{}
	params.Set("voice", normalizedVoice)
	params.Set("text", text)

	// Silero supports sample_rate parameter
	// Default to 48000 for best quality
	params.Set("sample_rate", "48000")

	// Note: Silero TTS doesn't support speed/pitch parameters directly
	// Speed/pitch would need SSML or post-processing

	reqURL := fmt.Sprintf("%s/api/tts?%s", p.serverURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the entire response into memory
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	return bytes.NewReader(audioData), nil
}

// TestConnection tests the connection to the Silero TTS server
func (p *SileroProvider) TestConnection() error {
	resp, err := p.httpClient.Get(p.serverURL + "/health")
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// GetServerURL returns the configured server URL
func (p *SileroProvider) GetServerURL() string {
	return p.serverURL
}

// RefreshVoices reloads the voice list from the server
func (p *SileroProvider) RefreshVoices() error {
	return p.loadVoices()
}
