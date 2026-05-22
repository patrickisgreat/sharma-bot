package review

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runEdit(t *testing.T, e *fileEditor, old, new string) (string, error) {
	t.Helper()
	in, _ := json.Marshal(editInput{OldString: old, NewString: new})
	return e.tool().Run(context.Background(), in)
}

func TestEditToolAppliesUniqueReplacement(t *testing.T) {
	e := newFileEditor("page.json", `{"heading":"Welcome","sub":"Welcome"}`)
	// "Welcome" appears twice — must disambiguate with context.
	if _, err := runEdit(t, e, `"heading":"Welcome"`, `"heading":"Make it yours"`); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if !e.changed() {
		t.Fatal("expected changed()")
	}
	if e.edits != 1 {
		t.Errorf("edits = %d, want 1", e.edits)
	}
	if !strings.Contains(e.content, `"heading":"Make it yours"`) || !strings.Contains(e.content, `"sub":"Welcome"`) {
		t.Errorf("unexpected content: %s", e.content)
	}
}

func TestEditToolRejectsNotFound(t *testing.T) {
	e := newFileEditor("page.json", `{"heading":"Welcome"}`)
	_, err := runEdit(t, e, "nonexistent", "x")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
	if e.changed() {
		t.Error("content should be unchanged after failed edit")
	}
}

func TestEditToolRejectsAmbiguous(t *testing.T) {
	e := newFileEditor("page.json", `{"a":"Welcome","b":"Welcome"}`)
	_, err := runEdit(t, e, "Welcome", "Hi")
	if err == nil || !strings.Contains(err.Error(), "appears 2 times") {
		t.Errorf("expected ambiguity error, got %v", err)
	}
}

func TestEditToolRejectsIdentical(t *testing.T) {
	e := newFileEditor("page.json", `{"a":"x"}`)
	_, err := runEdit(t, e, "x", "x")
	if err == nil || !strings.Contains(err.Error(), "identical") {
		t.Errorf("expected identical error, got %v", err)
	}
}

func TestEditToolSequentialEdits(t *testing.T) {
	e := newFileEditor("page.json", `{"a":"one","b":"two"}`)
	if _, err := runEdit(t, e, `"one"`, `"uno"`); err != nil {
		t.Fatal(err)
	}
	if _, err := runEdit(t, e, `"two"`, `"dos"`); err != nil {
		t.Fatal(err)
	}
	if e.edits != 2 {
		t.Errorf("edits = %d, want 2", e.edits)
	}
	if e.content != `{"a":"uno","b":"dos"}` {
		t.Errorf("content = %s", e.content)
	}
}
