package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var (
	browserMu sync.RWMutex
	browser   *rod.Browser
)

func (f *Fetcher) fetchHTMLBodyRod(ctx context.Context, url string) (*string, error) {
	timeout := f.BrowserTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	browserCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	browserMu.RLock()
	sharedBrowser := browser
	browserMu.RUnlock()
	if sharedBrowser == nil {
		return nil, fmt.Errorf("rod browser is not initialized")
	}

	page, err := sharedBrowser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("rod navigation failed: %w", err)
	}
	defer page.Close()
	page = page.Context(browserCtx)

	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("rod navigation failed: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("rod page load failed: %w", err)
	}
	body, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("rod body extraction failed: %w", err)
	}
	doctype, err := page.Eval(`() => {
		if (!document.doctype) {
			return "";
		}
		return new XMLSerializer().serializeToString(document.doctype);
	}`)
	if err != nil {
		return nil, fmt.Errorf("rod doctype extraction failed: %w", err)
	}
	if doctype.Value.String() != "" {
		body = doctype.Value.String() + "\n" + body
	}
	return &body, nil
}

func LaunchBrowser(chromePath string) error {
	browserMu.Lock()
	defer browserMu.Unlock()

	if browser != nil {
		return nil
	}

	browserLauncher := launcher.New().Headless(true).NoSandbox(true)
	if chromePath != "" {
		browserLauncher = browserLauncher.Bin(chromePath)
	}
	controlURL, err := browserLauncher.Launch()
	if err != nil {
		return fmt.Errorf("launch rod browser: %w", err)
	}

	browser = rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		browser = nil
		return fmt.Errorf("connect rod browser: %w", err)
	}
	return nil
}

func CloseBrowser() error {
	browserMu.Lock()
	defer browserMu.Unlock()

	if browser == nil {
		return nil
	}
	err := browser.Close()
	browser = nil
	return err
}
