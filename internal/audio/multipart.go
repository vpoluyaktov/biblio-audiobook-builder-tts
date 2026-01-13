package audio

import (
	"fmt"
	"os"
	"path/filepath"
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

// BuildMultiPartM4B builds multiple M4B files from parts
func BuildMultiPartM4B(parts []Part, outputDir string, baseFileName string, options M4BOptions) ([]string, error) {
	var m4bFiles []string

	for _, part := range parts {
		// Create part-specific options
		partOptions := options
		partOptions.Chapters = part.Chapters

		// Adjust title for multi-part
		if len(parts) > 1 {
			partOptions.Title = fmt.Sprintf("%s, Part %d", options.Title, part.Number)
			partOptions.Album = partOptions.Title
		}

		// Create builder
		builder, err := NewM4BBuilder(partOptions)
		if err != nil {
			return m4bFiles, fmt.Errorf("failed to create builder for part %d: %w", part.Number, err)
		}

		// Build M4B file
		var m4bFileName string
		if len(parts) > 1 {
			m4bFileName = fmt.Sprintf("%s, Part %d.m4b", baseFileName, part.Number)
		} else {
			m4bFileName = baseFileName + ".m4b"
		}
		m4bPath := filepath.Join(outputDir, m4bFileName)

		if err := builder.BuildFromFiles(part.ChapterFiles, m4bPath); err != nil {
			builder.Cleanup()
			return m4bFiles, fmt.Errorf("failed to build part %d: %w", part.Number, err)
		}

		builder.Cleanup()
		m4bFiles = append(m4bFiles, m4bPath)
	}

	return m4bFiles, nil
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
