package review

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// ExtractText pulls visible text out of an HTML document. It:
//
//   - skips contents of <script>, <style>, and <noscript>
//   - emits a newline when block-level elements close (so paragraphs survive)
//   - collapses runs of whitespace within a line and runs of blank lines
//
// The output is paragraph-shaped plain text suitable for LLM review.
func ExtractText(data []byte) string {
	z := html.NewTokenizer(bytes.NewReader(data))
	var sb strings.Builder
	skip := 0
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tag, _ := z.TagName()
			tagStr := string(tag)
			if isSkipTag(tagStr) {
				if tt == html.StartTagToken {
					skip++
				}
				continue
			}
			if tagStr == "br" {
				sb.WriteByte('\n')
			}
		case html.EndTagToken:
			tag, _ := z.TagName()
			tagStr := string(tag)
			if isSkipTag(tagStr) {
				if skip > 0 {
					skip--
				}
				continue
			}
			if isBlockTag(tagStr) {
				sb.WriteByte('\n')
			}
		case html.TextToken:
			if skip == 0 {
				sb.Write(z.Text())
			}
		}
	}
	return cleanText(sb.String())
}

func isSkipTag(t string) bool {
	switch t {
	case "script", "style", "noscript":
		return true
	}
	return false
}

func isBlockTag(t string) bool {
	switch t {
	case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
		"li", "tr", "td", "th",
		"section", "article", "header", "footer", "main",
		"nav", "aside", "blockquote", "pre", "hr":
		return true
	}
	return false
}

// cleanText normalizes whitespace: collapses runs of horizontal whitespace
// within a line, trims each line, and collapses runs of blank lines to one.
func cleanText(s string) string {
	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.Join(strings.Fields(l), " ")
		cleaned = append(cleaned, l)
	}
	var sb strings.Builder
	prevBlank := true
	for _, l := range cleaned {
		if l == "" {
			if !prevBlank {
				sb.WriteByte('\n')
				prevBlank = true
			}
			continue
		}
		sb.WriteString(l)
		sb.WriteByte('\n')
		prevBlank = false
	}
	return strings.TrimSpace(sb.String())
}
