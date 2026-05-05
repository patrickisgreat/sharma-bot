# CLAUDE.md

- Go 1.22+, single binary, `cmd/corpus/main.go` with subcommands
- SQLite via mattn/go-sqlite3
- No web frameworks. cobra for CLI is fine, stdlib flag is also fine.
- Anthropic SDK: github.com/anthropics/anthropic-sdk-go
- Tests: parse stage MUST have unit tests with sample fixtures.
  Other stages: skip tests for v1, manual inspection of output is faster.
- When in doubt, write to disk and let the next stage decide what to do.
- Don't add features not in SPEC.md without asking.