package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vpoluyaktov/biblio-ebook-parser/cover"
	_ "github.com/vpoluyaktov/biblio-ebook-parser/formats" // Register parsers
	ebookparser "github.com/vpoluyaktov/biblio-ebook-parser/parser"
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
		AddPeriods:    true, // Add periods to paragraphs
		InsertMarkers: true, // Insert SSML markers
		NormalizeText: true, // Normalize text for speech
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

// ParseFile parses an ebook file from a file path
func ParseFile(filePath string) (*Book, error) {
	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	var format string
	switch ext {
	case ".epub":
		format = "epub"
	case ".fb2":
		format = "fb2"
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	// Open file
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Get file info for size
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Parse using unified parser
	p, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("failed to get parser: %w", err)
	}

	unifiedBook, err := p.ParseReader(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Render to plain text for TTS
	renderer := plaintext.NewRenderer(plaintext.Config{
		AddPeriods:    true,
		InsertMarkers: false,
		NormalizeText: true,
	})

	content, err := renderer.RenderContent(unifiedBook)
	if err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}

	plaintextContent, ok := content.(*plaintext.Book)
	if !ok {
		return nil, fmt.Errorf("unexpected content type")
	}

	// Convert to audiobook builder format
	book := &Book{
		Title:          unifiedBook.Metadata.Title,
		Description:    unifiedBook.Metadata.Description,
		Series:         unifiedBook.Metadata.Series,
		CoverImage:     unifiedBook.Metadata.CoverData,
		CoverImageType: unifiedBook.Metadata.CoverType,
		Chapters:       make([]Chapter, len(plaintextContent.Chapters)),
	}

	// Set author
	if len(unifiedBook.Metadata.Authors) > 0 {
		book.Author = unifiedBook.Metadata.Authors[0].FullName()
	}

	// Set series number
	if unifiedBook.Metadata.SeriesIndex > 0 {
		book.SeriesNumber = fmt.Sprintf("%d", unifiedBook.Metadata.SeriesIndex)
	}

	// Generate placeholder cover if book has no cover
	if len(book.CoverImage) == 0 {
		placeholderCover, err := cover.GeneratePlaceholder(book.Title, book.Author)
		if err == nil {
			book.CoverImage = placeholderCover
			book.CoverImageType = "image/jpeg"
			book.CoverImageName = "cover.jpg"
		}
	}

	// Convert chapters
	for i, ch := range plaintextContent.Chapters {
		book.Chapters[i] = Chapter{
			ID:      ch.ID,
			Title:   ch.Title,
			Content: ch.Content,
		}
	}

	return book, nil
}

// ParseReader parses an ebook from an io.Reader
func ParseReader(reader io.Reader, format string) (*Book, error) {
	// Read all content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Parse using unified parser
	p, err := ebookparser.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("failed to get parser: %w", err)
	}

	unifiedBook, err := p.ParseReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse book: %w", err)
	}

	// Render to plain text for TTS
	renderer := plaintext.NewRenderer(plaintext.Config{
		AddPeriods:    true,
		InsertMarkers: false,
		NormalizeText: true,
	})

	content, err := renderer.RenderContent(unifiedBook)
	if err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}

	plaintextContent, ok := content.(*plaintext.Book)
	if !ok {
		return nil, fmt.Errorf("unexpected content type")
	}

	// Convert to audiobook builder format
	book := &Book{
		Title:          unifiedBook.Metadata.Title,
		Description:    unifiedBook.Metadata.Description,
		Series:         unifiedBook.Metadata.Series,
		CoverImage:     unifiedBook.Metadata.CoverData,
		CoverImageType: unifiedBook.Metadata.CoverType,
		Chapters:       make([]Chapter, len(plaintextContent.Chapters)),
	}

	// Set author
	if len(unifiedBook.Metadata.Authors) > 0 {
		book.Author = unifiedBook.Metadata.Authors[0].FullName()
	}

	// Set series number
	if unifiedBook.Metadata.SeriesIndex > 0 {
		book.SeriesNumber = fmt.Sprintf("%d", unifiedBook.Metadata.SeriesIndex)
	}

	// Generate placeholder cover if book has no cover
	if len(book.CoverImage) == 0 {
		placeholderCover, err := cover.GeneratePlaceholder(book.Title, book.Author)
		if err == nil {
			book.CoverImage = placeholderCover
			book.CoverImageType = "image/jpeg"
			book.CoverImageName = "cover.jpg"
		}
	}

	// Convert chapters
	for i, ch := range plaintextContent.Chapters {
		book.Chapters[i] = Chapter{
			ID:      ch.ID,
			Title:   ch.Title,
			Content: ch.Content,
		}
	}

	return book, nil
}
