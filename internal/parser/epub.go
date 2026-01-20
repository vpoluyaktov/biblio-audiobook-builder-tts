package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	htmlPkg "html"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for HTML to text conversion (performance optimization)
var (
	reHead     = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	reScript   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reBlock    = regexp.MustCompile(`(?i)</(p|div|br|h[1-6]|li|tr)>`)
	reBr       = regexp.MustCompile(`(?i)<br\s*/?>`)
	reTags     = regexp.MustCompile(`<[^>]+>`)
	reSpaces   = regexp.MustCompile(`[ \t]+`)
	reNewlines = regexp.MustCompile(`\n{2,}`)
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
	XMLName  xml.Name `xml:"package"`
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
		Title:       pkg.Metadata.Title,
		Author:      pkg.Metadata.Creator,
		Description: desc,
		Chapters:    make([]Chapter, 0),
		Metadata: map[string]string{
			"description": desc,
		},
	}

	// Find TOC (NCX or XHTML nav)
	baseDir := filepath.Dir(container.RootFile.FullPath)

	// Extract cover image
	coverHref := extractCoverHref(pkg, baseDir)
	if coverHref != "" {
		coverFile, err := findFile(zipReader, coverHref)
		if err == nil {
			coverData, err := ioutil.ReadAll(coverFile)
			if err == nil {
				book.CoverImage = coverData
				book.CoverImageName = filepath.Base(coverHref)
				if strings.HasSuffix(strings.ToLower(coverHref), ".png") {
					book.CoverImageType = "image/png"
				} else {
					book.CoverImageType = "image/jpeg"
				}
			}
		}
	}
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

	switch tocType {
	case "ncx":
		chapters, _ := parseNCXChapters(tocBytes)
		for i, ch := range chapters {
			// Get next chapter href to determine end boundary (if in same file)
			var nextHref string
			if i < len(chapters)-1 {
				nextHref = chapters[i+1].Href
			}
			chapterText := readEPUBChapterContentWithBoundary(zipReader, baseDir, ch.Href, nextHref)
			book.Chapters = append(book.Chapters, Chapter{
				Title:   ch.Title,
				Content: chapterText,
			})
		}
	case "nav":
		chapters, _ := parseNavChapters(tocBytes)
		for i, ch := range chapters {
			// Get next chapter href to determine end boundary (if in same file)
			var nextHref string
			if i < len(chapters)-1 {
				nextHref = chapters[i+1].Href
			}
			chapterText := readEPUBChapterContentWithBoundary(zipReader, baseDir, ch.Href, nextHref)
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
		XMLName  xml.Name `xml:"navPoint"`
		NavLabel struct {
			Text string `xml:"text"`
		} `xml:"navLabel"`
		Content struct {
			Src string `xml:"src,attr"`
		} `xml:"content"`
		NavPoints []navPoint `xml:"navPoint"`
	}
	type ncxRoot struct {
		XMLName xml.Name `xml:"ncx"`
		NavMap  struct {
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
		Href    string   `xml:"href,attr"`
		Text    string   `xml:",chardata"`
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
	return readEPUBChapterContentWithBoundary(zr, baseDir, href, "")
}

// readEPUBChapterContentWithBoundary reads chapter content with optional end boundary
// Based on Python EBook_audiobook_creator fetch_chapters_text logic
func readEPUBChapterContentWithBoundary(zr *zip.Reader, baseDir, href, nextHref string) string {
	// href may be "file.xhtml#anchor"
	parts := strings.SplitN(href, "#", 2)
	fileName := parts[0]
	file := filepath.Join(baseDir, fileName)
	f, err := findFile(zr, file)
	if err != nil {
		return ""
	}
	contentBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return ""
	}

	html := string(contentBytes)

	// Determine start position from current anchor
	var startPos int
	if len(parts) > 1 && parts[1] != "" {
		anchor := parts[1]
		startPos = findAnchorPosition(html, anchor)
	}

	// Determine end position from next chapter anchor (only if in same file)
	endPos := len(html)
	if nextHref != "" {
		nextParts := strings.SplitN(nextHref, "#", 2)
		nextFileName := nextParts[0]
		// Only set end boundary if next chapter is in the same file
		if nextFileName == fileName && len(nextParts) > 1 && nextParts[1] != "" {
			nextAnchor := nextParts[1]
			nextPos := findAnchorPosition(html, nextAnchor)
			if nextPos > startPos {
				endPos = nextPos
			}
		}
	}

	// Extract the chapter HTML
	chapterHTML := html[startPos:endPos]

	// Convert HTML to plain text
	return htmlToText(chapterHTML)
}

// findAnchorPosition finds the position of an anchor ID in HTML
// Returns the position of the opening tag containing the ID
func findAnchorPosition(html, anchor string) int {
	// Find the element with this ID
	idPattern := fmt.Sprintf(`id="%s"`, anchor)
	idPos := strings.Index(html, idPattern)
	if idPos == -1 {
		// Try with single quotes
		idPattern = fmt.Sprintf(`id='%s'`, anchor)
		idPos = strings.Index(html, idPattern)
	}
	if idPos == -1 {
		return 0 // anchor not found, return start
	}

	// Find the start of the tag containing this ID
	tagStart := strings.LastIndex(html[:idPos], "<")
	if tagStart == -1 {
		return idPos
	}

	return tagStart
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

// extractCoverHref finds the cover image href from the EPUB package
// Based on Python EBook_audiobook_creator logic
func extractCoverHref(pkg epubPackage, baseDir string) string {
	// Look for cover meta tag
	// In the Python version: <meta name="cover" content="cover-id"/>
	// Then find the item with that id in manifest

	// Build a map of manifest items by ID
	itemsByID := make(map[string]struct {
		Href      string
		MediaType string
	})
	for _, item := range pkg.Manifest.Items {
		itemsByID[item.ID] = struct {
			Href      string
			MediaType string
		}{item.Href, item.MediaType}
	}

	// Look for items that might be cover images
	for _, item := range pkg.Manifest.Items {
		id := strings.ToLower(item.ID)
		href := strings.ToLower(item.Href)
		if (strings.Contains(id, "cover") || strings.Contains(href, "cover")) &&
			(item.MediaType == "image/jpeg" || item.MediaType == "image/png" ||
				item.MediaType == "image/jpg") {
			return filepath.Join(baseDir, item.Href)
		}
	}

	return ""
}

// htmlToText converts HTML content to plain text
// Based on Python html2text library behavior
// Uses pre-compiled regex patterns for performance
func htmlToText(html string) string {
	// Remove head section (contains title, meta, etc.)
	html = reHead.ReplaceAllString(html, "")

	// Remove script and style tags with content
	html = reScript.ReplaceAllString(html, "")
	html = reStyle.ReplaceAllString(html, "")

	// Replace common block elements with newlines
	html = reBlock.ReplaceAllString(html, "\n")
	html = reBr.ReplaceAllString(html, "\n")

	// Remove all remaining HTML tags
	text := reTags.ReplaceAllString(html, "")

	// Decode all HTML entities (numeric like &#8197; and named like &nbsp;)
	text = htmlPkg.UnescapeString(text)
	// Replace non-breaking space with regular space
	text = strings.ReplaceAll(text, "\u00A0", " ")

	// Clean up whitespace (collapse 3+ newlines to single newline for tighter spacing)
	text = reSpaces.ReplaceAllString(text, " ")
	text = reNewlines.ReplaceAllString(text, "\n")

	// Add period at end of paragraphs if missing
	text = addPeriodToText(text)

	return strings.TrimSpace(text)
}

// addPeriodToText adds periods at the end of paragraphs that don't have punctuation
// Based on Python add_period function
func addPeriodToText(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			result = append(result, "")
			continue
		}
		// Get last rune to handle multi-byte characters
		runes := []rune(line)
		lastRune := runes[len(runes)-1]
		// Check for sentence-ending punctuation (including curly quotes U+201C and U+201D)
		if lastRune != '.' && lastRune != '?' && lastRune != '!' &&
			lastRune != ':' && lastRune != '"' && lastRune != 0x201C && lastRune != 0x201D {
			// Check for ellipsis
			if !strings.HasSuffix(line, "...") {
				line = line + "."
			}
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}
