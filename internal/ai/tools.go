package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ToolDef is the model-facing description of one tool. The agent loop builds
// a slice of these from internal/tools and passes them to Step.
type ToolDef struct {
	Name        string
	Description string
	InputSchema json.RawMessage // JSON Schema for the tool's input
}

// ToolUse is the model asking us to invoke a tool.
type ToolUse struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult is what we send back after executing a tool. One per ToolUse.
type ToolResult struct {
	ToolUseID string
	Content   string // tool output text the model will read
	IsError   bool
}

// Step is one assistant turn from the model. The agent loop appends a
// HistoryTurn made from this back into the conversation, then either:
//   - executes ToolUses (if any) and continues, or
//   - returns Text to the user (if no tool uses).
type Step struct {
	Text       string    // concatenated text-content blocks
	ToolUses   []ToolUse // tool_use blocks the model produced
	StopReason string    // "end_turn" | "tool_use" | "max_tokens" | "stop_sequence"
}

// HistoryTurn is one prior round in the conversation. The agent owns history
// bookkeeping — it constructs HistoryTurns from Steps it received and from
// ToolResults it produced.
type HistoryTurn struct {
	// Role is "assistant" (Text + ToolUses populated) or "user" (ToolResults
	// populated, in response to the previous assistant ToolUses).
	Role        string
	Text        string
	ToolUses    []ToolUse
	ToolResults []ToolResult
}

// History is the full conversation seen by the model on a Step call.
type History struct {
	UserPrompt string        // initial question (becomes the first user message)
	Turns      []HistoryTurn // alternating assistant / user turns since
}

// ToolCompleter is the tool-aware sibling of Completer. The agent loop calls
// Step repeatedly until the model returns plain text (no tool uses).
type ToolCompleter interface {
	Step(ctx context.Context, systemPrompt string, hist History, tools []ToolDef) (Step, Usage, error)
}

type sdkToolCompleter struct {
	client      anthropic.Client
	model       anthropic.Model
	maxTokens   int64
	longContext bool
}

// NewToolCompleter returns a ToolCompleter backed by the Anthropic SDK with
// prompt caching enabled on the system block.
func NewToolCompleter(model anthropic.Model, maxTokens int64, opts ...Option) ToolCompleter {
	o := resolveOptions(opts)
	return &sdkToolCompleter{client: Client(), model: model, maxTokens: maxTokens, longContext: o.longContext}
}

func (c *sdkToolCompleter) Step(ctx context.Context, systemPrompt string, hist History, tools []ToolDef) (Step, Usage, error) {
	var reqOpts []option.RequestOption
	if c.longContext {
		reqOpts = append(reqOpts, option.WithHeaderAdd("anthropic-beta", string(anthropic.AnthropicBetaContext1m2025_08_07)))
	}

	messages, err := historyToMessages(hist)
	if err != nil {
		return Step{}, Usage{}, err
	}
	sdkTools, err := toolDefsToSDK(tools)
	if err != nil {
		return Step{}, Usage{}, err
	}

	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System: []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: messages,
		Tools:    sdkTools,
	}, reqOpts...)

	msg := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return Step{}, Usage{}, err
		}
	}
	if err := stream.Err(); err != nil {
		return Step{}, Usage{}, err
	}

	usage := Usage{
		Model:               string(c.model),
		InputTokens:         msg.Usage.InputTokens,
		CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
		CacheReadTokens:     msg.Usage.CacheReadInputTokens,
		OutputTokens:        msg.Usage.OutputTokens,
	}

	step := Step{StopReason: string(msg.StopReason)}
	for _, b := range msg.Content {
		switch b.Type {
		case "text":
			step.Text += b.Text
		case "tool_use":
			step.ToolUses = append(step.ToolUses, ToolUse{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			})
		}
	}
	if msg.StopReason == anthropic.StopReasonMaxTokens {
		return step, usage, fmt.Errorf("hit max_tokens (%d) before stop", c.maxTokens)
	}
	return step, usage, nil
}

// historyToMessages translates our portable History into the SDK's MessageParam
// slice. Order: initial user prompt, then alternating assistant/user turns.
func historyToMessages(h History) ([]anthropic.MessageParam, error) {
	if h.UserPrompt == "" {
		return nil, fmt.Errorf("history.UserPrompt is empty")
	}
	out := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(h.UserPrompt)),
	}
	for _, t := range h.Turns {
		switch t.Role {
		case "assistant":
			var blocks []anthropic.ContentBlockParamUnion
			if t.Text != "" {
				blocks = append(blocks, anthropic.NewTextBlock(t.Text))
			}
			for _, u := range t.ToolUses {
				var input any
				if len(u.Input) > 0 {
					if err := json.Unmarshal(u.Input, &input); err != nil {
						return nil, fmt.Errorf("decode tool_use input for %s: %w", u.Name, err)
					}
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(u.ID, input, u.Name))
			}
			if len(blocks) == 0 {
				return nil, fmt.Errorf("assistant turn has no content")
			}
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		case "user":
			if len(t.ToolResults) == 0 {
				return nil, fmt.Errorf("user turn after the prompt must carry tool_results")
			}
			blocks := make([]anthropic.ContentBlockParamUnion, 0, len(t.ToolResults))
			for _, r := range t.ToolResults {
				blocks = append(blocks, anthropic.NewToolResultBlock(r.ToolUseID, r.Content, r.IsError))
			}
			out = append(out, anthropic.NewUserMessage(blocks...))
		default:
			return nil, fmt.Errorf("unknown role %q", t.Role)
		}
	}
	return out, nil
}

// toolDefsToSDK translates our portable []ToolDef into []anthropic.ToolUnionParam.
// JSON schemas are decoded from the raw bytes into the SDK's Properties/Required
// shape.
func toolDefsToSDK(tools []ToolDef) ([]anthropic.ToolUnionParam, error) {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var rs struct {
			Properties any      `json:"properties"`
			Required   []string `json:"required"`
		}
		if len(t.InputSchema) > 0 {
			if err := json.Unmarshal(t.InputSchema, &rs); err != nil {
				return nil, fmt.Errorf("tool %s: decode input schema: %w", t.Name, err)
			}
		}
		schema := anthropic.ToolInputSchemaParam{
			Properties: rs.Properties,
			Required:   rs.Required,
		}
		tp := &anthropic.ToolParam{
			Name:        t.Name,
			InputSchema: schema,
		}
		if t.Description != "" {
			tp.Description = anthropic.String(t.Description)
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: tp})
	}
	return out, nil
}
