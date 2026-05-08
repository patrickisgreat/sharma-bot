# CLAUDE.md — sharma-bot

## What is this project?

sharma-bot is a private knowledge agent for DTC ecommerce strategy. It transforms raw podcast transcripts (Limited Supply, Nik Sharma & Moiz Ali) and curated operator documents into a queryable corpus, then lets an operator ask Claude grounded questions about copy, PDPs, retention, email/SMS, paid social, and the operational realities of running consumer brands. Goal: replace "let me Google this" and "let me skim ten podcast episodes" with one fast, citation-backed answer.

## Requirements

1. Ingest podcast transcripts (timestamped `.txt`) into clean markdown
2. Track per-episode pipeline state in SQLite — idempotent, restartable, crash-safe
3. Auto-load podcast corpus + curated docs + articles when answering questions
4. Ground answers in specific, citation-backed claims from the corpus (not generic LLM filler)
5. Print cost telemetry (tokens, cache hit/miss, dollars) on every AI call
6. Tool-using agent loop that selectively reads only the corpus it needs
7. Review external content (Klaviyo emails, Shopify pages) against the corpus
8. Stay a single-binary CLI; keep a future-frontend door open via clean package boundaries

## Tech Stack

- **Language**: Go 1.23+
- **Storage**: SQLite via `github.com/mattn/go-sqlite3`
- **AI**: Anthropic SDK (`github.com/anthropics/anthropic-sdk-go`), Sonnet 4.6 with 1M-context beta and Haiku 4.5
- **CLI framework**: stdlib `flag` package (no cobra unless we cross 8+ commands)
- **Testing**: stdlib `testing` package, table-driven where it fits
- **No web frameworks, no ORMs, no LangChain / LangGraph / framework abstractions over the agent loop**

## Common Commands

```bash
# Build everything
go build ./...

# Run all tests
go test ./...

# Run a specific package's tests with verbose output
go test -v ./internal/strip

# Static check
go vet ./...

# Format
gofmt -w .

# Run a CLI subcommand
go run ./cmd/corpus discover
go run ./cmd/corpus parse
go run ./cmd/corpus strip
go run ./cmd/corpus ask "what about loyalty programs?"   # full corpus in context
go run ./cmd/corpus dig "what about loyalty programs?"   # agent loop with tools
go run ./cmd/corpus review email.md                      # critique user content
go run ./cmd/corpus review --batch klaviyo-emails/       # batch mode

# Every ask/dig/review-single auto-saves a transcript to chats/<date>/<slug>.md
go run ./cmd/corpus dig --no-save "throwaway question"
go run ./cmd/corpus dig --out my-chat.md "..."

# Pipe a file into ask/dig/review via stdin
cat my-pdp.html | go run ./cmd/corpus dig -

# Mirror a (password-gated) Shopify storefront for review --batch
go run ./cmd/corpus fetch-shopify          # uses $SHOPIFY_URL + $SHOPIFY_PASSWORD
go run ./cmd/corpus fetch-shopify https://other-store.myshopify.com

# Inspect pipeline state
sqlite3 corpus/state.db "SELECT stage, COUNT(*) FROM episodes GROUP BY stage"
```

## Project Structure

See [docs/architecture.md](docs/architecture.md) for the full diagram and rationale. High-level:

```
sharma-bot/
├── cmd/
│   └── corpus/                 # CLI entrypoint, thin dispatcher
├── internal/
│   ├── agent/                  # tool-using dispatch loop (the agentic part)
│   ├── ai/                     # Completer + ToolCompleter, SDK glue, pricing, telemetry, CallResult
│   ├── ask/                    # ask command — corpus-in-context Q&A
│   ├── chats/                  # markdown transcript persistence (frontmatter + Q/A/Trace)
│   ├── dig/                    # dig command — agent-loop Q&A
│   ├── discover/               # scan raw/, populate state.db
│   ├── envfile/                # tiny dotenv loader
│   ├── episode/                # filename parser
│   ├── parse/                  # cue-parse raw .txt -> cues.json
│   ├── review/                 # review command — content critique with HTML extraction
│   ├── state/                  # SQLite schema + queries
│   ├── strip/                  # Haiku cleanup pass
│   ├── tools/                  # corpus-reading tools (glob, grep, read_doc)
│   └── wrap/                   # terminal output wrap
├── prompts/                    # role/system prompts per stage (.md, editable)
├── corpus/
│   ├── raw/                    # podcast transcripts, docs, articles (source of truth)
│   ├── cues/                   # parser output (regenerable)
│   ├── clean_v1/               # Haiku-cleaned prose (regenerable but expensive)
│   └── state.db                # SQLite pipeline state
├── docs/                       # engineering + user docs
└── scripts/                    # ops helpers (e.g., fetch-shopify.sh)
```

## Architecture

Six pipeline stages (`discover` → `parse` → `strip` → `clean` → `enrich` → `validate`), only the first three implemented. Two query modes on top:

- **`ask`** stuffs the entire corpus into a cached system prompt and queries Sonnet 4.6 (1M context). Best for broad cross-corpus questions.
- **`dig`** runs the agent loop: model gets `glob`/`grep`/`read_doc` tools and pulls only the docs it needs. Per-call cost drops from ~$1.50 to pennies. Default for day-to-day use.

`review` runs `dig`'s machinery against user-supplied content (emails, PDPs, Shopify pages with HTML auto-extracted) with a critique-focused role prompt.

See [docs/architecture.md](docs/architecture.md) for component graphs, sequence diagrams, and design principles. See [docs/roadmap.md](docs/roadmap.md) for what's next.

## Environment

`.env` is loaded automatically at the start of every CLI invocation by `internal/envfile`. Variables already set in the process env win over `.env` (so `export ANTHROPIC_API_KEY=...` in the shell beats a stale `.env`).

```
ANTHROPIC_API_KEY=...           # required for strip / ask / dig / review
SHOPIFY_URL=...                 # used by corpus fetch-shopify
SHOPIFY_PASSWORD=...             # used by scripts/fetch-shopify.sh for password-gated stores
```

Never commit `.env`. It's in `.gitignore`. Use `.env.example` as the template.

## Conventions

### Go Style

- Run `gofmt` and `go vet` before every commit. CI will reject unformatted code.
- Idiomatic Go: short variable names in small scopes (`db`, `ep`, `ctx`), descriptive names in larger scopes.
- Lowercase package names, no underscores or camelCase.
- Use `internal/` for packages that aren't part of any public API (everything in this project except `cmd/`).
- Prefer pure functions when possible; isolate I/O and side effects to the edges of each package.
- Don't expose types or functions you don't need to expose. Default to lowercase (package-private).
- Use `context.Context` for cancellation/timeouts on anything I/O-bound. Pass it as the first parameter.
- Use `fmt.Errorf("...: %w", err)` to wrap errors with context.

### Package Boundaries

- Every `internal/*` package owns one concern. If a new responsibility doesn't fit, make a new package.
- `internal/ai` is the only seam between business logic and the Anthropic SDK. Stages depend on the `Completer` interface, not the SDK directly.
- `cmd/corpus` is a thin dispatcher. No business logic in main.go.

## Code Standards

### Clean Code

- **DRY, but not premature.** Three similar lines are fine. Extract on the second time you have to copy-paste, not the first.
- **SRP**: every function, type, and package does one thing. If a function description needs "and", split it.
- **No speculative abstractions.** Don't build for the future hypothetical. Build for the current request.
- **No commented-out code.** Delete it; git remembers.
- **No "// removed X" comments** explaining what was deleted.
- **No comments explaining what code does** — well-named identifiers do that. Only comment the *why*: a non-obvious constraint, a workaround, a surprising invariant.
- **No backwards-compatibility shims** unless we have actual external callers depending on the old shape. We don't.

### Error Handling

- Wrap errors with `fmt.Errorf("operation: %w", err)` so the chain is inspectable.
- Don't swallow errors. If an error is genuinely safe to ignore, write `_ = ...` so it's loud and intentional.
- At system boundaries (CLI entry, HTTP handlers, file I/O): convert internal errors into user-friendly messages with context.
- Prefer returning errors over panicking. Panic only for genuine programmer bugs (e.g., invariant violation in test setup).

### Security

- Never commit secrets, API keys, or credentials. `.env`, `secrets.*`, and `*.key` are gitignored.
- Validate and sanitize untrusted input at system boundaries (CLI args, file contents, network responses). Trust internal callers.
- Don't log secrets. Don't include them in error messages that get printed.
- Be careful with anything that constructs SQL — use parameterized queries (`?` placeholders), never string concatenation.

## Testing

No PR is mergeable without tests covering the behavior introduced or changed.

### What to test

- **Pure functions** (parsers, formatters, cost calculators): table-driven unit tests.
- **State** (anything that reads/writes SQLite): tests against a temp DB.
- **Stages that call the AI**: inject a fake `Completer` and assert on resulting filesystem + DB state. Never make real API calls in tests.
- **Tool implementations** (when we have them): unit tests with a tempdir as the corpus root.

### What not to test

- Third-party libraries. Trust the SDK, the database driver, and the standard library.
- The CLI dispatcher (`cmd/corpus/main.go`): plumbing, no logic worth testing in isolation.
- The Anthropic API itself. End-to-end real-API tests are flaky, slow, and expensive — manual inspection catches drift cheaper.

### Test naming

- `TestSomething_describesTheBehavior` (e.g., `TestRunWithErrorMarksFailed`, `TestParseFilenameRejectsBadFormat`).
- One assertion per test where reasonable. If a test has 12 assertions in a row, it's probably testing two things.

### Excuses that don't fly

- "It's just a small change." — Small changes are how regressions sneak in.
- "It's hard to test." — Difficulty testing is a design signal. Refactor for testability before writing the test.
- "I'll add tests in a follow-up PR." — Tests go in the same PR or the PR doesn't merge.

## Git Workflow

- **Always work from a feature branch.** Never commit directly to `main`. Branch naming:
  - `feat/<short-description>` — new feature or capability
  - `fix/<short-description>` — bug fix
  - `refactor/<short-description>` — code restructuring with no behavior change
  - `docs/<short-description>` — documentation-only change
  - `chore/<short-description>` — tooling, build, dependencies
- **Commit often.** Small, frequent commits that each represent one logical unit of work. Each commit should pass `go build` and `go test`.
- **Conventional commit messages:**
  - `feat:` — new feature or capability
  - `fix:` — bug fix
  - `refactor:` — code restructuring with no behavior change
  - `test:` — adding or updating tests
  - `chore:` — build, CI, dependency, tooling
  - `docs:` — documentation
  - `perf:` — performance improvements
  - `style:` — formatting, whitespace
- **Messages describe what and why, not how.** Example: `feat: add filesystem tools for agent loop` — not `update agent.go`.
- **Keep PRs reasonably sized.** A reviewer should be able to understand the full scope in one sitting. Break large work into incremental PRs that each deliver a coherent slice. Avoid monster PRs — they delay review, hide bugs, and are painful to revert.
- **Open PRs to `main` via `gh pr create`.** Include a summary and a test plan in the body.
- **The user reviews all PRs before merge.** Do not merge autonomously.
- **NEVER add `Co-Authored-By` or "Generated with Claude Code" to commits or PRs.**
- **Never bypass hooks** (`--no-verify`, `--no-gpg-sign`) unless the user explicitly asks. If a hook fails, investigate the underlying issue.
- **Never force-push to `main`.** Force-pushing to a feature branch is fine if you own it.

## Project-specific rules

- **Files on disk are the source of truth.** Every pipeline stage reads its input file, writes its output file. SQLite tracks state, not content. This makes everything debuggable (`cat` any intermediate output) and crash-safe.
- **Each stage is idempotent.** Reruns skip already-processed work via stage tracking in `state.db`. Restartable.
- **AI calls go through the `Completer` interface.** No package outside `internal/ai` imports the Anthropic SDK directly.
- **No vector DB, no embeddings, no chunking.** The corpus fits in 1M-context. Filesystem layout + per-episode metadata is the index.
- **Prompts live in `prompts/*.md`.** Never embed system prompts as Go string literals — editing one shouldn't require a recompile or PR.
- **When in doubt, write to disk and let the next stage decide what to do.** Disk is cheap. Pipe-in-memory is fragile.
- **Don't add features not on the [roadmap](docs/roadmap.md) without asking.** The roadmap is the priority document. New ideas go into the user's queue, not silent commits.
