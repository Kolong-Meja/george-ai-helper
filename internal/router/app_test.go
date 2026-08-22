package router

import (
	"testing"

	"github.com/Kolong-Meja/george/internal/config"
)

func TestTryHandle_AppOpen(t *testing.T) {
	origResolve, origStart := resolveBinary, startProcess
	defer func() { resolveBinary, startProcess = origResolve, origStart }()

	var launched string
	resolveBinary = func(candidates []string) string { return candidates[0] } // pretend installed
	startProcess = func(bin string) error { launched = bin; return nil }

	cfg := config.Default()
	cases := []struct {
		input       string
		wantHandled bool
	}{
		{"buka chrome", true},
		{"bukain vscode dong", true},
		{"jalanin spotify", true},
		{"start beekeeper studio", true},
		{"nggak usah buka chrome deh", false}, // negation -> falls through to LLM
		{"apa itu Indonesia?", false},         // unrelated -> falls through
	}
	for _, c := range cases {
		res := TryHandle(c.input, cfg)
		if res.Handled != c.wantHandled {
			t.Errorf("TryHandle(%q).Handled = %v, want %v (output: %q)", c.input, res.Handled, c.wantHandled, res.Output)
		}
	}

	if res := TryHandle("buka vscode", cfg); res.Handled && launched != "code" {
		t.Errorf("expected launched binary %q, got %q", "code", launched)
	}
}

func TestTryHandle_AppNotInstalled(t *testing.T) {
	origResolve := resolveBinary
	defer func() { resolveBinary = origResolve }()
	resolveBinary = func(candidates []string) string { return "" } // simulate not installed

	cfg := config.Default()
	res := TryHandle("buka steam", cfg)
	if !res.Handled {
		t.Fatalf("expected Handled=true even when app isn't installed (should report gracefully)")
	}
}
