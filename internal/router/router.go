// Package router handles deterministic commands (like file search) without
// calling the LLM at all — faster and lighter on an 8GB CPU-only machine.
package router

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Kolong-Meja/george/internal/config"
)

// searchPattern matches Indonesian and English phrasing for file/folder search requests,
// e.g. "cariin file ABC.txt", "cari folder Downloads", "find file report.pdf".
var searchPattern = regexp.MustCompile(`(?i)(?:cari(?:in)?|find)\s+(?:file|folder)\s+(\S+)`)

// negationWords are words that flip the meaning of the phrase right before the search
// verb, e.g. "nggak mau cari file apapun" - the user is declining a search, not asking
// for one. Exact whole-word matches only (via strings.Fields), so a word that merely
// *contains* one of these as a substring - like "juga" containing "ga" - never
// false-triggers.
var negationWords = map[string]bool{
	"nggak": true, "ga": true, "gak": true, "enggak": true, "kagak": true,
	"tidak": true, "jangan": true, "bukan": true, "ogah": true,
	"males": true, "malas": true,
	"don't": true, "dont": true, "won't": true, "wont": true, "not": true,
}

// nonFileWords are generic pronouns/fillers that sometimes land in the capture group
// when "cari file" is used loosely in conversation rather than as an actual command
// (e.g. "cari file apapun", "cari file itu"). Treating these as unhandled lets the
// sentence fall through to the LLM instead of George trying to `find` a file
// literally named "apapun".
var nonFileWords = map[string]bool{
	"apapun": true, "apaan": true, "apa": true, "itu": true, "ini": true,
	"aja": true, "saja": true, "semua": true, "semuanya": true,
	"anything": true, "something": true, "that": true, "this": true,
}

// Result represents the outcome of trying a rule-based command.
type Result struct {
	Handled bool
	Output  string
}

// TryHandle checks input against known deterministic patterns before it reaches the LLM.
// Handled=false means nothing matched, so the caller should fall back to the model.
// cfg is used only to keep George's tone (language + name) consistent with the rest
// of the app - it does not change which patterns match or how they're handled.
func TryHandle(input string, cfg config.Config) Result {
	loc := searchPattern.FindStringSubmatchIndex(input)
	if loc == nil {
		return Result{Handled: false}
	}

	// A negation word in the few words right before the match means the user is
	// declining a search, not requesting one - fall through to the LLM instead.
	if hasNearbyNegation(input[:loc[0]]) {
		return Result{Handled: false}
	}

	name := strings.TrimRight(input[loc[2]:loc[3]], ",.!?;:\"'")
	if nonFileWords[strings.ToLower(name)] {
		return Result{Handled: false}
	}

	return Result{Handled: true, Output: searchFile(name, cfg)}
}

// hasNearbyNegation reports whether one of the last few words in `before` is a
// negation word. Only a small window (last 4 words) is checked, so an unrelated
// "nggak" earlier in a long sentence doesn't block a genuine request later in the
// same sentence.
func hasNearbyNegation(before string) bool {
	words := strings.Fields(strings.ToLower(before))
	start := max(0, len(words)-4)
	for _, w := range words[start:] {
		if negationWords[strings.Trim(w, ",.!?;:\"'")] {
			return true
		}
	}
	return false
}

// searchFile looks for a file/folder by name under $HOME.
// Swap "/home" for a narrower root, or make it configurable, once you know which
// directories you actually want George searching.
// searchFile looks for a file/folder by name under $HOME.
func searchFile(name string, cfg config.Config) string {
	out, err := exec.Command("find", "/home", "-iname", name).Output()
	lines := strings.TrimSpace(string(out))

	// find(1) exits non-zero the moment it can't descend into ANY subdirectory
	// while walking the tree - another user's home folder, ~/.cache, ~/.var app
	// sandboxes, etc. That's routine and near-universal under /home, and it's an
	// *exec.ExitError: find still ran and reported whatever it legitimately could,
	// it just also hit something it wasn't allowed to open. That says nothing about
	// whether the target file exists, so it must never be treated as a search
	// failure - not when a match still came through on stdout (the original bug:
	// Output() keeps the matched bytes even on a non-zero exit, but the old code
	// discarded them anyway), and not when nothing matched either (still just means
	// "not found," not "broken").
	//
	// A real failure is when the *find command itself* couldn't run at all - e.g.
	// the binary is missing from PATH. That surfaces as a different error type
	// (*exec.Error, not *exec.ExitError), so it's the only case that still reaches
	// the hard-error message below.
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		if cfg.Language == config.English {
			return fmt.Sprintf("Couldn't search for '%s', bro: %v", name, err)
		}
		return fmt.Sprintf("Gagal nyari '%s' nih bro: %v", name, err)
	}

	if lines == "" {
		if cfg.Language == config.English {
			return fmt.Sprintf("Couldn't find anything named '%s', bro %s.", name, cfg.UserName)
		}
		return fmt.Sprintf("Nggak ketemu file/folder bernama '%s' nih, bro %s.", name, cfg.UserName)
	}

	if cfg.Language == config.English {
		return fmt.Sprintf("Found it, bro %s:\n%s", cfg.UserName, lines)
	}
	return fmt.Sprintf("Ketemu bro %s, ini dia:\n%s", cfg.UserName, lines)
}
