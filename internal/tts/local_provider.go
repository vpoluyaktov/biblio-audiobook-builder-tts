package tts

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// LocalProvider uses local TTS engines (espeak, festival, etc.)
type LocalProvider struct {
	BaseProvider
	engine string
}

// NewLocalProvider creates a new local TTS provider
func NewLocalProvider(engine string) Provider {
	return &LocalProvider{
		BaseProvider: BaseProvider{name: "local"},
		engine:       engine,
	}
}

// GetAvailableVoices returns a list of available voices for the local engine
func (p *LocalProvider) GetAvailableVoices() []Voice {
	voices := []Voice{}

	switch p.engine {
	case "espeak":
		// Add espeak voices
		voices = append(voices, []Voice{
			{ID: "en", Name: "English", Language: "en-US", Gender: "N/A", Provider: "espeak"},
			{ID: "en-gb", Name: "British English", Language: "en-GB", Gender: "N/A", Provider: "espeak"},
		}...)
	case "festival":
		// Add festival voices
		voices = append(voices, []Voice{
			{ID: "voice_kal_diphone", Name: "US Male", Language: "en-US", Gender: "male", Provider: "festival"},
		}...)
	}

	return voices
}

// ConvertToSpeech converts text to speech using the local engine
// Note: This method handles a single chunk of text. Use Adapter for automatic chunking.
func (p *LocalProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	var cmd *exec.Cmd
	var stderr bytes.Buffer

	switch p.engine {
	case "espeak":
		// Use --stdin to read text from stdin instead of command line
		// This avoids "argument list too long" errors for large texts
		cmd = exec.Command("espeak", "-v", voice, "-w", "/dev/stdout", "--stdin")
		cmd.Stdin = bytes.NewBufferString(text)
	case "festival":
		cmd = exec.Command("festival", "--tts")
		cmd.Stdin = bytes.NewBufferString(text)
	default:
		return nil, fmt.Errorf("unsupported TTS engine: %s", p.engine)
	}

	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("TTS conversion failed: %v (stderr: %s)", err, stderr.String())
	}

	return bytes.NewReader(output), nil
}

// GetAvailableLanguages returns available language codes
func (p *LocalProvider) GetAvailableLanguages() []string {
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

// GetModelsForLanguage returns available models (local providers have no models)
func (p *LocalProvider) GetModelsForLanguage(language string) []string {
	return []string{}
}

// GetVoicesFiltered returns voices filtered by language and model
func (p *LocalProvider) GetVoicesFiltered(language, model string) []Voice {
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

// RefreshVoices reloads the voice list (no-op for local provider)
func (p *LocalProvider) RefreshVoices() error {
	return nil
}
