package strip

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sharma-bot/internal/ai"
	"sharma-bot/internal/state"
)

// fakeCompleter records the (system, user) pair it was called with and
// returns a canned response (or error).
type fakeCompleter struct {
	resp     string
	err      error
	calls    int
	gotSys   string
	gotUser  string
}

func (f *fakeCompleter) Complete(_ context.Context, system, user string) (string, ai.Usage, error) {
	f.calls++
	f.gotSys = system
	f.gotUser = user
	if f.err != nil {
		return "", ai.Usage{}, f.err
	}
	return f.resp, ai.Usage{}, nil
}

var _ ai.Completer = (*fakeCompleter)(nil)

// fixture sets up a tempdir corpus with one parsed episode ready for strip.
func fixture(t *testing.T, episodeID string) (corpusDir, promptsDir string, db *sql.DB) {
	t.Helper()
	corpusDir = t.TempDir()
	promptsDir = t.TempDir()

	if err := os.WriteFile(filepath.Join(promptsDir, "strip.md"), []byte("clean this up"), 0o644); err != nil {
		t.Fatal(err)
	}

	cuesDir := filepath.Join(corpusDir, "cues", "limited-supply")
	if err := os.MkdirAll(cuesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cuesJSON := `{"cues":[
		{"start":0.0,"end":2.5,"text":"Hello there."},
		{"start":2.5,"end":5.0,"text":"This is a test."}
	]}`
	if err := os.WriteFile(filepath.Join(cuesDir, episodeID+".json"), []byte(cuesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	var err error
	db, err = state.Open(filepath.Join(corpusDir, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := state.InsertOrIgnore(db, state.Episode{
		ID:         "limited-supply-" + episodeID,
		Source:     "limited-supply",
		ExternalID: episodeID,
		AirDate:    "2022-07-20",
		TitleSlug:  "Title",
		RawPath:    "raw/limited-supply/" + episodeID + ".txt",
		Stage:      state.StageParsed,
	}); err != nil {
		t.Fatal(err)
	}
	return corpusDir, promptsDir, db
}

func currentStage(t *testing.T, db *sql.DB, id string) state.Stage {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT stage FROM episodes WHERE id = ?`, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return state.Stage(s)
}

func TestRunWithSuccessAdvancesStage(t *testing.T) {
	corpusDir, promptsDir, db := fixture(t, "abc123")
	fc := &fakeCompleter{resp: "Hello there. This is a test."}

	if err := RunWith(corpusDir, promptsDir, 0, fc, time.Second); err != nil {
		t.Fatal(err)
	}

	if fc.calls != 1 {
		t.Fatalf("expected 1 completer call, got %d", fc.calls)
	}
	if fc.gotSys != "clean this up" {
		t.Errorf("system prompt: %q", fc.gotSys)
	}
	if !strings.Contains(fc.gotUser, "Hello there.") || !strings.Contains(fc.gotUser, "This is a test.") {
		t.Errorf("user text missing cue contents: %q", fc.gotUser)
	}

	out, err := os.ReadFile(filepath.Join(corpusDir, "clean_v1", "limited-supply", "abc123.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "Hello there. This is a test." {
		t.Errorf("clean_v1 contents: %q", string(out))
	}
	if got := currentStage(t, db, "limited-supply-abc123"); got != state.StageStripped {
		t.Errorf("stage: %q", got)
	}
}

func TestRunWithErrorMarksFailed(t *testing.T) {
	corpusDir, promptsDir, db := fixture(t, "abc123")
	fc := &fakeCompleter{err: errors.New("api blew up")}

	if err := RunWith(corpusDir, promptsDir, 0, fc, time.Second); err != nil {
		t.Fatal(err)
	}

	if got := currentStage(t, db, "limited-supply-abc123"); got != state.StageFailed {
		t.Errorf("stage: %q (expected failed)", got)
	}
	// No output file should exist on failure.
	if _, err := os.Stat(filepath.Join(corpusDir, "clean_v1", "limited-supply", "abc123.md")); !os.IsNotExist(err) {
		t.Errorf("expected no clean_v1 file on failure, got err=%v", err)
	}
}

func TestRunWithEmptyResponseMarksFailed(t *testing.T) {
	corpusDir, promptsDir, db := fixture(t, "abc123")
	fc := &fakeCompleter{resp: "   \n  "}

	if err := RunWith(corpusDir, promptsDir, 0, fc, time.Second); err != nil {
		t.Fatal(err)
	}
	if got := currentStage(t, db, "limited-supply-abc123"); got != state.StageFailed {
		t.Errorf("stage: %q (expected failed for empty response)", got)
	}
}

func TestRunWithLimitHonored(t *testing.T) {
	corpusDir, promptsDir, db := fixture(t, "first")
	// Add a second parsed episode and matching cues file.
	cuesJSON := `{"cues":[{"start":0,"end":1,"text":"Two."}]}`
	if err := os.WriteFile(filepath.Join(corpusDir, "cues", "limited-supply", "second.json"), []byte(cuesJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := state.InsertOrIgnore(db, state.Episode{
		ID:         "limited-supply-second",
		Source:     "limited-supply",
		ExternalID: "second",
		RawPath:    "raw/limited-supply/second.txt",
		Stage:      state.StageParsed,
	}); err != nil {
		t.Fatal(err)
	}

	fc := &fakeCompleter{resp: "ok"}
	if err := RunWith(corpusDir, promptsDir, 1, fc, time.Second); err != nil {
		t.Fatal(err)
	}
	if fc.calls != 1 {
		t.Errorf("expected 1 call due to limit, got %d", fc.calls)
	}
}

func TestRunWithSkipsAlreadyStripped(t *testing.T) {
	// Episodes not at StageParsed should be ignored (idempotent re-runs).
	corpusDir, promptsDir, _ := fixture(t, "abc123")
	// Mark it stripped; RunWith should now find nothing pending.
	db, err := state.Open(filepath.Join(corpusDir, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := state.SetStage(db, "limited-supply-abc123", state.StageStripped, ""); err != nil {
		t.Fatal(err)
	}

	fc := &fakeCompleter{resp: "ok"}
	if err := RunWith(corpusDir, promptsDir, 0, fc, time.Second); err != nil {
		t.Fatal(err)
	}
	if fc.calls != 0 {
		t.Errorf("expected 0 calls for already-stripped episode, got %d", fc.calls)
	}
}
