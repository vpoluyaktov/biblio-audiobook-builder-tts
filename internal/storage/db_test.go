package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestConfigOperations(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test SetConfig and GetConfig
	err = db.SetConfig("test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	value, err := db.GetConfig("test_key")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	// Test update
	err = db.SetConfig("test_key", "updated_value")
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	value, err = db.GetConfig("test_key")
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}
	if value != "updated_value" {
		t.Errorf("Expected 'updated_value', got '%s'", value)
	}
}

func TestGetAllConfig(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Get default config
	cfg, err := db.GetAllConfig()
	if err != nil {
		t.Fatalf("Failed to get all config: %v", err)
	}

	// Verify defaults
	if cfg.ServerPort != "8080" {
		t.Errorf("Expected default server_port '8080', got '%s'", cfg.ServerPort)
	}
}

func TestSaveAllConfig(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create custom config
	cfg := &Config{
		LogFile:           "custom.log",
		TempDir:           "/custom/temp",
		DefaultVoice:      "en-GB",
		DefaultProvider:   "piper",
		ServerPort:        "9090",
		ServerHost:        "127.0.0.1",
		OpenBrowser:       false,
		DefaultSpeed:      1.5,
		DefaultPitch:      0.8,
		ChapterGapSeconds: 5,
		MaxFileSizeMB:     1000,
	}

	err = db.SaveAllConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Retrieve and verify
	loaded, err := db.GetAllConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.ServerPort != "9090" {
		t.Errorf("Expected server_port '9090', got '%s'", loaded.ServerPort)
	}
	if loaded.OpenBrowser != false {
		t.Error("Expected open_browser false")
	}
}

func TestJobOperations(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create job
	job := &Job{
		ID:        "test-job-123",
		Status:    "pending",
		FileName:  "test.epub",
		FilePath:  "/tmp/test.epub",
		Provider:  "espeak",
		Voice:     "en-US",
		Speed:     1.0,
		Pitch:     1.0,
		CreatedAt: time.Now(),
	}

	err = db.CreateJob(job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Get job
	loaded, err := db.GetJob("test-job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if loaded == nil {
		t.Fatal("Job not found")
	}
	if loaded.FileName != "test.epub" {
		t.Errorf("Expected file_name 'test.epub', got '%s'", loaded.FileName)
	}

	// Update job
	job.Status = "completed"
	job.BookTitle = "Test Book"
	now := time.Now()
	job.CompletedAt = &now

	err = db.UpdateJob(job)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	loaded, _ = db.GetJob("test-job-123")
	if loaded.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", loaded.Status)
	}
	if loaded.BookTitle != "Test Book" {
		t.Errorf("Expected book_title 'Test Book', got '%s'", loaded.BookTitle)
	}
}

func TestListJobs(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		job := &Job{
			ID:        "job-" + string(rune('a'+i)),
			Status:    "pending",
			FileName:  "test.epub",
			CreatedAt: time.Now(),
		}
		db.CreateJob(job)
	}

	// List all
	jobs, err := db.ListJobs("", 0)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}

	// List with limit
	jobs, err = db.ListJobs("", 3)
	if err != nil {
		t.Fatalf("Failed to list jobs with limit: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}

	// List by status
	db.UpdateJob(&Job{ID: "job-a", Status: "completed"})
	jobs, err = db.ListJobs("completed", 0)
	if err != nil {
		t.Fatalf("Failed to list jobs by status: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Expected 1 completed job, got %d", len(jobs))
	}
}

func TestDeleteJob(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	job := &Job{
		ID:        "delete-me",
		Status:    "pending",
		FileName:  "test.epub",
		CreatedAt: time.Now(),
	}
	db.CreateJob(job)

	err = db.DeleteJob("delete-me")
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	loaded, _ := db.GetJob("delete-me")
	if loaded != nil {
		t.Error("Job should have been deleted")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServerPort != "8080" {
		t.Errorf("Expected default server_port '8080', got '%s'", cfg.ServerPort)
	}
	if cfg.MaxFileSizeMB != 250 {
		t.Errorf("Expected default max_file_size_mb 250, got %d", cfg.MaxFileSizeMB)
	}
}
