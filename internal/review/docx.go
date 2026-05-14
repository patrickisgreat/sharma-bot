package review

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// writeDOCX converts an existing markdown file to .docx next to it using
// pandoc. The .docx imports cleanly into Google Docs with heading styles,
// bullets, bold/italic, and links preserved.
//
// Returns a clean error if pandoc isn't installed so callers can decide
// whether to warn-and-continue (typical) or fail.
func writeDOCX(mdPath string) error {
	pandoc, err := exec.LookPath("pandoc")
	if err != nil {
		return fmt.Errorf("pandoc not on PATH (brew install pandoc to enable)")
	}
	docxPath := strings.TrimSuffix(mdPath, ".md") + ".docx"
	cmd := exec.Command(pandoc, mdPath, "-o", docxPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
