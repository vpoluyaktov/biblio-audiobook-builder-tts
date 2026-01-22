package tts

import (
	"biblio-audiobook-builder-tts/internal/logger"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// OpenTTSProvider uses OpenTTS server for text-to-speech conversion
type OpenTTSProvider struct {
	BaseProvider
	serverURL  string
	httpClient *http.Client
	voices     []Voice
	voicesMap  map[string]openTTSVoice
}

// openTTSVoice represents a voice from OpenTTS API
type openTTSVoice struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Gender       string         `json:"gender"`
	Language     string         `json:"language"`
	Locale       string         `json:"locale"`
	TTSName      string         `json:"tts_name"`
	Multispeaker bool           `json:"multispeaker"`
	Speakers     map[string]int `json:"speakers"` // Speaker name -> index mapping
}

// NewOpenTTSProvider creates a new OpenTTS provider
func NewOpenTTSProvider(serverURL string) *OpenTTSProvider {
	// Ensure URL doesn't have trailing slash
	serverURL = strings.TrimSuffix(serverURL, "/")

	p := &OpenTTSProvider{
		BaseProvider: BaseProvider{name: "opentts"},
		serverURL:    serverURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Timeout for TTS conversion
		},
		voicesMap: make(map[string]openTTSVoice),
	}

	// Load voices on initialization
	if err := p.loadVoices(); err != nil {
		logger.Warn("Failed to load OpenTTS voices: %v", err)
	} else {
		logger.Debug("Loaded %d voices from OpenTTS server at %s", len(p.voices), serverURL)
	}

	return p
}

// loadVoices fetches available voices from the OpenTTS server
func (p *OpenTTSProvider) loadVoices() error {
	resp, err := p.httpClient.Get(p.serverURL + "/api/voices")
	if err != nil {
		return fmt.Errorf("failed to fetch voices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch voices: status %d", resp.StatusCode)
	}

	// OpenTTS returns voices as a map
	var voicesMap map[string]openTTSVoice
	if err := json.NewDecoder(resp.Body).Decode(&voicesMap); err != nil {
		return fmt.Errorf("failed to decode voices: %w", err)
	}

	p.voices = make([]Voice, 0, len(voicesMap))
	p.voicesMap = voicesMap

	for fullID, v := range voicesMap {
		voice := Voice{
			ID:       fullID, // e.g., "nanotts:en-US"
			Name:     formatVoiceName(v.TTSName, v.Name),
			Language: v.Language,
			Gender:   v.Gender,
			Provider: "opentts",
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

// formatVoiceName creates a readable voice name
func formatVoiceName(engine, name string) string {
	// Capitalize engine name
	engine = strings.Title(strings.ToLower(engine))
	// Clean up name
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return fmt.Sprintf("%s - %s", engine, name)
}

// GetAvailableVoices returns a list of available voices
func (p *OpenTTSProvider) GetAvailableVoices() []Voice {
	return p.voices
}

// GetVoicesFiltered returns voices filtered by language
func (p *OpenTTSProvider) GetVoicesFiltered(language, engine string) []Voice {
	if language == "" && engine == "" {
		return p.voices
	}

	var filtered []Voice
	for fullID, v := range p.voicesMap {
		matchLang := language == "" || v.Language == language || strings.HasPrefix(v.Language, language)
		matchEngine := engine == "" || v.TTSName == engine
		if matchLang && matchEngine {
			voice := Voice{
				ID:       fullID,
				Name:     formatVoiceName(v.TTSName, v.Name),
				Language: v.Language,
				Gender:   v.Gender,
				Provider: "opentts",
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
func (p *OpenTTSProvider) GetAvailableLanguages() []string {
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

// GetAvailableEngines returns available TTS engines
func (p *OpenTTSProvider) GetAvailableEngines() []string {
	engineMap := make(map[string]bool)
	for _, v := range p.voicesMap {
		engineMap[v.TTSName] = true
	}

	engines := make([]string, 0, len(engineMap))
	for engine := range engineMap {
		engines = append(engines, engine)
	}
	sort.Strings(engines)
	return engines
}

// ConvertToSpeech converts text to speech using OpenTTS
func (p *OpenTTSProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	// Build the TTS URL
	params := url.Values{}
	params.Set("voice", voice)
	params.Set("text", text)

	// Add optional parameters
	if options != nil {
		if options.Speed != 0 && options.Speed != 1.0 {
			// OpenTTS uses lengthScale where < 1 is faster, > 1 is slower
			// We invert our speed (where > 1 is faster) to match
			lengthScale := 1.0 / options.Speed
			params.Set("lengthScale", fmt.Sprintf("%.2f", lengthScale))
		}
	}

	reqURL := fmt.Sprintf("%s/api/tts?%s", p.serverURL, params.Encode())

	// Create request with context timeout for watchdog
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use a dedicated client with timeout for this request
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("TTS request timed out after 120s")
		}
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

// TestConnection tests the connection to the OpenTTS server
func (p *OpenTTSProvider) TestConnection() error {
	resp, err := p.httpClient.Get(p.serverURL + "/api/languages")
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
func (p *OpenTTSProvider) GetServerURL() string {
	return p.serverURL
}

// RefreshVoices reloads the voice list from the server
func (p *OpenTTSProvider) RefreshVoices() error {
	return p.loadVoices()
}
