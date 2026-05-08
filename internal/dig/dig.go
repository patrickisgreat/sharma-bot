// Package dig answers a question against the corpus using the agent loop.
// Unlike ask, which stuffs the entire corpus into the system prompt, dig
// gives the model glob/grep/read_doc tools and lets it pull only the docs
// it actually needs. Per-call cost drops from dollars to pennies.
package dig

import (
	"context"
	"fmt"
	"io"
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
	timeout   = 5 * time.Minute
)

// Run is the entry used by main: builds default tools and a real ToolCompleter,
// loads prompts/dig.md, and runs the agent loop. Returns the final answer text.
//
// trace receives the per-step trace lines from the agent (tool calls and
// results); pass nil to suppress.
func Run(corpusDir, promptsDir, question string, trace io.Writer) (string, error) {
	completer := ai.NewToolCompleter(model, maxTokens)
	return RunWith(corpusDir, promptsDir, question, completer, tools.NewCorpusTools(corpusDir), trace, timeout)
}

// RunWith is the testable form: accepts an injected ToolCompleter, an explicit
// tool list, and a per-call timeout. Tests fake the completer and tools.
func RunWith(corpusDir, promptsDir, question string, completer ai.ToolCompleter, ts []tools.Tool, trace io.Writer, perCallTimeout time.Duration) (string, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("question is empty")
	}

	role, err := os.ReadFile(filepath.Join(promptsDir, "dig.md"))
	if err != nil {
		return "", fmt.Errorf("read role prompt: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()

	start := time.Now()
	res, err := agent.Run(ctx, agent.Config{
		Completer: completer,
		Tools:     ts,
		MaxSteps:  maxSteps,
		Trace:     trace,
	}, string(role), question)
	elapsed := time.Since(start)

	if err != nil {
		return "", err
	}
	ai.PrintTelemetry(trace, res.Usage, elapsed, fmt.Sprintf("%d step(s)", res.Steps))
	return res.Answer, nil
}
