// Package router handles deterministic commands (like file search) without
// calling the LLM at all — faster and lighter on an 8GB CPU-only machine.
package router

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Kolong-Meja/george/internal/config"
)

// searchPattern matches Indonesian and English phrasing for file/folder search requests,
// e.g. "cariin file ABC.txt", "cari folder Downloads", "find file report.pdf".
var searchPattern = regexp.MustCompile(`(?i)(?:cari(?:in)?|find)\s+(?:file|folder)\s+(\S+)`)

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
	if m := searchPattern.FindStringSubmatch(input); m != nil {
		return Result{Handled: true, Output: searchFile(m[1], cfg)}
	}
	return Result{Handled: false}
}

// searchFile looks for a file/folder by name under $HOME.
// Swap "/home" for a narrower root, or make it configurable, once you know which
// directories you actually want George searching.
func searchFile(name string, cfg config.Config) string {
	out, err := exec.Command("find", "/home", "-iname", name).Output()
	if err != nil {
		if cfg.Language == config.English {
			return fmt.Sprintf("Couldn't search for '%s', bro: %v", name, err)
		}
		return fmt.Sprintf("Gagal nyari '%s' nih bro: %v", name, err)
	}

	lines := strings.TrimSpace(string(out))
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
