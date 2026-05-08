package ai

import (
	"math"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestEstimateCostSonnetStandard(t *testing.T) {
	u := Usage{
		InputTokens:         100_000,
		CacheCreationTokens: 0,
		CacheReadTokens:     0,
		OutputTokens:        1_000,
	}
	got := EstimateCost(anthropic.ModelClaudeSonnet4_6, u)
	// 100K * 3.00/M = 0.30, 1K * 15.00/M = 0.015 → 0.315
	want := 0.315
	if !approxEq(got, want) {
		t.Errorf("got %.6f, want %.6f", got, want)
	}
}

func TestEstimateCostSonnetLongTier(t *testing.T) {
	u := Usage{
		InputTokens:  300_000, // > 200K → long-context pricing
		OutputTokens: 1_000,
	}
	got := EstimateCost(anthropic.ModelClaudeSonnet4_6, u)
	// 300K * 6.00/M = 1.80, 1K * 22.50/M = 0.0225 → 1.8225
	want := 1.8225
	if !approxEq(got, want) {
		t.Errorf("got %.6f, want %.6f", got, want)
	}
}

func TestEstimateCostCacheReadDiscount(t *testing.T) {
	// Big system prompt, mostly cached: write once, read on subsequent call.
	u := Usage{
		InputTokens:     50,
		CacheReadTokens: 450_000, // counts toward 200K threshold via total input
		OutputTokens:    200,
	}
	got := EstimateCost(anthropic.ModelClaudeSonnet4_6, u)
	// totalInput = 450050 > 200K → long.
	// input: 50 * 6/M = 0.0003
	// cache read: 450000 * 0.60/M = 0.27
	// output: 200 * 22.5/M = 0.0045
	want := 0.0003 + 0.27 + 0.0045
	if !approxEq(got, want) {
		t.Errorf("got %.6f, want %.6f", got, want)
	}
}

func TestEstimateCostUnknownModelReturnsZero(t *testing.T) {
	if got := EstimateCost(anthropic.Model("totally-fake-model"), Usage{InputTokens: 1000}); got != 0 {
		t.Errorf("expected 0 for unknown model, got %v", got)
	}
}

func TestEstimateCostHaiku(t *testing.T) {
	u := Usage{InputTokens: 10_000, OutputTokens: 5_000}
	got := EstimateCost(anthropic.ModelClaudeHaiku4_5, u)
	// 10K * 1.00/M = 0.01, 5K * 5.00/M = 0.025 → 0.035
	want := 0.035
	if !approxEq(got, want) {
		t.Errorf("got %.6f, want %.6f", got, want)
	}
}
