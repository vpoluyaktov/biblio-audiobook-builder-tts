package parser

import (
	"biblio-audiobook-builder-tts/internal/normalize"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/ianaindex"
)

// Pre-compiled regex patterns for FB2 parsing (performance optimization)
var (
	reFB2Xmlns      = regexp.MustCompile(`xmlns[^=]*="[^"]*"`)
	reFB2NSOpen     = regexp.MustCompile(`<[a-zA-Z]+:`)
	reFB2NSClose    = regexp.MustCompile(`</[a-zA-Z]+:`)
	reFB2Section    = regexp.MustCompile(`(?is)<section[^>]*>.*?</section>`)
	reFB2Table      = regexp.MustCompile(`(?i)<table[^>]*>.*?</table>`)
	reFB2Image      = regexp.MustCompile(`(?i)<image[^>]*/?>`)
	reFB2EmptyLine  = regexp.MustCompile(`(?i)<empty-line\s*/?>`)
	reFB2Link       = regexp.MustCompile(`(?is)<a[^>]*>.*?</a>`)
	reFB2PClose     = regexp.MustCompile(`(?i)</p>`)
	reFB2POpen      = regexp.MustCompile(`(?i)<p[^>]*>`)
	reFB2TitleClose = regexp.MustCompile(`(?i)</title>`)
	reFB2TitleOpen  = regexp.MustCompile(`(?i)<title[^>]*>`)
	reFB2SubClose   = regexp.MustCompile(`(?i)</subtitle>`)
	reFB2SubOpen    = regexp.MustCompile(`(?i)<subtitle[^>]*>`)
	reFB2Tags       = regexp.MustCompile(`<[^>]+>`)
	reFB2Spaces     = regexp.MustCompile(`[ \t]+`)
	reFB2Newlines   = regexp.MustCompile(`\n{2,}`)
)

type fb2Parser struct {
	TOCMaxDepth int
	ParseNotes  bool
}

func NewFB2Parser() *fb2Parser {
	return &fb2Parser{
		TOCMaxDepth: 3,
		ParseNotes:  false,
	}
}

// ParseFB2File opens the file and parses it as FB2
func (p *fb2Parser) ParseFB2File(path string) (*Book, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return p.ParseFB2(f)
}

type fb2Document struct {
	XMLName     xml.Name `xml:"FictionBook"`
	Description struct {
		TitleInfo struct {
			Author struct {
				FirstName string `xml:"first-name"`
				LastName  string `xml:"last-name"`
			} `xml:"author"`
			BookTitle  string `xml:"book-title"`
			Annotation struct {
				Content string `xml:",innerxml"`
			} `xml:"annotation"`
			Coverpage struct {
				Image struct {
					Href string `xml:"href,attr"`
				} `xml:"image"`
			} `xml:"coverpage"`
		} `xml:"title-info"`
	} `xml:"description"`
	Bodies   []fb2Body   `xml:"body"`
	Binaries []fb2Binary `xml:"binary"`
}

type fb2Body struct {
	Name     string       `xml:"name,attr"`
	Title    fb2Title     `xml:"title"`
	Sections []fb2Section `xml:"section"`
}

type fb2Section struct {
	Title    fb2Title     `xml:"title"`
	Content  string       `xml:",innerxml"`
	Sections []fb2Section `xml:"section"`
}

type fb2Title struct {
	Content string `xml:",innerxml"`
}

type fb2Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"`
}

func (p *fb2Parser) ParseFB2(r io.Reader) (*Book, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read FB2 content: %v", err)
	}

	// Strip namespace prefixes for easier parsing (using pre-compiled patterns)
	contentStr := string(content)
	contentStr = reFB2Xmlns.ReplaceAllString(contentStr, "")
	contentStr = reFB2NSOpen.ReplaceAllStringFunc(contentStr, func(s string) string {
		return "<"
	})
	contentStr = reFB2NSClose.ReplaceAllStringFunc(contentStr, func(s string) string {
		return "</"
	})

	// Use xml.Decoder with CharsetReader to handle non-UTF-8 encodings (e.g., windows-1251)
	var fb2 fb2Document
	decoder := xml.NewDecoder(bytes.NewReader([]byte(contentStr)))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		enc, err := ianaindex.IANA.Encoding(charset)
		if err != nil {
			return nil, fmt.Errorf("unsupported charset %q: %v", charset, err)
		}
		if enc == nil {
			// nil encoding means UTF-8
			return input, nil
		}
		return enc.NewDecoder().Reader(input), nil
	}
	if err := decoder.Decode(&fb2); err != nil {
		return nil, fmt.Errorf("failed to parse FB2: %v", err)
	}

	// Extract annotation text
	annotation := fb2TreeToText(fb2.Description.TitleInfo.Annotation.Content)

	book := &Book{
		Title: fb2.Description.TitleInfo.BookTitle,
		Author: strings.TrimSpace(fmt.Sprintf("%s %s",
			fb2.Description.TitleInfo.Author.FirstName,
			fb2.Description.TitleInfo.Author.LastName)),
		Description: annotation,
		Chapters:    make([]Chapter, 0),
		Metadata: map[string]string{
			"description": annotation,
		},
	}

	// Extract cover image
	coverHref := fb2.Description.TitleInfo.Coverpage.Image.Href
	if coverHref != "" {
		coverHref = strings.TrimPrefix(coverHref, "#")
		for _, binary := range fb2.Binaries {
			if binary.ID == coverHref {
				if binary.ContentType == "image/jpeg" || binary.ContentType == "image/jpg" || binary.ContentType == "image/png" {
					decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(binary.Data))
					if err == nil {
						book.CoverImage = decoded
						book.CoverImageName = coverHref
						book.CoverImageType = binary.ContentType
					}
				}
				break
			}
		}
	}

	// Process all bodies
	for _, body := range fb2.Bodies {
		// Skip notes and comments sections unless configured
		if body.Name == "notes" || body.Name == "comments" {
			if !p.ParseNotes {
				continue
			}
		}

		// Add body title as chapter if present
		if body.Title.Content != "" {
			book.Chapters = append(book.Chapters, Chapter{
				Title:    fb2TreeToText(body.Title.Content),
				Content:  fb2TreeToText(body.Title.Content),
				TOCDepth: 0,
			})
		}

		// Process sections recursively
		p.addFB2Sections(book, body.Sections, 0)
	}

	return book, nil
}

func (p *fb2Parser) addFB2Sections(book *Book, sections []fb2Section, depth int) {
	depth++
	for i, section := range sections {
		title := fb2TreeToText(section.Title.Content)
		if title == "" {
			title = fmt.Sprintf("Chapter %d", len(book.Chapters)+1)
		}

		// Extract text content, excluding nested sections
		// The section.Content contains innerxml which includes nested <section> tags
		// fb2TreeToText already removes <section>...</section> blocks via regex
		content := fb2TreeToText(section.Content)

		// Only add chapter if it has actual content (not just title)
		// Sections that only contain nested sections should not be added as separate chapters
		hasNestedSections := len(section.Sections) > 0
		contentIsEmpty := strings.TrimSpace(content) == "" || strings.TrimSpace(content) == strings.TrimSpace(title)

		if !contentIsEmpty || !hasNestedSections {
			// Add this section as a chapter if it has content OR if it has no nested sections
			book.Chapters = append(book.Chapters, Chapter{
				Title:    strings.TrimSpace(title),
				Content:  content,
				ID:       fmt.Sprintf("section_%d_%d", depth, i),
				TOCDepth: depth,
			})
		}

		// Recursively process nested sections if within depth limit
		if depth < p.TOCMaxDepth && len(section.Sections) > 0 {
			p.addFB2Sections(book, section.Sections, depth)
		}
	}
}

// fb2TreeToText converts FB2 XML content to plain text
// Based on Python tree_to_text function
// Uses pre-compiled regex patterns for performance
func fb2TreeToText(xmlContent string) string {
	if xmlContent == "" {
		return ""
	}

	// Remove nested section tags (we process them separately)
	// Must loop to handle deeply nested sections since regex can't handle arbitrary nesting
	text := xmlContent
	for {
		newText := reFB2Section.ReplaceAllString(text, "")
		if newText == text {
			break // No more sections to remove
		}
		text = newText
	}

	// Handle special elements (single newline for tighter spacing)
	text = reFB2Table.ReplaceAllString(text, "\nTable omitted.\n")
	text = reFB2Image.ReplaceAllString(text, "\nIllustration.\n")
	text = reFB2EmptyLine.ReplaceAllString(text, "\n")

	// Skip footnotes and links
	text = reFB2Link.ReplaceAllString(text, "")

	// Handle paragraphs (single newline for tighter spacing)
	text = reFB2PClose.ReplaceAllString(text, "\n")
	text = reFB2POpen.ReplaceAllString(text, "")

	// Handle titles - insert title break marker for SSML pause after titles
	text = reFB2TitleClose.ReplaceAllString(text, normalize.TitleBreakMarker)
	text = reFB2TitleOpen.ReplaceAllString(text, "\n")
	text = reFB2SubClose.ReplaceAllString(text, "\n")
	text = reFB2SubOpen.ReplaceAllString(text, "\n")

	// Remove remaining XML tags
	text = reFB2Tags.ReplaceAllString(text, "")

	// Decode all HTML entities (numeric like &#8197; and named like &nbsp;)
	text = html.UnescapeString(text)

	// Clean up whitespace (collapse 3+ newlines to single newline for tighter spacing)
	text = strings.ReplaceAll(text, "\u00A0", " ")
	text = reFB2Spaces.ReplaceAllString(text, " ")
	text = reFB2Newlines.ReplaceAllString(text, "\n")

	// Add periods to paragraphs
	text = addPeriodToText(text)

	return strings.TrimSpace(text)
}
