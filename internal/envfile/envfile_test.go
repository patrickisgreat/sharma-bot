package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnv(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadSetsMissingVars(t *testing.T) {
	t.Setenv("ENVFILE_TEST_PRESET", "from-shell")
	// Ensure the unset key really is unset before we run.
	os.Unsetenv("ENVFILE_TEST_NEW")

	p := writeEnv(t, `
# a comment
ENVFILE_TEST_NEW=from-file
ENVFILE_TEST_PRESET=should-not-overwrite
`)
	if err := Load(p); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("ENVFILE_TEST_NEW"); got != "from-file" {
		t.Errorf("new var: %q", got)
	}
	if got := os.Getenv("ENVFILE_TEST_PRESET"); got != "from-shell" {
		t.Errorf("preset var should not be overridden: %q", got)
	}
}

func TestLoadStripsQuotesAndExport(t *testing.T) {
	os.Unsetenv("ENVFILE_TEST_QUOTED")
	os.Unsetenv("ENVFILE_TEST_SINGLE")
	os.Unsetenv("ENVFILE_TEST_EXPORT")

	p := writeEnv(t, `
ENVFILE_TEST_QUOTED="value with spaces"
ENVFILE_TEST_SINGLE='single quoted'
export ENVFILE_TEST_EXPORT=exported
`)
	if err := Load(p); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("ENVFILE_TEST_QUOTED"); got != "value with spaces" {
		t.Errorf("double-quoted: %q", got)
	}
	if got := os.Getenv("ENVFILE_TEST_SINGLE"); got != "single quoted" {
		t.Errorf("single-quoted: %q", got)
	}
	if got := os.Getenv("ENVFILE_TEST_EXPORT"); got != "exported" {
		t.Errorf("export prefix: %q", got)
	}
}

func TestLoadMissingFileIsNoError(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "does-not-exist.env")); err != nil {
		t.Errorf("missing file should not error: %v", err)
	}
}

func TestLoadRejectsMalformed(t *testing.T) {
	p := writeEnv(t, "this line has no equals\n")
	if err := Load(p); err == nil {
		t.Fatal("expected error for missing '='")
	}
}

func TestLoadIgnoresBlankAndComments(t *testing.T) {
	os.Unsetenv("ENVFILE_TEST_OK")
	p := writeEnv(t, `

# top comment
   # indented comment

ENVFILE_TEST_OK=yes

`)
	if err := Load(p); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("ENVFILE_TEST_OK"); got != "yes" {
		t.Errorf("expected yes, got %q", got)
	}
}
