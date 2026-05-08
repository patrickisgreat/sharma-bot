// Package agent runs the tool-using dispatch loop. The loop's job is small:
// while the model returns tool_use blocks, execute the tools, append the
// results back into the conversation, and call the model again. When the
// model returns plain text (no tool uses), that's the final answer.
//
// The loop is intentionally framework-free. Compare to LangGraph or similar:
// most of those abstractions hide what's happening here, which is a while
// loop and a switch statement.
package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/tools"
)

// Config bundles the dependencies for one agent.Run call.
type Config struct {
	// Completer is the tool-aware model client. Required.
	Completer ai.ToolCompleter

	// Tools is the registry of tools the agent may call. The model sees them
	// as a list of {name, description, schema}; we execute them by name when
	// the model produces a tool_use block.
	Tools []tools.Tool

	// MaxSteps caps the number of model turns to prevent runaway loops.
	// Each step is one model call (with possibly multiple tool calls inside).
	// Default 10.
	MaxSteps int

	// Trace, if non-nil, receives a human-readable trace of each step:
	// "[step 1] 2 tool call(s)", "→ grep(...)", "← grep: Found 3 docs...".
	// stderr is the typical destination.
	Trace io.Writer
}

// Result is what Run returns when the model finishes without hitting the
// step limit.
type Result struct {
	Answer  string     // final assistant text
	Steps   int        // number of model calls
	Usage   ai.Usage   // aggregated across steps; Model is set from the last call
	History ai.History // full conversation, useful for debugging
}

// Run executes the agent loop with the given system prompt and question.
// Returns the final answer text, total usage, and an error if anything blew up
// or the loop exceeded MaxSteps.
func Run(ctx context.Context, cfg Config, systemPrompt, question string) (*Result, error) {
	if cfg.Completer == nil {
		return nil, fmt.Errorf("agent: Completer is required")
	}
	maxSteps := cfg.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 10
	}

	toolDefs := make([]ai.ToolDef, 0, len(cfg.Tools))
	for _, t := range cfg.Tools {
		toolDefs = append(toolDefs, ai.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	hist := ai.History{UserPrompt: question}
	var total ai.Usage

	for step := 1; step <= maxSteps; step++ {
		s, usage, err := cfg.Completer.Step(ctx, systemPrompt, hist, toolDefs)
		total = mergeUsage(total, usage)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", step, err)
		}

		hist.Turns = append(hist.Turns, ai.HistoryTurn{
			Role:     "assistant",
			Text:     s.Text,
			ToolUses: s.ToolUses,
		})
		traceStep(cfg.Trace, step, s)

		if len(s.ToolUses) == 0 {
			return &Result{
				Answer:  s.Text,
				Steps:   step,
				Usage:   total,
				History: hist,
			}, nil
		}

		results := make([]ai.ToolResult, 0, len(s.ToolUses))
		for _, u := range s.ToolUses {
			out, isErr := dispatchTool(ctx, cfg.Tools, u)
			results = append(results, ai.ToolResult{
				ToolUseID: u.ID,
				Content:   out,
				IsError:   isErr,
			})
			traceToolResult(cfg.Trace, u, out, isErr)
		}

		hist.Turns = append(hist.Turns, ai.HistoryTurn{
			Role:        "user",
			ToolResults: results,
		})
	}

	return nil, fmt.Errorf("agent exceeded max steps (%d)", maxSteps)
}

// dispatchTool finds a tool by name and runs it. Tool errors and unknown-tool
// situations come back as IsError tool_results so the model can react and
// retry — they don't abort the loop.
func dispatchTool(ctx context.Context, all []tools.Tool, u ai.ToolUse) (string, bool) {
	t, ok := tools.ByName(all, u.Name)
	if !ok {
		return fmt.Sprintf("error: tool %q is not available", u.Name), true
	}
	out, err := t.Run(ctx, u.Input)
	if err != nil {
		return fmt.Sprintf("error: %s", err), true
	}
	return out, false
}

func mergeUsage(a, b ai.Usage) ai.Usage {
	model := a.Model
	if b.Model != "" {
		model = b.Model
	}
	return ai.Usage{
		Model:               model,
		InputTokens:         a.InputTokens + b.InputTokens,
		CacheCreationTokens: a.CacheCreationTokens + b.CacheCreationTokens,
		CacheReadTokens:     a.CacheReadTokens + b.CacheReadTokens,
		OutputTokens:        a.OutputTokens + b.OutputTokens,
	}
}

func traceStep(w io.Writer, step int, s ai.Step) {
	if w == nil {
		return
	}
	if len(s.ToolUses) == 0 {
		fmt.Fprintf(w, "[step %d] final answer (%d chars)\n", step, len(s.Text))
		return
	}
	var preview string
	if t := strings.TrimSpace(s.Text); t != "" {
		preview = ` — "` + truncate(t, 80) + `"`
	}
	fmt.Fprintf(w, "[step %d] %d tool call(s)%s\n", step, len(s.ToolUses), preview)
	for _, u := range s.ToolUses {
		fmt.Fprintf(w, "  → %s(%s)\n", u.Name, truncate(string(u.Input), 160))
	}
}

func traceToolResult(w io.Writer, u ai.ToolUse, output string, isErr bool) {
	if w == nil {
		return
	}
	label := "←"
	if isErr {
		label = "✗"
	}
	flat := strings.ReplaceAll(strings.TrimSpace(output), "\n", " ⏎ ")
	fmt.Fprintf(w, "  %s %s: %s\n", label, u.Name, truncate(flat, 200))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
