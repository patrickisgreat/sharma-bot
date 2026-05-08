package strip

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/parse"
	"sharma-bot/internal/state"
)

const (
	model     = anthropic.ModelClaudeHaiku4_5
	maxTokens = int64(32000)
	timeout   = 10 * time.Minute
)

// Run is the entry point used by main; it constructs a real Completer.
func Run(corpusDir, promptsDir string, limit int) error {
	return RunWith(corpusDir, promptsDir, limit, ai.NewCompleter(model, maxTokens), timeout)
}

// RunWith is the testable form: it accepts an injected Completer and per-call timeout.
func RunWith(corpusDir, promptsDir string, limit int, completer ai.Completer, perCallTimeout time.Duration) error {
	systemPrompt, err := os.ReadFile(filepath.Join(promptsDir, "strip.md"))
	if err != nil {
		return fmt.Errorf("read prompt: %w", err)
	}

	db, err := state.Open(filepath.Join(corpusDir, "state.db"))
	if err != nil {
		return err
	}
	defer db.Close()

	pending, err := state.Pending(db, state.StageParsed)
	if err != nil {
		return err
	}
	if limit > 0 && len(pending) > limit {
		pending = pending[:limit]
	}

	var written []string
	var ok, fail int
	for _, ep := range pending {
		cuesPath := filepath.Join(corpusDir, "cues", ep.Source, ep.ExternalID+".json")
		outPath := filepath.Join(corpusDir, "clean_v1", ep.Source, ep.ExternalID+".md")
		fmt.Printf("strip %s ... ", ep.ID)
		if err := stripOne(completer, perCallTimeout, string(systemPrompt), cuesPath, outPath); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			_ = state.SetStage(db, ep.ID, state.StageFailed, err.Error())
			fail++
			continue
		}
		if err := state.SetStage(db, ep.ID, state.StageStripped, ""); err != nil {
			return err
		}
		fmt.Println("ok")
		written = append(written, outPath)
		ok++
	}

	fmt.Printf("\nstrip: %d ok, %d failed (of %d pending)\n", ok, fail, len(pending))
	if len(written) > 0 {
		fmt.Println("\noutputs:")
		for _, p := range written {
			fmt.Println("  " + p)
		}
	}
	return nil
}

func stripOne(completer ai.Completer, perCallTimeout time.Duration, systemPrompt, in, out string) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	var cf parse.CuesFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return err
	}

	var sb strings.Builder
	for _, c := range cf.Cues {
		sb.WriteString(c.Text)
		sb.WriteByte('\n')
	}
	transcript := sb.String()

	ctx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()

	cleaned, _, err := completer.Complete(ctx, systemPrompt, transcript)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cleaned) == "" {
		return fmt.Errorf("empty cleaned output")
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, []byte(cleaned), 0o644)
}
