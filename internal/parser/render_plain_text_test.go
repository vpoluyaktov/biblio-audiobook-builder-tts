package parser

import "testing"

func TestRenderPlainText(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"empty", "", ""},
		{"blank lines only", "\n \n", ""},
		{"single line gets period", "Hello", "Hello."},
		{"punctuation kept", "Really?", "Really?"},
		{"lines become paragraphs", "Title\r\nFirst.\n\n\nSecond", "Title.\n\nFirst.\n\nSecond."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderPlainText(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("RenderPlainText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
