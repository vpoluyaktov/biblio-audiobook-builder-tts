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

// TTS pricing per 1 million characters
var ttsPricing = map[string]struct {
	PricePerMillion float64
	Note            string
}{
	"espeak":          {0, "Free (local)"},
	"festival":        {0, "Free (local)"},
	"google_standard": {4.00, "$4/1M chars"},
	"google_wavenet":  {16.00, "$16/1M chars"},
	"google_neural2":  {16.00, "$16/1M chars"},
	"azure_standard":  {4.00, "$4/1M chars"},
	"azure_neural":    {16.00, "$16/1M chars"},
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

	// Build chapter previews
	chapters := make([]ChapterPreview, len(book.Chapters))
	for i, ch := range book.Chapters {
		wordCount := len(splitWords(ch.Content))
		chapters[i] = ChapterPreview{
			Title:     ch.Title,
			WordCount: wordCount,
			CharCount: len(ch.Content),
			TOCDepth:  ch.TOCDepth,
		}
	}

	totalWords := book.GetTotalWords()
	totalChars := book.GetTotalCharacters()
	durationMinutes := totalWords / wordsPerMinute
	if durationMinutes == 0 && totalWords > 0 {
		durationMinutes = 1
	}

	// Calculate cost estimates
	costEstimates := make(map[string]CostEstimate)
	for provider, pricing := range ttsPricing {
		cost := float64(totalChars) / 1000000.0 * pricing.PricePerMillion
		costEstimates[provider] = CostEstimate{
			Cost:     cost,
			Currency: "USD",
			Note:     pricing.Note,
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
