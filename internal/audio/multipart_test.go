package audio

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestBuildMultiPartM4BParallel_EmptyParts(t *testing.T) {
	_, err := BuildMultiPartM4BParallel([]Part{}, "/tmp", "test", M4BOptions{}, 2)
	if err == nil {
		t.Error("Expected error for empty parts")
	}
}

func TestBuildMultiPartM4BParallel_WorkerLimits(t *testing.T) {
	// Test that worker count is properly limited
	parts := []Part{
		{Number: 1, ChapterFiles: []string{"/fake/file1.wav"}},
		{Number: 2, ChapterFiles: []string{"/fake/file2.wav"}},
	}

	// This will fail because files don't exist, but we're testing the worker limiting logic
	// The function should limit workers to number of parts (2)
	_, _ = BuildMultiPartM4BParallel(parts, "/tmp", "test", M4BOptions{}, 10)
	// No assertion needed - just verifying it doesn't panic with more workers than parts
}

func TestBuildMultiPartM4BParallel_ZeroWorkers(t *testing.T) {
	parts := []Part{
		{Number: 1, ChapterFiles: []string{"/fake/file1.wav"}},
	}

	// Zero workers should default to 1
	_, _ = BuildMultiPartM4BParallel(parts, "/tmp", "test", M4BOptions{}, 0)
	// No assertion needed - just verifying it doesn't panic
}

func TestBuildMultiPartM4BParallel_NegativeWorkers(t *testing.T) {
	parts := []Part{
		{Number: 1, ChapterFiles: []string{"/fake/file1.wav"}},
	}

	// Negative workers should default to 1
	_, _ = BuildMultiPartM4BParallel(parts, "/tmp", "test", M4BOptions{}, -5)
	// No assertion needed - just verifying it doesn't panic
}

func TestPartBuildResult(t *testing.T) {
	result := PartBuildResult{
		PartNumber: 1,
		FilePath:   "/path/to/file.m4b",
		Error:      nil,
	}

	if result.PartNumber != 1 {
		t.Errorf("Expected PartNumber 1, got %d", result.PartNumber)
	}
	if result.FilePath != "/path/to/file.m4b" {
		t.Errorf("Expected FilePath '/path/to/file.m4b', got '%s'", result.FilePath)
	}
	if result.Error != nil {
		t.Errorf("Expected nil error, got %v", result.Error)
	}
}

func TestBuildMultiPartM4B_CallsParallel(t *testing.T) {
	// Verify that BuildMultiPartM4B delegates to BuildMultiPartM4BParallel
	parts := []Part{
		{Number: 1, ChapterFiles: []string{"/fake/file1.wav"}},
	}

	// Both should fail the same way (file doesn't exist)
	_, err1 := BuildMultiPartM4B(parts, "/tmp", "test", M4BOptions{})
	_, err2 := BuildMultiPartM4BParallel(parts, "/tmp", "test", M4BOptions{}, 1)

	// Both should return errors (files don't exist)
	if (err1 == nil) != (err2 == nil) {
		t.Error("BuildMultiPartM4B and BuildMultiPartM4BParallel should behave the same")
	}
}

func TestSplitIntoParts_ChapterTitles(t *testing.T) {
	tmpDir := t.TempDir()
	files := createTestFiles(t, tmpDir, 3, 1024)

	// Test with matching titles
	titles := []string{"Chapter 1", "Chapter 2", "Chapter 3"}
	parts, err := SplitIntoParts(files, titles, 0)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	if len(parts[0].Chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(parts[0].Chapters))
	}

	for i, ch := range parts[0].Chapters {
		if ch.Title != titles[i] {
			t.Errorf("Chapter %d title = '%s', expected '%s'", i, ch.Title, titles[i])
		}
	}
}

func TestSplitIntoParts_FewerTitlesThanFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Create 5 files with 300KB each - will need splitting at 1MB
	files := createTestFiles(t, tmpDir, 5, 300*1024)

	// Fewer titles than files - function should use default titles for missing ones
	titles := []string{"Chapter 1", "Chapter 2"}
	// Use maxSizeMB=1 to trigger the splitting code path which handles default titles
	parts, err := SplitIntoParts(files, titles, 1)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	// Count total chapters across all parts
	totalChapters := 0
	for _, part := range parts {
		totalChapters += len(part.Chapters)
	}

	if totalChapters != 5 {
		t.Errorf("Expected 5 total chapters, got %d", totalChapters)
	}

	// Verify first part has correct titles
	if len(parts) > 0 && len(parts[0].Chapters) > 0 {
		if parts[0].Chapters[0].Title != "Chapter 1" {
			t.Errorf("Expected 'Chapter 1', got '%s'", parts[0].Chapters[0].Title)
		}
	}
}

func TestSplitIntoParts_TotalSize(t *testing.T) {
	tmpDir := t.TempDir()
	fileSize := 1024
	files := createTestFiles(t, tmpDir, 3, fileSize)

	titles := []string{"A", "B", "C"}
	// Use maxSizeMB=10 to trigger the code path that calculates TotalSize
	// (maxSizeMB=0 returns early without calculating sizes)
	parts, err := SplitIntoParts(files, titles, 10)
	if err != nil {
		t.Fatalf("SplitIntoParts failed: %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(parts))
	}

	expectedTotalSize := int64(3 * fileSize)
	if parts[0].TotalSize != expectedTotalSize {
		t.Errorf("Expected TotalSize %d, got %d", expectedTotalSize, parts[0].TotalSize)
	}
}

// TestBuildMultiPartM4B_MorePartsThanWorkers tests that building more parts than
// workers doesn't cause a deadlock. This is a regression test for a bug where
// defer inside a for loop caused encoder IDs to never be returned to the pool.
// Scenario: 12 parts with 5 workers should complete all 12 parts, not hang after 5.
func TestBuildMultiPartM4B_MorePartsThanWorkers(t *testing.T) {
	// Create 12 parts (more than 5 workers)
	numParts := 12
	numWorkers := 5
	parts := make([]Part, numParts)
	for i := 0; i < numParts; i++ {
		parts[i] = Part{
			Number:       i + 1,
			ChapterFiles: []string{"/fake/file.wav"}, // Will fail, but we're testing concurrency
		}
	}

	// Use a channel to signal completion or timeout
	done := make(chan bool, 1)

	go func() {
		// Call the parallel build function
		// Files don't exist so it will fail, but we're testing that it doesn't deadlock
		_, _ = BuildMultiPartM4BParallel(parts, t.TempDir(), "test", M4BOptions{}, numWorkers)
		done <- true
	}()

	// Wait for completion with timeout
	// If there's a deadlock, this will timeout
	select {
	case <-done:
		// Completed without deadlock - this is the expected behavior
		t.Log("Build completed without deadlock (errors expected due to fake files)")
	case <-time.After(10 * time.Second):
		t.Fatalf("DEADLOCK DETECTED: Build timed out after 10 seconds with %d parts and %d workers. "+
			"This indicates the encoder pool is not returning IDs correctly after each part.", numParts, numWorkers)
	}
}
