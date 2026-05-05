package discover

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sharma-bot/internal/episode"
	"sharma-bot/internal/state"
)

func Run(corpusDir string) error {
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		return err
	}
	rawDir := filepath.Join(corpusDir, "raw")
	sources, err := os.ReadDir(rawDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", rawDir, err)
	}

	db, err := state.Open(filepath.Join(corpusDir, "state.db"))
	if err != nil {
		return err
	}
	defer db.Close()

	var inserted, skipped, badName int
	for _, src := range sources {
		if !src.IsDir() {
			continue
		}
		source := src.Name()
		dir := filepath.Join(rawDir, source)
		files, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("read %s: %w", dir, err)
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".txt") {
				continue
			}
			meta, err := episode.ParseFilename(f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "skip %s/%s: %v\n", source, f.Name(), err)
				badName++
				continue
			}
			ep := state.Episode{
				ID:         source + "-" + meta.ExternalID,
				Source:     source,
				ExternalID: meta.ExternalID,
				AirDate:    meta.Date,
				TitleSlug:  meta.TitleSlug,
				RawPath:    filepath.Join("raw", source, f.Name()),
				Stage:      state.StageDiscovered,
			}
			if meta.Season > 0 {
				ep.Season = sql.NullInt64{Int64: int64(meta.Season), Valid: true}
			}
			if meta.EpisodeNum > 0 {
				ep.EpisodeNum = sql.NullInt64{Int64: int64(meta.EpisodeNum), Valid: true}
			}
			ok, err := state.InsertOrIgnore(db, ep)
			if err != nil {
				return fmt.Errorf("insert %s: %w", ep.ID, err)
			}
			if ok {
				inserted++
			} else {
				skipped++
			}
		}
	}
	fmt.Printf("discover: %d new, %d already known, %d filename errors\n", inserted, skipped, badName)
	return nil
}
