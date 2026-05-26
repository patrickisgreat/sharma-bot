package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sharma-bot/internal/tools"
)

// fileEditor holds the in-memory buffer for one file the agent is allowed to
// rewrite. Edits accumulate against content; the caller flushes content to
// disk after the agent loop finishes (and after any validation). original is
// retained so callers can detect whether anything changed and roll back.
//
// When validateJSON is set, each edit is checked against JSON validity and
// reverted in-place if it would break the file — so one malformed edit on a
// big template doesn't force an all-or-nothing rollback of every other (good)
// edit. The model gets the parse error back and can retry just that edit.
type fileEditor struct {
	path         string
	original     string
	content      string
	edits        int
	validateJSON bool
}

func newFileEditor(path, content string) *fileEditor {
	return &fileEditor{path: path, original: content, content: content}
}

func (e *fileEditor) changed() bool { return e.content != e.original }

type editInput struct {
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

const editSchema = `{
  "type": "object",
  "properties": {
    "old_string": {"type": "string", "description": "Exact text to replace. Must appear exactly once in the file — include enough surrounding context (whole lines, adjacent keys) to be unique."},
    "new_string": {"type": "string", "description": "Replacement text."}
  },
  "required": ["old_string", "new_string"]
}`

// tool exposes the editor to the agent loop. The model uses it to apply the
// changes it recommends — both copy (headings, answers, body) and structure
// (reordering an "order" or "block_order" array, moving or editing blocks).
func (e *fileEditor) tool() tools.Tool {
	return tools.Tool{
		Name: "edit_file",
		Description: "Replace an exact substring of the file under review with new text. " +
			"Use this to APPLY your recommendations, not just describe them — rewrite copy " +
			"and restructure (e.g. reorder sections by editing the \"order\" array, move a " +
			"block by editing \"block_order\"). old_string must match the current file " +
			"byte-for-byte and appear exactly once; include surrounding context to disambiguate. " +
			"For Shopify theme JSON, keep the result valid JSON — don't break quotes/commas/braces, " +
			"and leave SVG paths, color schemes, padding, and image refs untouched unless the edit is about them.",
		InputSchema: json.RawMessage(editSchema),
		Run: func(_ context.Context, raw json.RawMessage) (string, error) {
			var in editInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if in.OldString == "" {
				return "", fmt.Errorf("old_string is required")
			}
			if in.OldString == in.NewString {
				return "", fmt.Errorf("old_string and new_string are identical — no change")
			}
			n := strings.Count(e.content, in.OldString)
			if n == 0 {
				return "", fmt.Errorf("old_string not found in %s — it must match the current file exactly", e.path)
			}
			if n > 1 {
				return "", fmt.Errorf("old_string appears %d times — add surrounding context to make it unique", n)
			}
			next := strings.Replace(e.content, in.OldString, in.NewString, 1)
			if e.validateJSON {
				if err := validJSON([]byte(next)); err != nil {
					return "", fmt.Errorf("rejected: this edit breaks JSON validity (%v). The file must stay valid JSON — check for a dropped/extra comma, an unquoted value, or an unbalanced brace in new_string, and try again", err)
				}
			}
			e.content = next
			e.edits++
			return fmt.Sprintf("edit %d applied to %s", e.edits, e.path), nil
		},
	}
}
