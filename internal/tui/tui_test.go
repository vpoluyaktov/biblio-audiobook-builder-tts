package tui

import (
	"testing"
	"time"

	"biblio-audiobook-builder-tts/internal/storage"
)

// mockJobDB is a mock implementation of JobDB for testing
type mockJobDB struct{}

func (m *mockJobDB) ListJobs(status string, limit int) ([]*storage.Job, error) {
	return []*storage.Job{}, nil
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		progress float64
		width    int
		expected string
	}{
		{0.0, 8, "░░░░░░░░"},
		{0.5, 8, "████░░░░"},
		{1.0, 8, "████████"},
		{0.25, 8, "██░░░░░░"},
		{0.75, 8, "██████░░"},
		{0.0, 4, "░░░░"},
		{1.0, 4, "████"},
	}

	for _, tt := range tests {
		result := renderProgressBar(tt.progress, tt.width)
		if result != tt.expected {
			t.Errorf("renderProgressBar(%f, %d) = %q, want %q", tt.progress, tt.width, result, tt.expected)
		}
	}
}

func TestGetStatusIconFromString(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"pending", "⏳"},
		{"parsing", "📖"},
		{"converting", "🔄"},
		{"building", "📦"},
		{"uploading", "☁️"},
		{"completed", "✅"},
		{"failed", "❌"},
		{"cancelled", "🚫"},
	}

	for _, tt := range tests {
		result := getStatusIconFromString(tt.status)
		if result != tt.expected {
			t.Errorf("getStatusIconFromString(%s) = %q, want %q", tt.status, result, tt.expected)
		}
	}
}

func TestCalculateHeights(t *testing.T) {
	tests := []struct {
		terminalHeight int
		minTop         int
		minBottom      int
	}{
		{30, 5, 5},
		{50, 10, 10},
		{80, 20, 20},
	}

	for _, tt := range tests {
		top, bottom := calculateHeights(tt.terminalHeight)
		if top < tt.minTop {
			t.Errorf("calculateHeights(%d) top = %d, want >= %d", tt.terminalHeight, top, tt.minTop)
		}
		if bottom < tt.minBottom {
			t.Errorf("calculateHeights(%d) bottom = %d, want >= %d", tt.terminalHeight, bottom, tt.minBottom)
		}
	}
}

func TestInitialModel(t *testing.T) {
	db := &mockJobDB{}
	model := InitialModel("http://localhost:8080", db, nil, nil)

	if model.serverURL != "http://localhost:8080" {
		t.Errorf("Expected serverURL 'http://localhost:8080', got %q", model.serverURL)
	}

	if model.startTime.IsZero() {
		t.Error("Expected startTime to be set")
	}

	if time.Since(model.startTime) > time.Second {
		t.Error("startTime should be recent")
	}

	if model.quitting {
		t.Error("Model should not be quitting initially")
	}
}

func TestModelView(t *testing.T) {
	db := &mockJobDB{}
	model := InitialModel("http://localhost:8080", db, nil, nil)
	model.width = 100
	model.height = 30

	view := model.View()

	// Check that view contains expected elements
	if len(view) == 0 {
		t.Error("View should not be empty")
	}

	// Should contain the title
	if !containsString(view, "BIBLIO AUDIOBOOK BUILDER TTS") {
		t.Error("View should contain title")
	}

	// Should contain server URL
	if !containsString(view, "localhost:8080") {
		t.Error("View should contain server URL")
	}
}

func TestModelViewQuitting(t *testing.T) {
	db := &mockJobDB{}
	model := InitialModel("http://localhost:8080", db, nil, nil)
	model.quitting = true

	view := model.View()

	if !containsString(view, "Shutting down") {
		t.Error("Quitting view should contain shutdown message")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
