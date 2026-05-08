package ai

import "time"

// CallResult is the per-command return shape: the answer the user sees, plus
// the metadata needed for telemetry, logging, and chat persistence.
//
// Single-shot commands (ask) set Steps=1. Agent-loop commands (dig, review)
// set Steps to the number of model turns the loop took.
type CallResult struct {
	Answer  string
	Usage   Usage
	Elapsed time.Duration
	Steps   int
}
