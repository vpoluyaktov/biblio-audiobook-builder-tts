package parser

import (
	"io"
	"strings"
)

// Parser interface defines methods for parsing different book formats
type Parser interface {
	ParseEpub(r io.Reader) (*Book, error)
	ParseFB2(r io.Reader) (*Book, error)
	Parse(path string) (*Book, error)
}

// Book represents the parsed book content
type Book struct {
	Title          string
	Author         string
	Description    string
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

// NewParser is deprecated and should not be used. Use NewEpubParser or NewFB2Parser instead.
func NewParser() Parser {
	panic("NewParser is deprecated. Use NewEpubParser or NewFB2Parser instead.")
}
