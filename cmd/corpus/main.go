package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/ask"
	"sharma-bot/internal/chats"
	"sharma-bot/internal/dig"
	"sharma-bot/internal/discover"
	"sharma-bot/internal/envfile"
	"sharma-bot/internal/parse"
	"sharma-bot/internal/review"
	"sharma-bot/internal/strip"
	"sharma-bot/internal/wrap"
)

const usage = `usage: corpus <command> [flags] [args...]

commands:
  discover         scan raw/{source}/ and populate state.db
  parse            cue-parse raw .txt -> cues/{source}/{external_id}.json
  strip            Haiku-clean cues -> clean_v1/{source}/{external_id}.md
  ask              load full corpus into context, ask Claude (one big API call)
  dig              agent-loop: model uses tools to load only what it needs
  review           review user content (email, page, copy) against the corpus
  fetch-shopify    mirror a Shopify storefront (uses SHOPIFY_URL + SHOPIFY_PASSWORD)

flags:
  --corpus-dir   path to corpus root (default: ./corpus)
  --prompts-dir  path to prompts directory (default: ./prompts)
  --chats-dir    where to save chat transcripts (default: ./chats)
  --limit        max episodes/docs to process or load (0 = no limit)
  --width        wrap output to N columns (0 = no wrap; default: 100)
  --trace        print agent step trace to stderr (dig/review; default: true)
  --batch        directory of files to review (review only)
  --crawl        entry URL to crawl one hop and review each same-host page
  --max-pages    cap on pages reviewed during --crawl (default 20)
  --write        apply edits in place instead of only reporting (review only)
  --force        skip the clean-git-tree guard required by --write
  --out          override chat-save path (single-shot commands only)
  --no-save      skip writing the chat file

ask / dig:
  corpus dig "your question here"
  corpus dig -                    # read question from stdin
  corpus dig --no-save "..."      # don't save the chat file

review:
  corpus review email.md
  corpus review page.html             # HTML auto-extracted to text
  corpus review --batch shopify-mirror/
  corpus review --crawl https://brand.com [--max-pages 30]
  corpus review --batch theme/templates/ --write   # edit copy/structure in place
  corpus review theme/templates/page.faq.json --write

fetch-shopify:
  corpus fetch-shopify                          # uses $SHOPIFY_URL
  corpus fetch-shopify https://other-store...   # override
`

func main() {
	if err := envfile.Load(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: .env load failed:", err)
	}

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	corpusDir := fs.String("corpus-dir", "./corpus", "corpus root directory")
	promptsDir := fs.String("prompts-dir", "./prompts", "prompts directory")
	chatsDir := fs.String("chats-dir", "./chats", "where to save chat transcripts")
	limit := fs.Int("limit", 0, "process at most N items (0 = no limit)")
	width := fs.Int("width", 100, "wrap output to N columns (0 disables)")
	trace := fs.Bool("trace", true, "print agent step trace to stderr (dig/review)")
	batch := fs.String("batch", "", "directory of files to review (review only)")
	crawl := fs.String("crawl", "", "entry URL to crawl one hop and review (review only)")
	maxPages := fs.Int("max-pages", 0, "cap on pages reviewed during --crawl (0 = default 20)")
	write := fs.Bool("write", false, "apply edits in place instead of only reporting (review only)")
	force := fs.Bool("force", false, "skip the clean-git-tree guard required by --write")
	out := fs.String("out", "", "override chat-save path (single-shot commands)")
	noSave := fs.Bool("no-save", false, "skip writing the chat transcript")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	args := parseInterspersed(fs, os.Args[2:])

	var err error
	switch cmd {
	case "discover":
		err = discover.Run(*corpusDir)
	case "parse":
		err = parse.Run(*corpusDir)
	case "strip":
		err = strip.Run(*corpusDir, *promptsDir, *limit)
	case "ask":
		err = runAsk(*corpusDir, *promptsDir, *chatsDir, *out, *noSave, *width, *limit, args)
	case "dig":
		err = runDig(*corpusDir, *promptsDir, *chatsDir, *out, *noSave, *width, *trace, args)
	case "review":
		err = runReview(*corpusDir, *promptsDir, *chatsDir, *out, *noSave, *batch, *crawl, *maxPages, *write, *force, *trace, args)
	case "fetch-shopify":
		err = runFetchShopify(args)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// parseInterspersed parses flags that may appear before, after, or between
// positional args — Go's flag package otherwise stops at the first positional,
// so `review file --write` would silently drop --write. Returns the collected
// positional args in order.
func parseInterspersed(fs *flag.FlagSet, argv []string) []string {
	var positionals []string
	for {
		_ = fs.Parse(argv)
		rest := fs.Args()
		if len(rest) == 0 {
			return positionals
		}
		positionals = append(positionals, rest[0])
		argv = rest[1:]
	}
}

func runAsk(corpusDir, promptsDir, chatsDir, outPath string, noSave bool, width, limit int, args []string) error {
	question, err := readQuestion(args, os.Stdin)
	if err != nil {
		return err
	}
	res, err := ask.Run(corpusDir, promptsDir, question, limit)
	if err != nil {
		return err
	}
	fmt.Println(wrap.Text(res.Answer, width))
	saveChat(chatsDir, outPath, noSave, chats.Chat{
		Command:  "ask",
		Title:    titleFromQuestion(question),
		Question: question,
		Answer:   res.Answer,
		Result:   res,
	})
	return nil
}

func runDig(corpusDir, promptsDir, chatsDir, outPath string, noSave bool, width int, trace bool, args []string) error {
	question, err := readQuestion(args, os.Stdin)
	if err != nil {
		return err
	}
	traceW, traceBuf := traceWriter(trace, !noSave)
	res, err := dig.Run(corpusDir, promptsDir, question, traceW)
	if err != nil {
		return err
	}
	fmt.Println(wrap.Text(res.Answer, width))
	saveChat(chatsDir, outPath, noSave, chats.Chat{
		Command:  "dig",
		Title:    titleFromQuestion(question),
		Question: question,
		Answer:   res.Answer,
		Trace:    bufString(traceBuf),
		Result:   res,
	})
	return nil
}

func runReview(corpusDir, promptsDir, chatsDir, outPath string, noSave bool, batchDir, crawlURL string, maxPages int, write, force, trace bool, args []string) error {
	singleShot := batchDir == "" && crawlURL == ""
	traceW, traceBuf := traceWriter(trace, !noSave && singleShot)

	path := ""
	if len(args) > 0 {
		path = args[0]
	}

	res, err := review.Run(review.Options{
		CorpusDir:  corpusDir,
		PromptsDir: promptsDir,
		Path:       path,
		BatchDir:   batchDir,
		CrawlURL:   crawlURL,
		MaxPages:   maxPages,
		Write:      write,
		Force:      force,
		Stdin:      os.Stdin,
		Trace:      traceW,
	})
	if err != nil {
		return err
	}
	// Batch / crawl modes write per-page outputs; nothing to chat-save.
	if res == nil {
		return nil
	}
	saveChat(chatsDir, outPath, noSave, chats.Chat{
		Command:  "review",
		Title:    titleFromReview(path, batchDir),
		Question: titleFromReview(path, batchDir),
		Answer:   res.Answer,
		Trace:    bufString(traceBuf),
		Result:   res,
	})
	return nil
}

func runFetchShopify(args []string) error {
	url := os.Getenv("SHOPIFY_URL")
	if len(args) > 0 {
		url = args[0]
	}
	if url == "" {
		return fmt.Errorf("no Shopify URL: set SHOPIFY_URL in .env or pass as the first arg")
	}
	script := "./scripts/fetch-shopify.sh"
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("%s not found (run from repo root)", script)
	}
	cmd := exec.Command(script, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// traceWriter returns a writer for the agent trace stream and (optionally)
// a buffer that captures everything for chat saving. Either output may be
// nil depending on the flags:
//
//	showTrace=true,  capture=true  → MultiWriter(stderr, buf)
//	showTrace=true,  capture=false → stderr
//	showTrace=false, capture=true  → buf
//	showTrace=false, capture=false → nil
func traceWriter(showTrace, capture bool) (io.Writer, *bytes.Buffer) {
	var buf *bytes.Buffer
	if capture {
		buf = &bytes.Buffer{}
	}
	switch {
	case showTrace && capture:
		return io.MultiWriter(os.Stderr, buf), buf
	case showTrace:
		return os.Stderr, nil
	case capture:
		return buf, buf
	default:
		return nil, nil
	}
}

func bufString(b *bytes.Buffer) string {
	if b == nil {
		return ""
	}
	return b.String()
}

// saveChat writes the chat transcript and reports the resulting path on
// stderr. Save failures are warnings — they should never abort the command,
// since the user already got their answer on stdout.
func saveChat(chatsDir, outPath string, noSave bool, c chats.Chat) {
	if noSave {
		return
	}
	path, err := chats.Save(chatsDir, outPath, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: chat save failed:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "saved chat:", path)
}

// titleFromQuestion truncates the user question for use as a filename slug.
// 80 chars is the longest we want in a filename; slugify will further trim.
func titleFromQuestion(q string) string {
	q = strings.TrimSpace(q)
	if len(q) > 80 {
		q = q[:80]
	}
	return q
}

// titleFromReview makes a label from the review target. Used as both Title
// (for the slug) and Question (since the actual content is too long to dump
// into the saved file's Question section).
func titleFromReview(path, batchDir string) string {
	if batchDir != "" {
		return "review batch of " + filepath.Base(batchDir)
	}
	if path == "" || path == "-" {
		return "review of stdin"
	}
	return "review of " + filepath.Base(path)
}

// readQuestion takes positional args. If empty or `-`, it reads from r
// until EOF (so multi-line inputs like a pasted PDP work).
func readQuestion(args []string, r io.Reader) (string, error) {
	joined := strings.TrimSpace(strings.Join(args, " "))
	if joined != "" && joined != "-" {
		return joined, nil
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	q := strings.TrimSpace(string(data))
	if q == "" {
		return "", fmt.Errorf("no question provided (pass as args or pipe via stdin)")
	}
	return q, nil
}

// Keep the import alive — referenced from saveChat.
var _ = ai.CallResult{}
