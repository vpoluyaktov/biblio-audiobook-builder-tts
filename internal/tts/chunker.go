package tts

import (
	"regexp"
	"strings"
	"unicode"
)

// ChunkMode defines how text should be split
type ChunkMode int

const (
	ChunkBySentence ChunkMode = iota
	ChunkByParagraph
	ChunkBySize
)

// ChunkerConfig holds configuration for text chunking
type ChunkerConfig struct {
	Mode          ChunkMode
	MaxChunkSize  int  // Maximum characters per chunk (for ChunkBySize or as limit)
	PreserveWords bool // Don't split words when chunking by size
}

// DefaultChunkerConfig returns sensible defaults for chunking
func DefaultChunkerConfig() *ChunkerConfig {
	return &ChunkerConfig{
		Mode:          ChunkBySentence,
		MaxChunkSize:  900, // Silero TTS has 1000 char limit, leave margin for SSML overhead
		PreserveWords: true,
	}
}

// Chunker splits text into manageable pieces for TTS processing
type Chunker struct {
	config *ChunkerConfig
}

// NewChunker creates a new text chunker
func NewChunker(config *ChunkerConfig) *Chunker {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	return &Chunker{config: config}
}

// Chunk splits text into chunks based on the configured mode
func (c *Chunker) Chunk(text string) []string {
	// Clean up the text first
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var chunks []string

	switch c.config.Mode {
	case ChunkByParagraph:
		chunks = c.chunkByParagraph(text)
	case ChunkBySize:
		chunks = c.chunkBySize(text)
	default: // ChunkBySentence
		chunks = c.chunkBySentence(text)
	}

	// Ensure no chunk exceeds max size
	return c.enforceMaxSize(chunks)
}

// chunkBySentence splits text by sentences
func (c *Chunker) chunkBySentence(text string) []string {
	// Regex to match sentence endings: . ! ? followed by space or end
	// Also handles abbreviations like Mr. Mrs. Dr. etc.
	sentenceEnders := regexp.MustCompile(`([.!?]+)\s+`)

	// Split by sentence endings but keep the punctuation
	parts := sentenceEnders.Split(text, -1)
	matches := sentenceEnders.FindAllStringSubmatch(text, -1)

	var sentences []string
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Add back the punctuation
		if i < len(matches) {
			part += matches[i][1]
		}

		sentences = append(sentences, part)
	}

	// If no sentences found, return the whole text
	if len(sentences) == 0 && text != "" {
		return []string{text}
	}

	return sentences
}

// chunkByParagraph splits text by paragraphs (double newlines)
func (c *Chunker) chunkByParagraph(text string) []string {
	// Split by double newlines or multiple newlines
	paragraphSep := regexp.MustCompile(`\n\s*\n+`)
	parts := paragraphSep.Split(text, -1)

	var paragraphs []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Normalize internal whitespace
		part = regexp.MustCompile(`\s+`).ReplaceAllString(part, " ")
		if part != "" {
			paragraphs = append(paragraphs, part)
		}
	}

	return paragraphs
}

// chunkBySize splits text into chunks of approximately maxSize characters
func (c *Chunker) chunkBySize(text string) []string {
	if len(text) <= c.config.MaxChunkSize {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= c.config.MaxChunkSize {
			chunks = append(chunks, strings.TrimSpace(remaining))
			break
		}

		// Find a good break point
		breakPoint := c.findBreakPoint(remaining, c.config.MaxChunkSize)
		chunk := strings.TrimSpace(remaining[:breakPoint])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = remaining[breakPoint:]
	}

	return chunks
}

// findBreakPoint finds a good place to break the text (sentence end, word boundary)
func (c *Chunker) findBreakPoint(text string, maxPos int) int {
	if maxPos >= len(text) {
		return len(text)
	}

	// Try to find sentence end first (. ! ?)
	for i := maxPos; i > maxPos/2; i-- {
		if i < len(text) && (text[i] == '.' || text[i] == '!' || text[i] == '?') {
			// Make sure it's followed by space or end
			if i+1 >= len(text) || unicode.IsSpace(rune(text[i+1])) {
				return i + 1
			}
		}
	}

	// Try to find word boundary
	if c.config.PreserveWords {
		for i := maxPos; i > maxPos/2; i-- {
			if i < len(text) && unicode.IsSpace(rune(text[i])) {
				return i + 1
			}
		}
	}

	// Fall back to hard cut
	return maxPos
}

// enforceMaxSize ensures no chunk exceeds the maximum size
func (c *Chunker) enforceMaxSize(chunks []string) []string {
	var result []string

	for _, chunk := range chunks {
		if len(chunk) <= c.config.MaxChunkSize {
			result = append(result, chunk)
		} else {
			// Split oversized chunks
			subChunks := c.chunkBySize(chunk)
			result = append(result, subChunks...)
		}
	}

	return result
}

// MergeSmallChunks combines small chunks to reduce API calls
// minSize is the minimum chunk size to keep separate
func (c *Chunker) MergeSmallChunks(chunks []string, minSize int) []string {
	if len(chunks) <= 1 {
		return chunks
	}

	var result []string
	var current strings.Builder

	for _, chunk := range chunks {
		if current.Len() == 0 {
			current.WriteString(chunk)
			continue
		}

		// Check if adding this chunk would exceed max size
		if current.Len()+1+len(chunk) > c.config.MaxChunkSize {
			result = append(result, current.String())
			current.Reset()
			current.WriteString(chunk)
		} else if current.Len() < minSize {
			// Merge small chunks
			current.WriteString(" ")
			current.WriteString(chunk)
		} else {
			result = append(result, current.String())
			current.Reset()
			current.WriteString(chunk)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
