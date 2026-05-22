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
	"net/url"
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
	// editMaxSteps is much higher than maxSteps: write mode spends ~10 steps
	// grounding in the corpus, then one edit_file call per turn across a
	// multi-section file, plus a final summary. 12 wasn't enough to get past
	// research into editing.
	editMaxSteps = 40
	timeout      = 5 * time.Minute
	// editTimeout is longer than timeout: a write run does corpus research
	// plus many edit_file turns over a multi-section file.
	editTimeout     = 12 * time.Minute
	defaultMaxPages = 20
	renderTimeout   = 60 * time.Second
)

// Options configures one review run. Exactly one mode is selected: Path (a
// single file, or stdin when "" / "-"), BatchDir (a folder), or CrawlURL (a
// one-hop site crawl).
//
// Write turns reports into edits: instead of only critiquing, the agent
// rewrites the source files in place using the edit_file tool. Write applies
// to Path and BatchDir (local source), not CrawlURL (a remote mirror can't be
// edited). Force skips the clean-git-tree guard.
type Options struct {
	CorpusDir  string
	PromptsDir string
	Path       string
	BatchDir   string
	CrawlURL   string
	MaxPages   int
	Write      bool
	Force      bool
	Stdin      io.Reader
	Trace      io.Writer
}

// runConfig is the resolved, per-run state the mode functions share, so they
// don't each take a dozen positional arguments.
type runConfig struct {
	role      string // review.md (+ edit.md appended in write mode)
	write     bool
	completer ai.ToolCompleter
	baseTools []tools.Tool
	trace     io.Writer
	timeout   time.Duration
}

// Run reviews content per opts. Report mode returns a CallResult for single-
// file input (so the caller can persist the chat); batch/crawl modes write
// per-file outputs under reviews/<timestamp>/ and return (nil, nil).
//
// opts.Trace receives per-step trace lines and, for batch/crawl, per-file
// status. Pass nil to suppress.
func Run(opts Options) (*ai.CallResult, error) {
	completer := ai.NewToolCompleter(model, maxTokens)
	perCall := timeout
	if opts.Write {
		perCall = editTimeout
	}
	return RunWith(opts, completer, tools.NewCorpusTools(opts.CorpusDir), perCall)
}

// RunWith is the testable form: explicit completer + base tools + timeout.
// baseTools are the corpus-reading tools; in write mode a per-file edit_file
// tool is appended to them.
func RunWith(opts Options, completer ai.ToolCompleter, baseTools []tools.Tool, perCallTimeout time.Duration) (*ai.CallResult, error) {
	// Write mode uses a dedicated edit role (edit.md), not the reviewer role
	// (review.md): the reviewer prompt's "produce a report" output format
	// overpowers an appended override and the model never calls edit_file.
	rolePrompt := "review.md"
	if opts.Write {
		rolePrompt = "edit.md"
	}
	role, err := os.ReadFile(filepath.Join(opts.PromptsDir, rolePrompt))
	if err != nil {
		return nil, fmt.Errorf("read role prompt (%s): %w", rolePrompt, err)
	}
	cfg := runConfig{
		role:      string(role),
		write:     opts.Write,
		completer: completer,
		baseTools: baseTools,
		trace:     opts.Trace,
		timeout:   perCallTimeout,
	}

	if opts.Write {
		if opts.CrawlURL != "" {
			return nil, fmt.Errorf("--write applies to local files, not --crawl (a remote site can't be edited)")
		}
		if opts.Path == "" || opts.Path == "-" {
			return nil, fmt.Errorf("--write needs a file or --batch directory, not stdin")
		}
		guardDir := opts.BatchDir
		if guardDir == "" {
			guardDir = filepath.Dir(opts.Path)
		}
		if err := ensureCleanTree(guardDir, opts.Force); err != nil {
			return nil, err
		}
	}

	switch {
	case opts.CrawlURL != "":
		return nil, runCrawl(cfg, opts.CrawlURL, opts.MaxPages)
	case opts.BatchDir != "":
		return nil, runBatch(cfg, opts.BatchDir)
	default:
		return runSingle(cfg, opts.Path, opts.Stdin, os.Stdout)
	}
}

func runSingle(cfg runConfig, path string, stdinReader io.Reader, out io.Writer) (*ai.CallResult, error) {
	if cfg.write {
		er, res, err := editOne(cfg, path)
		if err != nil {
			return nil, err
		}
		reportEdit(out, filepath.Base(path), er)
		if _, err := fmt.Fprintln(out, res.Answer); err != nil {
			return nil, err
		}
		return res, nil
	}
	content, label, err := loadContent(path, stdinReader)
	if err != nil {
		return nil, err
	}
	res, err := reviewOne(content, label, cfg.role, cfg.completer, cfg.baseTools, cfg.trace, cfg.timeout)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(out, res.Answer); err != nil {
		return nil, err
	}
	return res, nil
}

func runBatch(cfg runConfig, batchDir string) error {
	files, err := listSupportedFiles(batchDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no supported files (%s) under %s", strings.Join(supportedExts, ", "), batchDir)
	}
	outDir := filepath.Join("reviews", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	verb := "reviewing"
	if cfg.write {
		verb = "reviewing + editing"
	}
	if cfg.trace != nil {
		fmt.Fprintf(cfg.trace, "%s %d file(s) → %s\n", verb, len(files), outDir)
	}
	var ok, fail int
	for _, f := range files {
		rel, _ := filepath.Rel(batchDir, f)
		if cfg.trace != nil {
			fmt.Fprintf(cfg.trace, "\n=== %s ===\n", rel)
		}
		answer, err := reviewFileForBatch(cfg, f, rel)
		if err != nil {
			if cfg.trace != nil {
				fmt.Fprintf(cfg.trace, "  error: %v\n", err)
			}
			fail++
			continue
		}
		outPath := filepath.Join(outDir, sanitizeForFilename(rel)+".review.md")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(answer), 0o644); err != nil {
			return err
		}
		ok++
	}
	if cfg.trace != nil {
		fmt.Fprintf(cfg.trace, "\nreview: %d ok, %d failed → %s\n", ok, fail, outDir)
	}
	return nil
}

// reviewFileForBatch reviews (and, in write mode, edits in place) one batch
// file and returns the review text to save. Per-file status goes to the
// trace writer.
func reviewFileForBatch(cfg runConfig, path, rel string) (string, error) {
	if cfg.write {
		er, res, err := editOne(cfg, path)
		if err != nil {
			return "", err
		}
		reportEdit(cfg.trace, rel, er)
		return res.Answer, nil
	}
	content, label, err := loadContent(path, nil)
	if err != nil {
		return "", err
	}
	res, err := reviewOne(content, label, cfg.role, cfg.completer, cfg.baseTools, cfg.trace, cfg.timeout)
	if err != nil {
		return "", err
	}
	return res.Answer, nil
}

func runCrawl(cfg runConfig, startURL string, maxPages int) error {
	if maxPages <= 0 {
		maxPages = defaultMaxPages
	}
	base, err := url.Parse(startURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return fmt.Errorf("crawl: invalid URL %q (need http(s)://host/...)", startURL)
	}

	status := cfg.trace
	if status == nil {
		status = io.Discard
	}

	outDir := filepath.Join("reviews", time.Now().Format("20060102-150405"))
	pagesDir := filepath.Join(outDir, "_pages")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		return err
	}

	fmt.Fprintf(status, "launching headless chrome...\n")
	r, err := newRenderer(context.Background(), renderSettle)
	if err != nil {
		return err
	}
	defer r.close()

	if pw := os.Getenv("SHOPIFY_PASSWORD"); pw != "" && shouldAuthenticate(base, os.Getenv("SHOPIFY_URL")) {
		fmt.Fprintf(status, "authenticating to %s...\n", base.Host)
		authCtx, cancel := context.WithTimeout(context.Background(), renderTimeout)
		err := r.authenticate(authCtx, base.Scheme+"://"+base.Host, pw)
		cancel()
		if err != nil {
			return fmt.Errorf("crawl auth: %w", err)
		}
	}

	fmt.Fprintf(status, "rendering %s (cap %d page(s))\n", startURL, maxPages)
	startData, err := renderWith(r, cfg.timeout, base.String())
	if err != nil {
		return fmt.Errorf("render homepage: %w", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, slugFromURL(base)+".html"), startData, 0o644); err != nil {
		return err
	}

	links := extractLinks(base, startData)
	queue := []string{base.String()}
	queue = append(queue, links...)
	if len(queue) > maxPages {
		queue = queue[:maxPages]
	}
	fmt.Fprintf(status, "found %d link(s); reviewing %d page(s) → %s\n", len(links), len(queue), outDir)

	var fetchErrs, reviewErrs, ok int
	for i, raw := range queue {
		u, _ := url.Parse(raw)
		slug := slugFromURL(u)
		htmlPath := filepath.Join(pagesDir, slug+".html")

		fmt.Fprintf(status, "\n[%d/%d] %s\n", i+1, len(queue), raw)

		var data []byte
		if i == 0 {
			data = startData
		} else {
			fetchStart := time.Now()
			data, err = renderWith(r, cfg.timeout, raw)
			if err != nil {
				fmt.Fprintf(status, "  render error: %v\n", err)
				fetchErrs++
				continue
			}
			fmt.Fprintf(status, "  rendered in %s (%d bytes)\n", time.Since(fetchStart).Round(time.Millisecond), len(data))
			if err := os.WriteFile(htmlPath, data, 0o644); err != nil {
				return err
			}
		}

		content := ExtractText(data)
		label := "page at " + raw + " (HTML extracted)"
		reviewStart := time.Now()
		res, err := reviewOne(content, label, cfg.role, cfg.completer, cfg.baseTools, cfg.trace, cfg.timeout)
		if err != nil {
			fmt.Fprintf(status, "  review error: %v\n", err)
			reviewErrs++
			continue
		}
		outPath := filepath.Join(outDir, slug+".review.md")
		header := fmt.Sprintf("# %s\n\nReview: %s\n\n---\n\n", titleFromURL(u), raw)
		body := stripPreamble(res.Answer)
		if err := os.WriteFile(outPath, []byte(header+body), 0o644); err != nil {
			return err
		}
		if err := writeDOCX(outPath); err != nil {
			fmt.Fprintf(status, "  docx: skipped (%v)\n", err)
		}
		fmt.Fprintf(status, "  reviewed in %s → %s\n", time.Since(reviewStart).Round(time.Millisecond), outPath)
		ok++
	}

	fmt.Fprintf(status, "\ncrawl: %d reviewed, %d render error(s), %d review error(s) → %s\n", ok, fetchErrs, reviewErrs, outDir)
	return nil
}

// renderWith wraps renderer.render with a fresh per-request context+timeout
// so the caller doesn't have to thread one through for every call.
func renderWith(r *renderer, perCallTimeout time.Duration, rawURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()
	return r.render(ctx, rawURL)
}

// shouldAuthenticate decides whether to attempt the Shopify password-gate
// flow. SHOPIFY_PASSWORD is paired with SHOPIFY_URL in the user's .env, so
// we only auth when the crawl target shares a host with SHOPIFY_URL —
// otherwise the password input never appears and the form-fill stalls until
// the per-call timeout fires.
func shouldAuthenticate(crawlBase *url.URL, shopifyURL string) bool {
	if shopifyURL == "" {
		return false
	}
	su, err := url.Parse(shopifyURL)
	if err != nil || su.Host == "" {
		return false
	}
	return strings.EqualFold(crawlBase.Host, su.Host)
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

// editResult summarizes what editOne did to a file.
type editResult struct {
	edits      int    // edits applied and written
	rolledBack bool   // edits were made but reverted (e.g. broke JSON)
	rollback   string // reason for rollback, when rolledBack
	truncated  bool   // agent loop errored (timeout/max-steps) but edits were salvaged
}

// editOne reviews the file at path against the corpus and applies the agent's
// edits in place via the edit_file tool. The returned CallResult.Answer is
// the review rationale (saved as the .review.md). For .json files, edits that
// would break JSON validity are rolled back rather than written.
//
// Edits accumulate in an in-memory buffer as the model calls edit_file, so if
// the agent loop errors out (timeout, max-steps) after making valid edits, we
// still flush them — the edits are real work; only the closing summary is
// lost. A hard error with no edits made is propagated.
func editOne(cfg runConfig, path string) (editResult, *ai.CallResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return editResult{}, nil, err
	}
	editor := newFileEditor(path, string(raw))
	ts := append(append([]tools.Tool{}, cfg.baseTools...), editor.tool())

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	userPrompt := fmt.Sprintf(
		"Review and revise this file (path: %s). Apply your changes with edit_file, "+
			"then write the review explaining what you changed and why.\n\n---\n%s\n---\n",
		path, string(raw))
	start := time.Now()
	res, agentErr := agent.Run(ctx, agent.Config{
		Completer: cfg.completer,
		Tools:     ts,
		MaxSteps:  editMaxSteps,
		Trace:     cfg.trace,
	}, cfg.role, userPrompt)

	// No edits made. Propagate a real error; otherwise report review-only.
	if !editor.changed() {
		if agentErr != nil {
			return editResult{}, nil, agentErr
		}
		ai.PrintTelemetry(cfg.trace, res.Usage, time.Since(start), fmt.Sprintf("%d step(s)", res.Steps))
		return editResult{edits: 0}, &ai.CallResult{Answer: res.Answer, Usage: res.Usage, Elapsed: time.Since(start), Steps: res.Steps}, nil
	}

	// Edits were made. Salvage them even if the loop errored mid-run.
	answer := ""
	var usage ai.Usage
	steps := 0
	if res != nil {
		answer, usage, steps = res.Answer, res.Usage, res.Steps
	}
	if agentErr != nil && answer == "" {
		answer = fmt.Sprintf("_(run truncated before summary: %v — edits below were still applied)_", agentErr)
	}
	ai.PrintTelemetry(cfg.trace, usage, time.Since(start), fmt.Sprintf("%d step(s)", steps))
	call := &ai.CallResult{Answer: answer, Usage: usage, Elapsed: time.Since(start), Steps: steps}

	if strings.EqualFold(filepath.Ext(path), ".json") {
		if err := validJSON([]byte(editor.content)); err != nil {
			return editResult{edits: editor.edits, rolledBack: true, rollback: err.Error()}, call, nil
		}
	}
	if err := os.WriteFile(path, []byte(editor.content), 0o644); err != nil {
		return editResult{}, nil, err
	}
	return editResult{edits: editor.edits, truncated: agentErr != nil}, call, nil
}

// reportEdit prints a one-line outcome for an edited file to w.
func reportEdit(w io.Writer, name string, er editResult) {
	if w == nil {
		return
	}
	switch {
	case er.rolledBack:
		fmt.Fprintf(w, "  ⚠ %s: %d edit(s) rolled back (%s)\n", name, er.edits, er.rollback)
	case er.edits == 0:
		fmt.Fprintf(w, "  %s: no edits (review only)\n", name)
	case er.truncated:
		fmt.Fprintf(w, "  ✎ %s: %d edit(s) applied (run truncated before summary)\n", name, er.edits)
	default:
		fmt.Fprintf(w, "  ✎ %s: %d edit(s) applied\n", name, er.edits)
	}
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

// supportedExts are the file extensions batch mode reviews. .json/.liquid
// cover Shopify theme templates and sections.
var supportedExts = []string{".html", ".htm", ".md", ".txt", ".json", ".liquid"}

func isSupportedExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range supportedExts {
		if ext == e {
			return true
		}
	}
	return false
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
		if isSupportedExt(path) {
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
