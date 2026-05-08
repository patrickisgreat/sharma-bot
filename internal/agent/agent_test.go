package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/tools"
)

// scriptedCompleter returns a pre-baked sequence of Steps. Each call to
// Step pops the next one. Useful for asserting agent loop behavior without
// hitting the API.
type scriptedCompleter struct {
	steps  []ai.Step
	usage  ai.Usage // returned each call
	err    error    // if non-nil, returned on every call
	calls  int
	gotSys []string
	gotHist []ai.History
	gotTools [][]ai.ToolDef
}

func (s *scriptedCompleter) Step(_ context.Context, sys string, hist ai.History, tds []ai.ToolDef) (ai.Step, ai.Usage, error) {
	s.calls++
	s.gotSys = append(s.gotSys, sys)
	s.gotHist = append(s.gotHist, hist)
	s.gotTools = append(s.gotTools, tds)
	if s.err != nil {
		return ai.Step{}, ai.Usage{}, s.err
	}
	if s.calls > len(s.steps) {
		return ai.Step{}, ai.Usage{}, fmt.Errorf("script exhausted (call %d, only %d steps)", s.calls, len(s.steps))
	}
	return s.steps[s.calls-1], s.usage, nil
}

// recordingTool is a Tool whose Run records its inputs and returns canned output.
func recordingTool(name string, output string, runErr error) (tools.Tool, *struct {
	Calls  int
	Inputs []json.RawMessage
}) {
	rec := &struct {
		Calls  int
		Inputs []json.RawMessage
	}{}
	return tools.Tool{
		Name:        name,
		Description: "test tool " + name,
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Run: func(_ context.Context, input json.RawMessage) (string, error) {
			rec.Calls++
			rec.Inputs = append(rec.Inputs, input)
			if runErr != nil {
				return "", runErr
			}
			return output, nil
		},
	}, rec
}

// ─────────────────────────── happy paths ───────────────────────────

func TestRunFinalAnswerImmediately(t *testing.T) {
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{Text: "the answer", StopReason: "end_turn"},
		},
		usage: ai.Usage{Model: "test", InputTokens: 100, OutputTokens: 50},
	}

	res, err := Run(context.Background(), Config{Completer: completer}, "system", "what?")
	if err != nil {
		t.Fatal(err)
	}
	if res.Answer != "the answer" {
		t.Errorf("answer: %q", res.Answer)
	}
	if res.Steps != 1 {
		t.Errorf("steps: %d", res.Steps)
	}
	if completer.calls != 1 {
		t.Errorf("expected 1 completer call, got %d", completer.calls)
	}
	if res.Usage.InputTokens != 100 || res.Usage.OutputTokens != 50 {
		t.Errorf("usage: %+v", res.Usage)
	}
}

func TestRunOneToolCallThenFinalAnswer(t *testing.T) {
	tool, rec := recordingTool("grep", "found 3 docs", nil)
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{
				Text:       "let me check",
				ToolUses:   []ai.ToolUse{{ID: "tu_1", Name: "grep", Input: json.RawMessage(`{"query":"loyalty"}`)}},
				StopReason: "tool_use",
			},
			{Text: "loyalty programs come up in episode X", StopReason: "end_turn"},
		},
	}

	res, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}}, "system", "tell me about loyalty")
	if err != nil {
		t.Fatal(err)
	}
	if res.Answer != "loyalty programs come up in episode X" {
		t.Errorf("answer: %q", res.Answer)
	}
	if res.Steps != 2 {
		t.Errorf("steps: %d", res.Steps)
	}
	if rec.Calls != 1 {
		t.Errorf("tool calls: %d", rec.Calls)
	}
	if string(rec.Inputs[0]) != `{"query":"loyalty"}` {
		t.Errorf("tool input: %q", string(rec.Inputs[0]))
	}
	// Second call should have seen 2 turns of history (assistant + user(tool_results)).
	if got := completer.gotHist[1]; len(got.Turns) != 2 {
		t.Errorf("second call history turns: %d", len(got.Turns))
	}
}

func TestRunMultipleToolCallsInOneStep(t *testing.T) {
	grep, gRec := recordingTool("grep", "grep result", nil)
	read, rRec := recordingTool("read_doc", "doc body", nil)
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{
				ToolUses: []ai.ToolUse{
					{ID: "tu_1", Name: "grep", Input: json.RawMessage(`{"q":"x"}`)},
					{ID: "tu_2", Name: "read_doc", Input: json.RawMessage(`{"id":"y"}`)},
				},
				StopReason: "tool_use",
			},
			{Text: "done", StopReason: "end_turn"},
		},
	}

	_, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{grep, read}}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	if gRec.Calls != 1 || rRec.Calls != 1 {
		t.Errorf("expected each tool called once: grep=%d read=%d", gRec.Calls, rRec.Calls)
	}
	// Second call's history should contain both tool_results in one user turn.
	hist := completer.gotHist[1]
	last := hist.Turns[len(hist.Turns)-1]
	if last.Role != "user" || len(last.ToolResults) != 2 {
		t.Errorf("expected user turn with 2 tool_results, got %+v", last)
	}
	if last.ToolResults[0].ToolUseID != "tu_1" || last.ToolResults[1].ToolUseID != "tu_2" {
		t.Errorf("tool_use_id mismatch: %v", last.ToolResults)
	}
}

func TestRunMultipleSequentialSteps(t *testing.T) {
	grep, _ := recordingTool("grep", "match", nil)
	read, _ := recordingTool("read_doc", "body", nil)
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{ToolUses: []ai.ToolUse{{ID: "1", Name: "grep", Input: json.RawMessage(`{}`)}}, StopReason: "tool_use"},
			{ToolUses: []ai.ToolUse{{ID: "2", Name: "read_doc", Input: json.RawMessage(`{}`)}}, StopReason: "tool_use"},
			{Text: "answer", StopReason: "end_turn"},
		},
	}
	res, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{grep, read}}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	if res.Steps != 3 {
		t.Errorf("steps: %d", res.Steps)
	}
}

// ───────────────────────── error & edge cases ─────────────────────────

func TestRunUnknownToolBecomesToolError(t *testing.T) {
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{ToolUses: []ai.ToolUse{{ID: "tu_1", Name: "imaginary", Input: json.RawMessage(`{}`)}}, StopReason: "tool_use"},
			{Text: "okay", StopReason: "end_turn"},
		},
	}
	_, err := Run(context.Background(), Config{Completer: completer}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	hist := completer.gotHist[1]
	r := hist.Turns[len(hist.Turns)-1].ToolResults[0]
	if !r.IsError {
		t.Errorf("expected IsError")
	}
	if !strings.Contains(r.Content, "imaginary") {
		t.Errorf("error content should mention tool name: %q", r.Content)
	}
}

func TestRunToolErrorBecomesToolResult(t *testing.T) {
	tool, _ := recordingTool("broken", "", errors.New("kaboom"))
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{ToolUses: []ai.ToolUse{{ID: "tu_1", Name: "broken", Input: json.RawMessage(`{}`)}}, StopReason: "tool_use"},
			{Text: "ok", StopReason: "end_turn"},
		},
	}
	_, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	r := completer.gotHist[1].Turns[1].ToolResults[0]
	if !r.IsError || !strings.Contains(r.Content, "kaboom") {
		t.Errorf("expected tool error to surface as IsError result, got %+v", r)
	}
}

func TestRunCompleterErrorBubbles(t *testing.T) {
	completer := &scriptedCompleter{err: errors.New("api blew up")}
	_, err := Run(context.Background(), Config{Completer: completer}, "sys", "q")
	if err == nil || !strings.Contains(err.Error(), "api blew up") {
		t.Errorf("expected API error to bubble, got %v", err)
	}
}

func TestRunMaxStepsLimit(t *testing.T) {
	tool, _ := recordingTool("grep", "result", nil)
	// Builds an infinite tool-loop: every Step asks for grep.
	steps := make([]ai.Step, 5)
	for i := range steps {
		steps[i] = ai.Step{
			ToolUses:   []ai.ToolUse{{ID: fmt.Sprintf("tu_%d", i), Name: "grep", Input: json.RawMessage(`{}`)}},
			StopReason: "tool_use",
		}
	}
	completer := &scriptedCompleter{steps: steps}
	_, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}, MaxSteps: 3}, "sys", "q")
	if err == nil || !strings.Contains(err.Error(), "max steps") {
		t.Errorf("expected max-steps error, got %v", err)
	}
}

func TestRunRequiresCompleter(t *testing.T) {
	_, err := Run(context.Background(), Config{}, "sys", "q")
	if err == nil || !strings.Contains(err.Error(), "Completer") {
		t.Errorf("expected Completer-required error, got %v", err)
	}
}

// ─────────────────────────── usage merging ───────────────────────────

func TestRunAggregatesUsageAcrossSteps(t *testing.T) {
	tool, _ := recordingTool("grep", "x", nil)
	completer := &scriptedCompleter{
		usage: ai.Usage{Model: "m", InputTokens: 100, OutputTokens: 50, CacheReadTokens: 10},
		steps: []ai.Step{
			{ToolUses: []ai.ToolUse{{ID: "1", Name: "grep", Input: json.RawMessage(`{}`)}}, StopReason: "tool_use"},
			{Text: "done", StopReason: "end_turn"},
		},
	}
	res, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	if res.Usage.InputTokens != 200 || res.Usage.OutputTokens != 100 || res.Usage.CacheReadTokens != 20 {
		t.Errorf("aggregate usage: %+v", res.Usage)
	}
	if res.Usage.Model != "m" {
		t.Errorf("model lost: %q", res.Usage.Model)
	}
}

// ─────────────────────────── tool defs ───────────────────────────

func TestRunPassesToolDefsToCompleter(t *testing.T) {
	tool, _ := recordingTool("grep", "x", nil)
	completer := &scriptedCompleter{
		steps: []ai.Step{{Text: "ok", StopReason: "end_turn"}},
	}
	_, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	if len(completer.gotTools[0]) != 1 || completer.gotTools[0][0].Name != "grep" {
		t.Errorf("tool defs not propagated: %+v", completer.gotTools[0])
	}
}

// ───────────────────────────── trace ─────────────────────────────

func TestRunTraceOutput(t *testing.T) {
	tool, _ := recordingTool("grep", "result text", nil)
	var buf bytes.Buffer
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{
				Text:       "let me look",
				ToolUses:   []ai.ToolUse{{ID: "tu_1", Name: "grep", Input: json.RawMessage(`{"q":"x"}`)}},
				StopReason: "tool_use",
			},
			{Text: "answer here", StopReason: "end_turn"},
		},
	}
	_, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}, Trace: &buf}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"[step 1]", "→ grep(", "← grep:", "[step 2] final answer"} {
		if !strings.Contains(got, want) {
			t.Errorf("trace missing %q:\n%s", want, got)
		}
	}
}

func TestRunNoTraceWhenWriterNil(t *testing.T) {
	tool, _ := recordingTool("grep", "x", nil)
	completer := &scriptedCompleter{
		steps: []ai.Step{
			{ToolUses: []ai.ToolUse{{ID: "1", Name: "grep", Input: json.RawMessage(`{}`)}}, StopReason: "tool_use"},
			{Text: "done", StopReason: "end_turn"},
		},
	}
	// Trace is nil; should not panic.
	_, err := Run(context.Background(), Config{Completer: completer, Tools: []tools.Tool{tool}}, "sys", "q")
	if err != nil {
		t.Fatal(err)
	}
}
