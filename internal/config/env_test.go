package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOr(t *testing.T) {
	const key = "GEORGE_TEST_ENVOR"
	os.Unsetenv(key)
	if got := envOr(key, "fallback"); got != "fallback" {
		t.Errorf("envOr with unset var = %q, want %q", got, "fallback")
	}

	os.Setenv(key, "set-value")
	defer os.Unsetenv(key)
	if got := envOr(key, "fallback"); got != "set-value" {
		t.Errorf("envOr with set var = %q, want %q", got, "set-value")
	}
}

func TestEnvInt(t *testing.T) {
	const key = "GEORGE_TEST_ENVINT"
	os.Unsetenv(key)
	if got := envInt(key, 7); got != 7 {
		t.Errorf("envInt with unset var = %d, want %d", got, 7)
	}

	os.Setenv(key, "12")
	defer os.Unsetenv(key)
	if got := envInt(key, 7); got != 12 {
		t.Errorf("envInt with valid int = %d, want %d", got, 12)
	}

	os.Setenv(key, "not-a-number")
	if got := envInt(key, 7); got != 7 {
		t.Errorf("envInt with invalid int = %d, want fallback %d", got, 7)
	}
}

func TestReadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	const key = "GEORGE_TEST_PRESET"
	os.Setenv(key, "already-set")
	defer os.Unsetenv(key)

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	readDotEnv(f)

	if got := os.Getenv(key); got != "already-set" {
		t.Errorf("readDotEnv overrode an existing env var: got %q, want %q", got, "already-set")
	}
}
