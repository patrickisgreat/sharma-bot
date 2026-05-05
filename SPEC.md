# Knowledge Corpus Pipeline

## Goal
Transform raw podcast transcripts (timestamped .txt files) into a clean,
front-matter-annotated markdown corpus suitable for use as Claude context.

## Input
Files in `corpus/raw/{source}/`, e.g. `corpus/raw/limited-supply/`.

Filename format:
`YYYYMMDD_S{n}_E{n}__{title_slug}-ep_{external_id}.txt`

Example:
`20220720_S1_E1__Nik_Sharma_and_Moiz_Ali__Why_Native_Rejected_Investors-ep_9lmar28vdkl4r2nw.txt`

File contents are line-per-cue, format:
`[HH:MM:SS.mmm --> HH:MM:SS.mmm] text of the cue`

## Output structure

```
corpus/
  state.db                          # SQLite, tracks per-episode pipeline stage
  raw/{source}/*.txt                # untouched originals
  cues/{source}/{id}.json           # parsed cues, seconds as floats
  clean_v1/{source}/{id}.md         # deterministic cleanup (boilerplate stripped)
  clean_v2/{source}/{id}.md         # AI-cleaned prose
  final/{source}/{id}.md            # clean_v2 + YAML front-matter
  patterns/{source}.json            # boilerplate regex list per source
```

## Pipeline stages
Each stage is a separate command. State machine in SQLite advances
on success. Idempotent. Restartable.

1. `corpus discover` — scan `raw/`, populate `episodes` table
2. `corpus parse`    — raw .txt → cues.json
3. `corpus strip`    — cues.json + patterns.json → clean_v1.md
4. `corpus clean`    — clean_v1.md → clean_v2.md (Haiku)
5. `corpus enrich`   — clean_v2.md + episode metadata → final.md (Sonnet)
6. `corpus validate` — sanity-check final.md (Haiku)

## State table

```sql
CREATE TABLE episodes (
  id TEXT PRIMARY KEY,         -- {source}-{external_id}
  source TEXT NOT NULL,
  external_id TEXT NOT NULL,   -- from filename, e.g. 9lmar28vdkl4r2nw
  air_date TEXT,               -- YYYY-MM-DD from filename
  season INTEGER,
  episode_num INTEGER,
  title_slug TEXT,
  raw_path TEXT NOT NULL,
  stage TEXT NOT NULL,         -- discovered|parsed|stripped|cleaned|enriched|validated|failed
  error TEXT,
  updated_at INTEGER
);
```

## Hard rules
- Never modify files in `raw/`. Source of truth, immutable.
- Each stage reads the previous stage's output, writes its own. No piping in memory.
- Anthropic API calls go through one shared client with prompt caching enabled.
- All AI prompts live in `prompts/{stage}.md`, not embedded in Go code.
- No vector DB, no embeddings, no chunking. Files + front-matter only.

## Out of scope (for v1)
- Web UI
- Incremental updates (just rerun the whole thing)
- Multiple sources (Limited Supply only first)
- RSS fetch / new episode detection