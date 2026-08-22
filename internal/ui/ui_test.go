package ui

import "testing"

func TestPickEmoji(t *testing.T) {
	cases := []struct{ text, want string }{
		{"makasih ya george", "🙌"},
		{"kenapa error mulu sih bug-nya", "💻"},
		{"apa itu Indonesia?", "🌍"},
		{"cariin file report.pdf", "📁"},
		{"halo", ""},
	}
	for _, c := range cases {
		if got := pickEmoji(c.text); got != c.want {
			t.Errorf("pickEmoji(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	for _, l := range wrapText("satu dua tiga empat lima", 11) {
		if len([]rune(l)) > 11 {
			t.Errorf("line %q exceeds width 11", l)
		}
	}
}
