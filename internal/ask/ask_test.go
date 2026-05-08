package ask

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sharma-bot/internal/ai"
)

type fakeCompleter struct {
	resp    string
	err     error
	calls   int
	gotSys  string
	gotUser string
}

func (f *fakeCompleter) Complete(_ context.Context, system, user string) (string, ai.Usage, error) {
	f.calls++
	f.gotSys = system
	f.gotUser = user
	if f.err != nil {
		return "", ai.Usage{}, f.err
	}
	return f.resp, ai.Usage{InputTokens: 100, CacheReadTokens: 200, OutputTokens: 50, Model: "fake"}, nil
}

var _ ai.Completer = (*fakeCompleter)(nil)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makePromptsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ask.md"), []byte("ROLE PROMPT"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunWithSingleSource(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, srcDir, "epA.md", "First body.")
	writeFile(t, srcDir, "epB.md", "Second body.")

	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "the answer"}

	got, err := RunWith(promptsDir, "what about pdps?", []Source{
		{Label: "limited-supply", Dir: srcDir},
	}, 0, fc, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answer != "the answer" {
		t.Errorf("answer: %q", got.Answer)
	}
	if got.Steps != 1 {
		t.Errorf("steps: %d", got.Steps)
	}
	if fc.calls != 1 {
		t.Fatalf("calls: %d", fc.calls)
	}
	if fc.gotUser != "what about pdps?" {
		t.Errorf("user: %q", fc.gotUser)
	}
	if !strings.HasPrefix(fc.gotSys, "ROLE PROMPT") {
		t.Errorf("system should start with role prompt: %q", fc.gotSys[:60])
	}
	for _, want := range []string{
		`<doc source="limited-supply" id="epA">`,
		"First body.",
		`<doc source="limited-supply" id="epB">`,
		"Second body.",
		"</doc>",
	} {
		if !strings.Contains(fc.gotSys, want) {
			t.Errorf("system missing %q", want)
		}
	}
}

func TestRunWithMultipleSourcesMerged(t *testing.T) {
	pcDir := t.TempDir()
	docsDir := t.TempDir()
	articlesDir := t.TempDir()
	writeFile(t, pcDir, "epA.md", "podcast body")
	writeFile(t, docsDir, "copy-guideline.txt", "doc body")
	writeFile(t, articlesDir, "teardown.md", "article body")

	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}

	_, err := RunWith(promptsDir, "q", []Source{
		{Label: "limited-supply", Dir: pcDir},
		{Label: "docs", Dir: docsDir},
		{Label: "articles", Dir: articlesDir},
	}, 0, fc, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<doc source="limited-supply" id="epA">`,
		`<doc source="docs" id="copy-guideline">`,
		`<doc source="articles" id="teardown">`,
		"podcast body",
		"doc body",
		"article body",
	} {
		if !strings.Contains(fc.gotSys, want) {
			t.Errorf("system missing %q", want)
		}
	}
}

func TestRunWithMissingSourceDirSkipped(t *testing.T) {
	pcDir := t.TempDir()
	writeFile(t, pcDir, "ep.md", "podcast")
	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}

	// docs and articles dirs deliberately do not exist.
	_, err := RunWith(promptsDir, "q", []Source{
		{Label: "limited-supply", Dir: pcDir},
		{Label: "docs", Dir: filepath.Join(t.TempDir(), "nope")},
		{Label: "articles", Dir: filepath.Join(t.TempDir(), "also-nope")},
	}, 0, fc, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fc.gotSys, "podcast") {
		t.Errorf("expected podcast body, got: %q", fc.gotSys)
	}
}

func TestRunWithRecursiveAndExtensions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "top.md", "top body")
	writeFile(t, dir, "nested/inner.txt", "nested body")
	writeFile(t, dir, "nested/skip.json", "should not load") // wrong extension
	writeFile(t, dir, "deep/dir/page.md", "deep body")

	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}
	_, err := RunWith(promptsDir, "q", []Source{
		{Label: "docs", Dir: dir},
	}, 0, fc, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`id="top"`,
		`id="nested/inner"`,
		`id="deep/dir/page"`,
		"top body",
		"nested body",
		"deep body",
	} {
		if !strings.Contains(fc.gotSys, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(fc.gotSys, "should not load") {
		t.Errorf("non-md/.txt file was loaded")
	}
}

func TestRunWithMetadataAttrsRendered(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "9lmar.md", "body")
	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}

	_, err := RunWith(promptsDir, "q", []Source{
		{
			Label: "limited-supply",
			Dir:   dir,
			Metadata: map[string]map[string]string{
				"9lmar": {
					"title":   "Why Native Rejected Investors",
					"season":  "1",
					"episode": "1",
					"date":    "2022-07-20",
				},
			},
		},
	}, 0, fc, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	want := `<doc source="limited-supply" id="9lmar" title="Why Native Rejected Investors" season="1" episode="1" date="2022-07-20">`
	if !strings.Contains(fc.gotSys, want) {
		t.Errorf("expected attrs:\n  want %s\n  got slice: %q", want, fc.gotSys)
	}
}

func TestRunWithLimitCapsTotal(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	writeFile(t, a, "a1.md", "1")
	writeFile(t, a, "a2.md", "2")
	writeFile(t, b, "b1.md", "3")

	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}
	_, err := RunWith(promptsDir, "q", []Source{
		{Label: "x", Dir: a},
		{Label: "y", Dir: b},
	}, 2, fc, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// Order: source x first (a1, a2), source y second (b1). Limit=2 → x's two only.
	if !strings.Contains(fc.gotSys, "a1") || !strings.Contains(fc.gotSys, "a2") {
		t.Errorf("expected a1/a2 included")
	}
	if strings.Contains(fc.gotSys, "b1") {
		t.Errorf("b1 should be excluded by limit")
	}
}

func TestRunWithNoDocsErrors(t *testing.T) {
	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}
	_, err := RunWith(promptsDir, "q", []Source{
		{Label: "docs", Dir: filepath.Join(t.TempDir(), "missing")},
	}, 0, fc, time.Second)
	if err == nil || !strings.Contains(err.Error(), "no documents") {
		t.Errorf("expected no-documents error, got %v", err)
	}
	if fc.calls != 0 {
		t.Errorf("should not have called completer")
	}
}

func TestRunWithEmptyQuestionErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ep.md", "body")
	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{resp: "ok"}

	_, err := RunWith(promptsDir, "   ", []Source{{Label: "x", Dir: dir}}, 0, fc, time.Second)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty-question error, got %v", err)
	}
}

func TestRunWithCompleterErrorBubbles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ep.md", "body")
	promptsDir := makePromptsDir(t)
	fc := &fakeCompleter{err: errors.New("boom")}

	_, err := RunWith(promptsDir, "q", []Source{{Label: "x", Dir: dir}}, 0, fc, time.Second)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected completer error, got %v", err)
	}
}

func TestHumanizeSlug(t *testing.T) {
	cases := map[string]string{
		"Nik_Sharma_and_Moiz_Ali__Why_Native_Rejected_Investors": "Nik Sharma and Moiz Ali — Why Native Rejected Investors",
		"Why_Nik_Sharma_Bought_Long_Wknd___How_to_Optimize":      "Why Nik Sharma Bought Long Wknd — How to Optimize",
		"Single":  "Single",
		"a__b__c": "a — b — c",
		"":        "",
	}
	for in, want := range cases {
		if got := humanizeSlug(in); got != want {
			t.Errorf("humanizeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStableAttrKeys(t *testing.T) {
	got := stableAttrKeys(map[string]string{
		"date":     "2022-07-20",
		"episode":  "1",
		"season":   "1",
		"title":    "T",
		"foo":      "bar",
		"alpha":    "first",
	})
	want := []string{"title", "season", "episode", "date", "alpha", "foo"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d]: %q != %q", i, got[i], want[i])
		}
	}
}
