package opds

import (
	"time"
)

// Source represents an OPDS catalog source
type Source struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	IsDefault   bool      `json:"is_default"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HasAuth returns true if the source has authentication configured
func (s *Source) HasAuth() bool {
	return s.Username != "" && s.Password != ""
}

// DefaultSources returns the list of default OPDS sources
func DefaultSources() []Source {
	return []Source{
		{
			ID:          "gutenberg",
			Name:        "Project Gutenberg",
			URL:         "https://www.gutenberg.org/ebooks.opds/",
			Description: "Free ebooks from Project Gutenberg. Over 70,000 free ebooks.",
			IsDefault:   true,
			Enabled:     true,
		},
		{
			ID:          "internet-archive",
			Name:        "Internet Archive",
			URL:         "https://archive.org/services/opds",
			Description: "Millions of free ebooks from Internet Archive.",
			IsDefault:   true,
			Enabled:     true,
		},
	}
}
