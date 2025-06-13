package parser

import (
	"io"
)

// Parser interface defines methods for parsing different book formats
type Parser interface {
	ParseEpub(r io.Reader) (*Book, error)
	ParseFB2(r io.Reader) (*Book, error)
	Parse(path string) (*Book, error)
}

// Book represents the parsed book content
type Book struct {
	Title       string
	Author      string
	Chapters    []Chapter
	Cover       []byte
	Metadata    map[string]string
}

// Chapter represents a single chapter in the book
type Chapter struct {
	Title   string
	Content string
}

// NewParser is deprecated and should not be used. Use NewEpubParser or NewFB2Parser instead.
func NewParser() Parser {
	panic("NewParser is deprecated. Use NewEpubParser or NewFB2Parser instead.")
}


