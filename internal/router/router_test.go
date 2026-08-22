package router

import "testing"

func TestWantsDetail(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"Tolong jelasin ke gw apa itu Jerman dengan detail", true},
		{"jelasin dong secara rinci soal itu", true},
		{"explain that in detail please", true},
		{"apa itu Indonesia?", false},
		{"cariin file report.pdf", false},
	}
	for _, c := range cases {
		if got := WantsDetail(c.input); got != c.want {
			t.Errorf("WantsDetail(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
