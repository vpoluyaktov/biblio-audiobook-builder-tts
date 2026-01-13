package opds

import (
	"testing"
)

func TestSource_HasAuth(t *testing.T) {
	tests := []struct {
		name     string
		source   Source
		expected bool
	}{
		{
			name:     "no auth",
			source:   Source{Username: "", Password: ""},
			expected: false,
		},
		{
			name:     "username only",
			source:   Source{Username: "user", Password: ""},
			expected: false,
		},
		{
			name:     "password only",
			source:   Source{Username: "", Password: "pass"},
			expected: false,
		},
		{
			name:     "both username and password",
			source:   Source{Username: "user", Password: "pass"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.HasAuth()
			if result != tt.expected {
				t.Errorf("HasAuth() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultSources(t *testing.T) {
	sources := DefaultSources()

	if len(sources) == 0 {
		t.Fatal("DefaultSources returned empty list")
	}

	// Check that Project Gutenberg is included
	var gutenbergFound bool
	for _, s := range sources {
		if s.ID == "gutenberg" {
			gutenbergFound = true
			if s.Name != "Project Gutenberg" {
				t.Errorf("expected name 'Project Gutenberg', got '%s'", s.Name)
			}
			if s.URL != "https://www.gutenberg.org/ebooks.opds/" {
				t.Errorf("expected URL 'https://www.gutenberg.org/ebooks.opds/', got '%s'", s.URL)
			}
			if !s.IsDefault {
				t.Error("expected IsDefault to be true")
			}
			if !s.Enabled {
				t.Error("expected Enabled to be true")
			}
		}
	}

	if !gutenbergFound {
		t.Error("Project Gutenberg not found in default sources")
	}
}

func TestSource_Fields(t *testing.T) {
	source := Source{
		ID:          "test-id",
		Name:        "Test Source",
		URL:         "https://example.com/opds",
		Description: "A test source",
		Username:    "testuser",
		Password:    "testpass",
		IsDefault:   true,
		Enabled:     true,
	}

	if source.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", source.ID)
	}
	if source.Name != "Test Source" {
		t.Errorf("expected Name 'Test Source', got '%s'", source.Name)
	}
	if source.URL != "https://example.com/opds" {
		t.Errorf("expected URL 'https://example.com/opds', got '%s'", source.URL)
	}
	if source.Description != "A test source" {
		t.Errorf("expected Description 'A test source', got '%s'", source.Description)
	}
	if source.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got '%s'", source.Username)
	}
	if source.Password != "testpass" {
		t.Errorf("expected Password 'testpass', got '%s'", source.Password)
	}
	if !source.IsDefault {
		t.Error("expected IsDefault to be true")
	}
	if !source.Enabled {
		t.Error("expected Enabled to be true")
	}
}
