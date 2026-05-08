package chats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sharma-bot/internal/ai"
)

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSaveDefaultPath(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 5, 8, 15, 5, 23, 0, time.UTC)
	path, err := Save(dir, "", Chat{
		Date:     date,
		Command:  "dig",
		Title:    "What about loyalty programs?",
		Question: "what about loyalty programs?",
		Answer:   "they're important.",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "2026-05-08", "15-05-23-what-about-loyalty-programs.md")
	if path != want {
		t.Errorf("path: got %q, want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestSaveOutOverride(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom", "place.md")
	got, err := Save("ignored", custom, Chat{
		Question: "q",
		Answer:   "a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != custom {
		t.Errorf("path: %q", got)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatal("file not written:", err)
	}
}

func TestRenderFrontmatter(t *testing.T) {
	c := Chat{
		Date:    time.Date(2026, 5, 8, 15, 5, 23, 0, time.UTC),
		Command: "dig",
		Title:   "loyalty",
		Result: &ai.CallResult{
			Answer:  "x",
			Steps:   3,
			Elapsed: 12_300 * time.Millisecond,
			Usage: ai.Usage{
				Model:               "claude-sonnet-4-6",
				InputTokens:         100,
				CacheCreationTokens: 200,
				CacheReadTokens:     50,
				OutputTokens:        20,
			},
		},
		Question: "loyalty?",
		Answer:   "the answer",
	}
	got := render(c)
	for _, want := range []string{
		"date: 2026-05-08T15:05:23Z",
		"command: dig",
		`title: "loyalty"`,
		"model: claude-sonnet-4-6",
		"steps: 3",
		"input_tokens: 100",
		"cache_creation_tokens: 200",
		"cache_read_tokens: 50",
		"output_tokens: 20",
		"cost_usd: 0.",
		"duration_sec: 12.30",
		"# Question",
		"loyalty?",
		"# Answer",
		"the answer",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered missing %q:\n%s", want, got)
		}
	}
}

func TestRenderOmitsCacheZeros(t *testing.T) {
	c := Chat{
		Result: &ai.CallResult{
			Usage: ai.Usage{
				Model:        "m",
				InputTokens:  100,
				OutputTokens: 50,
			},
		},
		Question: "q",
		Answer:   "a",
	}
	got := render(c)
	if strings.Contains(got, "cache_creation_tokens") {
		t.Error("zero cache_creation_tokens should be omitted")
	}
	if strings.Contains(got, "cache_read_tokens") {
		t.Error("zero cache_read_tokens should be omitted")
	}
}

func TestRenderTraceFenced(t *testing.T) {
	c := Chat{
		Question: "q",
		Answer:   "a",
		Trace:    "[step 1] ...\n[step 2] final answer",
	}
	got := render(c)
	if !strings.Contains(got, "# Trace") {
		t.Error("expected Trace section")
	}
	if !strings.Contains(got, "```\n[step 1]") {
		t.Error("expected fenced code block")
	}
}

func TestRenderNoTraceSectionWhenEmpty(t *testing.T) {
	c := Chat{Question: "q", Answer: "a", Trace: "   \n"}
	got := render(c)
	if strings.Contains(got, "# Trace") {
		t.Error("empty trace should not produce section")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"What about loyalty programs?":                    "what-about-loyalty-programs",
		"Hello, World!":                                   "hello-world",
		"   spaces   only  ":                              "spaces-only",
		"":                                                "",
		"!!!":                                             "",
		"AlReAdY-AlNuM":                                   "already-alnum",
		"this is going to be way too long for a slug because we have a lot of words here": "this-is-going-to-be-way-too-long-for-a-slug-because-we-have",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestYamlStringEscaping(t *testing.T) {
	cases := map[string]string{
		`simple`:           `"simple"`,
		`with "quotes"`:    `"with \"quotes\""`,
		"new\nline":        `"new\nline"`,
		`back\slash`:       `"back\\slash"`,
	}
	for in, want := range cases {
		if got := yamlString(in); got != want {
			t.Errorf("yamlString(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveWithEmptyTitleSlugFallsBack(t *testing.T) {
	dir := t.TempDir()
	path, err := Save(dir, "", Chat{
		Date:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Command: "dig",
		// Title is just punctuation -> slug becomes empty -> falls back to "chat"
		Title:    "!!!",
		Question: "q",
		Answer:   "a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "00-00-00-chat.md") {
		t.Errorf("expected fallback slug, got %q", path)
	}
}

func TestSaveCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "deeply", "nested", "out.md")
	if _, err := Save("", out, Chat{Question: "q", Answer: "a"}); err != nil {
		t.Fatal(err)
	}
	body := mustReadFile(t, out)
	if !strings.Contains(body, "# Question") {
		t.Error("body missing Question section")
	}
}
