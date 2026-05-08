// Package chats persists user/model exchanges to markdown files. The
// canonical layout is chats/<YYYY-MM-DD>/<HH-MM-SS>-<slug>.md, with a YAML
// frontmatter block carrying telemetry (model, tokens, cost, duration).
//
// Single-shot commands (dig, ask, review-single) call Save after each call.
// Batch review writes its own per-file outputs to reviews/<timestamp>/ and
// does not double-save.
package chats

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"sharma-bot/internal/ai"
)

// Chat captures everything needed to render one persisted exchange.
type Chat struct {
	Date     time.Time      // when the call happened (defaults to now)
	Command  string         // "dig", "ask", "review"
	Title    string         // short label; used as filename slug
	Question string         // full user prompt (may be longer than Title)
	Answer   string         // model's answer text
	Trace    string         // optional: captured agent trace, rendered fenced
	Result   *ai.CallResult // optional: usage/cost telemetry
}

// Save writes the chat and returns the path written. If outOverride is
// non-empty, the path is used as-is (parent dirs created if needed).
// Otherwise the path is derived from chatsDir + date + slug.
func Save(chatsDir, outOverride string, c Chat) (string, error) {
	if c.Date.IsZero() {
		c.Date = time.Now()
	}

	path := outOverride
	if path == "" {
		slug := slugify(c.Title)
		if slug == "" {
			slug = "chat"
		}
		path = filepath.Join(
			chatsDir,
			c.Date.Format("2006-01-02"),
			c.Date.Format("15-04-05")+"-"+slug+".md",
		)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(render(c)), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// render produces the file body: YAML frontmatter + Question/Answer/Trace.
func render(c Chat) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "date: %s\n", c.Date.Format(time.RFC3339))
	if c.Command != "" {
		fmt.Fprintf(&sb, "command: %s\n", c.Command)
	}
	if c.Title != "" {
		fmt.Fprintf(&sb, "title: %s\n", yamlString(c.Title))
	}
	if c.Result != nil {
		u := c.Result.Usage
		if u.Model != "" {
			fmt.Fprintf(&sb, "model: %s\n", u.Model)
		}
		if c.Result.Steps > 0 {
			fmt.Fprintf(&sb, "steps: %d\n", c.Result.Steps)
		}
		fmt.Fprintf(&sb, "input_tokens: %d\n", u.InputTokens)
		if u.CacheCreationTokens > 0 {
			fmt.Fprintf(&sb, "cache_creation_tokens: %d\n", u.CacheCreationTokens)
		}
		if u.CacheReadTokens > 0 {
			fmt.Fprintf(&sb, "cache_read_tokens: %d\n", u.CacheReadTokens)
		}
		fmt.Fprintf(&sb, "output_tokens: %d\n", u.OutputTokens)
		cost := ai.EstimateCost(anthropic.Model(u.Model), u)
		fmt.Fprintf(&sb, "cost_usd: %.4f\n", cost)
		if c.Result.Elapsed > 0 {
			fmt.Fprintf(&sb, "duration_sec: %.2f\n", c.Result.Elapsed.Seconds())
		}
	}
	sb.WriteString("---\n\n")

	sb.WriteString("# Question\n\n")
	sb.WriteString(strings.TrimSpace(c.Question))
	sb.WriteString("\n\n")

	sb.WriteString("# Answer\n\n")
	sb.WriteString(strings.TrimSpace(c.Answer))
	sb.WriteString("\n")

	if t := strings.TrimSpace(c.Trace); t != "" {
		sb.WriteString("\n# Trace\n\n```\n")
		sb.WriteString(t)
		sb.WriteString("\n```\n")
	}
	return sb.String()
}

// slugify turns a string into a filename-safe slug: lowercase, alphanumeric
// runs separated by hyphens, max 60 chars. Returns "" for inputs with no
// alphanumeric content.
func slugify(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && sb.Len() > 0 {
			sb.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(sb.String(), "-")
	if len(out) > 60 {
		out = out[:60]
		out = strings.TrimRight(out, "-")
	}
	return out
}

// yamlString returns a quoted YAML scalar safe for the title field. We use
// double-quotes with backslash escaping so newlines / quotes in the title
// don't break the frontmatter parser.
func yamlString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}
