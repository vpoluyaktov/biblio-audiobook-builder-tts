package parser

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEpubParser(t *testing.T) {
	parser := NewEpubParser()
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}
}

func TestHtmlToText(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello World</p>",
			expected: "Hello World.",
		},
		{
			name:     "multiple paragraphs",
			html:     "<p>First paragraph</p><p>Second paragraph</p>",
			expected: "First paragraph.\nSecond paragraph.",
		},
		{
			name:     "with head section",
			html:     "<html><head><title>freeLib</title></head><body><p>Content</p></body></html>",
			expected: "Content.",
		},
		{
			name:     "with script tags",
			html:     "<p>Before</p><script>alert('test');</script><p>After</p>",
			expected: "Before.\nAfter.",
		},
		{
			name:     "with style tags",
			html:     "<style>.test { color: red; }</style><p>Content</p>",
			expected: "Content.",
		},
		{
			name:     "with br tags",
			html:     "Line 1<br/>Line 2<br>Line 3",
			expected: "Line 1.\nLine 2.\nLine 3.",
		},
		{
			name:     "with div tags",
			html:     "<div>Block 1</div><div>Block 2</div>",
			expected: "Block 1.\nBlock 2.",
		},
		{
			name:     "with headings",
			html:     "<h1>Title</h1><p>Content</p>",
			expected: "Title.\nContent.",
		},
		{
			name:     "with HTML entities",
			html:     "<p>Tom &amp; Jerry &lt;3&gt;</p>",
			expected: "Tom & Jerry <3>.",
		},
		{
			name:     "with nbsp",
			html:     "<p>Hello&nbsp;World</p>",
			expected: "Hello World.",
		},
		{
			name:     "with numeric HTML entities for spaces",
			html:     "<p>Text&#160;with&#160;spaces</p>",
			expected: "Text with spaces.",
		},
		{
			name:     "with decimal entity for em dash",
			html:     "<p>Hello&#8212;World</p>",
			expected: "Hello\u2014World.",
		},
		{
			name:     "with hex entity",
			html:     "<p>Hello&#x2014;World</p>",
			expected: "Hello\u2014World.",
		},
		{
			name:     "with mixed entities",
			html:     "<p>&quot;Hello&#8217;s World&quot;</p>",
			expected: "\"Hello\u2019s World\"",
		},
		{
			name:     "with four-per-em space entity",
			html:     "<p>Text&#8197;here</p>",
			expected: "Text\u2005here.",
		},
		{
			name:     "empty content",
			html:     "",
			expected: "",
		},
		{
			name:     "nested tags",
			html:     "<p><strong>Bold</strong> and <em>italic</em></p>",
			expected: "Bold and italic.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlToText(tt.html)
			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			if result != expected {
				t.Errorf("htmlToText(%q) = %q, expected %q", tt.html, result, expected)
			}
		})
	}
}

func TestFindAnchorPosition(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		anchor   string
		expected int
	}{
		{
			name:     "double quotes",
			html:     `<div id="chapter1">Content</div>`,
			anchor:   "chapter1",
			expected: 0,
		},
		{
			name:     "single quotes",
			html:     `<div id='chapter1'>Content</div>`,
			anchor:   "chapter1",
			expected: 0,
		},
		{
			name:     "anchor in middle",
			html:     `<div>Before</div><div id="chapter2">Content</div>`,
			anchor:   "chapter2",
			expected: 17,
		},
		{
			name:     "anchor not found",
			html:     `<div id="chapter1">Content</div>`,
			anchor:   "nonexistent",
			expected: 0,
		},
		{
			name:     "tocref anchor",
			html:     `<div class="section"><div id="tocref5">Chapter</div></div>`,
			anchor:   "tocref5",
			expected: 21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findAnchorPosition(tt.html, tt.anchor)
			if result != tt.expected {
				t.Errorf("findAnchorPosition(%q, %q) = %d, expected %d", tt.html, tt.anchor, result, tt.expected)
			}
		})
	}
}

func TestAddPeriodToText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no punctuation",
			input:    "Hello World",
			expected: "Hello World.",
		},
		{
			name:     "ends with period",
			input:    "Hello World.",
			expected: "Hello World.",
		},
		{
			name:     "ends with question mark",
			input:    "How are you?",
			expected: "How are you?",
		},
		{
			name:     "ends with exclamation",
			input:    "Hello!",
			expected: "Hello!",
		},
		{
			name:     "ends with colon",
			input:    "Note:",
			expected: "Note:",
		},
		{
			name:     "ends with ellipsis",
			input:    "To be continued...",
			expected: "To be continued...",
		},
		{
			name:     "ends with quote",
			input:    `He said "hello"`,
			expected: `He said "hello"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple lines",
			input:    "Line 1\nLine 2",
			expected: "Line 1.\nLine 2.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addPeriodToText(tt.input)
			if result != tt.expected {
				t.Errorf("addPeriodToText(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseNCXChapters(t *testing.T) {
	ncxContent := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="navpoint1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter1.html#ch1"/>
    </navPoint>
    <navPoint id="navpoint2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="chapter2.html"/>
      <navPoint id="navpoint3" playOrder="3">
        <navLabel><text>Section 2.1</text></navLabel>
        <content src="chapter2.html#sec1"/>
      </navPoint>
    </navPoint>
  </navMap>
</ncx>`

	chapters, err := parseNCXChapters([]byte(ncxContent))
	if err != nil {
		t.Fatalf("parseNCXChapters failed: %v", err)
	}

	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(chapters))
	}

	expectedChapters := []struct {
		title string
		href  string
	}{
		{"Chapter 1", "chapter1.html#ch1"},
		{"Chapter 2", "chapter2.html"},
		{"Section 2.1", "chapter2.html#sec1"},
	}

	for i, expected := range expectedChapters {
		if i >= len(chapters) {
			break
		}
		if chapters[i].Title != expected.title {
			t.Errorf("Chapter %d title = %q, expected %q", i, chapters[i].Title, expected.title)
		}
		if chapters[i].Href != expected.href {
			t.Errorf("Chapter %d href = %q, expected %q", i, chapters[i].Href, expected.href)
		}
	}
}

func TestParseNavChapters(t *testing.T) {
	navContent := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li><a href="chapter1.html">Chapter 1</a></li>
      <li>
        <a href="chapter2.html">Chapter 2</a>
        <ol>
          <li><a href="chapter2.html#sec1">Section 2.1</a></li>
        </ol>
      </li>
    </ol>
  </nav>
</body>
</html>`

	chapters, err := parseNavChapters([]byte(navContent))
	if err != nil {
		t.Fatalf("parseNavChapters failed: %v", err)
	}

	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(chapters))
	}
}

func TestReadEPUBChapterContentWithBoundary(t *testing.T) {
	// Create a mock zip file in memory
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add a test HTML file with multiple sections
	htmlContent := `<html>
<head><title>Test</title></head>
<body>
<div id="tocref1"><h1>Section 1</h1><p>Content of section 1</p></div>
<div id="tocref2"><h1>Section 2</h1><p>Content of section 2</p></div>
<div id="tocref3"><h1>Section 3</h1><p>Content of section 3</p></div>
</body>
</html>`

	f, _ := w.Create("OEBPS/chapter.html")
	f.Write([]byte(htmlContent))
	w.Close()

	// Open the zip for reading
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	// Test extracting section 1 with boundary at section 2
	content := readEPUBChapterContentWithBoundary(zr, "OEBPS", "chapter.html#tocref1", "chapter.html#tocref2")
	if !strings.Contains(content, "Section 1") {
		t.Error("Expected content to contain 'Section 1'")
	}
	if !strings.Contains(content, "Content of section 1") {
		t.Error("Expected content to contain 'Content of section 1'")
	}
	if strings.Contains(content, "Section 2") {
		t.Error("Content should NOT contain 'Section 2' (should be cut off at boundary)")
	}

	// Test extracting section 2 with boundary at section 3
	content = readEPUBChapterContentWithBoundary(zr, "OEBPS", "chapter.html#tocref2", "chapter.html#tocref3")
	if !strings.Contains(content, "Section 2") {
		t.Error("Expected content to contain 'Section 2'")
	}
	if strings.Contains(content, "Section 3") {
		t.Error("Content should NOT contain 'Section 3'")
	}

	// Test extracting last section with no boundary
	content = readEPUBChapterContentWithBoundary(zr, "OEBPS", "chapter.html#tocref3", "")
	if !strings.Contains(content, "Section 3") {
		t.Error("Expected content to contain 'Section 3'")
	}
}

func TestReadEPUBChapterContentWithBoundaryDifferentFiles(t *testing.T) {
	// Create a mock zip file with multiple HTML files
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// File 1
	f1, _ := w.Create("OEBPS/chapter1.html")
	f1.Write([]byte(`<html><body><div id="ch1"><p>Chapter 1 content</p></div></body></html>`))

	// File 2
	f2, _ := w.Create("OEBPS/chapter2.html")
	f2.Write([]byte(`<html><body><div id="ch2"><p>Chapter 2 content</p></div></body></html>`))

	w.Close()

	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	// When next chapter is in a different file, should get all content from current file
	content := readEPUBChapterContentWithBoundary(zr, "OEBPS", "chapter1.html#ch1", "chapter2.html#ch2")
	if !strings.Contains(content, "Chapter 1 content") {
		t.Error("Expected content to contain 'Chapter 1 content'")
	}
}

func TestHeadSectionStripping(t *testing.T) {
	html := `<html>
<head>
<title>freeLib</title>
<meta charset="utf-8"/>
<style>.test { color: red; }</style>
</head>
<body>
<p>Actual content here</p>
</body>
</html>`

	result := htmlToText(html)

	if strings.Contains(result, "freeLib") {
		t.Error("Result should NOT contain 'freeLib' from head section")
	}
	if !strings.Contains(result, "Actual content") {
		t.Error("Result should contain 'Actual content'")
	}
}

func TestParseEpubWithRealFile(t *testing.T) {
	// Test with the actual EPUB file if it exists
	testFile := "/home/ubuntu/git/abb_tts/temp/1768329835622054344_book.epub"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test EPUB file not found, skipping integration test")
	}

	parser := NewEpubParser()
	book, err := parser.ParseEpubFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse EPUB: %v", err)
	}

	// Verify basic metadata
	if book.Title == "" {
		t.Error("Expected non-empty book title")
	}
	if book.Author == "" {
		t.Error("Expected non-empty book author")
	}
	if len(book.Chapters) == 0 {
		t.Error("Expected at least one chapter")
	}

	// Verify no "freeLib" in any chapter content
	for i, ch := range book.Chapters {
		if strings.Contains(ch.Content, "freeLib") {
			t.Errorf("Chapter %d (%s) contains 'freeLib' which should be stripped", i, ch.Title)
		}
	}

	// Verify chapter content is not duplicated (parent chapters should have less content than children)
	// Find "Крым" chapter and "Лист дела 1" chapter
	var krymLen, listDela1Len int
	for _, ch := range book.Chapters {
		if ch.Title == "Крым" {
			krymLen = len(ch.Content)
		}
		if ch.Title == "Лист дела 1" {
			listDela1Len = len(ch.Content)
		}
	}

	if krymLen > 0 && listDela1Len > 0 {
		// "Крым" is a parent section and should have minimal content (just title)
		// "Лист дела 1" is the actual chapter with content
		if krymLen > listDela1Len {
			t.Errorf("Parent section 'Крым' (%d chars) should not have more content than child 'Лист дела 1' (%d chars)", krymLen, listDela1Len)
		}
	}
}

func TestExtractCoverHref(t *testing.T) {
	pkg := epubPackage{}
	pkg.Manifest.Items = []struct {
		ID        string `xml:"id,attr"`
		Href      string `xml:"href,attr"`
		MediaType string `xml:"media-type,attr"`
	}{
		{ID: "cover-image", Href: "images/cover.jpg", MediaType: "image/jpeg"},
		{ID: "chapter1", Href: "chapter1.html", MediaType: "application/xhtml+xml"},
	}

	result := extractCoverHref(pkg, "OEBPS")
	expected := filepath.Join("OEBPS", "images/cover.jpg")
	if result != expected {
		t.Errorf("extractCoverHref() = %q, expected %q", result, expected)
	}
}

func TestExtractCoverHrefPNG(t *testing.T) {
	pkg := epubPackage{}
	pkg.Manifest.Items = []struct {
		ID        string `xml:"id,attr"`
		Href      string `xml:"href,attr"`
		MediaType string `xml:"media-type,attr"`
	}{
		{ID: "cover", Href: "cover.png", MediaType: "image/png"},
	}

	result := extractCoverHref(pkg, "")
	if result != "cover.png" {
		t.Errorf("extractCoverHref() = %q, expected 'cover.png'", result)
	}
}

func TestExtractCoverHrefNoCover(t *testing.T) {
	pkg := epubPackage{}
	pkg.Manifest.Items = []struct {
		ID        string `xml:"id,attr"`
		Href      string `xml:"href,attr"`
		MediaType string `xml:"media-type,attr"`
	}{
		{ID: "chapter1", Href: "chapter1.html", MediaType: "application/xhtml+xml"},
	}

	result := extractCoverHref(pkg, "OEBPS")
	if result != "" {
		t.Errorf("extractCoverHref() = %q, expected empty string", result)
	}
}

func TestReadEPUBChapterContentNoAnchor(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, _ := w.Create("OEBPS/chapter.html")
	f.Write([]byte(`<html><body><p>Full chapter content</p></body></html>`))
	w.Close()

	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	// Test reading without anchor - should get full content
	content := readEPUBChapterContent(zr, "OEBPS", "chapter.html")
	if !strings.Contains(content, "Full chapter content") {
		t.Error("Expected content to contain 'Full chapter content'")
	}
}

func TestReadEPUBChapterContentFileNotFound(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	w.Close()

	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	// Test reading non-existent file
	content := readEPUBChapterContent(zr, "OEBPS", "nonexistent.html")
	if content != "" {
		t.Errorf("Expected empty content for non-existent file, got %q", content)
	}
}

func TestParseNCXChaptersEmpty(t *testing.T) {
	ncxContent := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
  </navMap>
</ncx>`

	chapters, err := parseNCXChapters([]byte(ncxContent))
	if err != nil {
		t.Fatalf("parseNCXChapters failed: %v", err)
	}

	if len(chapters) != 0 {
		t.Errorf("Expected 0 chapters, got %d", len(chapters))
	}
}

func TestParseNCXChaptersDeepNesting(t *testing.T) {
	ncxContent := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Level 1</text></navLabel>
      <content src="ch1.html"/>
      <navPoint id="np2" playOrder="2">
        <navLabel><text>Level 2</text></navLabel>
        <content src="ch1.html#l2"/>
        <navPoint id="np3" playOrder="3">
          <navLabel><text>Level 3</text></navLabel>
          <content src="ch1.html#l3"/>
          <navPoint id="np4" playOrder="4">
            <navLabel><text>Level 4</text></navLabel>
            <content src="ch1.html#l4"/>
          </navPoint>
        </navPoint>
      </navPoint>
    </navPoint>
  </navMap>
</ncx>`

	chapters, err := parseNCXChapters([]byte(ncxContent))
	if err != nil {
		t.Fatalf("parseNCXChapters failed: %v", err)
	}

	// Should get all 4 levels
	if len(chapters) != 4 {
		t.Errorf("Expected 4 chapters (all nesting levels), got %d", len(chapters))
	}

	expectedTitles := []string{"Level 1", "Level 2", "Level 3", "Level 4"}
	for i, expected := range expectedTitles {
		if i < len(chapters) && chapters[i].Title != expected {
			t.Errorf("Chapter %d title = %q, expected %q", i, chapters[i].Title, expected)
		}
	}
}

func TestHtmlToTextWithCyrillicContent(t *testing.T) {
	html := `<html>
<head><title>freeLib</title></head>
<body>
<p>Аннотация</p>
<p>Неопознанное тело, найденное на южном шоссе, оказывается лишь первым звеном в цепи.</p>
</body>
</html>`

	result := htmlToText(html)

	if strings.Contains(result, "freeLib") {
		t.Error("Result should NOT contain 'freeLib'")
	}
	if !strings.Contains(result, "Аннотация") {
		t.Error("Result should contain 'Аннотация'")
	}
	if !strings.Contains(result, "Неопознанное тело") {
		t.Error("Result should contain 'Неопознанное тело'")
	}
}

func TestFindAnchorPositionWithComplexHTML(t *testing.T) {
	html := `<div class="titleblock" id="tocref0"><div class="h0"><p class="css">Author Name</p></div></div>
<div class="section"><div class="titleblock" id="tocref1"><div class="h1"><p class="css">Chapter Title</p></div></div>
<div class="section"><div class="titleblock" id="tocref2"><div class="h2"><p class="css">Section Title</p></div></div>
<p class="text">Actual content here.</p></div></div>`

	// Find tocref1
	pos := findAnchorPosition(html, "tocref1")
	if pos == 0 {
		t.Error("Expected non-zero position for tocref1")
	}

	// Verify the position is at the start of the div containing tocref1 (the titleblock div)
	if !strings.HasPrefix(html[pos:], "<div class=\"titleblock\" id=\"tocref1\"") {
		t.Errorf("Position should be at start of titleblock div, got: %s", html[pos:pos+50])
	}
}
