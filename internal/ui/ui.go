// Package ui renders George's replies for the terminal so a long answer doesn't
// just scroll past as a wall of text, and so a reply picks up a bit of the same
// emoji-flavored warmth Faisal's greeting/closing pools in config.go already have -
// this package extends that same feel to ordinary LLM replies, which had none.
package ui

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// boxWidth is the fixed interior width (in runes) used to wrap and border a long
// reply. A terminal-width-aware version would need golang.org/x/term (or a raw
// ioctl syscall) to query the real column count - both are outside George's
// current "standard library only, nothing to go-get" design (see README), so a
// safe fixed width is used instead. 64 fits comfortably even in a half-split
// Tilix pane.
const boxWidth = 64

// boxThreshold decides when a reply is "long enough" to box up instead of printing
// as a plain "George: ..." line - either long in raw character count, or already
// multi-line (e.g. router's file-search results, one path per line - those read
// far better boxed than run together after a single prefix).
const boxThreshold = 220

const (
	cyan  = "\x1b[36m"
	reset = "\x1b[0m"
)

// PrintReply prints one of George's replies: a plain colored line for a short
// answer, or a bordered box for a long/multi-line one, prefixed with an emoji
// picked deterministically from the conversation's own words (input + reply) -
// same keyword-matching approach as router.go's other classifiers, rather than
// asking the 3B model to choose its own emoji, which risks an off-tone pick that
// has nothing to do with what was actually said.
func PrintReply(input, reply string) {
	label := cyan + "George" + reset
	if emoji := pickEmoji(input + " " + reply); emoji != "" {
		label += " " + emoji
	}

	if shouldBox(reply) {
		fmt.Println(label + ":")
		fmt.Println(renderBox(reply))
		return
	}

	fmt.Println(label+":", reply)
}

func shouldBox(reply string) bool {
	return utf8.RuneCountInString(reply) > boxThreshold || strings.Count(reply, "\n") >= 2
}

// renderBox wraps body to boxWidth and borders it with box-drawing characters.
// Each existing line in body (router's search results are already newline-
// separated, one path per line) is wrapped independently, so a result list keeps
// one path per visual row instead of getting run together and re-wrapped mid-path.
//
// Known limitation: padding is computed by rune count, not display width, so a
// line containing a double-width emoji can misalign the right border by a
// character or two. Not fixed here to avoid pulling in a width-aware dependency
// for a cosmetic edge case - flag if it bugs you and we can revisit.
func renderBox(body string) string {
	top := "┌" + strings.Repeat("─", boxWidth+2) + "┐"
	bottom := "└" + strings.Repeat("─", boxWidth+2) + "┘"

	var out strings.Builder
	out.WriteString(top + "\n")
	for _, paragraph := range strings.Split(body, "\n") {
		for _, line := range wrapText(paragraph, boxWidth) {
			pad := boxWidth - utf8.RuneCountInString(line)
			if pad < 0 {
				pad = 0
			}
			out.WriteString("│ " + line + strings.Repeat(" ", pad) + " │\n")
		}
	}
	out.WriteString(bottom)
	return out.String()
}

// wrapText greedily word-wraps s to at most width runes per line. Always returns
// at least one (possibly empty) line, so renderBox never collapses an empty
// paragraph into zero border rows.
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, 4)
	current := words[0]
	for _, w := range words[1:] {
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(w) > width {
			lines = append(lines, current)
			current = w
			continue
		}
		current += " " + w
	}
	return append(lines, current)
}

// emojiRules is checked top to bottom; the first match wins. Order matters - more
// specific moods (thanks, sad, happy) come before the broad "?" fallback so a
// grateful or emotional message doesn't just get a generic 🤔.
var emojiRules = []struct {
	pattern *regexp.Regexp
	emoji   string
}{
	{regexp.MustCompile(`(?i)\b(makasih|mksh|thanks?|thx|terima\s*kasih)\b`), "🙌"},
	{regexp.MustCompile(`(?i)\b(sedih|capek|cape|lelah|down|stress|stres|galau)\b`), "🫂"},
	{regexp.MustCompile(`(?i)\b(seneng|senang|happy|mantap|asik|asyik|sukses)\b`), "🎉"},
	{regexp.MustCompile(`(?i)\b(code|coding|bug|error|debug|ngoding|programming)\b`), "💻"},
	{regexp.MustCompile(`(?i)\b(file|folder|cari(?:in|kan)?|find|search)\b`), "📁"},
	{regexp.MustCompile(`(?i)\b(makan|lapar|laper|kenyang|dinner|lunch)\b`), "🍽️"},
	{regexp.MustCompile(`(?i)\b(negara|dunia|indonesia|amerika|sejarah|geografi|country)\b`), "🌍"},
	{regexp.MustCompile(`\?`), "🤔"},
}

func pickEmoji(text string) string {
	for _, rule := range emojiRules {
		if rule.pattern.MatchString(text) {
			return rule.emoji
		}
	}
	return ""
}
