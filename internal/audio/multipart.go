package audio

import (
	"fmt"
	"os"
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
func SplitIntoParts(chapterFiles []string, chapterTitles []string, maxSizeMB int) ([]Part, error) {
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

	maxSizeBytes := int64(maxSizeMB) * 1024 * 1024
	var parts []Part
	var currentPart Part
	currentPart.Number = 1
	var currentSize int64

	for i, file := range chapterFiles {
		// Get file size
		info, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", file, err)
		}
		fileSize := info.Size()

		// Check if adding this file would exceed max size
		if currentSize+fileSize > maxSizeBytes && len(currentPart.ChapterFiles) > 0 {
			// Save current part and start new one
			parts = append(parts, currentPart)
			currentPart = Part{
				Number: len(parts) + 1,
			}
			currentSize = 0
		}

		// Add file to current part
		currentPart.ChapterFiles = append(currentPart.ChapterFiles, file)
		title := fmt.Sprintf("Chapter %d", i+1)
		if i < len(chapterTitles) {
			title = chapterTitles[i]
		}
		currentPart.Chapters = append(currentPart.Chapters, Chapter{Title: title})
		currentPart.TotalSize += fileSize
		currentSize += fileSize
	}

	// Add final part
	if len(currentPart.ChapterFiles) > 0 {
		parts = append(parts, currentPart)
	}

	return parts, nil
}

// BuildMultiPartM4B builds multiple M4B files from parts (sequential version)
// Deprecated: Use BuildMultiPartM4BParallel for better performance
func BuildMultiPartM4B(parts []Part, outputDir string, baseFileName string, options M4BOptions) ([]string, error) {
	return BuildMultiPartM4BParallel(parts, outputDir, baseFileName, options, 1)
}

// PartBuildResult holds the result of building a single M4B part
type PartBuildResult struct {
	PartNumber int
	FilePath   string
	Error      error
}

// BuildMultiPartM4BParallel builds multiple M4B files from parts using parallel workers
func BuildMultiPartM4BParallel(parts []Part, outputDir string, baseFileName string, options M4BOptions, numWorkers int) ([]string, error) {
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
		go func() {
			defer wg.Done()
			for partIdx := range partsChan {
				part := parts[partIdx]
				result := buildSinglePart(part, parts, outputDir, baseFileName, options)
				results[partIdx] = result
			}
		}()
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

// buildSinglePart builds a single M4B part
func buildSinglePart(part Part, allParts []Part, outputDir string, baseFileName string, options M4BOptions) PartBuildResult {
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
