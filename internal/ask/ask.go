package ask

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/state"
)

const (
	model     = anthropic.ModelClaudeSonnet4_6
	maxTokens = int64(8192)
	timeout   = 5 * time.Minute
)

// Source is one root directory of documents to load. Missing dirs are
// silently skipped so a user can omit `docs/` or `articles/` until they
// actually have content for them.
type Source struct {
	Label string // tag attribute, e.g. "limited-supply"
	Dir   string // root directory; recursively walked for .md and .txt
	// Metadata, optional, maps a document id to extra <doc> tag
	// attributes (e.g. title, season, episode, date). Used for podcast
	// sources where the state DB has the human-readable title.
	Metadata map[string]map[string]string
}

// Document is one loaded file with its rendering attributes.
type Document struct {
	Source string
	ID     string
	Body   string
	Attrs  map[string]string
}

// DefaultSources returns the sources `corpus ask` loads by default.
// Order is stable (limited-supply, docs, articles) so the cached system
// prompt stays cache-friendly across runs.
func DefaultSources(corpusDir string) []Source {
	return []Source{
		{
			Label:    "limited-supply",
			Dir:      filepath.Join(corpusDir, "clean_v1", "limited-supply"),
			Metadata: loadPodcastMetadata(corpusDir, "limited-supply"),
		},
		{Label: "docs", Dir: filepath.Join(corpusDir, "raw", "docs")},
		{Label: "articles", Dir: filepath.Join(corpusDir, "raw", "articles")},
	}
}

// Run is the entry used by main: it builds default sources and a real Completer
// with the 1M-context beta enabled (the corpus exceeds 200K tokens once
// stripped in full).
func Run(corpusDir, promptsDir, question string, limit int) (string, error) {
	sources := DefaultSources(corpusDir)
	completer := ai.NewCompleter(model, maxTokens, ai.WithLongContext())
	return RunWith(promptsDir, question, sources, limit, completer, timeout)
}

// RunWith is the testable form. It accepts an explicit source list, an
// injected Completer, and a per-call timeout.
func RunWith(promptsDir, question string, sources []Source, limit int, completer ai.Completer, perCallTimeout time.Duration) (string, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("question is empty")
	}

	role, err := os.ReadFile(filepath.Join(promptsDir, "ask.md"))
	if err != nil {
		return "", fmt.Errorf("read role prompt: %w", err)
	}

	docs, summary, err := loadDocuments(sources, limit)
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "", fmt.Errorf("no documents found in any source")
	}

	system := buildSystemPrompt(string(role), docs)
	fmt.Fprintf(os.Stderr, "loaded %s (~%d KB system prompt)\n", summary, len(system)/1024)

	ctx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()

	start := time.Now()
	answer, usage, err := completer.Complete(ctx, system, question)
	elapsed := time.Since(start)
	ai.PrintTelemetry(os.Stderr, usage, elapsed, "")
	return answer, err
}

func loadDocuments(sources []Source, limit int) ([]Document, string, error) {
	var all []Document
	parts := make([]string, 0, len(sources))
	for _, s := range sources {
		docs, err := walkSource(s)
		if err != nil {
			return nil, "", fmt.Errorf("walk %s: %w", s.Dir, err)
		}
		if len(docs) > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", len(docs), s.Label))
		}
		all = append(all, docs...)
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	summary := strings.Join(parts, ", ")
	if summary == "" {
		summary = "0 documents"
	}
	return all, summary, nil
}

func walkSource(s Source) ([]Document, error) {
	info, err := os.Stat(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", s.Dir)
	}
	var docs []Document
	walkErr := filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" {
			return nil
		}
		rel, err := filepath.Rel(s.Dir, path)
		if err != nil {
			return err
		}
		id := filepath.ToSlash(strings.TrimSuffix(rel, filepath.Ext(rel)))
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		docs = append(docs, Document{
			Source: s.Label,
			ID:     id,
			Body:   strings.TrimSpace(string(body)),
			Attrs:  s.Metadata[id],
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
	return docs, nil
}

func buildSystemPrompt(role string, docs []Document) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(role))
	sb.WriteString("\n\n# Knowledge corpus\n\n")
	for _, d := range docs {
		fmt.Fprintf(&sb, "<doc source=%q id=%q", d.Source, d.ID)
		// Stable attribute order so the cache key is stable.
		for _, k := range stableAttrKeys(d.Attrs) {
			fmt.Fprintf(&sb, " %s=%q", k, d.Attrs[k])
		}
		fmt.Fprintf(&sb, ">\n%s\n</doc>\n\n", d.Body)
	}
	return sb.String()
}

// stableAttrKeys returns the attribute keys in a fixed display order, with
// any unknown keys sorted alphabetically at the end.
func stableAttrKeys(attrs map[string]string) []string {
	if len(attrs) == 0 {
		return nil
	}
	preferred := []string{"title", "season", "episode", "date"}
	out := make([]string, 0, len(attrs))
	used := make(map[string]bool, len(attrs))
	for _, k := range preferred {
		if _, ok := attrs[k]; ok {
			out = append(out, k)
			used[k] = true
		}
	}
	var extra []string
	for k := range attrs {
		if !used[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// loadPodcastMetadata reads episode metadata from state.db and returns a map
// keyed by external_id. Failures (missing DB, empty DB) silently degrade to
// nil so ask still works without it.
func loadPodcastMetadata(corpusDir, source string) map[string]map[string]string {
	dbPath := filepath.Join(corpusDir, "state.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	db, err := state.Open(dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()
	eps, err := state.AllBySource(db, source)
	if err != nil {
		return nil
	}
	m := make(map[string]map[string]string, len(eps))
	for extID, ep := range eps {
		attrs := make(map[string]string)
		if ep.TitleSlug != "" {
			attrs["title"] = humanizeSlug(ep.TitleSlug)
		}
		if ep.Season.Valid {
			attrs["season"] = strconv.FormatInt(ep.Season.Int64, 10)
		}
		if ep.EpisodeNum.Valid {
			attrs["episode"] = strconv.FormatInt(ep.EpisodeNum.Int64, 10)
		}
		if ep.AirDate != "" {
			attrs["date"] = ep.AirDate
		}
		m[extID] = attrs
	}
	return m
}

// humanizeSlug turns "Nik_Sharma_and_Moiz_Ali__Why_Native_Rejected_Investors"
// into "Nik Sharma and Moiz Ali — Why Native Rejected Investors".
func humanizeSlug(s string) string {
	s = strings.ReplaceAll(s, "__", " — ")
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Join(strings.Fields(s), " ")
}
