package review

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sharma-bot/internal/ai"
)

type scriptedCompleter struct {
	steps   []ai.Step
	calls   int
	gotUser []string
}

func (s *scriptedCompleter) Step(_ context.Context, _ string, hist ai.History, _ []ai.ToolDef) (ai.Step, ai.Usage, error) {
	s.calls++
	s.gotUser = append(s.gotUser, hist.UserPrompt)
	if s.calls > len(s.steps) {
		return ai.Step{}, ai.Usage{}, errors.New("script exhausted")
	}
	return s.steps[s.calls-1], ai.Usage{Model: "test"}, nil
}

func writePrompt(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "review.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReviewSingleFileMarkdown(t *testing.T) {
	corpusDir := t.TempDir()
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")

	contentDir := t.TempDir()
	contentFile := filepath.Join(contentDir, "email.md")
	if err := os.WriteFile(contentFile, []byte("subject: hi\nbody: welcome"), 0o644); err != nil {
		t.Fatal(err)
	}

	completer := &scriptedCompleter{steps: []ai.Step{{Text: "review output", StopReason: "end_turn"}}}
	_, err := RunWith(corpusDir, promptsDir, contentFile, "", nil, completer, nil, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(completer.gotUser[0], "subject: hi") {
		t.Errorf("user prompt missing content: %q", completer.gotUser[0])
	}
	if !strings.Contains(completer.gotUser[0], "markdown content from email.md") {
		t.Errorf("user prompt missing label: %q", completer.gotUser[0])
	}
}

func TestReviewSingleFileHTMLExtracted(t *testing.T) {
	corpusDir := t.TempDir()
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")

	contentFile := filepath.Join(t.TempDir(), "page.html")
	html := `<!doctype html><html><head><script>alert("nope")</script></head>
		<body><h1>Title</h1><p>Body text here.</p></body></html>`
	if err := os.WriteFile(contentFile, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	completer := &scriptedCompleter{steps: []ai.Step{{Text: "ok", StopReason: "end_turn"}}}
	_, err := RunWith(corpusDir, promptsDir, contentFile, "", nil, completer, nil, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	user := completer.gotUser[0]
	if !strings.Contains(user, "Title") || !strings.Contains(user, "Body text here.") {
		t.Errorf("HTML body not extracted: %q", user)
	}
	if strings.Contains(user, "alert") || strings.Contains(user, "nope") {
		t.Errorf("script content leaked: %q", user)
	}
	if !strings.Contains(user, "HTML extracted from page.html") {
		t.Errorf("HTML label missing: %q", user)
	}
}

func TestReviewStdin(t *testing.T) {
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "ok", StopReason: "end_turn"}}}
	_, err := RunWith("", promptsDir, "", "", strings.NewReader("piped content"), completer, nil, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(completer.gotUser[0], "piped content") {
		t.Errorf("stdin not read: %q", completer.gotUser[0])
	}
}

func TestReviewBatch(t *testing.T) {
	corpusDir := t.TempDir()
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")

	batchDir := t.TempDir()
	files := map[string]string{
		"email-1.md":    "first email",
		"email-2.md":    "second email",
		"page.html":     "<html><body>html body</body></html>",
		"ignored.json":  `{"unsupported":true}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(batchDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	completer := &scriptedCompleter{
		steps: []ai.Step{
			{Text: "review of email-1", StopReason: "end_turn"},
			{Text: "review of email-2", StopReason: "end_turn"},
			{Text: "review of page", StopReason: "end_turn"},
		},
	}

	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	var trace bytes.Buffer
	_, err := RunWith(corpusDir, promptsDir, "", batchDir, nil, completer, nil, &trace, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if completer.calls != 3 {
		t.Errorf("expected 3 review calls (json ignored), got %d", completer.calls)
	}
	if !strings.Contains(trace.String(), "3 ok") {
		t.Errorf("trace summary missing: %s", trace.String())
	}
}

func TestReviewBatchEmptyDirErrors(t *testing.T) {
	corpusDir := t.TempDir()
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "x", StopReason: "end_turn"}}}
	_, err := RunWith(corpusDir, promptsDir, "", t.TempDir(), nil, completer, nil, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "no supported files") {
		t.Errorf("expected no-supported-files error, got %v", err)
	}
}

func TestReviewMissingRolePrompt(t *testing.T) {
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "x", StopReason: "end_turn"}}}
	_, err := RunWith("", t.TempDir(), "", "", nil, completer, nil, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "role prompt") {
		t.Errorf("expected role-prompt error, got %v", err)
	}
}

// ─────────────────────────── HTML extraction ───────────────────────────

func TestExtractTextBasicShape(t *testing.T) {
	html := `<html><body>
		<h1>Title</h1>
		<p>First paragraph.</p>
		<p>Second paragraph.</p>
		<script>const x = "noise";</script>
		<style>body { color: red; }</style>
		<div>Block content</div>
	</body></html>`
	got := ExtractText([]byte(html))
	for _, want := range []string{"Title", "First paragraph.", "Second paragraph.", "Block content"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
	for _, bad := range []string{"const x", "noise", "color: red"} {
		if strings.Contains(got, bad) {
			t.Errorf("unexpected %q in output, got:\n%s", bad, got)
		}
	}
}

func TestExtractTextPreservesParagraphs(t *testing.T) {
	html := `<p>First.</p><p>Second.</p>`
	got := ExtractText([]byte(html))
	// Paragraphs should be on different lines.
	if !strings.Contains(got, "First.\nSecond.") {
		t.Errorf("paragraphs not on separate lines: %q", got)
	}
}

func TestExtractTextHandlesBR(t *testing.T) {
	html := `<p>line one<br/>line two</p>`
	got := ExtractText([]byte(html))
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Errorf("br not handled: %q", got)
	}
}

func TestExtractTextCollapsesWhitespace(t *testing.T) {
	html := `<p>   lots   of    spaces   </p>`
	got := ExtractText([]byte(html))
	if got != "lots of spaces" {
		t.Errorf("whitespace not collapsed: %q", got)
	}
}

// ─────────────────────────── helpers ───────────────────────────

func TestLooksLikeHTML(t *testing.T) {
	cases := map[string]bool{
		"<!doctype html><html>...":        true,
		"<html><body>x</body></html>":     true,
		"  <html>x":                       true,
		"<HTML>UPPER":                     true,
		"plain text":                      false,
		"# markdown header":               false,
		`{"json": true}`:                  false,
	}
	for in, want := range cases {
		if got := looksLikeHTML([]byte(in)); got != want {
			t.Errorf("looksLikeHTML(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSanitizeForFilename(t *testing.T) {
	cases := map[string]string{
		"foo.html":                "foo",
		"products/foo.html":       "products-foo",
		"a/b/c.md":                "a-b-c",
		"plain":                   "plain",
	}
	for in, want := range cases {
		if got := sanitizeForFilename(in); got != want {
			t.Errorf("sanitizeForFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
