package parser

import (
	"fmt"
	"io"

	ebookparser "github.com/vpoluyaktov/biblio-ebook-parser/parser"
	_ "github.com/vpoluyaktov/biblio-ebook-parser/formats" // Register parsers
	"github.com/vpoluyaktov/biblio-ebook-parser/renderer/plaintext"
)

// ParseBook parses an ebook file using the unified parser library
func ParseBook(filePath string) (*Book, error) {
	// Determine format from file extension
	format := "epub"
	if len(filePath) > 4 {
		ext := filePath[len(filePath)-4:]
		if ext == ".fb2" {
			format = "fb2"
		}
	}

	parser, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format: %w", err)
	}

	book, err := parser.Parse(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Render to plain text for TTS
	renderer := plaintext.NewRenderer(plaintext.Config{
		AddPeriods:    true,  // Add periods to paragraphs
		InsertMarkers: true,  // Insert SSML markers
		NormalizeText: true,  // Normalize text for speech
	})

	content, err := renderer.RenderContent(book)
	if err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}

	// Convert to our Book format
	plaintextBook, ok := content.(*plaintext.Book)
	if !ok {
		return nil, fmt.Errorf("unexpected content type")
	}

	result := &Book{
		Title:        plaintextBook.Title,
		Author:       plaintextBook.Author,
		Series:       plaintextBook.Series,
		SeriesNumber: plaintextBook.SeriesNumber,
		Description:  plaintextBook.Description,
		Chapters:     make([]Chapter, len(plaintextBook.Chapters)),
		Metadata:     plaintextBook.Metadata,
	}

	// Copy cover image from metadata
	if book.Metadata.CoverData != nil {
		result.CoverImage = book.Metadata.CoverData
		result.CoverImageType = book.Metadata.CoverType
		// Extract filename from cover type
		if book.Metadata.CoverType == "image/png" {
			result.CoverImageName = "cover.png"
		} else {
			result.CoverImageName = "cover.jpg"
		}
	}

	for i, ch := range plaintextBook.Chapters {
		result.Chapters[i] = Chapter{
			Title:    ch.Title,
			Content:  ch.Content,
			ID:       ch.ID,
			TOCDepth: ch.TOCDepth,
		}
	}

	return result, nil
}

// ParseBookFromReader parses an ebook from an io.Reader using the unified parser library
func ParseBookFromReader(r io.Reader, format string) (*Book, error) {
	// For now, this is not implemented as the unified parser requires io.ReaderAt
	// The existing code can continue to use the old implementation if needed
	return nil, fmt.Errorf("ParseBookFromReader not yet implemented with unified parser")
}
