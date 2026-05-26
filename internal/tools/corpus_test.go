package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture builds a tempdir corpus matching the standard layout:
//
//	{corpus}/clean_v1/limited-supply/<id>.md
//	{corpus}/raw/docs/<id>.{md,txt}
//	{corpus}/raw/articles/<id>.{md,txt}
//
// and returns the corpusDir + a slice of tools.
func fixture(t *testing.T, files map[string]string) (corpusDir string, tools []Tool) {
	t.Helper()
	corpusDir = t.TempDir()
	for rel, body := range files {
		full := filepath.Join(corpusDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return corpusDir, NewCorpusTools(corpusDir)
}

func runTool(t *testing.T, tools []Tool, name, inputJSON string) string {
	t.Helper()
	tool, ok := ByName(tools, name)
	if !ok {
		t.Fatalf("tool %q not found", name)
	}
	out, err := tool.Run(context.Background(), json.RawMessage(inputJSON))
	if err != nil {
		t.Fatalf("%s.Run: %v", name, err)
	}
	return out
}

// ───────────────────────────── glob ─────────────────────────────

func TestGlobAcrossSources(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"clean_v1/limited-supply/abc123.md": "podcast about pdps",
		"clean_v1/limited-supply/xyz789.md": "podcast about retention",
		"raw/docs/pdp-guide.md":             "doc body",
		"raw/articles/pdp-teardown.md":      "article body",
	})

	got := runTool(t, tools, "glob", `{"pattern": "*pdp*"}`)
	for _, want := range []string{
		"docs / pdp-guide",
		"articles / pdp-teardown",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	if strings.Contains(got, "abc123") || strings.Contains(got, "xyz789") {
		t.Errorf("ids without 'pdp' should not match; got:\n%s", got)
	}
}

func TestGlobScopedToSource(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"clean_v1/limited-supply/abc.md": "x",
		"raw/docs/abc.md":                "y",
	})
	got := runTool(t, tools, "glob", `{"pattern": "*", "source": "docs"}`)
	if !strings.Contains(got, "docs / abc") {
		t.Errorf("expected docs match: %s", got)
	}
	if strings.Contains(got, "limited-supply") {
		t.Errorf("source filter should exclude limited-supply: %s", got)
	}
}

func TestGlobNoMatches(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/foo.md": "x",
	})
	got := runTool(t, tools, "glob", `{"pattern": "*nope*"}`)
	if !strings.Contains(got, "No documents match") {
		t.Errorf("expected no-match message: %s", got)
	}
}

func TestGlobRejectsEmptyPattern(t *testing.T) {
	_, tools := fixture(t, map[string]string{"raw/docs/x.md": "y"})
	tool, _ := ByName(tools, "glob")
	if _, err := tool.Run(context.Background(), json.RawMessage(`{"pattern": ""}`)); err == nil {
		t.Fatal("expected error for empty pattern")
	}
}

func TestGlobInvalidJSON(t *testing.T) {
	_, tools := fixture(t, map[string]string{"raw/docs/x.md": "y"})
	tool, _ := ByName(tools, "glob")
	if _, err := tool.Run(context.Background(), json.RawMessage(`not json`)); err == nil {
		t.Fatal("expected decode error")
	}
}

// ───────────────────────────── grep ─────────────────────────────

func TestGrepFindsMatchesWithSnippets(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/loyalty.md": `intro paragraph

a paragraph about loyalty programs and how they work.

something else entirely.

another mention of loyalty here.`,
	})

	got := runTool(t, tools, "grep", `{"query": "loyalty"}`)
	if !strings.Contains(got, "Found 1 document") {
		t.Errorf("expected 1 doc match: %s", got)
	}
	if !strings.Contains(got, "loyalty programs") {
		t.Errorf("snippet missing match line: %s", got)
	}
}

func TestGrepCaseInsensitive(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/x.md": "Loyalty Programs Are Important",
	})
	got := runTool(t, tools, "grep", `{"query": "LOYALTY"}`)
	if !strings.Contains(got, "Loyalty Programs") {
		t.Errorf("case-insensitive grep failed: %s", got)
	}
}

func TestGrepRespectsSourceFilter(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/x.md":     "loyalty in docs",
		"raw/articles/y.md": "loyalty in articles",
	})
	got := runTool(t, tools, "grep", `{"query": "loyalty", "source": "docs"}`)
	if !strings.Contains(got, "loyalty in docs") {
		t.Errorf("expected docs match: %s", got)
	}
	if strings.Contains(got, "loyalty in articles") {
		t.Errorf("source filter should exclude articles: %s", got)
	}
}

func TestGrepRespectsMaxResults(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 5; i++ {
		files["raw/docs/file"+string(rune('a'+i))+".md"] = "loyalty match"
	}
	_, tools := fixture(t, files)
	got := runTool(t, tools, "grep", `{"query": "loyalty", "max_results": 2}`)
	if !strings.Contains(got, "Found 2 document") {
		t.Errorf("expected exactly 2 hits: %s", got)
	}
}

func TestGrepNoMatchesMessage(t *testing.T) {
	_, tools := fixture(t, map[string]string{"raw/docs/x.md": "nothing useful"})
	got := runTool(t, tools, "grep", `{"query": "loyalty"}`)
	if !strings.Contains(got, "No matches") {
		t.Errorf("expected no-match message: %s", got)
	}
}

func TestGrepRequiresQuery(t *testing.T) {
	_, tools := fixture(t, map[string]string{"raw/docs/x.md": "y"})
	tool, _ := ByName(tools, "grep")
	if _, err := tool.Run(context.Background(), json.RawMessage(`{"query": ""}`)); err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestGrepMergesAdjacentMatches(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/x.md": "loyalty\nloyalty\nloyalty",
	})
	got := runTool(t, tools, "grep", `{"query": "loyalty"}`)
	if !strings.Contains(got, "loyalty\n  loyalty\n  loyalty") {
		t.Errorf("expected merged window with 3 lines, got:\n%s", got)
	}
}

// ─────────────────────────── read_doc ───────────────────────────

func TestReadDocReturnsBody(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/copy-guideline.md": "the body of the doc",
	})
	got := runTool(t, tools, "read_doc", `{"source": "docs", "id": "copy-guideline"}`)
	if !strings.Contains(got, "docs / copy-guideline") {
		t.Errorf("missing header: %s", got)
	}
	if !strings.Contains(got, "the body of the doc") {
		t.Errorf("missing body: %s", got)
	}
}

func TestReadDocFallsBackToTxt(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/docs/copy-guideline.txt": "txt body",
	})
	got := runTool(t, tools, "read_doc", `{"source": "docs", "id": "copy-guideline"}`)
	if !strings.Contains(got, "txt body") {
		t.Errorf("should find .txt fallback: %s", got)
	}
}

func TestReadDocSubdirectoryPath(t *testing.T) {
	_, tools := fixture(t, map[string]string{
		"raw/articles/retention/bfcm.md": "bfcm article",
	})
	got := runTool(t, tools, "read_doc", `{"source": "articles", "id": "retention/bfcm"}`)
	if !strings.Contains(got, "bfcm article") {
		t.Errorf("subdirectory id failed: %s", got)
	}
}

func TestReadDocMissing(t *testing.T) {
	_, tools := fixture(t, map[string]string{"raw/docs/x.md": "y"})
	tool, _ := ByName(tools, "read_doc")
	if _, err := tool.Run(context.Background(), json.RawMessage(`{"source": "docs", "id": "nope"}`)); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestReadDocUnknownSource(t *testing.T) {
	_, tools := fixture(t, map[string]string{"raw/docs/x.md": "y"})
	tool, _ := ByName(tools, "read_doc")
	if _, err := tool.Run(context.Background(), json.RawMessage(`{"source": "fakeland", "id": "x"}`)); err == nil {
		t.Fatal("expected unknown-source error")
	}
}

// ─────────────────────── helpers ───────────────────────

func TestByName(t *testing.T) {
	tools := []Tool{{Name: "alpha"}, {Name: "beta"}}
	if _, ok := ByName(tools, "beta"); !ok {
		t.Fatal("expected beta")
	}
	if _, ok := ByName(tools, "missing"); ok {
		t.Fatal("expected miss to return false")
	}
}

func TestHumanizeSlug(t *testing.T) {
	cases := map[string]string{
		"Nik_Sharma__Why_Native": "Nik Sharma — Why Native",
		"Single":                 "Single",
		"":                       "",
	}
	for in, want := range cases {
		if got := humanizeSlug(in); got != want {
			t.Errorf("humanizeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}
