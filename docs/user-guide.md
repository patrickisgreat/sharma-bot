# User Guide

CLI workflows, common commands, and the kinds of things that bite you the first time.

## Setup

You need:
- Go 1.23+
- An Anthropic API key (https://console.anthropic.com)
- macOS or Linux

```bash
cp .env.example .env
# edit .env, paste your ANTHROPIC_API_KEY
go build ./cmd/corpus    # optional; you can also use `go run`
```

The `.env` file is loaded automatically on every command. If you have `ANTHROPIC_API_KEY` exported in your shell already, that wins over `.env` (so a stale `.env` value can't accidentally override an explicit shell setting).

## The command surface

```
corpus discover    scan raw/<source>/ and populate state.db
corpus parse       cue-parse raw .txt -> cues/<source>/<id>.json
corpus strip       Haiku-clean cues -> clean_v1/<source>/<id>.md
corpus ask         load corpus, ask Claude a question
```

Common flags:
- `--corpus-dir`  path to corpus root (default: `./corpus`)
- `--prompts-dir` path to prompts directory (default: `./prompts`)
- `--limit`       process at most N items (0 = no limit)
- `--width`       wrap ask output to N columns (0 disables; default: 100)

## Adding new content

### A new podcast episode

1. Drop the timestamped `.txt` into `corpus/raw/limited-supply/` (or any source folder under `raw/`). Filename format: `YYYYMMDD_S{n}_E{n}__<title_slug>-ep_<external_id>.txt`. See [SPEC.md](../SPEC.md) for the full filename grammar.
2. Run the pipeline:
   ```bash
   go run ./cmd/corpus discover
   go run ./cmd/corpus parse
   go run ./cmd/corpus strip
   ```
3. The episode is now visible to `ask`.

Each stage is idempotent. `discover` ignores already-known files. `parse` and `strip` only process episodes at the previous stage. So you can always rerun the whole sequence safely.

### A new doc or article

Just drop a `.md` or `.txt` file into `corpus/raw/docs/` or `corpus/raw/articles/`. No pipeline. The next `corpus ask` invocation will pick it up automatically.

You can use subdirectories for organization:
```
corpus/raw/articles/
  copy/
    pdp-anatomy.md
    landing-pages.md
  retention/
    bfcm-retention.md
```

The doc id used for citation will be the path under the source dir without the extension (e.g. `copy/pdp-anatomy`).

## Asking questions

### Basic

```bash
go run ./cmd/corpus ask "what do nik and moiz say about loyalty programs?"
```

### Multi-line / paste-a-PDP

```bash
go run ./cmd/corpus ask -                    # then paste, then Ctrl-D
# or pipe:
cat my-pdp.html | go run ./cmd/corpus ask -
```

### Limit how much corpus is loaded

```bash
go run ./cmd/corpus ask --limit 10 "..."     # cap at 10 docs total
```

Useful for cheap exploratory questions or to keep total tokens low when iterating on the prompt.

### Disable terminal wrap

```bash
go run ./cmd/corpus ask --width 0 "..."
```

Useful when piping to a file or another tool.

## What makes a good question

The ask loop is good at:
- "What do they say about X?" — broad survey questions
- "I'm doing X, what would Nik & Moiz suggest?" — actionable recommendations
- "Look at this PDP / email / page and critique it" — paste content, get specific edits
- "Cite three episodes that disagree about Y" — comparative

The ask loop struggles with:
- Questions whose answer is not in the corpus (it'll either say so, or hallucinate — flag it if it does)
- Questions that need numerical precision the transcripts don't contain
- Questions about very recent events (the corpus is bounded by what's been ingested)

The role prompt in `prompts/ask.md` instructs the model to cite episodes by title (e.g. `("Why Native Rejected Investors", S1E1)`). If you see opaque hashes like `9lmar28vdkl4r2nw` in citations, it means that episode wasn't in `state.db` when `ask` ran — usually because `discover` was never run for it.

## State and reruns

Everything pipeline-related lives in `corpus/state.db` (SQLite). One row per episode, with a `stage` column tracking progress.

- See what's pending: `sqlite3 corpus/state.db "SELECT stage, COUNT(*) FROM episodes GROUP BY stage"`
- Force a re-strip of one episode: `sqlite3 corpus/state.db "UPDATE episodes SET stage='parsed' WHERE id='limited-supply-9lmar28vdkl4r2nw'"`, then `corpus strip`
- Nuke and restart: `rm corpus/state.db corpus/cues -rf corpus/clean_v1 -rf` then run the pipeline

## Cost and timing

- `discover`, `parse`: free, instant.
- `strip`: Haiku 4.5. ~$0.02-0.04 per episode. ~30-60 seconds per episode (sequential). For ~160 episodes: ~$3-6 and 2-3 hours wall-clock.
- `ask`: Sonnet 4.6 with prompt caching.
  - Cold call (corpus first time or 5+ min idle): ~$1-2 for input, ~$0.10 for output, with a 450K-token corpus
  - Warm call (within 5 min of last ask): ~$0.20 input + ~$0.10 output

Anthropic's ephemeral prompt cache has a 5-minute TTL. Reading a much larger corpus benefits from the 1M-context beta — see [Roadmap](roadmap.md) Phase 1.

## Troubleshooting

**"no documents found in any source"**
You haven't run the pipeline yet, or all docs/articles dirs are empty. Check `ls corpus/clean_v1/` and `ls corpus/raw/docs/`.

**Citations are opaque hashes, not titles**
The episode isn't in `state.db`, or `discover` was never run. Run `corpus discover`.

**Strip is stuck at "FAIL: hit max_tokens"**
The transcript is unusually long. Increase `maxTokens` in [internal/strip/strip.go](../internal/strip/strip.go) or split the episode.

**Ask returns "context_length_exceeded"**
Your corpus is bigger than the standard 200K Sonnet context. Either use `--limit N` to cap how many docs are loaded, or wait for the 1M-context beta to be wired in (Phase 1 on the roadmap).

**`.env` not loading**
The loader reads `.env` from the current working directory, not the binary's location. Run `corpus` commands from the repo root.

## Inspecting intermediate output

When something looks weird in an `ask` answer, the troubleshooting flow is:
1. Find which episode the citation points to.
2. `cat corpus/clean_v1/limited-supply/<id>.md` — does the cleaned prose look right?
3. If not: `cat corpus/cues/limited-supply/<id>.json | jq '.cues[:5]'` — does the parser output look right?
4. If not: `head corpus/raw/limited-supply/<original-filename>.txt` — is the source file actually what you think it is?

Three intermediate stages on disk = three places to check before assuming the model is the problem.
