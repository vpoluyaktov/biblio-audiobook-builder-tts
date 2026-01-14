package tts

import (
	"abb_tts/internal/logger"
	"bytes"
	"fmt"
	"io"
)

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

		// Convert this chunk
		reader, err := a.provider.ConvertToSpeech(chunk, voice, options)
		if err != nil {
			return nil, fmt.Errorf("failed to convert chunk %d/%d: %w", i+1, len(chunks), err)
		}

		// Read the audio data
		audioData, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read audio for chunk %d/%d: %w", i+1, len(chunks), err)
		}

		audioBuffers = append(audioBuffers, audioData)
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

		reader, err := a.provider.ConvertToSpeech(chunk, voice, options)
		if err != nil {
			return nil, fmt.Errorf("failed to convert chunk %d/%d: %w", i+1, len(chunks), err)
		}

		readers = append(readers, reader)
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

// concatenateWAV properly concatenates WAV files by handling headers
func concatenateWAV(buffers [][]byte) io.Reader {
	if len(buffers) == 0 {
		return bytes.NewReader(nil)
	}

	// WAV file structure:
	// Bytes 0-3: "RIFF"
	// Bytes 4-7: File size - 8
	// Bytes 8-11: "WAVE"
	// Bytes 12-15: "fmt "
	// Bytes 16-19: fmt chunk size (16 for PCM)
	// Bytes 20-35: fmt data
	// Bytes 36-39: "data"
	// Bytes 40-43: data size
	// Bytes 44+: audio data

	const headerSize = 44

	// Use the first file's header as template
	if len(buffers[0]) < headerSize {
		// Not a valid WAV, just concatenate
		var result []byte
		for _, buf := range buffers {
			result = append(result, buf...)
		}
		return bytes.NewReader(result)
	}

	// Calculate total data size
	var totalDataSize int
	for _, buf := range buffers {
		if len(buf) > headerSize {
			totalDataSize += len(buf) - headerSize
		}
	}

	// Create result buffer
	result := make([]byte, headerSize+totalDataSize)

	// Copy header from first file
	copy(result[:headerSize], buffers[0][:headerSize])

	// Update file size (bytes 4-7): total size - 8
	fileSize := uint32(headerSize + totalDataSize - 8)
	result[4] = byte(fileSize)
	result[5] = byte(fileSize >> 8)
	result[6] = byte(fileSize >> 16)
	result[7] = byte(fileSize >> 24)

	// Update data size (bytes 40-43)
	dataSize := uint32(totalDataSize)
	result[40] = byte(dataSize)
	result[41] = byte(dataSize >> 8)
	result[42] = byte(dataSize >> 16)
	result[43] = byte(dataSize >> 24)

	// Copy audio data from all buffers
	offset := headerSize
	for _, buf := range buffers {
		if len(buf) > headerSize {
			copy(result[offset:], buf[headerSize:])
			offset += len(buf) - headerSize
		}
	}

	return bytes.NewReader(result)
}
