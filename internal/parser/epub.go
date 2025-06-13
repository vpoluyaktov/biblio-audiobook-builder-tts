package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type epubParser struct{}

func NewEpubParser() *epubParser {
	return &epubParser{}
}

// ParseEpubFile opens the file and parses it as EPUB
func (p *epubParser) ParseEpubFile(path string) (*Book, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return p.ParseEpub(f)
}

type epubContainer struct {
	XMLName  xml.Name `xml:"container"`
	RootFile struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubPackage struct {
	XMLName xml.Name `xml:"package"`
	Metadata struct {
		Title       string   `xml:"title"`
		Creator     string   `xml:"creator"`
		Description string   `xml:"description"`
		Subjects    []string `xml:"subject"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Items []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func (p *epubParser) ParseEpub(r io.Reader) (*Book, error) {
	// Read the entire content into memory
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read epub content: %v", err)
	}

	// Open the epub file as a zip archive
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to open epub as zip: %v", err)
	}

	// Find and parse container.xml
	containerFile, err := findFile(zipReader, "META-INF/container.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to find container.xml: %v", err)
	}

	var container epubContainer
	if err := parseXML(containerFile, &container); err != nil {
		return nil, fmt.Errorf("failed to parse container.xml: %v", err)
	}

	// Find and parse the package file (content.opf)
	packageFile, err := findFile(zipReader, container.RootFile.FullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find package file: %v", err)
	}

	var pkg epubPackage
	if err := parseXML(packageFile, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package file: %v", err)
	}

	// Create book structure
	desc := pkg.Metadata.Description
	if desc == "" && len(pkg.Metadata.Subjects) > 0 {
		desc = strings.Join(pkg.Metadata.Subjects, ", ")
	}
	book := &Book{
		Title:    pkg.Metadata.Title,
		Author:   pkg.Metadata.Creator,
		Chapters: make([]Chapter, 0),
		Metadata: map[string]string{
			"description": desc,
		},
	}

	// Find TOC (NCX or XHTML nav)
	baseDir := filepath.Dir(container.RootFile.FullPath)
	var tocPath, tocType string
	for _, item := range pkg.Manifest.Items {
		if item.MediaType == "application/x-dtbncx+xml" {
			tocPath = filepath.Join(baseDir, item.Href)
			tocType = "ncx"
			break
		} else if item.MediaType == "application/xhtml+xml" && strings.Contains(strings.ToLower(item.Href), "nav") {
			tocPath = filepath.Join(baseDir, item.Href)
			tocType = "nav"
		}
	}

	if tocPath == "" {
		return book, nil // fallback: no TOC found, return metadata only
	}

	tocFile, err := findFile(zipReader, tocPath)
	if err != nil {
		return book, nil // fallback: no TOC file found
	}
	tocBytes, err := ioutil.ReadAll(tocFile)
	if err != nil {
		return book, nil
	}

	if tocType == "ncx" {
		chapters, _ := parseNCXChapters(tocBytes)
		for _, ch := range chapters {
			chapterText := readEPUBChapterContent(zipReader, baseDir, ch.Href)
			book.Chapters = append(book.Chapters, Chapter{
				Title:   ch.Title,
				Content: chapterText,
			})
		}
	} else if tocType == "nav" {
		chapters, _ := parseNavChapters(tocBytes)
		for _, ch := range chapters {
			chapterText := readEPUBChapterContent(zipReader, baseDir, ch.Href)
			book.Chapters = append(book.Chapters, Chapter{
				Title:   ch.Title,
				Content: chapterText,
			})
		}
	}


	return book, nil
}

// --- TOC-based chapter extraction logic ---
type tocChapter struct {
	Title string
	Href  string
}

func parseNCXChapters(ncx []byte) ([]tocChapter, error) {
	type navPoint struct {
		XMLName xml.Name `xml:"navPoint"`
		NavLabel struct {
			Text string `xml:"text"`
		} `xml:"navLabel"`
		Content struct {
			Src string `xml:"src,attr"`
		} `xml:"content"`
		NavPoints []navPoint `xml:"navPoint"`
	}
	type ncxRoot struct {
		XMLName  xml.Name   `xml:"ncx"`
		NavMap   struct {
			NavPoints []navPoint `xml:"navPoint"`
		} `xml:"navMap"`
	}
	var root ncxRoot
	if err := xml.Unmarshal(ncx, &root); err != nil {
		return nil, err
	}
	var chapters []tocChapter
	var walk func([]navPoint)
	walk = func(points []navPoint) {
		for _, np := range points {
			chapters = append(chapters, tocChapter{
				Title: np.NavLabel.Text,
				Href:  np.Content.Src,
			})
			if len(np.NavPoints) > 0 {
				walk(np.NavPoints)
			}
		}
	}
	walk(root.NavMap.NavPoints)
	return chapters, nil
}

func parseNavChapters(nav []byte) ([]tocChapter, error) {
	type navA struct {
		XMLName xml.Name `xml:"a"`
		Href   string   `xml:"href,attr"`
		Text   string   `xml:",chardata"`
	}
	type navLi struct {
		XMLName xml.Name `xml:"li"`
		A       navA     `xml:"a"`
		Lis     []navLi  `xml:"ol>li"`
	}
	type navRoot struct {
		XMLName xml.Name `xml:"html"`
		Navs    []struct {
			XMLName xml.Name `xml:"nav"`
			Ol      struct {
				Lis []navLi `xml:"li"`
			} `xml:"ol"`
		} `xml:"body>nav"`
	}
	var root navRoot
	if err := xml.Unmarshal(nav, &root); err != nil {
		return nil, err
	}
	var chapters []tocChapter
	var walk func([]navLi)
	walk = func(lis []navLi) {
		for _, li := range lis {
			chapters = append(chapters, tocChapter{
				Title: strings.TrimSpace(li.A.Text),
				Href:  li.A.Href,
			})
			if len(li.Lis) > 0 {
				walk(li.Lis)
			}
		}
	}
	for _, nav := range root.Navs {
		walk(nav.Ol.Lis)
	}
	return chapters, nil
}

func readEPUBChapterContent(zr *zip.Reader, baseDir, href string) string {
	// href may be "file.xhtml#anchor"
	parts := strings.SplitN(href, "#", 2)
	file := filepath.Join(baseDir, parts[0])
	f, err := findFile(zr, file)
	if err != nil {
		return ""
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return ""
	}
	return string(bytes) // TODO: extract only anchor section if present
}

func findFile(zr *zip.Reader, name string) (io.Reader, error) {
	for _, f := range zr.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

func parseXML(r io.Reader, v interface{}) error {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return xml.Unmarshal(content, v)
}
