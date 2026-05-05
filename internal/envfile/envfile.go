// Package envfile loads KEY=VALUE pairs from a .env file into the process
// environment. Variables already set in the environment are left alone, so
// terminal/machine env overrides .env. Missing file is a no-op, not an error.
package envfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load reads `path` and sets any KEY=VALUE pairs into os env that aren't
// already set. Lines starting with `#` and blank lines are ignored. Values
// may be quoted with single or double quotes; surrounding quotes are stripped.
// A non-existent file returns nil (so .env is optional).
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	lineNo := 0
	for s.Scan() {
		lineNo++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip optional `export ` prefix so users can copy-paste shell-style lines.
		line = strings.TrimPrefix(line, "export ")

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fmt.Errorf("%s:%d: missing '=' in %q", path, lineNo, line)
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", path, lineNo)
		}
		val := strings.TrimSpace(line[eq+1:])
		val = unquote(val)

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return s.Err()
}

func unquote(v string) string {
	if len(v) < 2 {
		return v
	}
	first, last := v[0], v[len(v)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return v[1 : len(v)-1]
	}
	return v
}
