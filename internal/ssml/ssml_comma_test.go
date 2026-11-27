package ssml

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func assertWellFormed(t *testing.T, fragment string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader("<speak>" + fragment + "</speak>"))
	for {
		if _, err := d.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Fatalf("malformed SSML %q: %v", fragment, err)
		}
	}
}

func TestCommaBreaks(t *testing.T) {
	opts := SSMLOptions{
		CommaBreakMs:          70,
		ColonBreakMs:          50,
		ParenthesesBreakMs:    40,
		ConvertDashesToBreaks: true,
		DashBreakDurationMs:   30,
		SentenceBreakMs:       10,
		ParagraphBreakMs:      20,
	}
	const comma = `<break time="70ms"/>`

	tests := []struct {
		name, text string
		commas     int
		breaks     int // all break tags
	}{
		{"simple", "Yes, indeed.", 1, 1},
		{"several", "One, two, three.", 2, 2},
		{"thousands separator", "It cost 1,000 dollars.", 0, 0},
		{"decimal comma", "Pi is 3,14 roughly.", 0, 0},
		{"unspaced words", "one,zero", 0, 0},
		{"number before spaced comma", "In 2020, 5 left.", 1, 1},
		{"dash after comma", "Well, - he said.", 0, 1},
		{"dash before comma", "Well - , he said.", 0, 1},
		{"ellipsis before comma", "Wait..., then go.", 0, 1},
		{"before parentheses", "Him, (the elder) left.", 0, 2},
		{"after parentheses", "Him (the elder), left.", 0, 2},
		{"inside parentheses", "Him (old, grey) left.", 1, 3},
		{"before colon", "Note, : this.", 0, 1},
		{"after colon", "Note:, this.", 0, 1},
		{"doubled comma", "So,, then.", 1, 1},
		{"spaced doubled comma", "So, , then.", 1, 1},
		{"end of sentence", "Hello,. Bye.", 0, 1},
		{"end of paragraph", "Hello,\nworld.", 0, 1},
		{"end of text", "Hello, ", 0, 0},
		{"start of text", ", hello", 0, 0},
		{"only punctuation", ", , ,", 0, 0},
		{"xml characters", `Tom, "Jerry" & <co>, left.`, 2, 2},
		{"cyrillic", "Да, конечно, это так.", 2, 2},
		{"empty", "", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddSSMLBreaks(tt.text, opts)
			if n := strings.Count(got, comma); n != tt.commas {
				t.Errorf("comma breaks = %d, want %d: %q", n, tt.commas, got)
			}
			if n := strings.Count(got, "<break "); n != tt.breaks {
				t.Errorf("total breaks = %d, want %d: %q", n, tt.breaks, got)
			}
			if strings.Contains(got, "\x01") {
				t.Errorf("placeholder left in %q", got)
			}
			assertWellFormed(t, got)
		})
	}
}

func TestCommaBreakKeepsComma(t *testing.T) {
	want := `Yes, <break time="70ms"/> indeed.`
	if got := AddSSMLBreaks("Yes, indeed.", SSMLOptions{CommaBreakMs: 70}); got != want {
		t.Errorf("AddSSMLBreaks = %q, want %q", got, want)
	}
	if got := WrapTextInSSMLWithOptions("Yes, indeed.", SSMLOptions{CommaBreakMs: 70}); got != "<speak>"+want+"</speak>" {
		t.Errorf("WrapTextInSSMLWithOptions = %q", got)
	}
}

func TestCommaBreakDisabled(t *testing.T) {
	for _, ms := range []int{0, -5} {
		if got := AddSSMLBreaks("Yes, indeed.", SSMLOptions{CommaBreakMs: ms}); got != "Yes, indeed." {
			t.Errorf("CommaBreakMs %d changed text: %q", ms, got)
		}
	}
}
