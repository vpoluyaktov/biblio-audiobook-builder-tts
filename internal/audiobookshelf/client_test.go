package audiobookshelf

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:13378")
	if client == nil {
		t.Error("NewClient should not return nil")
	}
	if client.url != "http://localhost:13378" {
		t.Errorf("Expected URL 'http://localhost:13378', got '%s'", client.url)
	}
}

func TestIsLoggedIn(t *testing.T) {
	client := NewClient("http://localhost:13378")

	// Not logged in initially
	if client.IsLoggedIn() {
		t.Error("Client should not be logged in initially")
	}

	// Simulate login
	client.loginResponse = &LoginResponse{
		User: User{
			Token: "test-token",
		},
	}

	if !client.IsLoggedIn() {
		t.Error("Client should be logged in after setting token")
	}
}

func TestGetLibraryID(t *testing.T) {
	client := NewClient("http://localhost:13378")

	libraries := []Library{
		{ID: "lib-1", Name: "Audiobooks"},
		{ID: "lib-2", Name: "Podcasts"},
		{ID: "lib-3", Name: "Music"},
	}

	// Find existing library
	id, err := client.GetLibraryID(libraries, "Audiobooks")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if id != "lib-1" {
		t.Errorf("Expected 'lib-1', got '%s'", id)
	}

	// Find non-existing library
	_, err = client.GetLibraryID(libraries, "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent library")
	}
}

func TestGetFolders(t *testing.T) {
	client := NewClient("http://localhost:13378")

	libraries := []Library{
		{
			ID:   "lib-1",
			Name: "Audiobooks",
			Folders: []Folder{
				{ID: "folder-1", FullPath: "/audiobooks"},
				{ID: "folder-2", FullPath: "/audiobooks/new"},
			},
		},
	}

	// Find folders for existing library
	folders, err := client.GetFolders(libraries, "Audiobooks")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(folders) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(folders))
	}

	// Find folders for non-existing library
	_, err = client.GetFolders(libraries, "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent library")
	}
}

func TestProgressReader(t *testing.T) {
	var lastPercent int
	callback := func(fileID int, fileName string, size int64, pos int64, percent int) {
		lastPercent = percent
	}

	// Create a simple reader with known content
	content := []byte("Hello, World!")
	pr := &progressReader{
		fileID:   0,
		fileName: "test.txt",
		reader:   &mockReader{data: content},
		size:     int64(len(content)),
		callback: callback,
	}

	buf := make([]byte, 5)
	n, err := pr.Read(buf)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to read 5 bytes, got %d", n)
	}

	// Progress should be ~38% (5/13)
	if lastPercent < 30 || lastPercent > 45 {
		t.Errorf("Expected progress around 38%%, got %d%%", lastPercent)
	}
}

// mockReader is a simple io.Reader for testing
type mockReader struct {
	data []byte
	pos  int
}

func (r *mockReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
