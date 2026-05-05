// Package wrap soft-wraps prose to a column width for terminal display.
package wrap

import "strings"

// Text wraps each input line to width columns, preserving newlines and any
// leading whitespace on continuation lines. width <= 0 disables wrapping.
//
// This is intentionally simple: it does not understand markdown structure
// (lists, code fences). It just word-wraps each line. Output stays close to
// the input for short content and never mangles structure.
func Text(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = wrapLine(line, width)
	}
	return strings.Join(out, "\n")
}

func wrapLine(line string, width int) string {
	leading := leadingWhitespace(line)
	rest := line[len(leading):]
	words := strings.Fields(rest)
	if len(words) == 0 {
		return line
	}
	var sb strings.Builder
	sb.WriteString(leading)
	sb.WriteString(words[0])
	col := len(leading) + len(words[0])
	for _, w := range words[1:] {
		if col+1+len(w) > width {
			sb.WriteByte('\n')
			sb.WriteString(leading)
			sb.WriteString(w)
			col = len(leading) + len(w)
		} else {
			sb.WriteByte(' ')
			sb.WriteString(w)
			col += 1 + len(w)
		}
	}
	return sb.String()
}

func leadingWhitespace(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return s[:i]
		}
	}
	return s
}
