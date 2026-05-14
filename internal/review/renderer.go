package review

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// Realistic Chrome on macOS UA — Cloudflare and friends look at this before
// deciding whether to challenge the request.
const browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// renderSettle is how long we wait after the load event before snapshotting
// the DOM. Most Shopify themes finish rendering reviews/Klaviyo embeds within
// 2-3 seconds; longer is wasted time × N pages.
const renderSettle = 3 * time.Second

// renderer drives a single headless Chrome instance for the duration of a
// crawl. One Chrome process is reused across all page renders so we don't
// pay browser-launch latency per URL.
type renderer struct {
	allocCtx    context.Context
	browserCtx  context.Context
	cancelAlloc context.CancelFunc
	cancelCtx   context.CancelFunc
	dataDir     string
	settle      time.Duration
}

// newRenderer launches headless Chrome with an isolated user-data-dir so
// crawls don't collide with the user's own Chrome profile or each other.
// The caller must call close() when done.
func newRenderer(parent context.Context, settle time.Duration) (*renderer, error) {
	dataDir, err := os.MkdirTemp("", "sharma-bot-chrome-*")
	if err != nil {
		return nil, fmt.Errorf("renderer: mkdir tmp: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(browserUA),
		chromedp.UserDataDir(dataDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	browserCtx, cancelCtx := chromedp.NewContext(allocCtx)

	// Force the browser to actually start so we fail fast if Chrome isn't
	// installed or can't launch — without this, chromedp lazily starts on
	// first Run() call and we'd see the error mid-crawl instead.
	if err := chromedp.Run(browserCtx); err != nil {
		cancelCtx()
		cancelAlloc()
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("renderer: launch chrome: %w", err)
	}

	return &renderer{
		allocCtx:    allocCtx,
		browserCtx:  browserCtx,
		cancelAlloc: cancelAlloc,
		cancelCtx:   cancelCtx,
		dataDir:     dataDir,
		settle:      settle,
	}, nil
}

// authenticate posts the Shopify storefront-password form by filling and
// submitting it inside the browser. Cookies persist in the user-data-dir for
// subsequent render() calls.
func (r *renderer) authenticate(ctx context.Context, baseURL, password string) error {
	postURL := strings.TrimRight(baseURL, "/") + "/password"
	c, cancel := mergeContexts(r.browserCtx, ctx)
	defer cancel()
	return chromedp.Run(c,
		chromedp.Navigate(postURL),
		chromedp.WaitVisible(`input[name="password"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, password, chromedp.ByQuery),
		chromedp.Submit(`input[name="password"]`, chromedp.ByQuery),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.Sleep(r.settle),
	)
}

// render navigates to rawURL, waits for the page to settle, scrolls top-to-
// bottom to trigger IntersectionObserver-based lazy loaders (review widgets,
// "as seen in" carousels, lazy product grids), then returns outerHTML.
func (r *renderer) render(ctx context.Context, rawURL string) ([]byte, error) {
	var html string
	c, cancel := mergeContexts(r.browserCtx, ctx)
	defer cancel()
	err := chromedp.Run(c,
		chromedp.Navigate(rawURL),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.Sleep(r.settle),
		chromedp.ActionFunc(scrollPage),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return nil, err
	}
	return []byte(html), nil
}

// scrollPage walks the viewport from top to bottom in steps, pausing between
// each so IntersectionObserver-driven widgets (Ryviu, Yotpo, lazy carousels)
// have time to fetch and render their content.
func scrollPage(ctx context.Context) error {
	steps := []float64{0.25, 0.5, 0.75, 1.0}
	for _, frac := range steps {
		js := fmt.Sprintf(`window.scrollTo(0, document.body.scrollHeight * %f)`, frac)
		if err := chromedp.Evaluate(js, nil).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(800 * time.Millisecond).Do(ctx); err != nil {
			return err
		}
	}
	if err := chromedp.Evaluate(`window.scrollTo(0, 0)`, nil).Do(ctx); err != nil {
		return err
	}
	return chromedp.Sleep(500 * time.Millisecond).Do(ctx)
}

func (r *renderer) close() {
	r.cancelCtx()
	r.cancelAlloc()
	_ = os.RemoveAll(r.dataDir)
}

// mergeContexts returns a context that cancels when either parent or deadline
// cancels. Used to apply a per-call timeout on top of the long-lived browser
// context without losing the browser context's chromedp metadata.
func mergeContexts(browser, deadline context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(browser)
	go func() {
		select {
		case <-deadline.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
