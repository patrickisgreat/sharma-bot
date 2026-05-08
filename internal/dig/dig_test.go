package dig

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/tools"
)

type scriptedCompleter struct {
	steps []ai.Step
	calls int
}

func (s *scriptedCompleter) Step(_ context.Context, _ string, _ ai.History, _ []ai.ToolDef) (ai.Step, ai.Usage, error) {
	if s.calls >= len(s.steps) {
		return ai.Step{}, ai.Usage{}, errors.New("script exhausted")
	}
	out := s.steps[s.calls]
	s.calls++
	return out, ai.Usage{Model: "test", InputTokens: 100, OutputTokens: 50}, nil
}

func writePrompt(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dig.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDigRunWithFinalAnswer(t *testing.T) {
	corpusDir := t.TempDir()
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")

	completer := &scriptedCompleter{
		steps: []ai.Step{{Text: "the answer", StopReason: "end_turn"}},
	}
	got, err := RunWith(corpusDir, promptsDir, "what?", completer, nil, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answer != "the answer" {
		t.Errorf("answer: %q", got.Answer)
	}
	if got.Steps != 1 {
		t.Errorf("steps: %d", got.Steps)
	}
}

func TestDigRunWithToolRoundTrip(t *testing.T) {
	corpusDir := t.TempDir()
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")

	tool := tools.Tool{
		Name:        "echo",
		Description: "echoes input back",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Run: func(_ context.Context, in json.RawMessage) (string, error) {
			return "echoed: " + string(in), nil
		},
	}
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{
				ToolUses:   []ai.ToolUse{{ID: "tu_1", Name: "echo", Input: json.RawMessage(`{"x":1}`)}},
				StopReason: "tool_use",
			},
			{Text: "done", StopReason: "end_turn"},
		},
	}
	got, err := RunWith(corpusDir, promptsDir, "q", completer, []tools.Tool{tool}, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answer != "done" {
		t.Errorf("answer: %q", got.Answer)
	}
	if got.Steps != 2 {
		t.Errorf("steps: %d", got.Steps)
	}
}

func TestDigRejectsEmptyQuestion(t *testing.T) {
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "x", StopReason: "end_turn"}}}
	_, err := RunWith("", promptsDir, "  ", completer, nil, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty-question error, got %v", err)
	}
}

func TestDigRejectsMissingPrompt(t *testing.T) {
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "x", StopReason: "end_turn"}}}
	_, err := RunWith("", t.TempDir(), "q", completer, nil, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "role prompt") {
		t.Errorf("expected role-prompt error, got %v", err)
	}
}
