package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sharma-bot/internal/ask"
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
  ask         load corpus (clean_v1 + raw/docs + raw/articles), ask Claude

flags:
  --corpus-dir   path to corpus root (default: ./corpus)
  --prompts-dir  path to prompts directory (default: ./prompts)
  --limit        max episodes/docs to process or load (0 = no limit)
  --width        wrap ask output to N columns (0 = no wrap; default: 100)

ask:
  corpus ask "your question here"
  corpus ask -                    # read question from stdin
  corpus ask --limit 20 "..."     # cap how many docs get loaded
  corpus ask --width 0 "..."      # raw output, no terminal wrap
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
	width := fs.Int("width", 100, "wrap ask output to N columns (0 disables)")
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
