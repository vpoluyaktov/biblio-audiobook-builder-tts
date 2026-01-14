package audio

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Chapter represents a chapter in the audiobook
type Chapter struct {
	Title     string
	StartTime time.Duration
	EndTime   time.Duration
}

// M4BOptions contains options for M4B creation
type M4BOptions struct {
	Title           string
	Author          string
	Album           string
	Genre           string
	Year            string
	Description     string
	CoverImage      []byte
	CoverImageType  string // "image/jpeg" or "image/png"
	Chapters        []Chapter
	BitRate         string // e.g., "128k"
	SampleRate      int    // e.g., 44100
	GapBetweenChaps time.Duration
}

// M4BBuilder builds M4B audiobook files from audio segments
type M4BBuilder struct {
	tempDir string
	options M4BOptions
}

// NewM4BBuilder creates a new M4B builder
func NewM4BBuilder(options M4BOptions) (*M4BBuilder, error) {
	tempDir, err := ioutil.TempDir("", "m4b_build_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	if options.BitRate == "" {
		options.BitRate = "128k"
	}
	if options.SampleRate == 0 {
		options.SampleRate = 44100
	}

	return &M4BBuilder{
		tempDir: tempDir,
		options: options,
	}, nil
}

// Cleanup removes temporary files
func (b *M4BBuilder) Cleanup() {
	os.RemoveAll(b.tempDir)
}

// BuildFromFiles creates an M4B file from a list of audio files
// audioFiles should be in order, one per chapter
func (b *M4BBuilder) BuildFromFiles(audioFiles []string, outputPath string) error {
	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files provided")
	}

	// Step 1: Get duration of each audio file and build chapter list
	chapters, err := b.buildChapterList(audioFiles)
	if err != nil {
		return fmt.Errorf("failed to build chapter list: %w", err)
	}
	b.options.Chapters = chapters

	// Step 2: Create concat file list
	concatFile := filepath.Join(b.tempDir, "concat.txt")
	if err := b.createConcatFile(audioFiles, concatFile); err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}

	// Step 3: Create FFMETADATA file with chapters
	metadataFile := filepath.Join(b.tempDir, "metadata.txt")
	if err := b.createMetadataFile(metadataFile); err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}

	// Step 4: Save cover image if provided
	var coverPath string
	if len(b.options.CoverImage) > 0 {
		ext := ".jpg"
		if b.options.CoverImageType == "image/png" {
			ext = ".png"
		}
		coverPath = filepath.Join(b.tempDir, "cover"+ext)
		if err := ioutil.WriteFile(coverPath, b.options.CoverImage, 0644); err != nil {
			return fmt.Errorf("failed to write cover image: %w", err)
		}
	}

	// Step 5: Build the M4B file using ffmpeg
	if err := b.runFFmpeg(concatFile, metadataFile, coverPath, outputPath); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	return nil
}

// buildChapterList gets durations and builds chapter metadata
func (b *M4BBuilder) buildChapterList(audioFiles []string) ([]Chapter, error) {
	chapters := make([]Chapter, len(audioFiles))
	var currentTime time.Duration

	for i, file := range audioFiles {
		duration, err := getAudioDuration(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get duration for %s: %w", file, err)
		}

		title := fmt.Sprintf("Chapter %d", i+1)
		if i < len(b.options.Chapters) && b.options.Chapters[i].Title != "" {
			title = b.options.Chapters[i].Title
		}

		chapters[i] = Chapter{
			Title:     title,
			StartTime: currentTime,
			EndTime:   currentTime + duration,
		}

		currentTime += duration
		if b.options.GapBetweenChaps > 0 && i < len(audioFiles)-1 {
			currentTime += b.options.GapBetweenChaps
		}
	}

	return chapters, nil
}

// createConcatFile creates the ffmpeg concat demuxer file
func (b *M4BBuilder) createConcatFile(audioFiles []string, outputPath string) error {
	var lines []string
	for _, file := range audioFiles {
		// FFmpeg concat demuxer requires escaping of special characters:
		// - backslash must be escaped first (\ -> \\)
		// - single quote must be escaped (' -> \')
		escaped := strings.ReplaceAll(file, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "'", "\\'")
		lines = append(lines, fmt.Sprintf("file '%s'", escaped))
	}
	content := strings.Join(lines, "\n")
	return ioutil.WriteFile(outputPath, []byte(content), 0644)
}

// createMetadataFile creates the FFMETADATA file with chapters
func (b *M4BBuilder) createMetadataFile(outputPath string) error {
	var sb strings.Builder

	sb.WriteString(";FFMETADATA1\n")

	// Global metadata
	if b.options.Title != "" {
		sb.WriteString(fmt.Sprintf("title=%s\n", escapeMetadata(b.options.Title)))
	}
	if b.options.Author != "" {
		sb.WriteString(fmt.Sprintf("artist=%s\n", escapeMetadata(b.options.Author)))
		sb.WriteString(fmt.Sprintf("album_artist=%s\n", escapeMetadata(b.options.Author)))
	}
	if b.options.Album != "" {
		sb.WriteString(fmt.Sprintf("album=%s\n", escapeMetadata(b.options.Album)))
	} else if b.options.Title != "" {
		sb.WriteString(fmt.Sprintf("album=%s\n", escapeMetadata(b.options.Title)))
	}
	if b.options.Genre != "" {
		sb.WriteString(fmt.Sprintf("genre=%s\n", escapeMetadata(b.options.Genre)))
	} else {
		sb.WriteString("genre=Audiobook\n")
	}
	if b.options.Year != "" {
		sb.WriteString(fmt.Sprintf("date=%s\n", b.options.Year))
	}
	if b.options.Description != "" {
		sb.WriteString(fmt.Sprintf("description=%s\n", escapeMetadata(b.options.Description)))
		sb.WriteString(fmt.Sprintf("comment=%s\n", escapeMetadata(b.options.Description)))
	}

	// Chapter markers
	for _, ch := range b.options.Chapters {
		sb.WriteString("\n[CHAPTER]\n")
		sb.WriteString("TIMEBASE=1/1000\n")
		sb.WriteString(fmt.Sprintf("START=%d\n", ch.StartTime.Milliseconds()))
		sb.WriteString(fmt.Sprintf("END=%d\n", ch.EndTime.Milliseconds()))
		sb.WriteString(fmt.Sprintf("title=%s\n", escapeMetadata(ch.Title)))
	}

	return ioutil.WriteFile(outputPath, []byte(sb.String()), 0644)
}

// runFFmpeg executes ffmpeg to create the M4B file
func (b *M4BBuilder) runFFmpeg(concatFile, metadataFile, coverPath, outputPath string) error {
	args := []string{
		"-y", // Overwrite output
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		"-i", metadataFile,
	}

	// Add cover image if available
	if coverPath != "" {
		args = append(args, "-i", coverPath)
	}

	// Map streams
	args = append(args,
		"-map", "0:a", // Audio from concat
		"-map_metadata", "1", // Metadata from file
	)

	// Map cover image
	if coverPath != "" {
		args = append(args,
			"-map", "2:v", // Video (cover) from image
			"-c:v", "copy",
			"-disposition:v:0", "attached_pic",
		)
	}

	// Audio encoding settings
	args = append(args,
		"-c:a", "aac",
		"-b:a", b.options.BitRate,
		"-ar", fmt.Sprintf("%d", b.options.SampleRate),
		"-ac", "2", // Stereo
	)

	// Output format
	args = append(args,
		"-f", "mp4",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// getAudioDuration uses ffprobe to get the duration of an audio file
func getAudioDuration(filePath string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe error: %w", err)
	}

	var seconds float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%f", &seconds)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}

// escapeMetadata escapes special characters for FFMETADATA format
func escapeMetadata(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "=", "\\=")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// CheckFFmpegAvailable checks if ffmpeg and ffprobe are available
func CheckFFmpegAvailable() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe not found in PATH")
	}
	return nil
}

// GenerateSilence creates a silent audio file of the specified duration
func GenerateSilence(duration time.Duration, outputPath string, sampleRate int) error {
	if sampleRate == 0 {
		sampleRate = 44100
	}

	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=%d:cl=stereo", sampleRate),
		"-t", fmt.Sprintf("%f", duration.Seconds()),
		"-c:a", "aac",
		"-b:a", "128k",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate silence: %w\nOutput: %s", err, string(output))
	}

	return nil
}
