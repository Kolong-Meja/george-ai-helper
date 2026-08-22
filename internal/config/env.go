package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// dotEnvCandidates returns, in priority order, the paths loadDotEnv checks for
// a .env file. Checking the working directory first supports running George
// straight out of the repo (`go run .`, `./george`) during development;
// falling back to ~/.config/george/.env supports the README's normal usage
// pattern of installing the binary to /usr/local/bin and calling it from
// anywhere, where the working directory is never the repo.
func dotEnvCandidates() []string {
	candidates := []string{".env"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".config", "george", ".env"))
	}
	return candidates
}

// loadDotEnv loads the first existing file from dotEnvCandidates() into the
// process environment. A missing .env at every candidate path is not an error -
// George just falls back to Default()'s neutral placeholder values, instead of
// silently greeting a stranger as "Faisal" on his birthday.
func loadDotEnv() {
	for _, path := range dotEnvCandidates() {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		readDotEnv(f)
		f.Close()
		return
	}
}

// readDotEnv parses simple KEY=VALUE lines from r into the process environment
// ("#" starts a comment, blank lines are skipped, surrounding quotes on the
// value are trimmed). A key that's already set in the environment - e.g. by a
// shell `export` or a systemd Environment= line - is left untouched, so .env
// behaves as a fallback/default rather than silently overriding an intentional
// override. This is the same convention most .env tooling follows (Node's
// dotenv, Python's python-dotenv).
func readDotEnv(f *os.File) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if _, alreadySet := os.LookupEnv(key); alreadySet {
			continue
		}
		os.Setenv(key, value)
	}
}

// envOr returns the environment variable named key, or fallback if it's unset
// or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt returns the environment variable named key parsed as an int, or
// fallback if it's unset, empty, or not a valid integer. BirthdayMonth/
// BirthdayDay/BirthYear follow the same zero-value-means-unset sentinel
// convention as the rest of the codebase (see config.go's birthdayLine): an
// absent or malformed value safely falls back to 0, which keeps the birthday
// greeting disabled rather than guessing at someone else's birthday.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
