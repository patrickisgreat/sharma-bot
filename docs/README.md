# Sharma-Bot Documentation

Engineering and product docs for the corpus + agent system.

## Contents

- **[Architecture](architecture.md)** — how the system fits together, with mermaid diagrams for components, data flow, and the Anthropic interaction.
- **[User Guide](user-guide.md)** — CLI commands, common workflows, troubleshooting.
- **[Roadmap](roadmap.md)** — what's built, what's next, in phased order with rationale.

## What is this?

A private knowledge agent built on the Limited Supply podcast catalog and a curated set of DTC marketing docs and articles. The pipeline transforms timestamped podcast transcripts into clean prose, ingests outside writing alongside it, and lets you ask Claude questions grounded in that corpus.

## One-paragraph mental model

Raw podcast `.txt` files live in `corpus/raw/<source>/`. A 4-stage Go pipeline (discover → parse → strip → ask) cleans them into prose markdown in `corpus/clean_v1/`. A SQLite state DB tracks per-episode progress so reruns are cheap and idempotent. The `ask` command loads the cleaned corpus plus any hand-curated docs and articles, stuffs them into a cached system prompt, and asks Sonnet 4.6 a question. No vector DB, no embeddings — the corpus is small enough to fit in context (with the 1M-context beta).

## Quick start

```bash
cp .env.example .env       # paste your ANTHROPIC_API_KEY
go run ./cmd/corpus ask "what do they say about loyalty programs?"
```

See [User Guide](user-guide.md) for the full command surface.
