# User Guide

CLI workflows, common commands, and the things that bite the first time.

## Setup

You need:
- Go 1.23+
- An Anthropic API key (https://console.anthropic.com)
- macOS or Linux
- `wget` (for the Shopify fetch script): `brew install wget`

```bash
cp .env.example .env
# edit .env, paste your ANTHROPIC_API_KEY
go build ./cmd/corpus    # optional; you can also use `go run`
```

`.env` is auto-loaded on every invocation. Shell-exported env vars (e.g. `export ANTHROPIC_API_KEY=...`) win over `.env`.

## The command surface

```
corpus discover    scan raw/<source>/ and populate state.db
corpus parse       cue-parse raw .txt -> cues/<source>/<id>.json
corpus strip       Haiku-clean cues -> clean_v1/<source>/<id>.md
corpus ask         load full corpus into context, ask Claude
corpus dig         agent-loop: model uses tools to load only what it needs
corpus review      review user content (email, page, ad copy) against the corpus
```

Common flags:
- `--corpus-dir`  path to corpus root (default: `./corpus`)
- `--prompts-dir` path to prompts directory (default: `./prompts`)
- `--limit`       max episodes/docs to process or load (0 = no limit)
- `--width`       wrap output to N columns (0 = no wrap; default: 100)
- `--trace`       print agent step trace to stderr (dig/review; default: true)
- `--batch`       directory of files to review (review only)

## Adding new content

### A new podcast episode

1. Drop the timestamped `.txt` into `corpus/raw/limited-supply/`. Filename format: `YYYYMMDD_S{n}_E{n}__<title_slug>-ep_<external_id>.txt`. See [SPEC.md](../SPEC.md).
2. Run the pipeline:
   ```bash
   go run ./cmd/corpus discover
   go run ./cmd/corpus parse
   go run ./cmd/corpus strip
   ```

Each stage is idempotent. Reruns process only what's new.

### A new doc or article

Drop a `.md` or `.txt` file into `corpus/raw/docs/` or `corpus/raw/articles/`. No pipeline. The next `ask` / `dig` invocation picks it up automatically.

Subdirectories work fine for organization:
```
corpus/raw/articles/
  copy/pdp-anatomy.md
  retention/bfcm.md
```

The doc id used in citations is the path under the source dir without the extension (e.g. `copy/pdp-anatomy`).

## Asking questions

### `corpus dig` — the daily driver

```bash
go run ./cmd/corpus dig "what do nik and moiz say about loyalty programs?"
```

The model autonomously uses `glob`/`grep`/`read_doc` to load only the docs it needs. Cost: pennies per call. Trace shows you what it did:

```
[step 1] 1 tool call(s) — "Let me search for loyalty program discussions"
  → grep({"query": "loyalty program"})
  ← grep: Found 4 documents matching "loyalty program"...
[step 2] 1 tool call(s)
  → read_doc({"source": "limited-supply", "id": "abc123"})
  ← read_doc: limited-supply / abc123  ("Why Native Rejected...
[step 3] final answer (1834 chars)
[claude-sonnet-4-6] in: 8,432 tok (cache write 6,200, read 0) | out: 612 tok | $0.0367 | 12.3s | 3 step(s)
```

### `corpus ask` — the heavy hammer

```bash
go run ./cmd/corpus ask "summarize every retention tactic mentioned across all 162 episodes"
```

Stuffs the entire corpus (~700K tokens) into the system prompt and asks. Cost: ~$1.50 cold, ~$0.20 cached within 5 min. Use when you want the model to consider everything in one pass.

### Multi-line / paste-a-PDP via stdin

Both `ask` and `dig` accept `-` to read from stdin:

```bash
go run ./cmd/corpus dig -                  # type, then Ctrl-D
cat my-pdp.html | go run ./cmd/corpus dig -
```

## Reviewing your own content

`corpus review` is `dig` with a different role — it reviews user copy and proposes specific edits, grounded in the corpus.

### Single file

```bash
go run ./cmd/corpus review email.md
go run ./cmd/corpus review page.html        # HTML auto-extracted to text
echo "subject: hi\nbody: ..." | go run ./cmd/corpus review -
```

Output goes to stdout — a structured review with sections: what's working, what's not (with verbatim quotes), suggested edits with replacement copy, corpus citations.

### Batch mode

```bash
go run ./cmd/corpus review --batch klaviyo-emails/
```

Reviews every `.html`/`.htm`/`.md`/`.txt` file under the directory. Per-file outputs go to `reviews/<timestamp>/<original-name>.review.md`. Trace summary at the end.

### Reviewing a Shopify storefront

```bash
# Set your Shopify password in .env (gitignored)
echo "SHOPIFY_PASSWORD=secretsauce" >> .env

# Mirror the storefront
./scripts/fetch-shopify.sh https://your-store.myshopify.com

# Review the HTML files (auto-extracted to text)
go run ./cmd/corpus review --batch tmp/shopify-mirror-<timestamp>/
```

The fetch script handles the password gate, mirrors with wget (rejecting binary assets), and tells you exactly what command to run next.

## State and reruns

- See pipeline state: `sqlite3 corpus/state.db "SELECT stage, COUNT(*) FROM episodes GROUP BY stage"`
- Force a re-strip of one episode: `sqlite3 corpus/state.db "UPDATE episodes SET stage='parsed' WHERE id='limited-supply-<id>'"`, then `corpus strip`
- Nuke and restart: `rm corpus/state.db corpus/cues -rf corpus/clean_v1 -rf` then run the pipeline

## Cost and timing

| Stage | Model | Cost (typical) | Time |
|-------|-------|----------------|------|
| `discover` | none | free | instant |
| `parse` | none | free | instant |
| `strip` (one episode) | Haiku 4.5 | ~$0.02–0.04 | ~30–60s |
| `ask` (cold) | Sonnet 4.6 (1M ctx) | ~$1.50–2.00 | ~10–20s |
| `ask` (cached, <5min) | Sonnet 4.6 (1M ctx) | ~$0.20 | ~10–20s |
| `dig` | Sonnet 4.6 | ~$0.05–0.20 | ~5–60s |
| `review` (single file) | Sonnet 4.6 | ~$0.05–0.20 | ~10–30s |

Anthropic's ephemeral cache TTL is 5 minutes. If you idle longer than that, the next call pays full input again.

## Troubleshooting

**"no documents found in any source"**
You haven't run the pipeline yet. Check `ls corpus/clean_v1/` — should have `.md` files.

**Citations are opaque hashes, not titles**
The episode isn't in `state.db`. Run `corpus discover` first.

**Strip says "FAIL: hit max_tokens"**
The transcript is unusually long. Increase `maxTokens` in [internal/strip/strip.go](../internal/strip/strip.go#L20) or split the episode.

**Ask returns "context_length_exceeded"**
Even with 1M context, the prompt is too big. Use `--limit N` to cap, or switch to `dig` (which only loads what it needs).

**Dig hits MaxSteps without answering**
The question is probably too broad, or the corpus genuinely doesn't have the answer. Try a more specific question, or increase MaxSteps in [internal/dig/dig.go](../internal/dig/dig.go) for that one run.

**Shopify fetch says "password authentication failed"**
Verify `SHOPIFY_PASSWORD` in `.env` matches the actual storefront password. Run with `bash -x scripts/fetch-shopify.sh ...` for verbose output.

**`.env` not loading**
The loader reads `.env` from the current working directory, not the binary's location. Run `corpus` commands from the repo root.

## Inspecting intermediate output

When something looks weird in an answer, the troubleshooting flow is:

1. Find which episode the citation points to.
2. `cat corpus/clean_v1/limited-supply/<id>.md` — does the cleaned prose look right?
3. If not: `cat corpus/cues/limited-supply/<id>.json | jq '.cues[:5]'` — does the parser output look right?
4. If not: `head corpus/raw/limited-supply/<original-filename>.txt` — is the source file actually what you think it is?

Three intermediate stages on disk = three places to check before assuming the model is the problem.
