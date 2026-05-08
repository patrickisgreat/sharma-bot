package ai

import "github.com/anthropics/anthropic-sdk-go"

// Pricing in USD per million tokens. Update when Anthropic adjusts rates.
// "Long" rates apply when the total input exceeds 200K tokens (1M-context tier).
type Pricing struct {
	Input            float64
	Output           float64
	CacheWrite5m     float64
	CacheRead        float64
	LongInput        float64 // 0 means model doesn't support 1M context
	LongOutput       float64
	LongCacheWrite5m float64
	LongCacheRead    float64
}

// pricingTable as of 2026-05. Constants live here so the cost calculator
// doesn't reach into ai package internals when callers want pricing for a
// model they aren't actively using.
var pricingTable = map[anthropic.Model]Pricing{
	anthropic.ModelClaudeSonnet4_6: {
		Input: 3.00, Output: 15.00,
		CacheWrite5m: 3.75, CacheRead: 0.30,
		LongInput: 6.00, LongOutput: 22.50,
		LongCacheWrite5m: 7.50, LongCacheRead: 0.60,
	},
	anthropic.ModelClaudeHaiku4_5: {
		Input: 1.00, Output: 5.00,
		CacheWrite5m: 1.25, CacheRead: 0.10,
	},
	anthropic.ModelClaudeOpus4_7: {
		Input: 15.00, Output: 75.00,
		CacheWrite5m: 18.75, CacheRead: 1.50,
	},
}

// EstimateCost returns the dollar cost of a single completion. Returns 0 if
// the model has no pricing entry rather than failing — telemetry should never
// break the actual call.
func EstimateCost(model anthropic.Model, u Usage) float64 {
	p, ok := pricingTable[model]
	if !ok {
		return 0
	}
	totalInput := u.InputTokens + u.CacheCreationTokens + u.CacheReadTokens
	long := totalInput > 200_000 && p.LongInput > 0

	in := p.Input
	out := p.Output
	cw := p.CacheWrite5m
	cr := p.CacheRead
	if long {
		in = p.LongInput
		out = p.LongOutput
		cw = p.LongCacheWrite5m
		cr = p.LongCacheRead
	}

	const million = 1_000_000.0
	cost := float64(u.InputTokens) / million * in
	cost += float64(u.CacheCreationTokens) / million * cw
	cost += float64(u.CacheReadTokens) / million * cr
	cost += float64(u.OutputTokens) / million * out
	return cost
}
