package audio

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestEscapeMetadata(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Title", "Simple Title"},
		{"Title=With=Equals", "Title\\=With\\=Equals"},
		{"Title;With;Semicolons", "Title\\;With\\;Semicolons"},
		{"Title#With#Hash", "Title\\#With\\#Hash"},
		{"Title\\With\\Backslash", "Title\\\\With\\\\Backslash"},
		{"Title\nWith\nNewlines", "Title With Newlines"},
		{"Complex=Title;With#All\\Chars", "Complex\\=Title\\;With\\#All\\\\Chars"},
	}

	for _, tt := range tests {
		result := escapeMetadata(tt.input)
		if result != tt.expected {
			t.Errorf("escapeMetadata(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewM4BBuilder(t *testing.T) {
	options := M4BOptions{
		Title:  "Test Book",
		Author: "Test Author",
	}

	builder, err := NewM4BBuilder(options)
	if err != nil {
		t.Fatalf("NewM4BBuilder failed: %v", err)
	}
	defer builder.Cleanup()

	if builder.tempDir == "" {
		t.Error("tempDir should not be empty")
	}

	// Check temp dir exists
	if _, err := os.Stat(builder.tempDir); os.IsNotExist(err) {
		t.Error("tempDir should exist")
	}
}

func TestM4BBuilderCleanup(t *testing.T) {
	options := M4BOptions{}
	builder, err := NewM4BBuilder(options)
	if err != nil {
		t.Fatalf("NewM4BBuilder failed: %v", err)
	}

	tempDir := builder.tempDir
	builder.Cleanup()

	// Check temp dir is removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("tempDir should be removed after Cleanup")
	}
}

func TestCreateMetadataFile(t *testing.T) {
	options := M4BOptions{
		Title:       "Test Book",
		Author:      "Test Author",
		Album:       "Test Album",
		Genre:       "Fiction",
		Year:        "2024",
		Description: "A test description",
		Chapters: []Chapter{
			{Title: "Chapter 1", StartTime: 0, EndTime: 60 * time.Second},
			{Title: "Chapter 2", StartTime: 60 * time.Second, EndTime: 120 * time.Second},
		},
	}

	builder, err := NewM4BBuilder(options)
	if err != nil {
		t.Fatalf("NewM4BBuilder failed: %v", err)
	}
	defer builder.Cleanup()

	metadataFile := builder.tempDir + "/metadata.txt"
	err = builder.createMetadataFile(metadataFile)
	if err != nil {
		t.Fatalf("createMetadataFile failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(metadataFile)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	expectedStrings := []string{
		";FFMETADATA1",
		"title=Test Book",
		"artist=Test Author",
		"album=Test Album",
		"genre=Fiction",
		"date=2024",
		"[CHAPTER]",
		"title=Chapter 1",
		"title=Chapter 2",
		"START=0",
		"END=60000",
		"START=60000",
		"END=120000",
	}

	for _, expected := range expectedStrings {
		if !containsString(contentStr, expected) {
			t.Errorf("Metadata file should contain %q", expected)
		}
	}
}

func TestCreateConcatFile(t *testing.T) {
	options := M4BOptions{}
	builder, err := NewM4BBuilder(options)
	if err != nil {
		t.Fatalf("NewM4BBuilder failed: %v", err)
	}
	defer builder.Cleanup()

	audioFiles := []string{
		"/path/to/chapter1.wav",
		"/path/to/chapter2.wav",
		"/path/to/file with spaces.wav",
		"/path/to/За секунду до взрыва/chapter1.wav",
		"/path/to/file's quote.wav",
	}

	concatFile := builder.tempDir + "/concat.txt"
	err = builder.createConcatFile(audioFiles, concatFile)
	if err != nil {
		t.Fatalf("createConcatFile failed: %v", err)
	}

	content, err := os.ReadFile(concatFile)
	if err != nil {
		t.Fatalf("Failed to read concat file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	if !containsString(contentStr, "file '/path/to/chapter1.wav'") {
		t.Error("Concat file should contain chapter1.wav")
	}
	if !containsString(contentStr, "file '/path/to/chapter2.wav'") {
		t.Error("Concat file should contain chapter2.wav")
	}
	if !containsString(contentStr, "file '/path/to/file with spaces.wav'") {
		t.Error("Concat file should contain file with spaces")
	}
	// Cyrillic characters should be preserved as-is
	if !containsString(contentStr, "file '/path/to/За секунду до взрыва/chapter1.wav'") {
		t.Error("Concat file should contain Cyrillic path")
	}
	// Single quotes should be escaped with backslash
	if !containsString(contentStr, "file '/path/to/file\\'s quote.wav'") {
		t.Error("Concat file should escape single quotes with backslash")
	}
}

func TestCheckFFmpegAvailable(t *testing.T) {
	// This test depends on system having ffmpeg installed
	err := CheckFFmpegAvailable()

	// Check if ffmpeg is available
	_, ffmpegErr := exec.LookPath("ffmpeg")
	_, ffprobeErr := exec.LookPath("ffprobe")

	if ffmpegErr == nil && ffprobeErr == nil {
		// ffmpeg should be available
		if err != nil {
			t.Errorf("CheckFFmpegAvailable should succeed when ffmpeg is installed: %v", err)
		}
	} else {
		// ffmpeg not available, error expected
		if err == nil {
			t.Error("CheckFFmpegAvailable should fail when ffmpeg is not installed")
		}
	}
}

func TestBuildChapterList(t *testing.T) {
	options := M4BOptions{
		Chapters: []Chapter{
			{Title: "Intro"},
			{Title: "Main Story"},
			{Title: "Epilogue"},
		},
		GapBetweenChaps: 2 * time.Second,
	}

	builder, err := NewM4BBuilder(options)
	if err != nil {
		t.Fatalf("NewM4BBuilder failed: %v", err)
	}
	defer builder.Cleanup()

	// We can't test buildChapterList directly without audio files
	// but we can verify the options are set correctly
	if builder.options.GapBetweenChaps != 2*time.Second {
		t.Errorf("GapBetweenChaps should be 2s, got %v", builder.options.GapBetweenChaps)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
