package state

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mkEpisode(id string, stage Stage) Episode {
	return Episode{
		ID:         id,
		Source:     "limited-supply",
		ExternalID: id,
		AirDate:    "2022-07-20",
		Season:     sql.NullInt64{Int64: 1, Valid: true},
		EpisodeNum: sql.NullInt64{Int64: 1, Valid: true},
		TitleSlug:  "Title",
		RawPath:    "raw/limited-supply/" + id + ".txt",
		Stage:      stage,
	}
}

func TestOpenCreatesSchema(t *testing.T) {
	db := openTempDB(t)
	// Schema creation is idempotent: opening twice on the same path must work.
	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='episodes'`)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("episodes table missing: %v", err)
	}
}

func TestInsertOrIgnoreIdempotent(t *testing.T) {
	db := openTempDB(t)
	ep := mkEpisode("abc", StageDiscovered)

	inserted, err := InsertOrIgnore(db, ep)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("first insert should report inserted=true")
	}

	inserted2, err := InsertOrIgnore(db, ep)
	if err != nil {
		t.Fatal(err)
	}
	if inserted2 {
		t.Fatal("second insert should be ignored")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM episodes`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestPendingFiltersByStage(t *testing.T) {
	db := openTempDB(t)
	if _, err := InsertOrIgnore(db, mkEpisode("a", StageDiscovered)); err != nil {
		t.Fatal(err)
	}
	if _, err := InsertOrIgnore(db, mkEpisode("b", StageDiscovered)); err != nil {
		t.Fatal(err)
	}
	if _, err := InsertOrIgnore(db, mkEpisode("c", StageParsed)); err != nil {
		t.Fatal(err)
	}

	discovered, err := Pending(db, StageDiscovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovered) != 2 {
		t.Fatalf("expected 2 discovered, got %d", len(discovered))
	}
	// Order is by id, so a before b.
	if discovered[0].ID != "a" || discovered[1].ID != "b" {
		t.Errorf("order: %v", []string{discovered[0].ID, discovered[1].ID})
	}

	parsed, err := Pending(db, StageParsed)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 1 || parsed[0].ID != "c" {
		t.Errorf("parsed: %+v", parsed)
	}

	// And a stage with no rows returns empty.
	stripped, err := Pending(db, StageStripped)
	if err != nil {
		t.Fatal(err)
	}
	if len(stripped) != 0 {
		t.Errorf("expected 0 stripped, got %d", len(stripped))
	}
}

func TestSetStageAdvancesAndClearsError(t *testing.T) {
	db := openTempDB(t)
	if _, err := InsertOrIgnore(db, mkEpisode("a", StageDiscovered)); err != nil {
		t.Fatal(err)
	}

	if err := SetStage(db, "a", StageFailed, "boom"); err != nil {
		t.Fatal(err)
	}
	stage, errMsg := readStageAndError(t, db, "a")
	if stage != string(StageFailed) || errMsg != "boom" {
		t.Errorf("after fail: stage=%q err=%q", stage, errMsg)
	}

	// Re-running advances and clears the error.
	if err := SetStage(db, "a", StageParsed, ""); err != nil {
		t.Fatal(err)
	}
	stage, errMsg = readStageAndError(t, db, "a")
	if stage != string(StageParsed) || errMsg != "" {
		t.Errorf("after parsed: stage=%q err=%q", stage, errMsg)
	}
}

func readStageAndError(t *testing.T, db *sql.DB, id string) (string, string) {
	t.Helper()
	var stage string
	var errMsg sql.NullString
	row := db.QueryRow(`SELECT stage, error FROM episodes WHERE id = ?`, id)
	if err := row.Scan(&stage, &errMsg); err != nil {
		t.Fatal(err)
	}
	return stage, errMsg.String
}

func TestAllBySourceKeyedByExternalID(t *testing.T) {
	db := openTempDB(t)
	if _, err := InsertOrIgnore(db, mkEpisode("alpha", StageParsed)); err != nil {
		t.Fatal(err)
	}
	if _, err := InsertOrIgnore(db, mkEpisode("beta", StageStripped)); err != nil {
		t.Fatal(err)
	}
	// Episode from a different source should not appear.
	other := mkEpisode("gamma", StageDiscovered)
	other.Source = "other-show"
	other.ID = "other-show-gamma"
	if _, err := InsertOrIgnore(db, other); err != nil {
		t.Fatal(err)
	}

	got, err := AllBySource(db, "limited-supply")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(got))
	}
	if _, ok := got["alpha"]; !ok {
		t.Errorf("missing alpha")
	}
	if _, ok := got["beta"]; !ok {
		t.Errorf("missing beta")
	}
	if _, ok := got["gamma"]; ok {
		t.Errorf("gamma from other source should be excluded")
	}
	if got["alpha"].TitleSlug != "Title" {
		t.Errorf("title_slug not loaded: %q", got["alpha"].TitleSlug)
	}
}

func TestAllBySourceEmpty(t *testing.T) {
	db := openTempDB(t)
	got, err := AllBySource(db, "anything")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}
