// Package router handles deterministic commands (like file search) without
// calling the LLM at all — faster and lighter on an 8GB CPU-only machine.
package router

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
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
func TryHandle(input string) Result {
	if m := searchPattern.FindStringSubmatch(input); m != nil {
		return Result{Handled: true, Output: searchFile(m[1])}
	}
	return Result{Handled: false}
}

// searchFile looks for a file/folder by name under $HOME.
// Swap "/home" for a narrower root, or make it configurable, once you know which
// directories you actually want George searching.
func searchFile(name string) string {
	out, err := exec.Command("find", "/home", "-iname", name).Output()
	if err != nil {
		return fmt.Sprintf("Gagal mencari '%s': %v", name, err)
	}
	lines := strings.TrimSpace(string(out))
	if lines == "" {
		return fmt.Sprintf("Tidak ditemukan file/folder bernama '%s', tuan.", name)
	}
	return fmt.Sprintf("Ditemukan, tuan:\n%s", lines)
}