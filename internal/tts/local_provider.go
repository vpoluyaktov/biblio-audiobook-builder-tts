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
		engine:      engine,
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
func (p *LocalProvider) ConvertToSpeech(text string, voice string, options *ConversionOptions) (io.Reader, error) {
	var cmd *exec.Cmd

	switch p.engine {
	case "espeak":
		cmd = exec.Command("espeak", "-v", voice, "-w", "-", text)
	case "festival":
		cmd = exec.Command("festival", "--tts")
		cmd.Stdin = bytes.NewBufferString(text)
	default:
		return nil, fmt.Errorf("unsupported TTS engine: %s", p.engine)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("TTS conversion failed: %v", err)
	}

	return bytes.NewReader(output), nil
}
