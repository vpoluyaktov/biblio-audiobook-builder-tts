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

// PiperProvider uses Piper TTS server for text-to-speech conversion
type PiperProvider struct {
	BaseProvider
	serverURL  string
	httpClient *http.Client
	voices     []Voice
	voicesMap  map[string]piperVoice
}

// piperVoice represents a voice from Piper TTS API
type piperVoice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
	Locale   string `json:"locale"`
	TTSName  string `json:"tts_name"`
	ModelID  string `json:"model_id"`
}

// NewPiperProvider creates a new Piper TTS provider
func NewPiperProvider(serverURL string) *PiperProvider {
	// Ensure URL doesn't have trailing slash
	serverURL = strings.TrimSuffix(serverURL, "/")

	p := &PiperProvider{
		BaseProvider: BaseProvider{name: "piper"},
		serverURL:    serverURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // TTS can take a while for long text
			Transport: &http.Transport{
				DisableKeepAlives: true, // Disable keep-alive to force new connection per request for load balancing
			},
		},
		voicesMap: make(map[string]piperVoice),
	}

	// Load voices with retry logic (Piper server may not be ready yet)
	go p.loadVoicesWithRetry()

	return p
}

// loadVoicesWithRetry attempts to load voices with exponential backoff
func (p *PiperProvider) loadVoicesWithRetry() {
	maxRetries := 10
	baseDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := p.loadVoices(); err != nil {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff: 2s, 4s, 8s, 16s...
			if delay > 60*time.Second {
				delay = 60 * time.Second // Cap at 60 seconds
			}
			logger.Warn("Failed to load Piper voices (attempt %d/%d): %v. Retrying in %v...", attempt+1, maxRetries, err, delay)
			time.Sleep(delay)
		} else {
			logger.Info("Loaded %d voices from Piper TTS server at %s", len(p.voices), p.serverURL)
			return
		}
	}
	logger.Error("Failed to load Piper voices after %d attempts. Provider will have no voices.", maxRetries)
}

// loadVoices fetches available voices from the Piper TTS server
func (p *PiperProvider) loadVoices() error {
	resp, err := p.httpClient.Get(p.serverURL + "/api/voices")
	if err != nil {
		return fmt.Errorf("failed to fetch voices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch voices: status %d", resp.StatusCode)
	}

	// Piper returns voices as a map (same format as Silero/OpenVoice)
	var voicesMap map[string]piperVoice
	if err := json.NewDecoder(resp.Body).Decode(&voicesMap); err != nil {
		return fmt.Errorf("failed to decode voices: %w", err)
	}

	p.voices = make([]Voice, 0, len(voicesMap))
	p.voicesMap = make(map[string]piperVoice)

	for fullID, v := range voicesMap {
		p.voicesMap[fullID] = v
		voice := Voice{
			ID:       fullID, // e.g., "piper:en_US-lessac-medium#default"
			Name:     formatPiperVoiceName(v.Name, v.ModelID),
			Language: v.Language,
			Gender:   v.Gender,
			Provider: "piper",
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

// formatPiperVoiceName creates a readable voice name
func formatPiperVoiceName(name, modelID string) string {
	// Clean up name: replace underscores and hyphens with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	if modelID != "" {
		return fmt.Sprintf("%s (%s)", name, modelID)
	}
	return name
}

// GetAvailableVoices returns a list of available voices
func (p *PiperProvider) GetAvailableVoices() []Voice {
	return p.voices
}

// GetVoicesFiltered returns voices filtered by language and model
func (p *PiperProvider) GetVoicesFiltered(language, model string) []Voice {
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
				Name:     formatPiperVoiceName(v.Name, v.ModelID),
				Language: v.Language,
				Gender:   v.Gender,
				Provider: "piper",
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
func (p *PiperProvider) GetAvailableLanguages() []string {
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

// GetAvailableModels returns available Piper models
func (p *PiperProvider) GetAvailableModels() []string {
	return p.GetModelsForLanguage("")
}

// GetModelsForLanguage returns models filtered by language
func (p *PiperProvider) GetModelsForLanguage(language string) []string {
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

// ConvertToSpeech converts text to speech using Piper TTS
func (p *PiperProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	// Get speed from options (Piper supports speed parameter)
	speed := 1.0
	if options != nil && options.Speed != 0 {
		speed = options.Speed
	}

	logger.Debug("PiperProvider.ConvertToSpeech: voice='%s', speed=%.2f, text_len=%d", voice, speed, len(text))

	// Build the TTS URL
	params := url.Values{}
	params.Set("voice", voice)
	params.Set("text", text)
	params.Set("speed", fmt.Sprintf("%.2f", speed))

	// Piper supports sample_rate parameter
	// Default to 22050 (Piper's common output rate)
	params.Set("sample_rate", "22050")

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

// TestConnection tests the connection to the Piper TTS server
func (p *PiperProvider) TestConnection() error {
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
func (p *PiperProvider) GetServerURL() string {
	return p.serverURL
}

// RefreshVoices reloads the voice list from the server
func (p *PiperProvider) RefreshVoices() error {
	return p.loadVoices()
}
