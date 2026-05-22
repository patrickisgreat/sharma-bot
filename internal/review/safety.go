package review

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ensureCleanTree refuses to proceed when dir has uncommitted changes, so
// --write edits always land as their own reviewable diff. force bypasses the
// check. A dir that isn't inside a git work tree is treated as unsafe unless
// forced — the whole safety story is "review and revert via git".
func ensureCleanTree(dir string, force bool) error {
	if force {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found; --write needs a git work tree for safety (or pass --force): %w", err)
	}

	inside := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	if out, err := inside.Output(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("%s is not inside a git work tree; --write needs one so edits are revertable (or pass --force)", dir)
	}

	status := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain", "--", ".")
	var stdout, stderr bytes.Buffer
	status.Stdout = &stdout
	status.Stderr = &stderr
	if err := status.Run(); err != nil {
		return fmt.Errorf("git status: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	if strings.TrimSpace(stdout.String()) != "" {
		return fmt.Errorf("uncommitted changes under %s — commit or stash first so the edits are isolated (or pass --force)", dir)
	}
	return nil
}

// validJSON reports whether data parses as JSON after stripping a leading
// block comment. Shopify theme template files open with an auto-generated
// /* ... */ banner before the JSON object; everything else must be strict
// JSON. Used to reject (and roll back) edits that would corrupt a .json file.
func validJSON(data []byte) error {
	return json.Unmarshal(stripLeadingBlockComment(data), &json.RawMessage{})
}

// stripLeadingBlockComment removes a single leading /* ... */ comment and any
// surrounding whitespace, returning the remainder. If there's no leading
// block comment, the input is returned unchanged.
func stripLeadingBlockComment(data []byte) []byte {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("/*")) {
		return data
	}
	end := bytes.Index(trimmed, []byte("*/"))
	if end < 0 {
		return data
	}
	return bytes.TrimLeft(trimmed[end+2:], " \t\r\n")
}
