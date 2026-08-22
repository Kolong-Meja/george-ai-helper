// This file adds "open app X" as another deterministic command alongside file
// search in router.go - matching "buka chrome", "bukain vscode dong", "jalanin
// spotify", etc. and launching the app directly, without ever going through
// the LLM. Same rationale as file search: a 3B model can't actually launch a
// process, so letting it "handle" this conversationally would mean either an
// honest refusal every time or (worse) a hallucinated "done!" that did nothing.
package router

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/Kolong-Meja/george/internal/config"
)

// appAliasPattern is the alternation of every recognized way to name a
// launchable app. Longer, more specific aliases (e.g. "visual studio code")
// are listed before shorter ones that could be a prefix of them (e.g.
// "vscode") - Go's regexp package (RE2) picks the first alternative that
// leads to an overall match rather than the longest one, so ordering here
// matters.
const appAliasPattern = `visual\s+studio\s+code|vs\s+code|vscode|` +
	`google\s+chrome|chrome|` +
	`mozilla\s+firefox|firefox|` +
	`beekeeper\s+studio|beekeeper|` +
	`spotify|steam`

// appOpenPattern matches Indonesian and English phrasing for launching a
// desktop application, e.g. "buka chrome", "bukain vscode dong", "jalanin
// spotify", "nyalain steam", "open firefox", "start beekeeper studio".
//
// Same shape as searchPattern in router.go: the verb and the app name don't
// have to sit right next to each other - up to 3 filler words are allowed in
// between (non-greedy), since that's how people actually phrase requests
// ("bukain dong aplikasi chrome-nya").
//
// Capture group 1 = the matched alias text, looked up in appAliases below to
// find which appRegistry entry to launch.
var appOpenPattern = regexp.MustCompile(
	`(?i)\b(?:buka(?:in)?|jalanin|nyalain|start|open|launch)\b` +
		`(?:\s+\S+){0,3}?` +
		`\s+(` + appAliasPattern + `)\b`,
)

// appAliases maps every alias appOpenPattern can capture (after whitespace
// normalization by normalizeAlias below) to its appRegistry key.
var appAliases = map[string]string{
	"visual studio code": "vscode",
	"vs code":            "vscode",
	"vscode":             "vscode",
	"google chrome":      "chrome",
	"chrome":             "chrome",
	"mozilla firefox":    "firefox",
	"firefox":            "firefox",
	"beekeeper studio":   "beekeeper",
	"beekeeper":          "beekeeper",
	"spotify":            "spotify",
	"steam":              "steam",
}

// appInfo describes one launchable application: a display name used in
// George's replies, and an ordered list of candidate binary names to try on
// PATH - different install methods (apt .deb, snap, flatpak) sometimes ship
// the same app under a different binary name, so the first one found wins
// (same fallback approach as fdBinary() in router.go for fd/fdfind).
type appInfo struct {
	DisplayName string
	Candidates  []string
}

// appRegistry is the fixed allow-list of apps George is willing to launch.
// Deliberately NOT extensible from within a chat message - the binary that
// actually gets exec'd always comes from this hardcoded list, never from
// anything typed by the user, so there's no path for a chat message to make
// George run an arbitrary command.
var appRegistry = map[string]appInfo{
	"vscode":    {DisplayName: "VS Code", Candidates: []string{"code", "code-insiders"}},
	"chrome":    {DisplayName: "Chrome", Candidates: []string{"google-chrome-stable", "google-chrome"}},
	"firefox":   {DisplayName: "Firefox", Candidates: []string{"firefox"}},
	"spotify":   {DisplayName: "Spotify", Candidates: []string{"spotify"}},
	"beekeeper": {DisplayName: "Beekeeper Studio", Candidates: []string{"beekeeper-studio"}},
	"steam":     {DisplayName: "Steam", Candidates: []string{"steam"}},
}

// resolveBinary returns the first candidate found on $PATH, or "" if none are
// installed. A package-level var (not a plain func) so router_test.go can
// stub it out - the real lookup depends on what happens to be installed on
// whichever machine runs `go test`, which a unit test must not depend on.
var resolveBinary = func(candidates []string) string {
	for _, name := range candidates {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

// startProcess launches bin as a new, fully detached process - Start (not
// Run), so George doesn't block waiting for a GUI app to exit, and Setsid so
// the app survives independently of George's own process (closing the
// terminal George runs in must not also kill the app it just opened). Linux-
// only (SysProcAttr.Setsid isn't defined on every GOOS), which matches the
// rest of this project's Pop!_OS/Ubuntu-only scope.
//
// A package-level var for the same testability reason as resolveBinary above
// - a test run must never actually pop open a browser or a game client.
var startProcess = func(bin string) error {
	cmd := exec.Command(bin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}

// normalizeAlias collapses whatever whitespace appOpenPattern actually
// matched (a real message could have "vs   code" with extra spaces, or a
// stray tab) down to the single-space form used as keys in appAliases.
func normalizeAlias(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// tryOpenApp checks input against appOpenPattern. The second return value
// reports whether the pattern matched at all: false means TryHandle should
// keep checking other rules (e.g. file search); true means TryHandle should
// return the paired Result as-is, whether that's a launch outcome or
// Handled=false for a nearby negation ("jangan buka chrome dong" - declining,
// not requesting).
//
// Known limitation: only the first app mentioned gets launched. "buka chrome
// sama spotify" only opens Chrome - a deliberate simplification rather than
// an oversight, since supporting several apps in one command would need a
// different Result shape than the rest of router.go currently assumes (one
// action -> one reply). Flag if this matters in practice and it can be
// extended.
func tryOpenApp(input string, cfg config.Config) (Result, bool) {
	loc := appOpenPattern.FindStringSubmatchIndex(input)
	if loc == nil {
		return Result{}, false
	}

	if hasNearbyNegation(input[:loc[0]]) {
		return Result{Handled: false}, true
	}

	appKey := appAliases[normalizeAlias(input[loc[2]:loc[3]])]
	app, ok := appRegistry[appKey]
	if !ok {
		// Matched the alternation but somehow isn't in appAliases - shouldn't
		// happen since both are hand-kept in sync, but fall through to the
		// LLM rather than panic on the mismatch.
		return Result{Handled: false}, true
	}

	bin := resolveBinary(app.Candidates)
	if bin == "" {
		if cfg.Language == config.English {
			return Result{Handled: true, Output: fmt.Sprintf("Can't find %s installed on this machine, bro.", app.DisplayName)}, true
		}
		return Result{Handled: true, Output: fmt.Sprintf("%s kayaknya belum ke-install di mesin ini, bro.", app.DisplayName)}, true
	}

	if err := startProcess(bin); err != nil {
		if cfg.Language == config.English {
			return Result{Handled: true, Output: fmt.Sprintf("Tried to open %s but it failed, bro: %v", app.DisplayName, err)}, true
		}
		return Result{Handled: true, Output: fmt.Sprintf("Gagal buka %s nih bro: %v", app.DisplayName, err)}, true
	}

	if cfg.Language == config.English {
		return Result{Handled: true, Output: fmt.Sprintf("Opening %s for you, bro %s.", app.DisplayName, cfg.UserName)}, true
	}
	return Result{Handled: true, Output: fmt.Sprintf("Oke bro %s, %s lagi dibuka.", cfg.UserName, app.DisplayName)}, true
}
