package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sharma-bot/internal/ask"
	"sharma-bot/internal/dig"
	"sharma-bot/internal/discover"
	"sharma-bot/internal/envfile"
	"sharma-bot/internal/parse"
	"sharma-bot/internal/strip"
	"sharma-bot/internal/wrap"
)

const usage = `usage: corpus <command> [flags] [args...]

commands:
  discover    scan raw/{source}/ and populate state.db
  parse       cue-parse raw .txt -> cues/{source}/{external_id}.json
  strip       Haiku-clean cues -> clean_v1/{source}/{external_id}.md
  ask         load full corpus into context, ask Claude (one big API call)
  dig         agent-loop: model uses tools to load only what it needs (cheap)

flags:
  --corpus-dir   path to corpus root (default: ./corpus)
  --prompts-dir  path to prompts directory (default: ./prompts)
  --limit        max episodes/docs to process or load (0 = no limit)
  --width        wrap output to N columns (0 = no wrap; default: 100)
  --trace        write agent step trace to stderr (dig only; default: true)

ask:
  corpus ask "your question here"
  corpus ask -                    # read question from stdin
  corpus ask --limit 20 "..."     # cap how many docs get loaded
  corpus ask --width 0 "..."      # raw output, no terminal wrap

dig:
  corpus dig "your question here"
  corpus dig -                    # read question from stdin
  cat my-pdp.html | corpus dig -  # pipe content in
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
	limit := fs.Int("limit", 0, "process at most N items (0 = no limit)")
	width := fs.Int("width", 100, "wrap output to N columns (0 disables)")
	trace := fs.Bool("trace", true, "print agent step trace to stderr (dig)")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	_ = fs.Parse(os.Args[2:])

	var err error
	switch cmd {
	case "discover":
		err = discover.Run(*corpusDir)
	case "parse":
		err = parse.Run(*corpusDir)
	case "strip":
		err = strip.Run(*corpusDir, *promptsDir, *limit)
	case "ask":
		question, qerr := readQuestion(fs.Args(), os.Stdin)
		if qerr != nil {
			err = qerr
			break
		}
		var answer string
		answer, err = ask.Run(*corpusDir, *promptsDir, question, *limit)
		if err == nil {
			fmt.Println(wrap.Text(answer, *width))
		}
	case "dig":
		question, qerr := readQuestion(fs.Args(), os.Stdin)
		if qerr != nil {
			err = qerr
			break
		}
		var traceW io.Writer
		if *trace {
			traceW = os.Stderr
		}
		var answer string
		answer, err = dig.Run(*corpusDir, *promptsDir, question, traceW)
		if err == nil {
			fmt.Println(wrap.Text(answer, *width))
		}
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

// readQuestion takes positional args after `corpus ask`. If empty or `-`, it
// reads from r until EOF (so multi-line inputs like a pasted PDP work).
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
