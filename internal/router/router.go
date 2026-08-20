// Package router handles deterministic commands (like file search) without
// calling the LLM at all — faster and lighter on an 8GB CPU-only machine.
package router

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Kolong-Meja/george/internal/config"
)

// searchPattern matches Indonesian and English phrasing for file/folder search
// requests, e.g. "cariin file ABC.txt", "cari folder Downloads", "find file
// report.pdf", "carikan aku folder Documents".
//
// The verb (cari/cariin/carikan/mencari/temuin/find/search) and the object
// keyword (file/folder/dir/directory/direktori) don't have to sit right next to
// each other - up to 4 filler words are allowed in between (non-greedy, so it
// stops at the nearest file/folder keyword), because that's how people actually
// phrase requests: "cariin gw file X", "tolong carikan aku folder Y". Requiring
// strict adjacency was the root cause of search requests silently falling
// through to the LLM, which then hallucinated a fake answer instead of
// admitting it can't touch the filesystem.
//
// Capture group 1 = the object keyword, used below to decide file vs. folder
// search (previously not distinguished at all, so a folder search returned
// every file AND folder matching the name).
// Capture group 2 = the target name to search for.
var searchPattern = regexp.MustCompile(
	`(?i)\b(?:cari(?:in|kan)?|mencari(?:kan)?|temu(?:in|kan)?|find|search(?:\s+for)?)\b` +
		`(?:\s+\S+){0,4}?` +
		`\s+(file|folder|dir(?:ectory)?|direktori)\s+(\S+)`,
)

// folderKeywords lists which values of capture group 1 mean "search for a
// directory" rather than "search for a file". Anything else captured by the
// pattern (i.e. "file") is treated as a file search.
var folderKeywords = map[string]bool{
	"folder": true, "dir": true, "directory": true, "direktori": true,
}

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

// excludeDirs are directory names George skips while walking the filesystem
// with `find` (the `fd` path skips almost all of these for free by ignoring
// hidden directories by default - see buildSearchCommand). These are near-always
// noise for a "find my file" request on a desktop Linux box: Wine/PlayOnLinux/
// Steam Proton prefixes each carry their own fake "Documents" folder, sandboxed
// Flatpak app data (.var) mirrors real folders again, and dependency trees
// (node_modules, vendor) can be enormous. Walking into all of them on every
// search was the direct cause of both the noisy/irrelevant results and the
// multi-minute search times.
var excludeDirs = []string{
	".cache", ".var", ".wine", ".PlayOnLinux",
	".npm", ".cargo", ".rustup", ".git",
	"node_modules", "vendor",
	".local/share/Trash",
	".steam/steamapps/compatdata",
	".local/share/Steam/steamapps/compatdata",
}

// searchTimeout bounds how long a single file/folder search may run. With
// excludeDirs pruning heavy trees this should normally finish in well under a
// second, but the timeout is a hard backstop so an unusual directory structure
// can never hang George's terminal indefinitely.
const searchTimeout = 12 * time.Second

// maxSearchResults caps how many matched paths are shown - and, just as
// importantly, how many get written into conversation history in main.go. An
// uncapped 50-line dump (as with a generic name like "Documents") would bloat
// every LLM prompt for the rest of the session, since router output is stored
// in history too, slowing down every reply after it, not just this one.
const maxSearchResults = 15

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

	kind := strings.ToLower(input[loc[2]:loc[3]])
	name := strings.TrimRight(input[loc[4]:loc[5]], ",.!?;:\"'")
	if nonFileWords[strings.ToLower(name)] {
		return Result{Handled: false}
	}

	findType := "f"
	if folderKeywords[kind] {
		findType = "d"
	}

	return Result{Handled: true, Output: searchFile(name, findType, cfg)}
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

// searchFile looks for a file (findType "f") or folder (findType "d") named
// `name` under the current user's home directory, bounded by searchTimeout.
func searchFile(name, findType string, cfg config.Config) string {
	root, err := os.UserHomeDir()
	if err != nil || root == "" {
		root = "/home" // fall back to the previous, broader scope
	}

	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	defer cancel()

	out, runErr := buildSearchCommand(ctx, root, name, findType).Output()
	lines := strings.TrimSpace(string(out))

	if ctx.Err() == context.DeadlineExceeded {
		if cfg.Language == config.English {
			return fmt.Sprintf("Search for '%s' took too long and got cut off, bro - try a more specific name.", name)
		}
		return fmt.Sprintf("Nyari '%s' kelamaan jadi gw stop, bro - coba nama yang lebih spesifik ya.", name)
	}

	// find(1)/fd(1) can exit non-zero the moment they can't descend into ANY
	// subdirectory while walking the tree - another user's home folder,
	// permission-restricted app sandboxes, etc. That's routine and near-universal
	// under a home directory, and it's an *exec.ExitError: the tool still ran and
	// reported whatever it legitimately could, it just also hit something it
	// wasn't allowed to open. That says nothing about whether the target exists,
	// so it must never be treated as a search failure - not when a match still
	// came through on stdout (Output() keeps matched bytes even on a non-zero
	// exit), and not when nothing matched either (still just means "not found,"
	// not "broken").
	//
	// A real failure is when the command itself couldn't run at all - e.g. the
	// binary is missing from PATH. That surfaces as a different error type
	// (*exec.Error, not *exec.ExitError), so it's the only case that still
	// reaches the hard-error message below.
	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		if cfg.Language == config.English {
			return fmt.Sprintf("Couldn't search for '%s', bro: %v", name, runErr)
		}
		return fmt.Sprintf("Gagal nyari '%s' nih bro: %v", name, runErr)
	}

	if lines == "" {
		if cfg.Language == config.English {
			return fmt.Sprintf("Couldn't find anything named '%s', bro %s.", name, cfg.UserName)
		}
		return fmt.Sprintf("Nggak ketemu file/folder bernama '%s' nih, bro %s.", name, cfg.UserName)
	}

	result := capResults(lines, cfg)

	if cfg.Language == config.English {
		return fmt.Sprintf("Found it, bro %s:\n%s", cfg.UserName, result)
	}
	return fmt.Sprintf("Ketemu bro %s, ini dia:\n%s", cfg.UserName, result)
}

// capResults trims a matched-paths list down to maxSearchResults lines, noting
// how many were omitted so a broad query (e.g. a common folder name matched in
// several places) doesn't flood the reply or bloat conversation history.
func capResults(lines string, cfg config.Config) string {
	matches := strings.Split(lines, "\n")
	if len(matches) <= maxSearchResults {
		return lines
	}
	extra := len(matches) - maxSearchResults
	shown := strings.Join(matches[:maxSearchResults], "\n")
	if cfg.Language == config.English {
		return fmt.Sprintf("%s\n… and %d more - try a more specific name for a shorter list.", shown, extra)
	}
	return fmt.Sprintf("%s\n… dan %d lagi - coba nama yang lebih spesifik biar hasilnya lebih ringkas.", shown, extra)
}

// fdBinary returns the name of an installed fd binary - "fd", or "fdfind" (the
// name Debian/Ubuntu's `fd-find` package installs it under, since "fd" was
// already taken by another tool in their repos) - or "" if neither is on PATH.
func fdBinary() string {
	for _, name := range []string{"fd", "fdfind"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

// buildSearchCommand picks fd when it's installed - it walks the tree with
// multiple threads and, by default, already skips hidden directories (which
// covers most of excludeDirs: .cache, .var, .wine, .git, ...) without needing
// an explicit prune list - and falls back to `find` with excludeDirs pruned
// explicitly otherwise, since that's what a stock Ubuntu install has out of
// the box.
func buildSearchCommand(ctx context.Context, root, name, findType string) *exec.Cmd {
	if bin := fdBinary(); bin != "" {
		// -i: case-insensitive. -g: match `name` as a glob against the whole
		// path's basename, same semantics as find's -iname (so wildcards like
		// "*.txt" keep working). -a: print absolute paths. -t: type filter.
		return exec.CommandContext(ctx, bin, "-i", "-g", "-a", "-t", findType, "--", name, root)
	}

	args := []string{root, "("}
	for i, dir := range excludeDirs {
		if i > 0 {
			args = append(args, "-o")
		}
		args = append(args, "-path", "*/"+dir)
	}
	args = append(args, ")", "-prune", "-o", "-type", findType, "-iname", name, "-print")
	return exec.CommandContext(ctx, "find", args...)
}
