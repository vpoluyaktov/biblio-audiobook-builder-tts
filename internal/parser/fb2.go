package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type fb2Parser struct{}

func NewFB2Parser() *fb2Parser {
	return &fb2Parser{}
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

type fb2Book struct {
	XMLName     xml.Name `xml:"FictionBook"`
	Description struct {
		TitleInfo struct {
			Author struct {
				FirstName string `xml:"first-name"`
				LastName  string `xml:"last-name"`
			} `xml:"author"`
			BookTitle string `xml:"book-title"`
			Annotation fb2Annotation `xml:"annotation"`
		} `xml:"title-info"`
	} `xml:"description"`
	Body struct {
		Sections []fb2Section `xml:"section"`
	} `xml:"body"`
}

type fb2Annotation struct {
	Text string   `xml:",chardata"`
	Paragraphs []string `xml:"p"`
}


type fb2Section struct {
	Title    fb2Title      `xml:"title"`
	Paragraphs []string    `xml:"p"`
	Sections []fb2Section  `xml:"section"`
}

type fb2Title struct {
	Paragraphs []string `xml:"p"`
}


func (p *fb2Parser) ParseFB2(r io.Reader) (*Book, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read FB2 content: %v", err)
	}

	var fb2 fb2Book
	if err := xml.Unmarshal(content, &fb2); err != nil {
		return nil, fmt.Errorf("failed to parse FB2: %v", err)
	}

	annotation := strings.TrimSpace(fb2.Description.TitleInfo.Annotation.Text)
if annotation == "" && len(fb2.Description.TitleInfo.Annotation.Paragraphs) > 0 {
	annotation = strings.Join(fb2.Description.TitleInfo.Annotation.Paragraphs, "\n")
}
book := &Book{
	Title:  fb2.Description.TitleInfo.BookTitle,
	Author: fmt.Sprintf("%s %s",
		fb2.Description.TitleInfo.Author.FirstName,
		fb2.Description.TitleInfo.Author.LastName),
	Chapters: make([]Chapter, 0),
	Metadata: map[string]string{
		"description": annotation,
	},
}

// Recursively extract chapters from all sections
	var addSections func(sections []fb2Section, depth int)
	addSections = func(sections []fb2Section, depth int) {
		for _, section := range sections {
			title := strings.TrimSpace(strings.Join(section.Title.Paragraphs, " "))
			if title == "" {
				title = fmt.Sprintf("Chapter %d", len(book.Chapters)+1)
			}
			content := strings.Join(section.Paragraphs, "\n")
			book.Chapters = append(book.Chapters, Chapter{
				Title:   title,
				Content: content,
			})
			if len(section.Sections) > 0 {
				addSections(section.Sections, depth+1)
			}
		}
	}
	addSections(fb2.Body.Sections, 0)

	return book, nil
}
