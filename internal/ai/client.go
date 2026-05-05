package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
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

// Completer turns (system, user) into model output. Stages depend on this
// interface, not the SDK directly, so tests can swap in a fake.
type Completer interface {
	Complete(ctx context.Context, systemPrompt, userText string) (string, error)
}

type sdkCompleter struct {
	client    anthropic.Client
	model     anthropic.Model
	maxTokens int64
}

// NewCompleter returns a Completer backed by the Anthropic SDK with prompt
// caching enabled on the system block.
func NewCompleter(model anthropic.Model, maxTokens int64) Completer {
	return &sdkCompleter{client: Client(), model: model, maxTokens: maxTokens}
}

func (c *sdkCompleter) Complete(ctx context.Context, systemPrompt, userText string) (string, error) {
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
	})
	msg := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return "", err
		}
	}
	if err := stream.Err(); err != nil {
		return "", err
	}
	var out string
	for _, b := range msg.Content {
		if b.Type == "text" {
			out += b.Text
		}
	}
	if msg.StopReason == anthropic.StopReasonMaxTokens {
		return out, fmt.Errorf("hit max_tokens (%d) before stop", c.maxTokens)
	}
	if out == "" {
		return "", fmt.Errorf("empty response from %s", c.model)
	}
	return out, nil
}
