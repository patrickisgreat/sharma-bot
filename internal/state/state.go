package state

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Stage string

const (
	StageDiscovered Stage = "discovered"
	StageParsed     Stage = "parsed"
	StageStripped   Stage = "stripped"
	StageCleaned    Stage = "cleaned"
	StageEnriched   Stage = "enriched"
	StageValidated  Stage = "validated"
	StageFailed     Stage = "failed"
)

const schema = `
CREATE TABLE IF NOT EXISTS episodes (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  external_id TEXT NOT NULL,
  air_date TEXT,
  season INTEGER,
  episode_num INTEGER,
  title_slug TEXT,
  raw_path TEXT NOT NULL,
  stage TEXT NOT NULL,
  error TEXT,
  updated_at INTEGER
);
`

type Episode struct {
	ID         string
	Source     string
	ExternalID string
	AirDate    string
	Season     sql.NullInt64
	EpisodeNum sql.NullInt64
	TitleSlug  string
	RawPath    string
	Stage      Stage
	Error      string
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func InsertOrIgnore(db *sql.DB, ep Episode) (bool, error) {
	res, err := db.Exec(`INSERT OR IGNORE INTO episodes
		(id, source, external_id, air_date, season, episode_num, title_slug, raw_path, stage, error, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ep.ID, ep.Source, ep.ExternalID, nullableStr(ep.AirDate), ep.Season, ep.EpisodeNum,
		ep.TitleSlug, ep.RawPath, string(ep.Stage), nullableStr(ep.Error), time.Now().Unix(),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func Pending(db *sql.DB, stage Stage) ([]Episode, error) {
	rows, err := db.Query(`SELECT id, source, external_id, COALESCE(air_date, ''),
		season, episode_num, title_slug, raw_path, stage, COALESCE(error, '')
		FROM episodes WHERE stage = ? ORDER BY id`, string(stage))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Episode
	for rows.Next() {
		var ep Episode
		var stageStr string
		if err := rows.Scan(&ep.ID, &ep.Source, &ep.ExternalID, &ep.AirDate,
			&ep.Season, &ep.EpisodeNum, &ep.TitleSlug, &ep.RawPath, &stageStr, &ep.Error); err != nil {
			return nil, err
		}
		ep.Stage = Stage(stageStr)
		out = append(out, ep)
	}
	return out, rows.Err()
}

func SetStage(db *sql.DB, id string, stage Stage, errMsg string) error {
	_, err := db.Exec(`UPDATE episodes SET stage = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(stage), nullableStr(errMsg), time.Now().Unix(), id)
	return err
}

// AllBySource returns every episode for a given source, keyed by external_id.
// Used by the ask command to enrich <doc> tags with title/season/episode/date.
func AllBySource(db *sql.DB, source string) (map[string]Episode, error) {
	rows, err := db.Query(`SELECT id, source, external_id, COALESCE(air_date, ''),
		season, episode_num, title_slug, raw_path, stage, COALESCE(error, '')
		FROM episodes WHERE source = ?`, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]Episode)
	for rows.Next() {
		var ep Episode
		var stageStr string
		if err := rows.Scan(&ep.ID, &ep.Source, &ep.ExternalID, &ep.AirDate,
			&ep.Season, &ep.EpisodeNum, &ep.TitleSlug, &ep.RawPath, &stageStr, &ep.Error); err != nil {
			return nil, err
		}
		ep.Stage = Stage(stageStr)
		out[ep.ExternalID] = ep
	}
	return out, rows.Err()
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
