package tts

import (
	"biblio-audiobook-builder-tts/internal/logger"
	"biblio-audiobook-builder-tts/internal/sanitize"
	"biblio-audiobook-builder-tts/internal/ssml"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	// maxRetries is the maximum number of retry attempts for transient TTS failures
	maxRetries = 3
	// baseRetryDelay is the initial delay between retries
	baseRetryDelay = 500 * time.Millisecond
	// chunkTimeout is the maximum time allowed for a single chunk TTS conversion
	chunkTimeout = 120 * time.Second
)

// ttsResult holds the result of a TTS conversion attempt
type ttsResult struct {
	reader io.Reader
	err    error
}

// convertWithTimeout wraps a TTS conversion with a timeout watchdog
func (a *Adapter) convertWithTimeout(chunk string, voice string, options *ConversionOptions) (io.Reader, error) {
	resultCh := make(chan ttsResult, 1)

	go func() {
		reader, err := a.provider.ConvertToSpeech(chunk, voice, options)
		resultCh <- ttsResult{reader: reader, err: err}
	}()

	select {
	case result := <-resultCh:
		return result.reader, result.err
	case <-time.After(chunkTimeout):
		logger.Warn("TTS conversion timed out after %v - TTS backend may be hanging", chunkTimeout)
		return nil, fmt.Errorf("TTS conversion timed out after %v", chunkTimeout)
	}
}

// isRetryableError checks if an error is transient and worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Retry on server errors (5xx), tensor errors, and connection issues
	retryablePatterns := []string{
		"status 500",
		"status 502",
		"status 503",
		"status 504",
		"tensor",
		"RuntimeError",
		"connection refused",
		"connection reset",
		"timeout",
		"timed out",
		"EOF",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// Adapter wraps a TTS provider with chunking and audio concatenation
type Adapter struct {
	provider Provider
	chunker  *Chunker
}

// NewAdapter creates a new TTS adapter for a provider
func NewAdapter(provider Provider, chunkerConfig *ChunkerConfig) *Adapter {
	return &Adapter{
		provider: provider,
		chunker:  NewChunker(chunkerConfig),
	}
}

// ProgressCallback is called after each chunk is processed
// chunkIndex is 0-based, totalChunks is the total number of chunks
type ProgressCallback func(chunkIndex, totalChunks int, chunkText string)

// ConvertToSpeech converts text to speech, handling chunking automatically
// Returns concatenated audio data from all chunks
func (a *Adapter) ConvertToSpeech(text string, voice string, options *ConversionOptions, progressCb ProgressCallback) (io.Reader, error) {
	// Split text into chunks
	chunks := a.chunker.Chunk(text)

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no text to convert")
	}

	// Merge very small chunks to reduce API calls (min 100 chars)
	chunks = a.chunker.MergeSmallChunks(chunks, 100)

	logger.Debug("Converting text in %d chunks", len(chunks))

	// Process each chunk and collect audio
	var audioBuffers [][]byte

	for i, chunk := range chunks {
		if progressCb != nil {
			progressCb(i, len(chunks), chunk)
		}

		// Skip empty chunks or chunks with no speakable content for the target language
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			logger.Debug("Skipping empty chunk %d/%d", i+1, len(chunks))
			continue
		}
		// Use language-aware check if language is specified
		lang := ""
		if options != nil {
			lang = options.Language
		}
		if lang != "" && !sanitize.HasSpeakableContentForLanguage(chunk, lang) {
			logger.Debug("Skipping chunk %d/%d with no speakable content for language '%s': '%s'", i+1, len(chunks), lang, chunk)
			continue
		} else if lang == "" && !sanitize.HasSpeakableContent(chunk) {
			logger.Debug("Skipping chunk %d/%d with no speakable content: '%s'", i+1, len(chunks), chunk)
			continue
		}

		// Apply SSML wrapping per-chunk if provider supports it
		chunkToConvert := chunk
		if options != nil && options.SSMLSupport {
			chunkToConvert = ssml.WrapTextInSSML(chunk, options.UseSentencePauses)
			logger.Debug("Applied SSML wrapping to chunk %d/%d (sentence pauses: %v)", i+1, len(chunks), options.UseSentencePauses)
		}

		// Convert this chunk with retry logic for transient failures
		var audioData []byte
		var lastErr error
		for retry := 0; retry <= maxRetries; retry++ {
			if retry > 0 {
				delay := baseRetryDelay * time.Duration(1<<(retry-1)) // Exponential backoff
				logger.Warn("Retrying chunk %d/%d (attempt %d/%d) after %v: %v", i+1, len(chunks), retry+1, maxRetries+1, delay, lastErr)
				time.Sleep(delay)
			}

			// Text is already sanitized by the worker before reaching here
			// Use watchdog timeout to prevent hanging on stuck TTS backend
			reader, err := a.convertWithTimeout(chunkToConvert, voice, options)
			if err != nil {
				lastErr = err
				// Check if this is a retryable error (server errors, tensor errors, etc.)
				if isRetryableError(err) {
					continue
				}
				// Non-retryable error, fail immediately
				return nil, fmt.Errorf("failed to convert chunk %d/%d: %w", i+1, len(chunks), err)
			}

			// Read the audio data
			audioData, err = io.ReadAll(reader)
			if err != nil {
				lastErr = err
				continue
			}

			// Success
			lastErr = nil
			break
		}

		if lastErr != nil {
			return nil, fmt.Errorf("failed to convert chunk %d/%d after %d retries: %w", i+1, len(chunks), maxRetries+1, lastErr)
		}

		audioBuffers = append(audioBuffers, audioData)
	}

	// Check if we have any audio to return
	if len(audioBuffers) == 0 {
		return nil, fmt.Errorf("no speakable content for the target language - all chunks were skipped")
	}

	// Concatenate all audio buffers
	return concatenateAudio(audioBuffers), nil
}

// ConvertToSpeechWithChunks converts text and returns individual chunk results
// Useful when you need to process chunks separately (e.g., for streaming)
func (a *Adapter) ConvertToSpeechWithChunks(text string, voice string, options *ConversionOptions, progressCb ProgressCallback) ([]io.Reader, error) {
	chunks := a.chunker.Chunk(text)

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no text to convert")
	}

	chunks = a.chunker.MergeSmallChunks(chunks, 100)

	var readers []io.Reader

	for i, chunk := range chunks {
		if progressCb != nil {
			progressCb(i, len(chunks), chunk)
		}

		// Skip empty chunks or chunks with no speakable content for the target language
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			logger.Debug("Skipping empty chunk %d/%d", i+1, len(chunks))
			continue
		}
		// Use language-aware check if language is specified
		lang := ""
		if options != nil {
			lang = options.Language
		}
		if lang != "" && !sanitize.HasSpeakableContentForLanguage(chunk, lang) {
			logger.Debug("Skipping chunk %d/%d with no speakable content for language '%s': '%s'", i+1, len(chunks), lang, chunk)
			continue
		} else if lang == "" && !sanitize.HasSpeakableContent(chunk) {
			logger.Debug("Skipping chunk %d/%d with no speakable content: '%s'", i+1, len(chunks), chunk)
			continue
		}

		// Apply SSML wrapping per-chunk if provider supports it
		chunkToConvert := chunk
		if options != nil && options.SSMLSupport {
			chunkToConvert = ssml.WrapTextInSSML(chunk, options.UseSentencePauses)
			logger.Debug("Applied SSML wrapping to chunk %d/%d (sentence pauses: %v)", i+1, len(chunks), options.UseSentencePauses)
		}

		// Convert this chunk with retry logic for transient failures
		var reader io.Reader
		var lastErr error
		for retry := 0; retry <= maxRetries; retry++ {
			if retry > 0 {
				delay := baseRetryDelay * time.Duration(1<<(retry-1))
				logger.Warn("Retrying chunk %d/%d (attempt %d/%d) after %v: %v", i+1, len(chunks), retry+1, maxRetries+1, delay, lastErr)
				time.Sleep(delay)
			}

			// Text is already sanitized by the worker before reaching here
			// Use watchdog timeout to prevent hanging on stuck TTS backend
			var err error
			reader, err = a.convertWithTimeout(chunkToConvert, voice, options)
			if err != nil {
				lastErr = err
				if isRetryableError(err) {
					continue
				}
				return nil, fmt.Errorf("failed to convert chunk %d/%d: %w", i+1, len(chunks), err)
			}
			lastErr = nil
			break
		}

		if lastErr != nil {
			return nil, fmt.Errorf("failed to convert chunk %d/%d after %d retries: %w", i+1, len(chunks), maxRetries+1, lastErr)
		}

		readers = append(readers, reader)
	}

	// Check if we have any audio to return
	if len(readers) == 0 {
		return nil, fmt.Errorf("no speakable content for the target language - all chunks were skipped")
	}

	return readers, nil
}

// GetProvider returns the underlying provider
func (a *Adapter) GetProvider() Provider {
	return a.provider
}

// GetChunker returns the chunker for configuration
func (a *Adapter) GetChunker() *Chunker {
	return a.chunker
}

// concatenateAudio combines multiple audio buffers into one
// For WAV files, this needs special handling of headers
// For MP3/raw audio, simple concatenation works
func concatenateAudio(buffers [][]byte) io.Reader {
	if len(buffers) == 0 {
		return bytes.NewReader(nil)
	}

	if len(buffers) == 1 {
		return bytes.NewReader(buffers[0])
	}

	// Check if this is WAV audio (starts with "RIFF")
	if len(buffers[0]) > 4 && string(buffers[0][:4]) == "RIFF" {
		return concatenateWAV(buffers)
	}

	// For other formats (MP3, raw PCM), just concatenate
	var total int
	for _, buf := range buffers {
		total += len(buf)
	}

	result := make([]byte, 0, total)
	for _, buf := range buffers {
		result = append(result, buf...)
	}

	return bytes.NewReader(result)
}

// findDataChunkOffset finds the offset where the "data" chunk's audio data begins in a WAV file
// Returns the offset to the audio data (after "data" + size bytes) and the header to use
func findDataChunkOffset(buf []byte) int {
	// Minimum WAV: RIFF(4) + size(4) + WAVE(4) + fmt (4) + fmtSize(4) + fmtData(16+) + data(4) + dataSize(4)
	if len(buf) < 44 {
		return -1
	}

	// Verify RIFF header
	if string(buf[0:4]) != "RIFF" || string(buf[8:12]) != "WAVE" {
		return -1
	}

	// Search for "data" chunk starting after RIFF header (12 bytes)
	offset := 12
	for offset < len(buf)-8 {
		chunkID := string(buf[offset : offset+4])
		chunkSize := int(buf[offset+4]) | int(buf[offset+5])<<8 | int(buf[offset+6])<<16 | int(buf[offset+7])<<24

		if chunkID == "data" {
			// Return offset to actual audio data (after "data" + size)
			return offset + 8
		}

		// Move to next chunk (chunk header is 8 bytes + chunk data)
		offset += 8 + chunkSize
	}

	return -1
}

// concatenateWAV properly concatenates WAV files by handling headers
func concatenateWAV(buffers [][]byte) io.Reader {
	if len(buffers) == 0 {
		return bytes.NewReader(nil)
	}

	// Find data offset for first buffer to use as template
	firstDataOffset := findDataChunkOffset(buffers[0])
	if firstDataOffset < 0 {
		// Not a valid WAV, just concatenate
		var result []byte
		for _, buf := range buffers {
			result = append(result, buf...)
		}
		return bytes.NewReader(result)
	}

	// Calculate total data size by finding data offset in each buffer
	var totalDataSize int
	dataOffsets := make([]int, len(buffers))
	for i, buf := range buffers {
		dataOffsets[i] = findDataChunkOffset(buf)
		if dataOffsets[i] > 0 && len(buf) > dataOffsets[i] {
			totalDataSize += len(buf) - dataOffsets[i]
		}
	}

	// Create result buffer: use first file's header up to data chunk + all audio data
	result := make([]byte, firstDataOffset+totalDataSize)

	// Copy header from first file (everything up to and including "data" + size)
	copy(result[:firstDataOffset], buffers[0][:firstDataOffset])

	// Update RIFF file size (bytes 4-7): total size - 8
	fileSize := uint32(firstDataOffset + totalDataSize - 8)
	result[4] = byte(fileSize)
	result[5] = byte(fileSize >> 8)
	result[6] = byte(fileSize >> 16)
	result[7] = byte(fileSize >> 24)

	// Update data chunk size (4 bytes before firstDataOffset)
	dataSizeOffset := firstDataOffset - 4
	result[dataSizeOffset] = byte(totalDataSize)
	result[dataSizeOffset+1] = byte(totalDataSize >> 8)
	result[dataSizeOffset+2] = byte(totalDataSize >> 16)
	result[dataSizeOffset+3] = byte(totalDataSize >> 24)

	// Copy audio data from all buffers
	offset := firstDataOffset
	for i, buf := range buffers {
		if dataOffsets[i] > 0 && len(buf) > dataOffsets[i] {
			copy(result[offset:], buf[dataOffsets[i]:])
			offset += len(buf) - dataOffsets[i]
		}
	}

	return bytes.NewReader(result)
}
