package review

import (
	"bytes"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// extractLinks pulls <a href> values from data, resolves each against base,
// keeps only same-host URLs (excluding base itself), drops mailto:/tel:/
// javascript:/anchor-only links and static-asset extensions, strips fragments,
// and dedupes in discovery order.
func extractLinks(base *url.URL, data []byte) []string {
	z := html.NewTokenizer(bytes.NewReader(data))
	seen := map[string]bool{normalizeURL(base): true}
	var out []string

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		tag, hasAttr := z.TagName()
		if string(tag) != "a" || !hasAttr {
			continue
		}
		href := readHref(z)
		if href == "" {
			continue
		}
		u, ok := resolveSameHost(base, href)
		if !ok {
			continue
		}
		key := normalizeURL(u)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func readHref(z *html.Tokenizer) string {
	for {
		k, v, more := z.TagAttr()
		if string(k) == "href" {
			return strings.TrimSpace(string(v))
		}
		if !more {
			return ""
		}
	}
}

func resolveSameHost(base *url.URL, href string) (*url.URL, bool) {
	if href == "" || strings.HasPrefix(href, "#") {
		return nil, false
	}
	lower := strings.ToLower(href)
	for _, bad := range []string{"mailto:", "tel:", "javascript:", "sms:", "data:"} {
		if strings.HasPrefix(lower, bad) {
			return nil, false
		}
	}
	u, err := base.Parse(href)
	if err != nil {
		return nil, false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, false
	}
	if !strings.EqualFold(u.Host, base.Host) {
		return nil, false
	}
	if isAssetPath(u.Path) {
		return nil, false
	}
	u.Fragment = ""
	return u, true
}

func isAssetPath(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case ".css", ".js", ".mjs", ".map",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp",
		".woff", ".woff2", ".ttf", ".otf", ".eot",
		".mp4", ".mov", ".webm", ".mp3", ".wav",
		".pdf", ".zip", ".gz", ".tar",
		".xml", ".json", ".rss", ".atom", ".txt":
		return true
	}
	return false
}

// normalizeURL returns the canonical string form used for dedup keys and
// output. Fragments are dropped and an empty path is coerced to "/", so the
// homepage matches whether the link is written as "https://brand.com" or
// "https://brand.com/".
func normalizeURL(u *url.URL) string {
	c := *u
	c.Fragment = ""
	if c.Path == "" {
		c.Path = "/"
	}
	return c.String()
}

// titleFromURL derives a human-readable page title from a URL path for the
// top of the review markdown. Root becomes "Home"; otherwise the final
// path segment is hyphen/underscore-split and Title Cased ("pages/about-us"
// → "About Us"). Any trailing extension like ".html" is dropped.
func titleFromURL(u *url.URL) string {
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return "Home"
	}
	parts := strings.Split(p, "/")
	last := parts[len(parts)-1]
	if dot := strings.LastIndex(last, "."); dot > 0 {
		last = last[:dot]
	}
	last = strings.ReplaceAll(last, "_", "-")
	words := strings.Split(last, "-")
	for i, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// stripPreamble removes everything before the first "## " (h2) heading in s.
// The review prompt asks for output that opens with `## What's working`, but
// models occasionally narrate their tool-use process first ("I have what I
// need, here's the review.") — that content is noise in the saved file.
// If no h2 heading is found, returns s unchanged.
func stripPreamble(s string) string {
	idx := strings.Index(s, "\n## ")
	if idx < 0 {
		if strings.HasPrefix(s, "## ") {
			return s
		}
		return s
	}
	return strings.TrimLeft(s[idx+1:], "\n")
}

// slugFromURL turns a URL path into a filesystem-safe slug for naming saved
// HTML and review files. Root path becomes "index"; nested paths join with
// hyphens. Query string is dropped — same path with different params dedupes
// upstream by full URL, so collisions here are rare and we treat them as the
// same page for filename purposes.
func slugFromURL(u *url.URL) string {
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return "index"
	}
	parts := strings.Split(p, "/")
	cleaned := make([]string, 0, len(parts))
	for _, seg := range parts {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		seg = strings.ReplaceAll(seg, " ", "-")
		cleaned = append(cleaned, seg)
	}
	if len(cleaned) == 0 {
		return "index"
	}
	return strings.Join(cleaned, "-")
}
