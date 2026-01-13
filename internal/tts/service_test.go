package tts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"abb_tts/internal/config"
)

func TestNewService_WithOpenTTS(t *testing.T) {
	// Create mock OpenTTS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {
					ID:       "en-US",
					Name:     "en-US",
					Gender:   "F",
					Language: "en",
					TTSName:  "nanotts",
				},
				"espeak:de-DE": {
					ID:       "de-DE",
					Name:     "de-DE",
					Gender:   "M",
					Language: "de",
					TTSName:  "espeak",
				},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenTTSURL: server.URL,
	}

	svc := NewService(cfg)

	// Verify OpenTTS provider is registered
	providers := svc.GetAvailableProviders()
	found := false
	for _, p := range providers {
		if p == "opentts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("OpenTTS provider should be registered when URL is configured")
	}
}

func TestNewService_WithoutOpenTTS(t *testing.T) {
	cfg := &config.Config{
		OpenTTSURL: "", // Empty URL
	}

	svc := NewService(cfg)

	// Verify OpenTTS provider is NOT registered
	providers := svc.GetAvailableProviders()
	for _, p := range providers {
		if p == "opentts" {
			t.Error("OpenTTS provider should not be registered when URL is empty")
		}
	}
}

func TestService_GetAvailableModels_OpenTTS(t *testing.T) {
	// Create mock OpenTTS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US":  {TTSName: "nanotts", Language: "en"},
				"espeak:en-GB":   {TTSName: "espeak", Language: "en"},
				"festival:en-US": {TTSName: "festival", Language: "en"},
				"coqui-tts:de":   {TTSName: "coqui-tts", Language: "de"},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenTTSURL: server.URL,
	}

	svc := NewService(cfg)
	models := svc.GetAvailableModels("opentts")

	// Should return unique engine names
	if len(models) != 4 {
		t.Errorf("expected 4 models (engines), got %d", len(models))
	}

	// Verify expected engines are present
	engineMap := make(map[string]bool)
	for _, m := range models {
		engineMap[m] = true
	}

	expectedEngines := []string{"nanotts", "espeak", "festival", "coqui-tts"}
	for _, e := range expectedEngines {
		if !engineMap[e] {
			t.Errorf("expected engine %s to be in models", e)
		}
	}
}

func TestService_GetVoicesFiltered_OpenTTS(t *testing.T) {
	// Create mock OpenTTS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {ID: "en-US", Name: "en-US", TTSName: "nanotts", Language: "en"},
				"nanotts:de-DE": {ID: "de-DE", Name: "de-DE", TTSName: "nanotts", Language: "de"},
				"espeak:en-GB":  {ID: "en-GB", Name: "en-GB", TTSName: "espeak", Language: "en"},
				"espeak:fr-FR":  {ID: "fr-FR", Name: "fr-FR", TTSName: "espeak", Language: "fr"},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenTTSURL: server.URL,
	}

	svc := NewService(cfg)

	// Filter by language only
	enVoices := svc.GetVoicesFiltered("opentts", "en", "")
	if len(enVoices) != 2 {
		t.Errorf("expected 2 English voices, got %d", len(enVoices))
	}

	// Filter by engine only
	nanottsVoices := svc.GetVoicesFiltered("opentts", "", "nanotts")
	if len(nanottsVoices) != 2 {
		t.Errorf("expected 2 nanotts voices, got %d", len(nanottsVoices))
	}

	// Filter by both language and engine
	enEspeakVoices := svc.GetVoicesFiltered("opentts", "en", "espeak")
	if len(enEspeakVoices) != 1 {
		t.Errorf("expected 1 English espeak voice, got %d", len(enEspeakVoices))
	}

	// No filter - should return all
	allVoices := svc.GetVoicesFiltered("opentts", "", "")
	if len(allVoices) != 4 {
		t.Errorf("expected 4 voices with no filter, got %d", len(allVoices))
	}
}

func TestService_GetAvailableLanguages_OpenTTS(t *testing.T) {
	// Create mock OpenTTS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {Language: "en", TTSName: "nanotts"},
				"nanotts:de-DE": {Language: "de", TTSName: "nanotts"},
				"espeak:en-GB":  {Language: "en", TTSName: "espeak"},
				"espeak:fr-FR":  {Language: "fr", TTSName: "espeak"},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenTTSURL: server.URL,
	}

	svc := NewService(cfg)
	languages := svc.GetAvailableLanguages("opentts")

	// Should return unique languages
	if len(languages) != 3 {
		t.Errorf("expected 3 unique languages, got %d", len(languages))
	}

	// Verify expected languages are present
	langMap := make(map[string]bool)
	for _, l := range languages {
		langMap[l] = true
	}

	expectedLangs := []string{"en", "de", "fr"}
	for _, l := range expectedLangs {
		if !langMap[l] {
			t.Errorf("expected language %s to be in languages", l)
		}
	}
}

func TestService_ReloadProviders_OpenTTS(t *testing.T) {
	// Create mock OpenTTS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/voices" {
			voices := map[string]openTTSVoice{
				"nanotts:en-US": {TTSName: "nanotts", Language: "en"},
			}
			json.NewEncoder(w).Encode(voices)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenTTSURL: "", // Start without OpenTTS
	}

	svc := NewService(cfg)

	// Verify OpenTTS is not registered initially
	providers := svc.GetAvailableProviders()
	for _, p := range providers {
		if p == "opentts" {
			t.Error("OpenTTS should not be registered initially")
		}
	}

	// Update config and reload
	cfg.OpenTTSURL = server.URL
	svc.ReloadProviders()

	// Verify OpenTTS is now registered
	providers = svc.GetAvailableProviders()
	found := false
	for _, p := range providers {
		if p == "opentts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("OpenTTS should be registered after reload with URL configured")
	}
}
