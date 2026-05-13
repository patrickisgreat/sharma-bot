package review

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Realistic Chrome on macOS — many DTC storefronts and Cloudflare front-ends
// look at User-Agent before deciding whether to challenge a request.
const browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// fetcher fetches HTML over HTTP by shelling out to curl. curl handles
// redirects, cookies and TLS without us reimplementing them, and it's the
// same tool path our existing fetch-shopify.sh trusts.
type fetcher struct {
	curlPath    string
	cookiesPath string // empty when no auth was set up
	delay       time.Duration
	last        time.Time
}

// newFetcher returns a fetcher with default Chrome UA and inter-request delay.
// cookiesPath may be empty (no cookie jar); authenticate populates it.
func newFetcher(cookiesPath string, delay time.Duration) (*fetcher, error) {
	p, err := exec.LookPath("curl")
	if err != nil {
		return nil, fmt.Errorf("curl not found on PATH: %w", err)
	}
	return &fetcher{curlPath: p, cookiesPath: cookiesPath, delay: delay}, nil
}

// authenticate posts to <baseURL>/password (the Shopify storefront-password
// form) and saves the resulting session cookie. Mirrors the wget flow in
// scripts/fetch-shopify.sh.
func (f *fetcher) authenticate(ctx context.Context, baseURL, password string) error {
	if f.cookiesPath == "" {
		return fmt.Errorf("authenticate: no cookies path set")
	}
	postURL := strings.TrimRight(baseURL, "/") + "/password"
	form := "form_type=storefront_password&utf8=%E2%9C%93&password=" + password
	cmd := exec.CommandContext(ctx, f.curlPath,
		"-sS", "-L",
		"-A", browserUA,
		"--max-time", "30",
		"-c", f.cookiesPath,
		"-b", f.cookiesPath,
		"--data", form,
		"-o", os.DevNull,
		postURL,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("password auth: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// fetch GETs rawURL and returns the body. Returns a non-nil error on any
// HTTP-level failure or non-2xx status, with the status code in the message.
func (f *fetcher) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if !f.last.IsZero() && f.delay > 0 {
		wait := f.delay - time.Since(f.last)
		if wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	defer func() { f.last = time.Now() }()

	args := []string{
		"-sS", "-L",
		"-A", browserUA,
		"--max-time", "30",
		"-w", "\n__HTTP_STATUS__=%{http_code}",
	}
	if f.cookiesPath != "" {
		args = append(args, "-b", f.cookiesPath, "-c", f.cookiesPath)
	}
	args = append(args, rawURL)

	cmd := exec.CommandContext(ctx, f.curlPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("curl: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	body, status := splitStatus(stdout.Bytes())
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return body, nil
}

// splitStatus pulls the trailing "__HTTP_STATUS__=NNN" sentinel that curl's
// -w flag appends, and returns (body, status). If parsing fails, returns
// (raw, 0) so callers treat it as a non-2xx failure.
func splitStatus(raw []byte) ([]byte, int) {
	const sentinel = "\n__HTTP_STATUS__="
	idx := bytes.LastIndex(raw, []byte(sentinel))
	if idx < 0 {
		return raw, 0
	}
	body := raw[:idx]
	tail := strings.TrimSpace(string(raw[idx+len(sentinel):]))
	var status int
	if _, err := fmt.Sscanf(tail, "%d", &status); err != nil {
		return raw, 0
	}
	return body, status
}
