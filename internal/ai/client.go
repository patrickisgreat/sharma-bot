package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

var (
	once   sync.Once
	shared anthropic.Client
)

func Client() anthropic.Client {
	once.Do(func() {
		shared = anthropic.NewClient()
	})
	return shared
}

// Usage describes the token accounting returned by a completion. It mirrors
// the fields the SDK exposes so callers don't need to import the SDK to log
// or price a call.
type Usage struct {
	Model               string
	InputTokens         int64
	CacheCreationTokens int64 // cache write (5m TTL with current settings)
	CacheReadTokens     int64
	OutputTokens        int64
}

// Completer turns (system, user) into model output. Stages depend on this
// interface, not the SDK directly, so tests can swap in a fake.
type Completer interface {
	Complete(ctx context.Context, systemPrompt, userText string) (string, Usage, error)
}

// completerOpts is the shared shape both Completer and ToolCompleter take
// at construction time.
type completerOpts struct {
	longContext bool
}

// Option mutates a completer's behavior at construction time. Same option
// value can be passed to NewCompleter or NewToolCompleter.
type Option func(*completerOpts)

// WithLongContext enables Anthropic's 1M-context beta. Required when the
// system prompt + conversation will exceed 200K tokens.
func WithLongContext() Option {
	return func(o *completerOpts) { o.longContext = true }
}

func resolveOptions(opts []Option) completerOpts {
	var o completerOpts
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

type sdkCompleter struct {
	client      anthropic.Client
	model       anthropic.Model
	maxTokens   int64
	longContext bool
}

// NewCompleter returns a Completer backed by the Anthropic SDK with prompt
// caching enabled on the system block. Pass WithLongContext() to allow >200K
// token prompts.
func NewCompleter(model anthropic.Model, maxTokens int64, opts ...Option) Completer {
	o := resolveOptions(opts)
	return &sdkCompleter{client: Client(), model: model, maxTokens: maxTokens, longContext: o.longContext}
}

func (c *sdkCompleter) Complete(ctx context.Context, systemPrompt, userText string) (string, Usage, error) {
	var reqOpts []option.RequestOption
	if c.longContext {
		reqOpts = append(reqOpts, option.WithHeaderAdd("anthropic-beta", string(anthropic.AnthropicBetaContext1m2025_08_07)))
	}

	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System: []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userText)),
		},
	}, reqOpts...)

	msg := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return "", Usage{}, err
		}
	}
	if err := stream.Err(); err != nil {
		return "", Usage{}, err
	}

	usage := Usage{
		Model:               string(c.model),
		InputTokens:         msg.Usage.InputTokens,
		CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
		CacheReadTokens:     msg.Usage.CacheReadInputTokens,
		OutputTokens:        msg.Usage.OutputTokens,
	}

	var out string
	for _, b := range msg.Content {
		if b.Type == "text" {
			out += b.Text
		}
	}
	if msg.StopReason == anthropic.StopReasonMaxTokens {
		return out, usage, fmt.Errorf("hit max_tokens (%d) before stop", c.maxTokens)
	}
	if out == "" {
		return "", usage, fmt.Errorf("empty response from %s", c.model)
	}
	return out, usage, nil
}
