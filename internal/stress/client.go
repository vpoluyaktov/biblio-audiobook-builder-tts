package stress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"biblio-audiobook-builder-tts/internal/logger"
)

// Client is a client for the Silero Stress server
type Client struct {
	baseURL    string
	httpClient *http.Client
	available  bool
	mu         sync.RWMutex
}

// StressRequest represents a request to add stress markers
type StressRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

// StressResponse represents the response with stressed text
type StressResponse struct {
	Text string `json:"text"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	ModelLoaded bool   `json:"model_loaded"`
}

// LanguageInfo represents information about a supported language
type LanguageInfo struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	HomographSupport bool   `json:"homograph_support"`
}

// LanguagesResponse represents the response from /api/languages
type LanguagesResponse struct {
	Languages []LanguageInfo `json:"languages"`
}

// NewClient creates a new stress server client
func NewClient(baseURL string) *Client {
	// Normalize URL - remove trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true, // Disable keep-alive to force new connection per request for load balancing
			},
		},
		available: false,
	}
}

// CheckHealth checks if the stress server is available
func (c *Client) CheckHealth() error {
	url := c.baseURL + "/health"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.setAvailable(false)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.setAvailable(false)
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		c.setAvailable(false)
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	if health.Status != "ok" || !health.ModelLoaded {
		c.setAvailable(false)
		return fmt.Errorf("stress server not ready: status=%s, model_loaded=%v", health.Status, health.ModelLoaded)
	}

	c.setAvailable(true)
	return nil
}

// IsAvailable returns whether the stress server is available
// If not currently available, it will retry the health check once
func (c *Client) IsAvailable() bool {
	c.mu.RLock()
	available := c.available
	c.mu.RUnlock()

	if !available {
		// Retry health check if not available
		if err := c.CheckHealth(); err == nil {
			return true
		}
	}
	return available
}

func (c *Client) setAvailable(available bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.available = available
}

// GetLanguages returns the list of supported languages
func (c *Client) GetLanguages() ([]LanguageInfo, error) {
	url := c.baseURL + "/api/languages"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get languages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get languages returned status %d", resp.StatusCode)
	}

	var langResp LanguagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&langResp); err != nil {
		return nil, fmt.Errorf("failed to decode languages response: %w", err)
	}

	return langResp.Languages, nil
}

// SupportsLanguage checks if the given language is supported
func (c *Client) SupportsLanguage(lang string) bool {
	languages, err := c.GetLanguages()
	if err != nil {
		return false
	}

	for _, l := range languages {
		if l.Code == lang {
			return true
		}
	}
	return false
}

// AddStress adds stress markers to the given text
func (c *Client) AddStress(text, language string) (string, error) {
	if text == "" {
		return "", nil
	}

	url := c.baseURL + "/api/stress"

	reqBody := StressRequest{
		Text:     text,
		Language: language,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("stress request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("stress request returned status %d: %s", resp.StatusCode, string(body))
	}

	var stressResp StressResponse
	if err := json.NewDecoder(resp.Body).Decode(&stressResp); err != nil {
		return "", fmt.Errorf("failed to decode stress response: %w", err)
	}

	return stressResp.Text, nil
}

// AddStressToSentences adds stress markers to text by processing it sentence by sentence
// This is useful for parallel processing with multiple stress server replicas
func (c *Client) AddStressToSentences(text, language string) (string, error) {
	if text == "" {
		return "", nil
	}

	// Split text into sentences
	sentences := splitIntoSentences(text)
	if len(sentences) == 0 {
		return text, nil
	}

	// Process each sentence
	var result strings.Builder
	for i, sentence := range sentences {
		if strings.TrimSpace(sentence) == "" {
			result.WriteString(sentence)
			continue
		}

		stressed, err := c.AddStress(sentence, language)
		if err != nil {
			logger.Warn("Failed to add stress to sentence %d: %v", i+1, err)
			result.WriteString(sentence) // Keep original on error
			continue
		}

		result.WriteString(stressed)
	}

	return result.String(), nil
}

// splitIntoSentences splits text into sentences while preserving whitespace
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence-ending punctuation
		if r == '.' || r == '!' || r == '?' || r == '…' {
			// Look ahead to see if this is really end of sentence
			// (not abbreviation, not ellipsis in middle of sentence)
			isEndOfSentence := true

			// Check if followed by space and uppercase letter (or end of text)
			if i+1 < len(runes) {
				next := runes[i+1]
				if next == ' ' || next == '\n' || next == '\r' || next == '\t' {
					// Consume the whitespace
					current.WriteRune(next)
					i++
					// This looks like end of sentence
				} else if next == '.' || next == '!' || next == '?' {
					// Multiple punctuation, continue
					isEndOfSentence = false
				} else {
					// No space after punctuation, probably not end of sentence
					isEndOfSentence = false
				}
			}

			if isEndOfSentence {
				sentences = append(sentences, current.String())
				current.Reset()
			}
		}
	}

	// Add remaining text
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}
