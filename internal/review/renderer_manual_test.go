package review

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestRenderManual is a manual gate for sanity-checking the headless render
// against a live URL. Run with:
//
//	RENDER_URL=https://getringstring.com go test -v -run TestRenderManual ./internal/review/
//
// Skipped by default so CI and the normal test suite never make a network
// call.
func TestRenderManual(t *testing.T) {
	rawURL := os.Getenv("RENDER_URL")
	if rawURL == "" {
		t.Skip("set RENDER_URL=https://... to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	r, err := newRenderer(ctx, renderSettle)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, err := r.render(ctx, rawURL)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	text := ExtractText(html)

	if err := os.WriteFile("/tmp/rendered.html", html, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/rendered.txt", []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("rendered HTML: %d bytes; extracted text: %d chars\n", len(html), len([]rune(text)))
	fmt.Println("wrote /tmp/rendered.html and /tmp/rendered.txt")
}
