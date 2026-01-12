package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitIntoParts_NoSplitting(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	files := createTestFiles(t, tmpDir, 3, 1024) // 3 files, 1KB each

	titles := []string{"Chapter 1", "Chapter 2", "Chapter 3"}

	// No splitting (maxSize = 0)
	parts, err := SplitIntoParts(files, titles, 0)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	if len(parts) != 1 {
		t.Errorf("Expected 1 part, got %d", len(parts))
	}

	if len(parts[0].ChapterFiles) != 3 {
		t.Errorf("Expected 3 chapter files, got %d", len(parts[0].ChapterFiles))
	}
}

func TestSplitIntoParts_WithSplitting(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	// Create 5 files of 500KB each = 2.5MB total
	files := createTestFiles(t, tmpDir, 5, 500*1024)

	titles := []string{"Ch 1", "Ch 2", "Ch 3", "Ch 4", "Ch 5"}

	// Split at 1MB - should create 3 parts (2 files per part, last part has 1)
	parts, err := SplitIntoParts(files, titles, 1)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	if len(parts) < 2 {
		t.Errorf("Expected at least 2 parts, got %d", len(parts))
	}

	// Verify all files are accounted for
	totalFiles := 0
	for _, part := range parts {
		totalFiles += len(part.ChapterFiles)
	}
	if totalFiles != 5 {
		t.Errorf("Expected 5 total files, got %d", totalFiles)
	}
}

func TestSplitIntoParts_SingleLargeFile(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	// Create 1 file larger than max size
	files := createTestFiles(t, tmpDir, 1, 2*1024*1024) // 2MB file

	titles := []string{"Big Chapter"}

	// Max size 1MB - file is larger but should still be in one part
	parts, err := SplitIntoParts(files, titles, 1)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	if len(parts) != 1 {
		t.Errorf("Expected 1 part (single large file), got %d", len(parts))
	}
}

func TestSplitIntoParts_PartNumbering(t *testing.T) {
	tmpDir := t.TempDir()
	files := createTestFiles(t, tmpDir, 4, 300*1024) // 4 files, 300KB each

	titles := []string{"A", "B", "C", "D"}

	// Split at 500KB - should create 2-3 parts
	parts, err := SplitIntoParts(files, titles, 1) // 1MB max
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	// Verify part numbering
	for i, part := range parts {
		if part.Number != i+1 {
			t.Errorf("Part %d has wrong number: %d", i, part.Number)
		}
	}
}

func TestEstimateM4BSize(t *testing.T) {
	tests := []struct {
		durationSec float64
		bitRateKbps int
		minExpected int64
		maxExpected int64
	}{
		{60, 128, 900000, 1100000},      // 1 minute at 128kbps ~= 960KB
		{3600, 128, 54000000, 65000000}, // 1 hour at 128kbps ~= 57.6MB + overhead
		{0, 128, 0, 100},                // 0 duration
	}

	for _, tt := range tests {
		result := EstimateM4BSize(tt.durationSec, tt.bitRateKbps)
		if result < tt.minExpected || result > tt.maxExpected {
			t.Errorf("EstimateM4BSize(%f, %d) = %d, want between %d and %d",
				tt.durationSec, tt.bitRateKbps, result, tt.minExpected, tt.maxExpected)
		}
	}
}

func TestSplitIntoParts_EmptyInput(t *testing.T) {
	parts, err := SplitIntoParts([]string{}, []string{}, 100)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	if len(parts) != 0 {
		t.Errorf("Expected 0 parts for empty input, got %d", len(parts))
	}
}

func TestSplitIntoParts_NonexistentFile(t *testing.T) {
	files := []string{"/nonexistent/file.wav"}
	titles := []string{"Chapter 1"}

	_, err := SplitIntoParts(files, titles, 100)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// Helper function to create test files
func createTestFiles(t *testing.T, dir string, count int, sizeBytes int) []string {
	var files []string
	for i := 0; i < count; i++ {
		path := filepath.Join(dir, "chapter_"+string(rune('a'+i))+".wav")
		data := make([]byte, sizeBytes)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		files = append(files, path)
	}
	return files
}
