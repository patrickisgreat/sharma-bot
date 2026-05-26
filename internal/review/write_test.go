package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sharma-bot/internal/ai"
)

// editStep builds an assistant step that calls edit_file once.
func editStep(old, new string) ai.Step {
	in, _ := json.Marshal(editInput{OldString: old, NewString: new})
	return ai.Step{
		StopReason: "tool_use",
		ToolUses:   []ai.ToolUse{{ID: "e1", Name: "edit_file", Input: in}},
	}
}

func writeCfg(completer ai.ToolCompleter) runConfig {
	return runConfig{
		role:      "ROLE",
		write:     true,
		completer: completer,
		timeout:   2 * time.Second,
	}
}

func TestEditOneAppliesAndWrites(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "page.json")
	if err := os.WriteFile(target, []byte(`{"heading":"Welcome to our store"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	completer := &scriptedCompleter{steps: []ai.Step{
		editStep("Welcome to our store", "Make your String Ring"),
		{Text: "## What's working\n- sharpened the hero", StopReason: "end_turn"},
	}}

	er, res, err := editOne(writeCfg(completer), target)
	if err != nil {
		t.Fatal(err)
	}
	if er.edits != 1 || er.truncated {
		t.Errorf("editResult = %+v, want 1 edit, not truncated", er)
	}
	got, _ := os.ReadFile(target)
	if string(got) != `{"heading":"Make your String Ring"}` {
		t.Errorf("file not edited: %s", got)
	}
	if res.Answer == "" {
		t.Error("expected review answer")
	}
}

func TestEditOneRejectsBadEditKeepsGoodOnes(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "page.json")
	if err := os.WriteFile(target, []byte(`{"a":"one","b":"two"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// A good edit, then one that breaks JSON (dropped closing brace), then end.
	// Per-edit validation rejects the bad one; the good one still lands and the
	// file stays valid — no all-or-nothing rollback.
	completer := &scriptedCompleter{steps: []ai.Step{
		editStep(`"one"`, `"uno"`),
		editStep(`"two"}`, `"two`),
		{Text: "review", StopReason: "end_turn"},
	}}

	er, _, err := editOne(writeCfg(completer), target)
	if err != nil {
		t.Fatal(err)
	}
	if er.edits != 1 {
		t.Errorf("expected 1 edit (bad one rejected), got %+v", er)
	}
	got, _ := os.ReadFile(target)
	if string(got) != `{"a":"uno","b":"two"}` {
		t.Errorf("good edit lost or file corrupted: %s", got)
	}
	if err := validJSON(got); err != nil {
		t.Errorf("written file should be valid JSON: %v", err)
	}
}

func TestEditOneSalvagesEditsOnTruncation(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "page.json")
	if err := os.WriteFile(target, []byte(`{"heading":"Welcome"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// One edit step and no final text step: the agent loop runs out of script
	// and errors, simulating a timeout/max-steps after a valid edit landed.
	completer := &scriptedCompleter{steps: []ai.Step{
		editStep(`"Welcome"`, `"Make your String Ring"`),
	}}

	er, _, err := editOne(writeCfg(completer), target)
	if err != nil {
		t.Fatalf("salvaged edits should not surface the loop error: %v", err)
	}
	if er.edits != 1 || !er.truncated {
		t.Errorf("editResult = %+v, want 1 edit + truncated", er)
	}
	got, _ := os.ReadFile(target)
	if string(got) != `{"heading":"Make your String Ring"}` {
		t.Errorf("salvaged edit not written: %s", got)
	}
}

func TestEditOneNoEditsLeavesFileAlone(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "page.json")
	original := `{"heading":"Already great"}`
	if err := os.WriteFile(target, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	completer := &scriptedCompleter{steps: []ai.Step{
		{Text: "## What's working\n- nothing to change", StopReason: "end_turn"},
	}}

	er, _, err := editOne(writeCfg(completer), target)
	if err != nil {
		t.Fatal(err)
	}
	if er.edits != 0 {
		t.Errorf("editResult = %+v, want 0 edits", er)
	}
	got, _ := os.ReadFile(target)
	if string(got) != original {
		t.Errorf("file changed unexpectedly: %s", got)
	}
}

func TestRunWriteRejectsStdin(t *testing.T) {
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")
	if err := os.WriteFile(filepath.Join(promptsDir, "edit.md"), []byte("EDIT"), 0o644); err != nil {
		t.Fatal(err)
	}
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "x", StopReason: "end_turn"}}}
	_, err := RunWith(Options{PromptsDir: promptsDir, Write: true, Force: true}, completer, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "stdin") {
		t.Errorf("expected stdin rejection, got %v", err)
	}
}

func TestRunWriteRejectsCrawl(t *testing.T) {
	promptsDir := t.TempDir()
	writePrompt(t, promptsDir, "ROLE")
	if err := os.WriteFile(filepath.Join(promptsDir, "edit.md"), []byte("EDIT"), 0o644); err != nil {
		t.Fatal(err)
	}
	completer := &scriptedCompleter{steps: []ai.Step{{Text: "x", StopReason: "end_turn"}}}
	_, err := RunWith(Options{PromptsDir: promptsDir, CrawlURL: "https://x.com", Write: true, Force: true}, completer, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "--write applies to local files") {
		t.Errorf("expected crawl rejection, got %v", err)
	}
}
