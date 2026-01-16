package audio

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// Part represents a part of a multi-part audiobook
type Part struct {
	Number       int
	ChapterFiles []string
	Chapters     []Chapter
	TotalSize    int64
	Duration     time.Duration
}

// SplitIntoParts splits chapter files into parts based on max file size
// Returns a slice of Parts, each containing chapter files that fit within maxSizeMB
// Deprecated: Use SplitIntoPartsByEstimatedSize for accurate M4B size estimation
func SplitIntoParts(chapterFiles []string, chapterTitles []string, maxSizeMB int) ([]Part, error) {
	// Use default bitrate of 128kbps for backward compatibility
	return SplitIntoPartsByEstimatedSize(chapterFiles, chapterTitles, maxSizeMB, 128)
}

// SplitIntoPartsByEstimatedSize splits chapter files into parts based on estimated M4B output size
// It uses audio duration and bitrate to estimate the compressed M4B size, not the WAV file size
// Returns a slice of Parts, each containing chapter files that fit within maxSizeMB
func SplitIntoPartsByEstimatedSize(chapterFiles []string, chapterTitles []string, maxSizeMB int, bitRateKbps int) ([]Part, error) {
	if maxSizeMB <= 0 {
		// No splitting, return single part with all chapters
		chapters := make([]Chapter, len(chapterTitles))
		for i, title := range chapterTitles {
			chapters[i] = Chapter{Title: title}
		}
		return []Part{{
			Number:       1,
			ChapterFiles: chapterFiles,
			Chapters:     chapters,
		}}, nil
	}

	if bitRateKbps <= 0 {
		bitRateKbps = 128 // Default to 128kbps
	}

	maxSizeBytes := int64(maxSizeMB) * 1024 * 1024
	var parts []Part
	var currentPart Part
	currentPart.Number = 1
	var currentEstimatedSize int64

	for i, file := range chapterFiles {
		// Get audio duration and estimate M4B size
		duration, err := getAudioDuration(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get duration for %s: %w", file, err)
		}

		// Estimate M4B size based on duration and bitrate
		estimatedSize := EstimateM4BSize(duration.Seconds(), bitRateKbps)

		// Check if adding this file would exceed max size
		if currentEstimatedSize+estimatedSize > maxSizeBytes && len(currentPart.ChapterFiles) > 0 {
			// Save current part and start new one
			parts = append(parts, currentPart)
			currentPart = Part{
				Number: len(parts) + 1,
			}
			currentEstimatedSize = 0
		}

		// Add file to current part
		currentPart.ChapterFiles = append(currentPart.ChapterFiles, file)
		title := fmt.Sprintf("Chapter %d", i+1)
		if i < len(chapterTitles) {
			title = chapterTitles[i]
		}
		currentPart.Chapters = append(currentPart.Chapters, Chapter{Title: title})
		currentPart.TotalSize += estimatedSize
		currentPart.Duration += duration
		currentEstimatedSize += estimatedSize
	}

	// Add final part
	if len(currentPart.ChapterFiles) > 0 {
		parts = append(parts, currentPart)
	}

	return parts, nil
}

// M4BProgressCallback is called with progress updates during M4B building
// partNum is 1-indexed, progress is 0.0 to 1.0
type M4BProgressCallback func(partNum int, totalParts int, progress float64)

// EncoderProgressCallback is called with per-encoder progress updates
// encoderID is 0-indexed, partNum is 1-indexed, progress is 0.0 to 1.0
type EncoderProgressCallback func(encoderID int, partNum int, totalParts int, progress float64)

// BuildMultiPartM4B builds multiple M4B files from parts (sequential version)
// Deprecated: Use BuildMultiPartM4BParallel for better performance
func BuildMultiPartM4B(parts []Part, outputDir string, baseFileName string, options M4BOptions) ([]string, error) {
	return BuildMultiPartM4BParallel(parts, outputDir, baseFileName, options, 1)
}

// BuildMultiPartM4BWithProgress builds M4B files with progress callback
func BuildMultiPartM4BWithProgress(parts []Part, outputDir string, baseFileName string, options M4BOptions, numWorkers int, progressCb M4BProgressCallback) ([]string, error) {
	return buildMultiPartM4BInternal(parts, outputDir, baseFileName, options, numWorkers, nil, progressCb)
}

// BuildMultiPartM4BWithEncoderProgress builds M4B files with per-encoder progress callback
func BuildMultiPartM4BWithEncoderProgress(parts []Part, outputDir string, baseFileName string, options M4BOptions, numWorkers int, encoderCb EncoderProgressCallback) ([]string, error) {
	return buildMultiPartM4BInternal(parts, outputDir, baseFileName, options, numWorkers, encoderCb, nil)
}

// PartBuildResult holds the result of building a single M4B part
type PartBuildResult struct {
	PartNumber int
	FilePath   string
	Error      error
}

// BuildMultiPartM4BParallel builds multiple M4B files from parts using parallel workers
func BuildMultiPartM4BParallel(parts []Part, outputDir string, baseFileName string, options M4BOptions, numWorkers int) ([]string, error) {
	return buildMultiPartM4BInternal(parts, outputDir, baseFileName, options, numWorkers, nil, nil)
}

// buildMultiPartM4BInternal is the internal implementation with optional progress callbacks
func buildMultiPartM4BInternal(parts []Part, outputDir string, baseFileName string, options M4BOptions, numWorkers int, encoderCb EncoderProgressCallback, progressCb M4BProgressCallback) ([]string, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("no parts to build")
	}

	// Limit workers to number of parts
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if numWorkers > len(parts) {
		numWorkers = len(parts)
	}

	// Results slice
	results := make([]PartBuildResult, len(parts))

	// Channel for distributing work
	partsChan := make(chan int, len(parts))
	for i := range parts {
		partsChan <- i
	}
	close(partsChan)

	// WaitGroup for workers
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for partIdx := range partsChan {
				part := parts[partIdx]
				result := buildSinglePartWithEncoderProgress(part, parts, outputDir, baseFileName, options, workerID, encoderCb, progressCb)
				results[partIdx] = result
			}
		}(w)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Collect results in order
	var m4bFiles []string
	for i, result := range results {
		if result.Error != nil {
			return m4bFiles, fmt.Errorf("failed to build part %d: %w", i+1, result.Error)
		}
		m4bFiles = append(m4bFiles, result.FilePath)
	}

	return m4bFiles, nil
}

// buildSinglePartWithEncoderProgress builds a single M4B part with optional encoder and progress callbacks
func buildSinglePartWithEncoderProgress(part Part, allParts []Part, outputDir string, baseFileName string, options M4BOptions, encoderID int, encoderCb EncoderProgressCallback, progressCb M4BProgressCallback) PartBuildResult {
	result := PartBuildResult{PartNumber: part.Number}

	// Create part-specific options
	partOptions := options
	partOptions.Chapters = part.Chapters

	// Adjust title for multi-part
	if len(allParts) > 1 {
		partOptions.Title = fmt.Sprintf("%s, Part %d", options.Title, part.Number)
		partOptions.Album = partOptions.Title
	}

	// Create builder
	builder, err := NewM4BBuilder(partOptions)
	if err != nil {
		result.Error = fmt.Errorf("failed to create builder: %w", err)
		return result
	}
	defer builder.Cleanup()

	// Set progress callback if provided
	if encoderCb != nil {
		builder.SetProgressCallback(func(progress float64) {
			encoderCb(encoderID, part.Number, len(allParts), progress)
		})
	} else if progressCb != nil {
		builder.SetProgressCallback(func(progress float64) {
			progressCb(part.Number, len(allParts), progress)
		})
	}

	// Build M4B file
	var m4bFileName string
	if len(allParts) > 1 {
		m4bFileName = fmt.Sprintf("%s, Part %d.m4b", baseFileName, part.Number)
	} else {
		m4bFileName = baseFileName + ".m4b"
	}
	m4bPath := filepath.Join(outputDir, m4bFileName)

	if err := builder.BuildFromFiles(part.ChapterFiles, m4bPath); err != nil {
		result.Error = err
		return result
	}

	result.FilePath = m4bPath
	return result
}

// EstimateM4BSize estimates the output M4B file size based on audio duration and bitrate
// Returns size in bytes
func EstimateM4BSize(durationSeconds float64, bitRateKbps int) int64 {
	// Size = duration * bitrate / 8 (convert bits to bytes)
	// Add 5% overhead for container format
	bytesPerSecond := float64(bitRateKbps) * 1000 / 8
	estimatedSize := durationSeconds * bytesPerSecond * 1.05
	return int64(estimatedSize)
}
