package wrap

import (
	"strings"
	"testing"
)

func TestTextZeroWidthNoop(t *testing.T) {
	in := "hello world this is a sentence"
	if got := Text(in, 0); got != in {
		t.Errorf("zero width should be no-op, got %q", got)
	}
}

func TestTextShortLineUnchanged(t *testing.T) {
	in := "short line"
	if got := Text(in, 80); got != in {
		t.Errorf("short line changed: %q", got)
	}
}

func TestTextWrapsLongLine(t *testing.T) {
	in := "alpha beta gamma delta epsilon zeta eta theta"
	got := Text(in, 20)
	for _, line := range strings.Split(got, "\n") {
		if len(line) > 20 {
			t.Errorf("line too long: %q (%d > 20)", line, len(line))
		}
	}
	// Words should be in original order, joined by space or newline.
	flattened := strings.Join(strings.Fields(got), " ")
	if flattened != in {
		t.Errorf("words rearranged: %q vs %q", flattened, in)
	}
}

func TestTextPreservesEmptyLines(t *testing.T) {
	in := "para one\n\npara two"
	if got := Text(in, 80); got != in {
		t.Errorf("empty line not preserved: %q", got)
	}
}

func TestTextPreservesLeadingIndent(t *testing.T) {
	in := "    indented line with many words that should stay indented when wrapped"
	got := Text(in, 30)
	for _, line := range strings.Split(got, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "    ") {
			t.Errorf("indent dropped: %q", line)
		}
	}
}

func TestTextDoesNotBreakSingleLongWord(t *testing.T) {
	// A word longer than width should be emitted as-is (no hyphenation),
	// not lost.
	in := "supercalifragilisticexpialidocious is long"
	got := Text(in, 10)
	if !strings.Contains(got, "supercalifragilisticexpialidocious") {
		t.Errorf("long word lost: %q", got)
	}
}

func TestTextMultipleLinesIndependently(t *testing.T) {
	in := "first short\nsecond also short\nthird"
	if got := Text(in, 80); got != in {
		t.Errorf("multi-line short text changed: %q", got)
	}
}
