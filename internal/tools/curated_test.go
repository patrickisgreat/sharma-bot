package tools

import (
	"strings"
	"testing"
)

func TestCuratedContextIncludesDocsAndArticles(t *testing.T) {
	corpusDir, _ := fixture(t, map[string]string{
		"raw/docs/copy-guideline.txt":    "Always lead with the benefit.",
		"raw/articles/landpage-audit.md": "Audit: most heroes are too vague.",
		"clean_v1/limited-supply/ep1.md": "podcast transcript that must NOT be inlined",
	})

	got, err := CuratedContext(corpusDir, "docs", "articles")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"docs / copy-guideline", "Always lead with the benefit.",
		"articles / landpage-audit", "Audit: most heroes are too vague.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("curated context missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(got, "podcast transcript") {
		t.Error("limited-supply leaked into curated context")
	}
}

func TestCuratedContextRespectsLabelFilter(t *testing.T) {
	corpusDir, _ := fixture(t, map[string]string{
		"raw/docs/d1.txt":     "doc one",
		"raw/articles/a1.txt": "article one",
	})
	got, err := CuratedContext(corpusDir, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "doc one") || strings.Contains(got, "article one") {
		t.Errorf("label filter not respected: %q", got)
	}
}

func TestCuratedContextEmptyWhenNoDocs(t *testing.T) {
	corpusDir, _ := fixture(t, map[string]string{
		"clean_v1/limited-supply/ep1.md": "only a podcast",
	})
	got, err := CuratedContext(corpusDir, "docs", "articles")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestWithCuratedContext(t *testing.T) {
	if got := WithCuratedContext("ROLE", ""); got != "ROLE" {
		t.Errorf("empty curated should return role unchanged, got %q", got)
	}
	got := WithCuratedContext("ROLE", "docs / x\n\nbody")
	if !strings.HasPrefix(got, "ROLE\n\n") {
		t.Errorf("role should come first: %q", got)
	}
	if !strings.Contains(got, "already loaded") || !strings.Contains(got, "docs / x") {
		t.Errorf("missing header or curated body: %q", got)
	}
}
