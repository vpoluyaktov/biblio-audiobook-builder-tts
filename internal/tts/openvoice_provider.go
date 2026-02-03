package tts

import (
	"biblio-audiobook-builder-tts/internal/logger"
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

// OpenVoiceProvider uses OpenVoice/MeloTTS server for text-to-speech conversion
type OpenVoiceProvider struct {
	BaseProvider
	serverURL  string
	httpClient *http.Client
	voices     []Voice
	voicesMap  map[string]openvoiceVoice
}

// openvoiceVoice represents a voice from OpenVoice TTS API
type openvoiceVoice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
	Locale   string `json:"locale"`
	TTSName  string `json:"tts_name"`
	ModelID  string `json:"model_id"`
}

// NewOpenVoiceProvider creates a new OpenVoice TTS provider
func NewOpenVoiceProvider(serverURL string) *OpenVoiceProvider {
	// Ensure URL doesn't have trailing slash
	serverURL = strings.TrimSuffix(serverURL, "/")

	p := &OpenVoiceProvider{
		BaseProvider: BaseProvider{name: "openvoice"},
		serverURL:    serverURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // TTS can take a while for long text
			Transport: &http.Transport{
				DisableKeepAlives: true, // Disable keep-alive to force new connection per request for load balancing
			},
		},
		voicesMap: make(map[string]openvoiceVoice),
	}

	// Load voices with retry logic (OpenVoice server may not be ready yet)
	go p.loadVoicesWithRetry()

	return p
}

// loadVoicesWithRetry attempts to load voices with exponential backoff
func (p *OpenVoiceProvider) loadVoicesWithRetry() {
	maxRetries := 10
	baseDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := p.loadVoices(); err != nil {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff: 2s, 4s, 8s, 16s...
			if delay > 60*time.Second {
				delay = 60 * time.Second // Cap at 60 seconds
			}
			logger.Warn("Failed to load OpenVoice voices (attempt %d/%d): %v. Retrying in %v...", attempt+1, maxRetries, err, delay)
			time.Sleep(delay)
		} else {
			logger.Info("Loaded %d voices from OpenVoice TTS server at %s", len(p.voices), p.serverURL)
			return
		}
	}
	logger.Error("Failed to load OpenVoice voices after %d attempts. Provider will have no voices.", maxRetries)
}

// loadVoices fetches available voices from the OpenVoice TTS server
func (p *OpenVoiceProvider) loadVoices() error {
	resp, err := p.httpClient.Get(p.serverURL + "/api/voices")
	if err != nil {
		return fmt.Errorf("failed to fetch voices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch voices: status %d", resp.StatusCode)
	}

	// OpenVoice returns voices as a map (same format as Silero)
	var voicesMap map[string]openvoiceVoice
	if err := json.NewDecoder(resp.Body).Decode(&voicesMap); err != nil {
		return fmt.Errorf("failed to decode voices: %w", err)
	}

	p.voices = make([]Voice, 0, len(voicesMap))
	p.voicesMap = make(map[string]openvoiceVoice)

	for fullID, v := range voicesMap {
		p.voicesMap[fullID] = v
		voice := Voice{
			ID:       fullID, // e.g., "openvoice:EN-US#default"
			Name:     formatOpenVoiceVoiceName(v.Name, v.ModelID),
			Language: v.Language,
			Gender:   v.Gender,
			Provider: "openvoice",
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

// formatOpenVoiceVoiceName creates a readable voice name
func formatOpenVoiceVoiceName(name, modelID string) string {
	// Clean up name
	name = strings.ReplaceAll(name, "_", " ")
	if modelID != "" {
		return fmt.Sprintf("%s (%s)", name, modelID)
	}
	return name
}

// GetAvailableVoices returns a list of available voices
func (p *OpenVoiceProvider) GetAvailableVoices() []Voice {
	return p.voices
}

// GetVoicesFiltered returns voices filtered by language and model
func (p *OpenVoiceProvider) GetVoicesFiltered(language, model string) []Voice {
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
				Name:     formatOpenVoiceVoiceName(v.Name, v.ModelID),
				Language: v.Language,
				Gender:   v.Gender,
				Provider: "openvoice",
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
func (p *OpenVoiceProvider) GetAvailableLanguages() []string {
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

// GetAvailableModels returns available OpenVoice models
func (p *OpenVoiceProvider) GetAvailableModels() []string {
	return p.GetModelsForLanguage("")
}

// GetModelsForLanguage returns models filtered by language
func (p *OpenVoiceProvider) GetModelsForLanguage(language string) []string {
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

// ConvertToSpeech converts text to speech using OpenVoice TTS
func (p *OpenVoiceProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	// Get speed from options (OpenVoice supports speed parameter)
	speed := 1.0
	if options != nil && options.Speed != 0 {
		speed = options.Speed
	}

	logger.Debug("OpenVoiceProvider.ConvertToSpeech: voice='%s', speed=%.2f, text_len=%d", voice, speed, len(text))

	// Build the TTS URL
	params := url.Values{}
	params.Set("voice", voice)
	params.Set("text", text)
	params.Set("speed", fmt.Sprintf("%.2f", speed))

	// OpenVoice supports sample_rate parameter
	// Default to 44100 (OpenVoice default)
	params.Set("sample_rate", "44100")

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

// TestConnection tests the connection to the OpenVoice TTS server
func (p *OpenVoiceProvider) TestConnection() error {
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
func (p *OpenVoiceProvider) GetServerURL() string {
	return p.serverURL
}

// RefreshVoices reloads the voice list from the server
func (p *OpenVoiceProvider) RefreshVoices() error {
	return p.loadVoices()
}
