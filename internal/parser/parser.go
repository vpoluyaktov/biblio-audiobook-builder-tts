package parser

import (
	"strings"
)

// Book represents the parsed book content
type Book struct {
	Title          string
	Author         string
	Series         string
	SeriesNumber   string
	Description    string
	Genre          string            // comma-separated genre string
	Chapters       []Chapter
	CoverImage     []byte
	CoverImageName string
	CoverImageType string // "image/jpeg" or "image/png"
	Metadata       map[string]string
}

// Chapter represents a single chapter in the book
type Chapter struct {
	Title    string
	Content  string
	ID       string
	TOCDepth int
}

// GetTotalCharacters returns the total character count across all chapters
func (b *Book) GetTotalCharacters() int {
	total := 0
	for _, ch := range b.Chapters {
		total += len(ch.Content)
	}
	return total
}

// GetTotalWords returns the approximate word count across all chapters
func (b *Book) GetTotalWords() int {
	total := 0
	for _, ch := range b.Chapters {
		total += len(strings.Fields(ch.Content))
	}
	return total
}
