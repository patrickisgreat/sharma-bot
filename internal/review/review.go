// Package review reviews user-supplied marketing content (emails, landing
// pages, PDPs) against the corpus, using the agent loop. Single-file or
// batch (folder of files); stdin supported. HTML inputs are text-extracted
// before sending to the model.
package review

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"sharma-bot/internal/agent"
	"sharma-bot/internal/ai"
	"sharma-bot/internal/tools"
)

const (
	model     = anthropic.ModelClaudeSonnet4_6
	maxTokens = int64(8192)
	maxSteps  = 12
	timeout   = 5 * time.Minute
)

// Run reviews either one file (path != "") or every supported file in
// batchDir. If both are empty, content is read from stdinReader.
//
// Returns a CallResult for single-file mode (so the caller can persist the
// chat). Batch mode writes per-file outputs to reviews/<timestamp>/ and
// returns (nil, nil) on success — there's no single chat to save.
//
// trace receives per-step trace lines (tool calls, results, telemetry).
// Pass nil to suppress.
func Run(corpusDir, promptsDir, path, batchDir string, stdinReader io.Reader, trace io.Writer) (*ai.CallResult, error) {
	completer := ai.NewToolCompleter(model, maxTokens)
	return RunWith(corpusDir, promptsDir, path, batchDir, stdinReader, completer, tools.NewCorpusTools(corpusDir), trace, timeout)
}

// RunWith is the testable form: explicit completer + tools + timeout.
func RunWith(corpusDir, promptsDir, path, batchDir string, stdinReader io.Reader, completer ai.ToolCompleter, ts []tools.Tool, trace io.Writer, perCallTimeout time.Duration) (*ai.CallResult, error) {
	role, err := os.ReadFile(filepath.Join(promptsDir, "review.md"))
	if err != nil {
		return nil, fmt.Errorf("read role prompt: %w", err)
	}
	if batchDir != "" {
		if err := runBatch(batchDir, string(role), completer, ts, trace, perCallTimeout); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return runSingle(path, stdinReader, string(role), completer, ts, trace, perCallTimeout, os.Stdout)
}

func runSingle(path string, stdinReader io.Reader, role string, completer ai.ToolCompleter, ts []tools.Tool, trace io.Writer, perCallTimeout time.Duration, out io.Writer) (*ai.CallResult, error) {
	content, label, err := loadContent(path, stdinReader)
	if err != nil {
		return nil, err
	}
	res, err := reviewOne(content, label, role, completer, ts, trace, perCallTimeout)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(out, res.Answer); err != nil {
		return nil, err
	}
	return res, nil
}

func runBatch(batchDir, role string, completer ai.ToolCompleter, ts []tools.Tool, trace io.Writer, perCallTimeout time.Duration) error {
	files, err := listSupportedFiles(batchDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no supported files (.html, .htm, .md, .txt) under %s", batchDir)
	}
	outDir := filepath.Join("reviews", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if trace != nil {
		fmt.Fprintf(trace, "reviewing %d file(s) → %s\n", len(files), outDir)
	}
	var ok, fail int
	for _, f := range files {
		rel, _ := filepath.Rel(batchDir, f)
		if trace != nil {
			fmt.Fprintf(trace, "\n=== %s ===\n", rel)
		}
		content, label, err := loadContent(f, nil)
		if err != nil {
			if trace != nil {
				fmt.Fprintf(trace, "  load error: %v\n", err)
			}
			fail++
			continue
		}
		res, err := reviewOne(content, label, role, completer, ts, trace, perCallTimeout)
		if err != nil {
			if trace != nil {
				fmt.Fprintf(trace, "  review error: %v\n", err)
			}
			fail++
			continue
		}
		outPath := filepath.Join(outDir, sanitizeForFilename(rel)+".review.md")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(res.Answer), 0o644); err != nil {
			return err
		}
		ok++
	}
	if trace != nil {
		fmt.Fprintf(trace, "\nreview: %d ok, %d failed → %s\n", ok, fail, outDir)
	}
	return nil
}

func reviewOne(content, label, role string, completer ai.ToolCompleter, ts []tools.Tool, trace io.Writer, perCallTimeout time.Duration) (*ai.CallResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()

	userPrompt := buildUserPrompt(content, label)
	start := time.Now()
	res, err := agent.Run(ctx, agent.Config{
		Completer: completer,
		Tools:     ts,
		MaxSteps:  maxSteps,
		Trace:     trace,
	}, role, userPrompt)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}
	ai.PrintTelemetry(trace, res.Usage, elapsed, fmt.Sprintf("%d step(s)", res.Steps))
	return &ai.CallResult{
		Answer:  res.Answer,
		Usage:   res.Usage,
		Elapsed: elapsed,
		Steps:   res.Steps,
	}, nil
}

func buildUserPrompt(content, label string) string {
	var sb strings.Builder
	sb.WriteString("Review the following ")
	if label == "" {
		label = "content"
	}
	sb.WriteString(label)
	sb.WriteString(":\n\n---\n")
	sb.WriteString(content)
	sb.WriteString("\n---\n")
	return sb.String()
}

// loadContent reads input and returns (text, label, error). Label is a short
// human-readable description used in the user prompt ("page (HTML extracted
// from foo.html)" etc.).
func loadContent(path string, stdinReader io.Reader) (string, string, error) {
	if path == "" || path == "-" {
		if stdinReader == nil {
			return "", "", fmt.Errorf("no input provided")
		}
		data, err := io.ReadAll(stdinReader)
		if err != nil {
			return "", "", fmt.Errorf("read stdin: %w", err)
		}
		if looksLikeHTML(data) {
			return ExtractText(data), "page (HTML extracted from stdin)", nil
		}
		return string(data), "content from stdin", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	base := filepath.Base(path)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return ExtractText(data), "page (HTML extracted from " + base + ")", nil
	case ".md":
		return string(data), "markdown content from " + base, nil
	case ".txt":
		return string(data), "content from " + base, nil
	default:
		if looksLikeHTML(data) {
			return ExtractText(data), "page (HTML extracted from " + base + ")", nil
		}
		return string(data), "content from " + base, nil
	}
}

func looksLikeHTML(data []byte) bool {
	n := len(data)
	if n > 200 {
		n = 200
	}
	head := strings.TrimSpace(strings.ToLower(string(data[:n])))
	return strings.HasPrefix(head, "<!doctype html") ||
		strings.HasPrefix(head, "<html") ||
		strings.Contains(head, "<head") ||
		strings.Contains(head, "<body")
}

func listSupportedFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".html", ".htm", ".md", ".txt":
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

// sanitizeForFilename turns a relative path into something safe for use as
// the basename of an output file. "products/foo bar.html" -> "products-foo bar".
func sanitizeForFilename(s string) string {
	s = strings.TrimSuffix(s, filepath.Ext(s))
	s = strings.ReplaceAll(s, string(filepath.Separator), "-")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}
