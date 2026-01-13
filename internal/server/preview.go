package server

import (
	"fmt"
	"sync"
	"time"

	"abb_tts/internal/parser"

	"github.com/google/uuid"
)

// Preview represents a parsed book preview with metadata and cost estimates
type Preview struct {
	ID                         string                  `json:"id"`
	FileName                   string                  `json:"file_name"`
	FilePath                   string                  `json:"file_path,omitempty"`
	BookTitle                  string                  `json:"book_title"`
	BookAuthor                 string                  `json:"book_author"`
	Description                string                  `json:"description"`
	CoverImageURL              string                  `json:"cover_image_url,omitempty"`
	Chapters                   []ChapterPreview        `json:"chapters"`
	TotalChapters              int                     `json:"total_chapters"`
	TotalWords                 int                     `json:"total_words"`
	TotalCharacters            int                     `json:"total_characters"`
	EstimatedDurationMinutes   int                     `json:"estimated_duration_minutes"`
	EstimatedDurationFormatted string                  `json:"estimated_duration_formatted"`
	CostEstimates              map[string]CostEstimate `json:"cost_estimates"`
	CreatedAt                  time.Time               `json:"created_at"`
	ExpiresAt                  time.Time               `json:"expires_at"`
}

// ChapterPreview represents a chapter in the preview
type ChapterPreview struct {
	Title     string `json:"title"`
	WordCount int    `json:"word_count"`
	CharCount int    `json:"char_count"`
	TOCDepth  int    `json:"toc_depth"`
}

// CostEstimate represents the cost for a specific TTS provider
type CostEstimate struct {
	Cost     float64 `json:"cost"`
	Currency string  `json:"currency"`
	Note     string  `json:"note"`
}

// TTSModelPricing contains pricing info for a TTS model
type TTSModelPricing struct {
	PricePerMillion  float64
	Note             string
	FreeMonthlyChars int // Free tier characters per month (0 = no free tier)
}

// TTS pricing per 1 million characters by provider and model
// Google pricing: https://cloud.google.com/text-to-speech/pricing (Updated January 2025)
// Note: Prices are in USD per 1 million characters
var ttsPricing = map[string]map[string]TTSModelPricing{
	"espeak": {
		"default": {0, "Free (local)", 0},
	},
	"festival": {
		"default": {0, "Free (local)", 0},
	},
	"google": {
		// Latest TTS models
		"Chirp3-HD": {30.00, "$30/1M chars", 1000000}, // 1M free/month
		// Legacy TTS models
		"Standard": {4.00, "$4/1M chars", 4000000},     // 4M free/month
		"Wavenet":  {4.00, "$4/1M chars", 4000000},     // 4M free/month (same as Standard)
		"Neural2":  {16.00, "$16/1M chars", 1000000},   // 1M free/month
		"Studio":   {160.00, "$160/1M chars", 1000000}, // 1M free/month
		"Polyglot": {16.00, "$16/1M chars", 1000000},   // 1M free/month (Preview)
		// Other models (fallback pricing)
		"News":   {16.00, "$16/1M chars", 1000000},
		"Chirp":  {16.00, "$16/1M chars", 1000000},
		"Casual": {16.00, "$16/1M chars", 1000000},
	},
	"azure": {
		"Standard": {4.00, "$4/1M chars", 500000},
		"Neural":   {16.00, "$16/1M chars", 500000},
	},
}

// ModelCostInfo represents cost info for a single model
type ModelCostInfo struct {
	Model           string  `json:"model"`
	PricePerMillion float64 `json:"price_per_million"`
	Cost            float64 `json:"cost"`
	Note            string  `json:"note"`
}

// GetAllModelPricing returns pricing for all models of a provider
func GetAllModelPricing(provider string, chars int) []ModelCostInfo {
	providerPricing, ok := ttsPricing[provider]
	if !ok {
		return []ModelCostInfo{}
	}

	var results []ModelCostInfo
	for model, pricing := range providerPricing {
		// Skip "default" entries
		if model == "default" {
			continue
		}
		// Skip free (local) providers
		if pricing.PricePerMillion == 0 {
			continue
		}
		cost := float64(chars) / 1000000.0 * pricing.PricePerMillion
		results = append(results, ModelCostInfo{
			Model:           model,
			PricePerMillion: pricing.PricePerMillion,
			Cost:            cost,
			Note:            pricing.Note,
		})
	}

	// Sort by price (cheapest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Cost < results[i].Cost {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// GetPricingForVoice returns pricing info for a specific voice ID
func GetPricingForVoice(provider, voiceID string) TTSModelPricing {
	providerPricing, ok := ttsPricing[provider]
	if !ok {
		return TTSModelPricing{0, "Unknown provider", 0}
	}

	// For local providers, return default pricing
	if provider == "espeak" || provider == "festival" {
		return providerPricing["default"]
	}

	// Extract model type from voice ID (e.g., "en-US-Wavenet-A" -> "Wavenet")
	modelType := extractModelTypeFromVoice(voiceID)
	if pricing, ok := providerPricing[modelType]; ok {
		return pricing
	}

	// Fallback to Standard pricing if model not found
	if pricing, ok := providerPricing["Standard"]; ok {
		return pricing
	}

	return TTSModelPricing{0, "Unknown model", 0}
}

// extractModelTypeFromVoice extracts the model type from a voice ID
// e.g., "en-US-Wavenet-A" -> "Wavenet", "en-US-Chirp3-HD-Achernar" -> "Chirp3-HD"
func extractModelTypeFromVoice(voiceID string) string {
	parts := splitByDash(voiceID)
	if len(parts) < 3 {
		return ""
	}

	// Handle multi-part model names like "Chirp3-HD"
	if len(parts) >= 4 && parts[2] == "Chirp3" && parts[3] == "HD" {
		return "Chirp3-HD"
	}

	return parts[2]
}

// splitByDash splits a string by dash character
func splitByDash(s string) []string {
	var parts []string
	var current []rune
	for _, r := range s {
		if r == '-' {
			if len(current) > 0 {
				parts = append(parts, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	return parts
}

// Average speech rate: ~150 words per minute
const wordsPerMinute = 150

// PreviewStore manages book previews with TTL cleanup
type PreviewStore struct {
	previews map[string]*Preview
	covers   map[string][]byte // preview ID -> cover image data
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewPreviewStore creates a new preview store with the given TTL
func NewPreviewStore(ttl time.Duration) *PreviewStore {
	ps := &PreviewStore{
		previews: make(map[string]*Preview),
		covers:   make(map[string][]byte),
		ttl:      ttl,
	}
	// Start cleanup goroutine
	go ps.cleanupLoop()
	return ps
}

// CreatePreview creates a preview from a parsed book
func (ps *PreviewStore) CreatePreview(book *parser.Book, fileName string) *Preview {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	// Build chapter previews and calculate totals in a single pass
	chapters := make([]ChapterPreview, len(book.Chapters))
	totalWords := 0
	totalChars := 0
	for i, ch := range book.Chapters {
		wordCount := countWords(ch.Content)
		charCount := len(ch.Content)
		chapters[i] = ChapterPreview{
			Title:     ch.Title,
			WordCount: wordCount,
			CharCount: charCount,
			TOCDepth:  ch.TOCDepth,
		}
		totalWords += wordCount
		totalChars += charCount
	}
	durationMinutes := totalWords / wordsPerMinute
	if durationMinutes == 0 && totalWords > 0 {
		durationMinutes = 1
	}

	// Calculate cost estimates for each provider/model combination
	costEstimates := make(map[string]CostEstimate)
	for provider, models := range ttsPricing {
		for model, pricing := range models {
			// Skip "default" entries for display, use provider name instead
			key := provider
			if model != "default" {
				key = fmt.Sprintf("%s_%s", provider, model)
			}
			cost := float64(totalChars) / 1000000.0 * pricing.PricePerMillion
			costEstimates[key] = CostEstimate{
				Cost:     cost,
				Currency: "USD",
				Note:     pricing.Note,
			}
		}
	}

	preview := &Preview{
		ID:                         id,
		FileName:                   fileName,
		BookTitle:                  book.Title,
		BookAuthor:                 book.Author,
		Description:                book.Description,
		Chapters:                   chapters,
		TotalChapters:              len(chapters),
		TotalWords:                 totalWords,
		TotalCharacters:            totalChars,
		EstimatedDurationMinutes:   durationMinutes,
		EstimatedDurationFormatted: formatDuration(durationMinutes),
		CostEstimates:              costEstimates,
		CreatedAt:                  now,
		ExpiresAt:                  now.Add(ps.ttl),
	}

	// Set cover image URL if available
	if len(book.CoverImage) > 0 {
		preview.CoverImageURL = fmt.Sprintf("/api/preview/%s/cover", id)
		ps.covers[id] = book.CoverImage
	}

	ps.previews[id] = preview
	return preview
}

// GetPreview returns a preview by ID
func (ps *PreviewStore) GetPreview(id string) (*Preview, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	preview, exists := ps.previews[id]
	return preview, exists
}

// GetCover returns the cover image data for a preview
func (ps *PreviewStore) GetCover(id string) ([]byte, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	cover, exists := ps.covers[id]
	return cover, exists
}

// DeletePreview removes a preview
func (ps *PreviewStore) DeletePreview(id string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.previews, id)
	delete(ps.covers, id)
}

// cleanupLoop periodically removes expired previews
func (ps *PreviewStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ps.cleanup()
	}
}

func (ps *PreviewStore) cleanup() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	for id, preview := range ps.previews {
		if now.After(preview.ExpiresAt) {
			delete(ps.previews, id)
			delete(ps.covers, id)
		}
	}
}

// formatDuration formats minutes into "Xh Ym" format
func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// countWords counts words in text efficiently without allocating a slice
func countWords(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			if inWord {
				count++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		count++
	}
	return count
}

// splitWords splits text into words (simple implementation)
func splitWords(text string) []string {
	var words []string
	var word []rune
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			if len(word) > 0 {
				words = append(words, string(word))
				word = nil
			}
		} else {
			word = append(word, r)
		}
	}
	if len(word) > 0 {
		words = append(words, string(word))
	}
	return words
}
