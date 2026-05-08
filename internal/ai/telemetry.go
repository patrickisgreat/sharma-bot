package ai

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// PrintTelemetry writes a one-line summary of token usage and estimated cost
// to w. Safe to call with a zero Usage (e.g. when the call failed before we
// got a usage block) — it returns silently in that case.
//
// extra is appended in brackets at the end if non-empty (used by the agent
// loop to add "X steps" or similar).
func PrintTelemetry(w io.Writer, u Usage, elapsed time.Duration, extra string) {
	if w == nil {
		return
	}
	if u.Model == "" && u.InputTokens == 0 && u.OutputTokens == 0 {
		return
	}
	cost := EstimateCost(anthropic.Model(u.Model), u)
	tail := ""
	if extra != "" {
		tail = " | " + extra
	}
	fmt.Fprintf(w,
		"[%s] in: %s tok (cache write %s, read %s) | out: %s tok | $%.4f | %s%s\n",
		u.Model,
		formatCommas(u.InputTokens), formatCommas(u.CacheCreationTokens), formatCommas(u.CacheReadTokens),
		formatCommas(u.OutputTokens), cost, elapsed.Round(time.Millisecond), tail,
	)
}

// formatCommas formats an int64 with thousands separators. 12345 -> "12,345".
func formatCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return "-" + formatCommas(-n)
	}
	if len(s) <= 3 {
		return s
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
		if len(s) > rem {
			out = append(out, ',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, ',')
		}
	}
	return string(out)
}
