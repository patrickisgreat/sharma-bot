package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"plain object", `{"a":1}`, true},
		{"with shopify banner", "/* auto-generated\n * caution\n */\n{\"a\":1}", true},
		{"banner then array", "/* x */\n[1,2,3]", true},
		{"broken: trailing comma", `{"a":1,}`, false},
		{"broken: missing brace", `{"a":1`, false},
		{"banner then broken", "/* x */\n{\"a\":}", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validJSON([]byte(tc.in))
			if tc.ok && err != nil {
				t.Errorf("expected valid, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected invalid, got nil")
			}
		})
	}
}

func TestStripLeadingBlockComment(t *testing.T) {
	got := string(stripLeadingBlockComment([]byte("/* hi */\n  {\"a\":1}")))
	if got != `{"a":1}` {
		t.Errorf("got %q", got)
	}
	// No comment: unchanged.
	if got := string(stripLeadingBlockComment([]byte(`{"a":1}`))); got != `{"a":1}` {
		t.Errorf("got %q", got)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
}

func gitCommitAll(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", "x"}} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
}

func TestEnsureCleanTree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Run("not a repo errors", func(t *testing.T) {
		if err := ensureCleanTree(t.TempDir(), false); err == nil {
			t.Error("expected error for non-repo")
		}
	})

	t.Run("force skips all checks", func(t *testing.T) {
		if err := ensureCleanTree(t.TempDir(), true); err != nil {
			t.Errorf("force should bypass: %v", err)
		}
	})

	t.Run("clean tree passes", func(t *testing.T) {
		dir := t.TempDir()
		gitInit(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitCommitAll(t, dir)
		if err := ensureCleanTree(dir, false); err != nil {
			t.Errorf("clean tree should pass: %v", err)
		}
	})

	t.Run("dirty tree errors", func(t *testing.T) {
		dir := t.TempDir()
		gitInit(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitCommitAll(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := ensureCleanTree(dir, false)
		if err == nil || !strings.Contains(err.Error(), "uncommitted") {
			t.Errorf("expected uncommitted error, got %v", err)
		}
	})
}
