package sanitize

import (
"strings"
"testing"
)

func TestFileName_BasicSanitization(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Simple name", "hello", "hello"},
{"With spaces", "hello world", "hello_world"},
{"With slash", "hello/world", "hello_world"},
{"With backslash", "hello\\world", "hello_world"},
{"With colon", "hello:world", "hello_world"},
{"With asterisk", "hello*world", "hello_world"},
{"With question mark", "hello?world", "hello_world"},
{"With double quotes", "hello\"world", "hello_world"},
{"With less than", "hello<world", "hello_world"},
{"With greater than", "hello>world", "hello_world"},
{"With pipe", "hello|world", "hello_world"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_ShellSpecialCharacters(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Single quote", "hello'world", "hello_world"},
{"Backtick", "hello`world", "hello_world"},
{"Dollar sign", "hello$world", "hello_world"},
{"Ampersand", "hello&world", "hello_world"},
{"Semicolon", "hello;world", "hello_world"},
{"Left paren", "hello(world", "hello_world"},
{"Right paren", "hello)world", "hello_world"},
{"Left bracket", "hello[world", "hello_world"},
{"Right bracket", "hello]world", "hello_world"},
{"Left brace", "hello{world", "hello_world"},
{"Right brace", "hello}world", "hello_world"},
{"Exclamation", "hello!world", "hello_world"},
{"Hash", "hello#world", "hello_world"},
{"Tilde", "hello~world", "hello_world"},
{"Caret", "hello^world", "hello_world"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_WhitespaceHandling(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Tab", "hello\tworld", "hello_world"},
{"Newline", "hello\nworld", "hello_world"},
{"Carriage return", "hello\rworld", "hello_world"},
{"Multiple spaces", "hello   world", "hello_world"},
{"Mixed whitespace", "hello \t\n world", "hello_world"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_MultipleUnderscores(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Double underscore", "hello__world", "hello_world"},
{"Triple underscore", "hello___world", "hello_world"},
{"Multiple special chars", "hello!@#world", "hello_world"},
{"Spaces and special", "hello ! world", "hello_world"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_TrimEnds(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Leading underscore", "_hello", "hello"},
{"Trailing underscore", "hello_", "hello"},
{"Both underscores", "_hello_", "hello"},
{"Leading dot", ".hello", "hello"},
{"Trailing dot", "hello.", "hello"},
{"Both dots", ".hello.", "hello"},
{"Mixed leading", "_.hello", "hello"},
{"Mixed trailing", "hello._", "hello"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_LengthLimit(t *testing.T) {
t.Run("Long ASCII name truncated", func(t *testing.T) {
input := strings.Repeat("a", 150)
result := FileName(input)
if len(result) > 100 {
t.Errorf("FileName should truncate to 100 chars, got %d", len(result))
}
})

t.Run("Long UTF-8 name truncated by runes", func(t *testing.T) {
input := strings.Repeat("Я", 150)
result := FileName(input)
runes := []rune(result)
if len(runes) > 100 {
t.Errorf("FileName should truncate to 100 runes, got %d", len(runes))
}
})

t.Run("Exactly 100 chars unchanged", func(t *testing.T) {
input := strings.Repeat("a", 100)
result := FileName(input)
if result != input {
t.Errorf("100 char name should be unchanged")
}
})
}

func TestFileName_EmptyAndEdgeCases(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Empty string", "", "untitled"},
{"Only spaces", "   ", "untitled"},
{"Only special chars", "!@#$%", "untitled"},
{"Only underscores", "___", "untitled"},
{"Only dots", "...", "untitled"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_PreservesUnicode(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Russian", "Привет мир", "Привет_мир"},
{"Chinese", "你好世界", "你好世界"},
{"Japanese", "こんにちは", "こんにちは"},
{"Korean", "안녕하세요", "안녕하세요"},
{"Arabic", "مرحبا", "مرحبا"},
{"Hebrew", "שלום", "שלום"},
{"Greek", "Γειά", "Γειά"},
{"French accents", "café", "café"},
{"German umlauts", "Größe", "Größe"},
{"Mixed", "Hello Мир 世界", "Hello_Мир_世界"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_RealWorldExamples(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Book title with colon", "Pride and Prejudice: A Novel", "Pride_and_Prejudice_A_Novel"},
{"Author - Title format", "Jane Austen - Pride and Prejudice", "Jane_Austen_-_Pride_and_Prejudice"},
{"Russian book title", "Война и мир", "Война_и_мир"},
{"Title with quotes", "\"Hello World\"", "Hello_World"},
{"Title with parentheses", "Book (Volume 1)", "Book_Volume_1"},
{"Complex title", "Book: Part 1 - Chapter (1)", "Book_Part_1_-_Chapter_1"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestDirectoryName(t *testing.T) {
// DirectoryName is an alias for FileName
result := DirectoryName("hello world")
expected := "hello_world"
if result != expected {
t.Errorf("DirectoryName(%q) = %q, want %q", "hello world", result, expected)
}
}

func TestFileNameWithExtension(t *testing.T) {
tests := []struct {
name     string
input    string
ext      string
expected string
}{
{"Simple with ext", "hello", ".txt", "hello.txt"},
{"With spaces", "hello world", ".txt", "hello_world.txt"},
{"Ext without dot", "hello", "txt", "hello.txt"},
{"Empty ext", "hello", "", "hello"},
{"Complex name", "My Book: Chapter 1", ".mp3", "My_Book_Chapter_1.mp3"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileNameWithExtension(tt.input, tt.ext)
if result != tt.expected {
t.Errorf("FileNameWithExtension(%q, %q) = %q, want %q", tt.input, tt.ext, result, tt.expected)
}
})
}
}

func TestPathComponent(t *testing.T) {
tests := []struct {
name     string
input    string
expected string
}{
{"Simple name", "hello", "hello"},
{"Leading dot removed", ".hidden", "hidden"},
{"Multiple leading dots", "...test", "test"},
{"With spaces", "hello world", "hello_world"},
{"Empty becomes untitled", "", "untitled"},
{"Only dots", "...", "untitled"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := PathComponent(tt.input)
if result != tt.expected {
t.Errorf("PathComponent(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}

func TestFileName_CurlyQuotes(t *testing.T) {
// Test that Unicode curly quotes are also replaced
tests := []struct {
name     string
input    string
expected string
}{
{"Right single quote U+2019", "hello\u2019world", "hello_world"},
{"Left single quote U+2018", "hello\u2018world", "hello_world"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := FileName(tt.input)
if result != tt.expected {
t.Errorf("FileName(%q) = %q, want %q", tt.input, result, tt.expected)
}
})
}
}
