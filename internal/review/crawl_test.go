package review

import (
	"net/url"
	"reflect"
	"testing"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestExtractLinksSameHostOnly(t *testing.T) {
	base := mustURL(t, "https://brand.com/")
	html := `
		<a href="/products/foo">Foo</a>
		<a href="https://brand.com/products/bar">Bar</a>
		<a href="https://other.com/spam">Other</a>
		<a href="https://cdn.brand.com/asset">CDN</a>
	`
	got := extractLinks(base, []byte(html))
	want := []string{
		"https://brand.com/products/foo",
		"https://brand.com/products/bar",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractLinksSkipsNonHTTPAndAssets(t *testing.T) {
	base := mustURL(t, "https://brand.com/")
	html := `
		<a href="mailto:hi@brand.com">Mail</a>
		<a href="tel:+15555550100">Phone</a>
		<a href="javascript:void(0)">JS</a>
		<a href="#top">Anchor</a>
		<a href="/static/main.css">CSS</a>
		<a href="/img/hero.jpg">Image</a>
		<a href="/feed.xml">Feed</a>
		<a href="/products/keep">Keep</a>
	`
	got := extractLinks(base, []byte(html))
	want := []string{"https://brand.com/products/keep"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractLinksDedupesAndStripsFragments(t *testing.T) {
	base := mustURL(t, "https://brand.com/")
	html := `
		<a href="/products/foo">A</a>
		<a href="/products/foo#features">A again with anchor</a>
		<a href="https://brand.com/products/foo">A absolute</a>
		<a href="/products/bar?utm=x">B</a>
		<a href="/products/bar?utm=x">B dup</a>
	`
	got := extractLinks(base, []byte(html))
	want := []string{
		"https://brand.com/products/foo",
		"https://brand.com/products/bar?utm=x",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractLinksExcludesBaseItself(t *testing.T) {
	base := mustURL(t, "https://brand.com/")
	html := `
		<a href="/">Home</a>
		<a href="https://brand.com">Home absolute</a>
		<a href="/products/foo">Foo</a>
	`
	got := extractLinks(base, []byte(html))
	want := []string{"https://brand.com/products/foo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShouldAuthenticate(t *testing.T) {
	cases := []struct {
		name       string
		crawl      string
		shopifyURL string
		want       bool
	}{
		{"matching host", "https://staging.brand.com/", "https://staging.brand.com", true},
		{"matching host with trailing slash", "https://staging.brand.com/", "https://staging.brand.com/", true},
		{"matching host case-insensitive", "https://Brand.com/", "https://brand.com", true},
		{"different host", "https://getringstring.com/", "https://staging.brand.com", false},
		{"empty SHOPIFY_URL", "https://anything.com/", "", false},
		{"malformed SHOPIFY_URL", "https://anything.com/", "::not a url::", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := mustURL(t, tc.crawl)
			if got := shouldAuthenticate(u, tc.shopifyURL); got != tc.want {
				t.Errorf("shouldAuthenticate(%q, %q) = %v, want %v",
					tc.crawl, tc.shopifyURL, got, tc.want)
			}
		})
	}
}

func TestTitleFromURL(t *testing.T) {
	cases := map[string]string{
		"https://brand.com/":                 "Home",
		"https://brand.com":                  "Home",
		"https://brand.com/pages/catalog":    "Catalog",
		"https://brand.com/pages/about-us":   "About Us",
		"https://brand.com/products/foo_bar": "Foo Bar",
		"https://brand.com/policies/privacy": "Privacy",
		"https://brand.com/contact.html":     "Contact",
		"https://brand.com/a/b/c-d":          "C D",
	}
	for in, want := range cases {
		u := mustURL(t, in)
		if got := titleFromURL(u); got != want {
			t.Errorf("titleFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStripPreamble(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "drops process narration",
			in:   "I have everything I need. Here's the full review.\n\n---\n\n## What's working\n\n- a\n",
			want: "## What's working\n\n- a\n",
		},
		{
			name: "leading h2 stays put",
			in:   "## What's working\n\n- a\n",
			want: "## What's working\n\n- a\n",
		},
		{
			name: "no h2 returns input unchanged",
			in:   "just some prose with no headings",
			want: "just some prose with no headings",
		},
		{
			name: "drops single-line preamble",
			in:   "Here's my review:\n## What's working\n",
			want: "## What's working\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripPreamble(tc.in); got != tc.want {
				t.Errorf("stripPreamble:\ngot  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestSlugFromURL(t *testing.T) {
	cases := map[string]string{
		"https://brand.com/":                 "index",
		"https://brand.com":                  "index",
		"https://brand.com/products/foo":     "products-foo",
		"https://brand.com/products/foo/":    "products-foo",
		"https://brand.com/pages/about-us":   "pages-about-us",
		"https://brand.com/a/b/c":            "a-b-c",
		"https://brand.com/products/foo?x=1": "products-foo",
	}
	for in, want := range cases {
		u := mustURL(t, in)
		if got := slugFromURL(u); got != want {
			t.Errorf("slugFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}
