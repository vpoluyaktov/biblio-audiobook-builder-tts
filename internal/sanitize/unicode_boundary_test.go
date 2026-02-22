package sanitize

import (
	"testing"
)

func TestApplyRuleUnicode_CyrillicAbbreviations(t *testing.T) {
	s := NewTextSanitizer()
	s.LoadDefaultRules()

	tests := []struct {
		input string
		want  string
	}{
		{
			"атаки террористов на США одиннадцатого сентября",
			"атаки террористов на сэ шэ а одиннадцатого сентября",
		},
		{
			"президент США Кеннеди",
			"президент сэ шэ а Кеннеди",
		},
		{
			"в ФСБ сообщили",
			"в эф эс бэ сообщили",
		},
		{
			"СССР и США в холодной войне",
			"эс эс эс эр и сэ шэ а в холодной войне",
		},
	}

	for _, tt := range tests {
		got := s.SanitizeWithOptions(tt.input, "ru", false)
		if got != tt.want {
			t.Errorf("SanitizeWithOptions(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.want)
		}
	}
}
