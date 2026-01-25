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

// RHVoiceProvider uses RHVoice REST server for text-to-speech conversion
type RHVoiceProvider struct {
	BaseProvider
	serverURL  string
	httpClient *http.Client
	voices     []Voice
	voicesMap  map[string]rhVoiceInfo
}

// rhVoiceInfo represents voice information from RHVoice /info endpoint
type rhVoiceInfo struct {
	Name    string `json:"name"`
	Lang    string `json:"lang"`
	Country string `json:"country"`
	Gender  string `json:"gender"`
	No      int    `json:"no"`
}

// rhVoiceServerInfo represents the /info endpoint response
type rhVoiceServerInfo struct {
	DefaultVoice  string                 `json:"DEFAULT_VOICE"`
	DefaultFormat string                 `json:"DEFAULT_FORMAT"`
	Formats       map[string]string      `json:"FORMATS"`
	SupportVoices []string               `json:"SUPPORT_VOICES"`
	VoicesInfo    map[string]rhVoiceInfo `json:"rhvoice_wrapper_voices_info"`
}

// NewRHVoiceProvider creates a new RHVoice provider
func NewRHVoiceProvider(serverURL string) *RHVoiceProvider {
	// Ensure URL doesn't have trailing slash
	serverURL = strings.TrimSuffix(serverURL, "/")

	p := &RHVoiceProvider{
		BaseProvider: BaseProvider{name: "rhvoice"},
		serverURL:    serverURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // TTS can take a while for long text
		},
		voicesMap: make(map[string]rhVoiceInfo),
	}

	// Load voices on initialization
	if err := p.loadVoices(); err != nil {
		logger.Warn("Failed to load RHVoice voices: %v", err)
	} else {
		logger.Debug("Loaded %d voices from RHVoice server at %s", len(p.voices), serverURL)
	}

	return p
}

// loadVoices fetches available voices from the RHVoice server
func (p *RHVoiceProvider) loadVoices() error {
	resp, err := p.httpClient.Get(p.serverURL + "/info")
	if err != nil {
		return fmt.Errorf("failed to fetch server info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch server info: status %d", resp.StatusCode)
	}

	var serverInfo rhVoiceServerInfo
	if err := json.NewDecoder(resp.Body).Decode(&serverInfo); err != nil {
		return fmt.Errorf("failed to decode server info: %w", err)
	}

	p.voices = make([]Voice, 0, len(serverInfo.VoicesInfo))
	p.voicesMap = serverInfo.VoicesInfo

	for voiceID, info := range serverInfo.VoicesInfo {
		voice := Voice{
			ID:       voiceID,
			Name:     formatRHVoiceName(info.Name, info.Lang),
			Language: info.Lang,
			Gender:   info.Gender,
			Provider: "rhvoice",
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

// formatRHVoiceName creates a readable voice name
func formatRHVoiceName(name, lang string) string {
	// Map language codes to readable names
	langNames := map[string]string{
		"en": "English",
		"ru": "Russian",
		"uk": "Ukrainian",
		"pl": "Polish",
		"cs": "Czech",
		"sk": "Slovak",
		"ka": "Georgian",
		"ky": "Kyrgyz",
		"mk": "Macedonian",
		"pt": "Portuguese",
		"sq": "Albanian",
		"uz": "Uzbek",
		"eo": "Esperanto",
		"tt": "Tatar",
	}

	langName := langNames[lang]
	if langName == "" {
		langName = strings.ToUpper(lang)
	}

	return fmt.Sprintf("%s (%s)", name, langName)
}

// GetAvailableVoices returns a list of available voices
func (p *RHVoiceProvider) GetAvailableVoices() []Voice {
	return p.voices
}

// GetVoicesFiltered returns voices filtered by language
func (p *RHVoiceProvider) GetVoicesFiltered(language, model string) []Voice {
	if language == "" {
		return p.voices
	}

	var filtered []Voice
	for _, v := range p.voices {
		if v.Language == language || strings.HasPrefix(v.Language, language) {
			filtered = append(filtered, v)
		}
	}

	// Sort by name
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	return filtered
}

// GetAvailableLanguages returns available languages
func (p *RHVoiceProvider) GetAvailableLanguages() []string {
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

// GetAvailableModels returns a fake model for UI compatibility
func (p *RHVoiceProvider) GetAvailableModels() []string {
	return []string{"RHVoice"}
}

// GetModelsForLanguage returns models filtered by language (RHVoice has single model for all languages)
func (p *RHVoiceProvider) GetModelsForLanguage(language string) []string {
	return []string{"RHVoice"}
}

// ConvertToSpeech converts text to speech using RHVoice
// Uses POST to /rhasspy endpoint for long text support
func (p *RHVoiceProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	// Build the TTS URL with query parameters
	params := url.Values{}
	params.Set("voice", voice)

	// Add optional parameters
	// RHVoice uses 0-100 scale where 50 is default
	if options != nil {
		if options.Speed != 0 && options.Speed != 1.0 {
			// Convert our speed (0.5-2.0 where 1.0 is normal) to RHVoice rate (0-100 where 50 is normal)
			// Speed 0.5 -> rate 0, Speed 1.0 -> rate 50, Speed 2.0 -> rate 100
			rate := int((options.Speed - 0.5) * 100 / 1.5)
			if rate < 0 {
				rate = 0
			}
			if rate > 100 {
				rate = 100
			}
			params.Set("rate", fmt.Sprintf("%d", rate))
		}
		if options.Pitch != 0 && options.Pitch != 1.0 {
			// Convert our pitch (0.5-2.0 where 1.0 is normal) to RHVoice pitch (0-100 where 50 is normal)
			pitch := int((options.Pitch - 0.5) * 100 / 1.5)
			if pitch < 0 {
				pitch = 0
			}
			if pitch > 100 {
				pitch = 100
			}
			params.Set("pitch", fmt.Sprintf("%d", pitch))
		}
	}

	// Use POST to /rhasspy endpoint (text in body, always returns WAV)
	reqURL := fmt.Sprintf("%s/rhasspy?%s", p.serverURL, params.Encode())

	// Check if text is already SSML-wrapped
	trimmedText := strings.TrimSpace(text)
	contentType := "text/plain; charset=utf-8"
	if strings.HasPrefix(trimmedText, "<speak>") && strings.HasSuffix(trimmedText, "</speak>") {
		// Text is SSML - use appropriate content type
		contentType = "application/ssml+xml; charset=utf-8"
	}

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	logger.Debug("RHVoiceProvider.ConvertToSpeech: voice='%s', content_type='%s', text_len=%d", voice, contentType, len(text))
	logger.Debug("RHVoiceProvider text: %s", text)

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

// TestConnection tests the connection to the RHVoice server
func (p *RHVoiceProvider) TestConnection() error {
	resp, err := p.httpClient.Get(p.serverURL + "/info")
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
func (p *RHVoiceProvider) GetServerURL() string {
	return p.serverURL
}

// RefreshVoices reloads the voice list from the server
func (p *RHVoiceProvider) RefreshVoices() error {
	return p.loadVoices()
}
