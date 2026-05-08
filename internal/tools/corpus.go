package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"sharma-bot/internal/state"
)

// NewCorpusTools returns the three corpus-reading tools the agent loop uses
// to navigate sharma-bot's filesystem-backed knowledge base.
//
// Sources are discovered from the standard layout:
//
//	clean_v1/limited-supply/   (podcast transcripts; metadata from state.db)
//	raw/docs/                  (operator notes, copy guidelines, etc.)
//	raw/articles/              (collected outside writing)
//
// Missing source directories are tolerated — they just produce no results.
func NewCorpusTools(corpusDir string) []Tool {
	sources := defaultSources(corpusDir)
	meta := loadPodcastMeta(corpusDir, "limited-supply")
	return []Tool{
		globTool(sources, meta),
		grepTool(sources, meta),
		readDocTool(sources, meta),
	}
}

// source maps a label the model can use to a directory on disk. Order is
// stable so tool output is deterministic across runs.
type source struct {
	Label string
	Dir   string
}

func defaultSources(corpusDir string) []source {
	return []source{
		{Label: "limited-supply", Dir: filepath.Join(corpusDir, "clean_v1", "limited-supply")},
		{Label: "docs", Dir: filepath.Join(corpusDir, "raw", "docs")},
		{Label: "articles", Dir: filepath.Join(corpusDir, "raw", "articles")},
	}
}

// docInfo is one document discovered while walking the corpus.
type docInfo struct {
	Source string
	ID     string
	Path   string
	Title  string
}

// header renders a one-line label like:
//
//	limited-supply / 9lmar28vdkl4r2nw  ("Why Native Rejected Investors", S1E1, 2022-07-20)
//
// Used in glob/grep output so the model can cite by something readable.
func (d docInfo) header(meta map[string]map[string]string) string {
	attrs := meta[d.ID]
	parts := []string{fmt.Sprintf("%s / %s", d.Source, d.ID)}
	if title := attrs["title"]; title != "" {
		seasonEp := ""
		if s, e := attrs["season"], attrs["episode"]; s != "" && e != "" {
			seasonEp = fmt.Sprintf(", S%sE%s", s, e)
		}
		date := ""
		if dt := attrs["date"]; dt != "" {
			date = ", " + dt
		}
		parts = append(parts, fmt.Sprintf(`("%s"%s%s)`, title, seasonEp, date))
	}
	return strings.Join(parts, "  ")
}

// ───────────────────────────── glob ─────────────────────────────

type globInput struct {
	Source  string `json:"source,omitempty"`
	Pattern string `json:"pattern"`
}

func globTool(sources []source, meta map[string]map[string]string) Tool {
	return Tool{
		Name: "glob",
		Description: `List documents whose id matches a glob pattern. Use this to discover what's in the corpus before reading.

Patterns support shell-style wildcards: * matches any run of characters, ? matches one character, [abc] matches a character set.

Examples:
  pattern="*pdp*"               — docs about PDPs across all sources
  pattern="*", source="docs"    — every doc in the docs source
  pattern="retention/*"         — articles in a retention/ subfolder
  pattern="9lmar*"              — episodes whose external id starts with 9lmar

Sources available: limited-supply (podcast episodes), docs (operator notes), articles (outside writing). If source is omitted, all sources are searched.`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"pattern": {
					"type": "string",
					"description": "Glob pattern to match against document ids (filename without extension). Required."
				},
				"source": {
					"type": "string",
					"enum": ["limited-supply", "docs", "articles"],
					"description": "Restrict to one source. Omit to search all sources."
				}
			},
			"required": ["pattern"]
		}`),
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var in globInput
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("decode input: %w", err)
			}
			if in.Pattern == "" {
				return "", fmt.Errorf("pattern is required")
			}
			docs, err := walkSources(sources, in.Source)
			if err != nil {
				return "", err
			}
			var matches []docInfo
			for _, d := range docs {
				ok, err := filepath.Match(in.Pattern, d.ID)
				if err != nil {
					return "", fmt.Errorf("invalid pattern: %w", err)
				}
				if ok {
					matches = append(matches, d)
				}
			}
			if len(matches) == 0 {
				return fmt.Sprintf("No documents match pattern %q.", in.Pattern), nil
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "Found %d document(s):\n\n", len(matches))
			for _, m := range matches {
				sb.WriteString(m.header(meta))
				sb.WriteByte('\n')
			}
			return sb.String(), nil
		},
	}
}

// ───────────────────────────── grep ─────────────────────────────

type grepInput struct {
	Query      string `json:"query"`
	Source     string `json:"source,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

const (
	defaultMaxGrepResults  = 10
	defaultMaxSnippetsPerDoc = 3
	contextLines           = 1
)

func grepTool(sources []source, meta map[string]map[string]string) Tool {
	return Tool{
		Name: "grep",
		Description: `Search document bodies for a substring or phrase. Case-insensitive. Returns matching documents with context snippets so you can decide whether to read the full document with read_doc.

Use this when you want to find which documents discuss a topic. Be specific in your query — "loyalty program" is better than "loyalty".

Sources: limited-supply, docs, articles. Omit to search all.`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Substring or phrase to search for. Case-insensitive. Required."
				},
				"source": {
					"type": "string",
					"enum": ["limited-supply", "docs", "articles"],
					"description": "Restrict to one source. Omit to search all sources."
				},
				"max_results": {
					"type": "integer",
					"description": "Max number of matching documents to return. Default 10."
				}
			},
			"required": ["query"]
		}`),
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var in grepInput
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("decode input: %w", err)
			}
			if in.Query == "" {
				return "", fmt.Errorf("query is required")
			}
			max := in.MaxResults
			if max <= 0 {
				max = defaultMaxGrepResults
			}
			docs, err := walkSources(sources, in.Source)
			if err != nil {
				return "", err
			}
			needle := strings.ToLower(in.Query)

			type hit struct {
				info     docInfo
				snippets []string
			}
			var hits []hit
			for _, d := range docs {
				if len(hits) >= max {
					break
				}
				snips, err := findSnippets(d.Path, needle, defaultMaxSnippetsPerDoc, contextLines)
				if err != nil {
					return "", fmt.Errorf("grep %s: %w", d.Path, err)
				}
				if len(snips) > 0 {
					hits = append(hits, hit{info: d, snippets: snips})
				}
			}
			if len(hits) == 0 {
				return fmt.Sprintf("No matches for %q.", in.Query), nil
			}

			var sb strings.Builder
			fmt.Fprintf(&sb, "Found %d document(s) matching %q:\n", len(hits), in.Query)
			for _, h := range hits {
				sb.WriteString("\n")
				sb.WriteString(h.info.header(meta))
				sb.WriteString("\n")
				for _, s := range h.snippets {
					sb.WriteString("  ")
					sb.WriteString(strings.ReplaceAll(s, "\n", "\n  "))
					sb.WriteString("\n")
				}
			}
			return sb.String(), nil
		},
	}
}

// findSnippets reads the file line-by-line and returns up to maxSnips
// snippets, each containing the matched line plus `ctx` lines on either side.
// Adjacent or overlapping match windows are merged so one big paragraph with
// many hits doesn't burn the whole budget.
func findSnippets(path, needleLower string, maxSnips, ctx int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	type window struct{ start, end int } // inclusive
	var windows []window
	for i, line := range lines {
		if !strings.Contains(strings.ToLower(line), needleLower) {
			continue
		}
		w := window{start: max0(i - ctx), end: min(len(lines)-1, i+ctx)}
		if n := len(windows); n > 0 && windows[n-1].end >= w.start-1 {
			if windows[n-1].end < w.end {
				windows[n-1].end = w.end
			}
			continue
		}
		windows = append(windows, w)
		if len(windows) >= maxSnips {
			break
		}
	}

	out := make([]string, 0, len(windows))
	for _, w := range windows {
		out = append(out, strings.Join(lines[w.start:w.end+1], "\n"))
	}
	return out, nil
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─────────────────────────── read_doc ───────────────────────────

type readDocInput struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

func readDocTool(sources []source, meta map[string]map[string]string) Tool {
	return Tool{
		Name: "read_doc",
		Description: `Read the full content of a single document. Use this after glob or grep narrows you to a specific doc you want the full text of.

Source must be one of: limited-supply, docs, articles. The id is the value returned by glob or grep (no file extension).`,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"source": {
					"type": "string",
					"enum": ["limited-supply", "docs", "articles"],
					"description": "The source containing the document."
				},
				"id": {
					"type": "string",
					"description": "Document id (filename without extension, may include subdirectory path for docs/articles)."
				}
			},
			"required": ["source", "id"]
		}`),
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			var in readDocInput
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("decode input: %w", err)
			}
			if in.Source == "" || in.ID == "" {
				return "", fmt.Errorf("source and id are required")
			}
			var dir string
			for _, s := range sources {
				if s.Label == in.Source {
					dir = s.Dir
					break
				}
			}
			if dir == "" {
				return "", fmt.Errorf("unknown source %q", in.Source)
			}
			// Try .md first, then .txt.
			var data []byte
			var err error
			for _, ext := range []string{".md", ".txt"} {
				data, err = os.ReadFile(filepath.Join(dir, in.ID+ext))
				if err == nil {
					break
				}
				if !os.IsNotExist(err) {
					return "", err
				}
			}
			if err != nil {
				return "", fmt.Errorf("document not found: %s/%s", in.Source, in.ID)
			}
			info := docInfo{Source: in.Source, ID: in.ID}
			var sb strings.Builder
			sb.WriteString(info.header(meta))
			sb.WriteString("\n\n")
			sb.Write(data)
			return sb.String(), nil
		},
	}
}

// ─────────────────────── shared helpers ────────────────────────

// walkSources returns every .md/.txt file under the given sources. If
// `onlySource` is non-empty, restricts to that one. Missing directories are
// skipped silently (matches ask's behavior).
func walkSources(sources []source, onlySource string) ([]docInfo, error) {
	var out []docInfo
	for _, s := range sources {
		if onlySource != "" && s.Label != onlySource {
			continue
		}
		info, err := os.Stat(s.Dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
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
			out = append(out, docInfo{Source: s.Label, ID: id, Path: path})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// loadPodcastMeta reads state.db (if it exists) and returns a map keyed by
// external_id holding human-readable attrs. Returns nil if state.db is missing
// or unreadable — tools must keep working without it.
func loadPodcastMeta(corpusDir, source string) map[string]map[string]string {
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

func humanizeSlug(s string) string {
	s = strings.ReplaceAll(s, "__", " — ")
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Join(strings.Fields(s), " ")
}
