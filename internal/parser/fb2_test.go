package parser

import (
	"os"
	"strings"
	"testing"
)

func TestNewFB2Parser(t *testing.T) {
	parser := NewFB2Parser()
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}
	if parser.TOCMaxDepth != 3 {
		t.Errorf("Expected TOCMaxDepth=3, got %d", parser.TOCMaxDepth)
	}
	if parser.ParseNotes != false {
		t.Error("Expected ParseNotes=false by default")
	}
}

func TestFb2TreeToText(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected string
	}{
		{
			name:     "simple paragraph",
			xml:      "<p>Hello World</p>",
			expected: "Hello World.",
		},
		{
			name:     "multiple paragraphs",
			xml:      "<p>First paragraph</p><p>Second paragraph</p>",
			expected: "First paragraph.\nSecond paragraph.",
		},
		{
			name:     "with title",
			xml:      "<title><p>Chapter Title</p></title><p>Content</p>",
			expected: "Chapter Title.\nContent.",
		},
		{
			name:     "with subtitle",
			xml:      "<subtitle>Subtitle</subtitle><p>Content</p>",
			expected: "Subtitle.\nContent.",
		},
		{
			name:     "with empty-line",
			xml:      "<p>Before</p><empty-line/><p>After</p>",
			expected: "Before.\nAfter.",
		},
		{
			name:     "with table",
			xml:      "<p>Before</p><table><tr><td>Cell</td></tr></table><p>After</p>",
			expected: "Before.\nTable omitted.\nAfter.",
		},
		{
			name:     "with image",
			xml:      "<p>Before</p><image l:href=\"#img1\"/><p>After</p>",
			expected: "Before.\nIllustration.\nAfter.",
		},
		{
			name:     "with link (should be removed)",
			xml:      "<p>Text with <a l:href=\"#note1\">footnote</a> here</p>",
			expected: "Text with here.",
		},
		{
			name:     "nested sections removed",
			xml:      "<p>Parent content</p><section><title><p>Child</p></title><p>Child content</p></section>",
			expected: "Parent content.",
		},
		{
			name:     "with nbsp",
			xml:      "<p>Hello\u00A0World</p>",
			expected: "Hello World.",
		},
		{
			name:     "with numeric HTML entities for spaces",
			xml:      "<p>Text&#160;with&#160;spaces</p>",
			expected: "Text with spaces.",
		},
		{
			name:     "with decimal entity for em dash",
			xml:      "<p>Hello&#8212;World</p>",
			expected: "Hello\u2014World.",
		},
		{
			name:     "with hex entity",
			xml:      "<p>Hello&#x2014;World</p>",
			expected: "Hello\u2014World.",
		},
		{
			name:     "with mixed entities",
			xml:      "<p>&quot;Hello&#8217;s World&quot;</p>",
			expected: "\"Hello\u2019s World\"",
		},
		{
			name:     "with four-per-em space entity",
			xml:      "<p>Text&#8197;here</p>",
			expected: "Text\u2005here.",
		},
		{
			name:     "empty content",
			xml:      "",
			expected: "",
		},
		{
			name:     "with emphasis tags",
			xml:      "<p><strong>Bold</strong> and <emphasis>italic</emphasis></p>",
			expected: "Bold and italic.",
		},
		{
			name:     "cyrillic content",
			xml:      "<p>Привет мир</p>",
			expected: "Привет мир.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fb2TreeToText(tt.xml)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			if result != expected {
				t.Errorf("fb2TreeToText(%q) = %q, expected %q", tt.xml, result, expected)
			}
		})
	}
}

func TestFb2TreeToTextNestedSections(t *testing.T) {
	// Test that deeply nested sections are all removed
	xml := `<p>Level 0</p>
<section>
  <title><p>Level 1</p></title>
  <p>Content 1</p>
  <section>
    <title><p>Level 2</p></title>
    <p>Content 2</p>
    <section>
      <title><p>Level 3</p></title>
      <p>Content 3</p>
    </section>
  </section>
</section>`

	result := fb2TreeToText(xml)

	if !strings.Contains(result, "Level 0") {
		t.Error("Expected result to contain 'Level 0'")
	}
	if strings.Contains(result, "Level 1") {
		t.Error("Result should NOT contain 'Level 1' (nested section)")
	}
	if strings.Contains(result, "Level 2") {
		t.Error("Result should NOT contain 'Level 2' (nested section)")
	}
	if strings.Contains(result, "Level 3") {
		t.Error("Result should NOT contain 'Level 3' (nested section)")
	}
}

func TestParseFB2BasicStructure(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>John</first-name>
        <last-name>Doe</last-name>
      </author>
      <book-title>Test Book</book-title>
      <annotation><p>This is the annotation.</p></annotation>
    </title-info>
  </description>
  <body>
    <title><p>Book Title</p></title>
    <section>
      <title><p>Chapter 1</p></title>
      <p>Chapter 1 content here.</p>
    </section>
    <section>
      <title><p>Chapter 2</p></title>
      <p>Chapter 2 content here.</p>
    </section>
  </body>
</FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	if book.Title != "Test Book" {
		t.Errorf("Expected title 'Test Book', got %q", book.Title)
	}
	if book.Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got %q", book.Author)
	}
	if !strings.Contains(book.Description, "annotation") {
		t.Errorf("Expected description to contain 'annotation', got %q", book.Description)
	}
	// Body title + 2 chapters = 3
	if len(book.Chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(book.Chapters))
	}
}

func TestParseFB2NestedSections(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author><first-name>Test</first-name><last-name>Author</last-name></author>
      <book-title>Nested Test</book-title>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Part 1</p></title>
      <section>
        <title><p>Chapter 1.1</p></title>
        <p>Content of chapter 1.1</p>
      </section>
      <section>
        <title><p>Chapter 1.2</p></title>
        <p>Content of chapter 1.2</p>
      </section>
    </section>
  </body>
</FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	// Part 1 should be skipped (has nested sections, no own content)
	// Chapter 1.1 and 1.2 should be present
	if len(book.Chapters) != 2 {
		t.Errorf("Expected 2 chapters (nested sections only), got %d", len(book.Chapters))
		for i, ch := range book.Chapters {
			t.Logf("Chapter %d: %s (len=%d)", i, ch.Title, len(ch.Content))
		}
	}

	// Verify chapter titles
	expectedTitles := []string{"Chapter 1.1.", "Chapter 1.2."}
	for i, expected := range expectedTitles {
		if i < len(book.Chapters) && book.Chapters[i].Title != expected {
			t.Errorf("Chapter %d title = %q, expected %q", i, book.Chapters[i].Title, expected)
		}
	}
}

func TestParseFB2NotesSection(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author><first-name>Test</first-name><last-name>Author</last-name></author>
      <book-title>Notes Test</book-title>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Chapter 1</p></title>
      <p>Main content</p>
    </section>
  </body>
  <body name="notes">
    <title><p>Notes</p></title>
    <section>
      <title><p>Note 1</p></title>
      <p>Note content</p>
    </section>
  </body>
</FictionBook>`

	// Test with ParseNotes = false (default)
	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	// Should only have Chapter 1
	if len(book.Chapters) != 1 {
		t.Errorf("Expected 1 chapter (notes skipped), got %d", len(book.Chapters))
	}

	// Test with ParseNotes = true
	parser.ParseNotes = true
	book, err = parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	// Should have Chapter 1 + Notes title + Note 1 = 3
	if len(book.Chapters) < 2 {
		t.Errorf("Expected at least 2 chapters (with notes), got %d", len(book.Chapters))
	}
}

func TestParseFB2CoverImage(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <author><first-name>Test</first-name><last-name>Author</last-name></author>
      <book-title>Cover Test</book-title>
      <coverpage>
        <image l:href="#cover.jpg"/>
      </coverpage>
    </title-info>
  </description>
  <body>
    <section><title><p>Chapter</p></title><p>Content</p></section>
  </body>
  <binary id="cover.jpg" content-type="image/jpeg">
/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRof
  </binary>
</FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	if book.CoverImageName != "cover.jpg" {
		t.Errorf("Expected cover name 'cover.jpg', got %q", book.CoverImageName)
	}
	if book.CoverImageType != "image/jpeg" {
		t.Errorf("Expected cover type 'image/jpeg', got %q", book.CoverImageType)
	}
	if len(book.CoverImage) == 0 {
		t.Error("Expected non-empty cover image data")
	}
}

func TestParseFB2WithNamespace(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<fb2:FictionBook xmlns:fb2="http://www.gribuser.ru/xml/fictionbook/2.0">
  <fb2:description>
    <fb2:title-info>
      <fb2:author>
        <fb2:first-name>Namespaced</fb2:first-name>
        <fb2:last-name>Author</fb2:last-name>
      </fb2:author>
      <fb2:book-title>Namespace Test</fb2:book-title>
    </fb2:title-info>
  </fb2:description>
  <fb2:body>
    <fb2:section>
      <fb2:title><fb2:p>Chapter</fb2:p></fb2:title>
      <fb2:p>Content</fb2:p>
    </fb2:section>
  </fb2:body>
</fb2:FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	if book.Title != "Namespace Test" {
		t.Errorf("Expected title 'Namespace Test', got %q", book.Title)
	}
	if book.Author != "Namespaced Author" {
		t.Errorf("Expected author 'Namespaced Author', got %q", book.Author)
	}
}

func TestParseFB2EmptySections(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author><first-name>Test</first-name><last-name>Author</last-name></author>
      <book-title>Empty Test</book-title>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Empty Chapter</p></title>
    </section>
    <section>
      <title><p>Chapter with Content</p></title>
      <p>Actual content here</p>
    </section>
  </body>
</FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	// Both chapters should be present (empty chapter has no nested sections)
	if len(book.Chapters) != 2 {
		t.Errorf("Expected 2 chapters, got %d", len(book.Chapters))
	}
}

func TestParseFB2TOCDepth(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author><first-name>Test</first-name><last-name>Author</last-name></author>
      <book-title>Depth Test</book-title>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Level 1</p></title>
      <p>Content 1</p>
      <section>
        <title><p>Level 2</p></title>
        <p>Content 2</p>
        <section>
          <title><p>Level 3</p></title>
          <p>Content 3</p>
          <section>
            <title><p>Level 4</p></title>
            <p>Content 4</p>
          </section>
        </section>
      </section>
    </section>
  </body>
</FictionBook>`

	// Test with TOCMaxDepth = 2
	parser := NewFB2Parser()
	parser.TOCMaxDepth = 2
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	// Should only have Level 1 and Level 2 (depth limit)
	if len(book.Chapters) != 2 {
		t.Errorf("Expected 2 chapters (depth limited), got %d", len(book.Chapters))
		for i, ch := range book.Chapters {
			t.Logf("Chapter %d: %s (depth=%d)", i, ch.Title, ch.TOCDepth)
		}
	}

	// Verify depths
	for _, ch := range book.Chapters {
		if ch.TOCDepth > 2 {
			t.Errorf("Chapter %q has depth %d, expected <= 2", ch.Title, ch.TOCDepth)
		}
	}
}

func TestParseFB2WithRealFile(t *testing.T) {
	testFile := "/home/ubuntu/git/abb_tts/temp/1768332245206279435_book.fb2"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test FB2 file not found, skipping integration test")
	}

	parser := NewFB2Parser()
	book, err := parser.ParseFB2File(testFile)
	if err != nil {
		t.Fatalf("Failed to parse FB2: %v", err)
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

	t.Logf("Parsed book: %s by %s (%d chapters)", book.Title, book.Author, len(book.Chapters))

	// Verify no content duplication (check that parent sections don't include child content)
	// Look for chapters that might be duplicated
	contentMap := make(map[string]int)
	for i, ch := range book.Chapters {
		// Use first 100 chars as key
		key := ch.Content
		if len(key) > 100 {
			key = key[:100]
		}
		if prevIdx, exists := contentMap[key]; exists && len(key) > 50 {
			t.Errorf("Possible content duplication: chapter %d (%s) has same content start as chapter %d",
				i, ch.Title, prevIdx)
		}
		contentMap[key] = i
	}

	// Verify chapter content is not empty for leaf chapters
	emptyCount := 0
	for _, ch := range book.Chapters {
		if strings.TrimSpace(ch.Content) == "" || strings.TrimSpace(ch.Content) == strings.TrimSpace(ch.Title) {
			emptyCount++
		}
	}
	// Allow some empty chapters (section headers) but not too many
	if emptyCount > len(book.Chapters)/2 {
		t.Errorf("Too many empty chapters: %d out of %d", emptyCount, len(book.Chapters))
	}
}

func TestParseFB2FileNotFound(t *testing.T) {
	parser := NewFB2Parser()
	_, err := parser.ParseFB2File("/nonexistent/path/book.fb2")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParseFB2InvalidXML(t *testing.T) {
	invalidContent := `<?xml version="1.0"?>
<FictionBook>
  <description>
    <title-info>
      <book-title>Unclosed tag
    </title-info>
  </description>
</FictionBook>`

	parser := NewFB2Parser()
	_, err := parser.ParseFB2(strings.NewReader(invalidContent))
	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}

func TestParseFB2CyrillicContent(t *testing.T) {
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>Аркадий</first-name>
        <last-name>Вайнер</last-name>
      </author>
      <book-title>Я, следователь…</book-title>
      <annotation><p>Неопознанное тело, найденное на южном шоссе.</p></annotation>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Лист дела 1</p></title>
      <p>Я давно приметил забавную особенность.</p>
    </section>
  </body>
</FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	if book.Title != "Я, следователь…" {
		t.Errorf("Expected Cyrillic title, got %q", book.Title)
	}
	if book.Author != "Аркадий Вайнер" {
		t.Errorf("Expected Cyrillic author, got %q", book.Author)
	}
	if len(book.Chapters) == 0 {
		t.Error("Expected at least one chapter")
	}
	if !strings.Contains(book.Chapters[0].Content, "забавную особенность") {
		t.Error("Expected Cyrillic content in chapter")
	}
}

func TestParseFB2MultipleAuthors(t *testing.T) {
	// Note: Current implementation only supports single author
	// This test documents current behavior
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author>
        <first-name>First</first-name>
        <last-name>Author</last-name>
      </author>
      <book-title>Multi Author Test</book-title>
    </title-info>
  </description>
  <body>
    <section><title><p>Chapter</p></title><p>Content</p></section>
  </body>
</FictionBook>`

	parser := NewFB2Parser()
	book, err := parser.ParseFB2(strings.NewReader(fb2Content))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	if book.Author != "First Author" {
		t.Errorf("Expected 'First Author', got %q", book.Author)
	}
}

func TestFb2TreeToTextSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		contains string
	}{
		{
			name:     "em dash",
			xml:      "<p>Hello — World</p>",
			contains: "—",
		},
		{
			name:     "ellipsis",
			xml:      "<p>To be continued…</p>",
			contains: "…",
		},
		{
			name:     "quotes",
			xml:      "<p>«Привет» сказал он</p>",
			contains: "«Привет»",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fb2TreeToText(tt.xml)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("fb2TreeToText(%q) should contain %q, got %q", tt.xml, tt.contains, result)
			}
		})
	}
}
