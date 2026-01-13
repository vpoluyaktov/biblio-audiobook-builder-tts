package dto

import parser "abb_tts/internal/parser"

// ParseBookCommand is sent from BookPage to BookController to request parsing an ebook file
// Result is sent as BookParsedResult
type ParseBookCommand struct {
	FilePath string
}

type BookParsedResult struct {
	Book   *parser.Book
	Error  string // empty if no error
}
