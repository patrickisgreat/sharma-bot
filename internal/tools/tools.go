// Package tools defines the Tool abstraction used by the agent loop. A Tool
// is a name, a JSON-Schema-shaped input contract, and a Run function that
// produces a text result the model can read.
//
// The Tool type is intentionally SDK-agnostic. Wiring tools into a model call
// (i.e. translating to anthropic.ToolUnionParam) lives in internal/ai.
package tools

import (
	"context"
	"encoding/json"
)

// Tool is one capability the model can invoke during an agent loop turn.
type Tool struct {
	// Name is the identifier the model uses to call the tool. Must be unique
	// within a registry and match the regex `[a-zA-Z0-9_-]+`.
	Name string

	// Description tells the model what the tool does and when to use it. The
	// quality of this string is the single biggest lever on tool-selection
	// quality — write it for the model, not for humans.
	Description string

	// InputSchema is the JSON Schema describing the tool's input. The model
	// will produce inputs that match this shape; we'll JSON-decode them in Run.
	InputSchema json.RawMessage

	// Run executes the tool with the given JSON-encoded input and returns the
	// text the model will see in the corresponding tool_result block. Errors
	// returned here become tool-result errors, surfaced to the model so it can
	// react (e.g. retry with different inputs).
	Run func(ctx context.Context, input json.RawMessage) (string, error)
}

// ByName looks up a tool by name. Returns (nil, false) if not found.
func ByName(tools []Tool, name string) (Tool, bool) {
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}
