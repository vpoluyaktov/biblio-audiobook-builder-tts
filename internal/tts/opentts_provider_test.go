package tts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOpenTTSProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {
					ID:       "en-US",
					Name:     "en-US",
					Gender:   "F",
					Language: "en",
					Locale:   "en-us",
					TTSName:  "nanotts",
				},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	if provider == nil {
		t.Fatal("NewOpenTTSProvider returned nil")
	}
	if provider.GetName() != "opentts" {
		t.Errorf("expected name 'opentts', got '%s'", provider.GetName())
	}
	if provider.serverURL != server.URL {
		t.Errorf("expected serverURL '%s', got '%s'", server.URL, provider.serverURL)
	}
}

func TestOpenTTSProvider_GetAvailableVoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {
					ID:       "en-US",
					Name:     "en-US",
					Gender:   "F",
					Language: "en",
					Locale:   "en-us",
					TTSName:  "nanotts",
				},
				"larynx:ljspeech": {
					ID:       "ljspeech",
					Name:     "ljspeech",
					Gender:   "F",
					Language: "en",
					Locale:   "en-us",
					TTSName:  "larynx",
				},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	voices := provider.GetAvailableVoices()

	if len(voices) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(voices))
	}

	// Check that voices have correct provider
	for _, v := range voices {
		if v.Provider != "opentts" {
			t.Errorf("expected provider 'opentts', got '%s'", v.Provider)
		}
	}
}

func TestOpenTTSProvider_GetVoicesFiltered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {
					ID:       "en-US",
					Name:     "en-US",
					Gender:   "F",
					Language: "en",
					Locale:   "en-us",
					TTSName:  "nanotts",
				},
				"nanotts:de-DE": {
					ID:       "de-DE",
					Name:     "de-DE",
					Gender:   "F",
					Language: "de",
					Locale:   "de-de",
					TTSName:  "nanotts",
				},
				"espeak:en-GB": {
					ID:       "en-GB",
					Name:     "en-GB",
					Gender:   "M",
					Language: "en",
					Locale:   "en-gb",
					TTSName:  "espeak",
				},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)

	// Filter by English only
	enVoices := provider.GetVoicesFiltered("en", "")
	if len(enVoices) != 2 {
		t.Errorf("expected 2 English voices, got %d", len(enVoices))
	}

	// Filter by German only
	deVoices := provider.GetVoicesFiltered("de", "")
	if len(deVoices) != 1 {
		t.Errorf("expected 1 German voice, got %d", len(deVoices))
	}

	// Filter by engine only
	nanottsVoices := provider.GetVoicesFiltered("", "nanotts")
	if len(nanottsVoices) != 2 {
		t.Errorf("expected 2 nanotts voices, got %d", len(nanottsVoices))
	}

	// Filter by both language and engine
	enNanottsVoices := provider.GetVoicesFiltered("en", "nanotts")
	if len(enNanottsVoices) != 1 {
		t.Errorf("expected 1 English nanotts voice, got %d", len(enNanottsVoices))
	}

	// No filter
	allVoices := provider.GetVoicesFiltered("", "")
	if len(allVoices) != 3 {
		t.Errorf("expected 3 voices with no filter, got %d", len(allVoices))
	}
}

func TestOpenTTSProvider_GetAvailableLanguages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {Language: "en", TTSName: "nanotts"},
				"nanotts:de-DE": {Language: "de", TTSName: "nanotts"},
				"larynx:en-GB":  {Language: "en", TTSName: "larynx"},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	languages := provider.GetAvailableLanguages()

	if len(languages) != 2 {
		t.Errorf("expected 2 unique languages, got %d", len(languages))
	}
}

func TestOpenTTSProvider_GetAvailableEngines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {TTSName: "nanotts"},
				"larynx:en-US":  {TTSName: "larynx"},
				"marytts:en-US": {TTSName: "marytts"},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	engines := provider.GetAvailableEngines()

	if len(engines) != 3 {
		t.Errorf("expected 3 engines, got %d", len(engines))
	}
}

func TestOpenTTSProvider_ConvertToSpeech(t *testing.T) {
	audioData := []byte("fake audio data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{}
			json.NewEncoder(w).Encode(voices)
			return
		}
		if r.URL.Path == "/api/tts" {
			voice := r.URL.Query().Get("voice")
			text := r.URL.Query().Get("text")

			if voice == "" || text == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "audio/wav")
			w.Write(audioData)
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	reader, err := provider.ConvertToSpeech("Hello world", "nanotts:en-US", nil)
	if err != nil {
		t.Fatalf("ConvertToSpeech failed: %v", err)
	}

	// Read the audio data
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	if string(buf[:n]) != string(audioData) {
		t.Errorf("expected audio data '%s', got '%s'", audioData, buf[:n])
	}
}

func TestOpenTTSProvider_ConvertToSpeech_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{}
			json.NewEncoder(w).Encode(voices)
			return
		}
		if r.URL.Path == "/api/tts" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("TTS engine error"))
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	_, err := provider.ConvertToSpeech("Hello world", "invalid:voice", nil)
	if err == nil {
		t.Error("expected error for failed TTS request")
	}
}

func TestOpenTTSProvider_TestConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{}
			json.NewEncoder(w).Encode(voices)
			return
		}
		if r.URL.Path == "/api/languages" {
			json.NewEncoder(w).Encode([]string{"en", "de", "fr"})
		}
	}))
	defer server.Close()

	provider := NewOpenTTSProvider(server.URL)
	err := provider.TestConnection()
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestOpenTTSProvider_TestConnection_Failure(t *testing.T) {
	provider := NewOpenTTSProvider("http://localhost:99999")
	err := provider.TestConnection()
	if err == nil {
		t.Error("expected error for connection to non-existent server")
	}
}

func TestFormatVoiceName(t *testing.T) {
	tests := []struct {
		engine   string
		name     string
		expected string
	}{
		{"nanotts", "en-US", "Nanotts - en US"},
		{"larynx", "ljspeech_glow_tts", "Larynx - ljspeech glow tts"},
		{"MARYTTS", "cmu-slt-hsmm", "Marytts - cmu slt hsmm"},
	}

	for _, tt := range tests {
		result := formatVoiceName(tt.engine, tt.name)
		if result != tt.expected {
			t.Errorf("formatVoiceName(%s, %s) = %s, expected %s", tt.engine, tt.name, result, tt.expected)
		}
	}
}
