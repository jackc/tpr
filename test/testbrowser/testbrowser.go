package testbrowser

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/sync/semaphore"
)

type Manager struct {
	baseBrowser *rod.Browser
	sem         *semaphore.Weighted

	Timeout time.Duration
}

type ManagerConfig struct {
	MaxConcurrentTests int64
	Timeout            time.Duration
}

func NewManager(config ManagerConfig) (*Manager, error) {
	browser := rod.New()
	err := browser.Connect()
	if err != nil {
		return nil, fmt.Errorf("connect to browser failed: %w", err)
	}

	maxConcurrentTests := int64(1)
	if config.MaxConcurrentTests != 0 {
		maxConcurrentTests = config.MaxConcurrentTests
	} else if n, err := strconv.ParseInt(os.Getenv("TESTBROWSER_MAX_CONCURRENT_TESTS"), 10, 32); err == nil {
		maxConcurrentTests = n
	}
	if maxConcurrentTests <= 0 {
		return nil, fmt.Errorf("invalid MaxConcurrentTests: %v", maxConcurrentTests)
	}

	timeout := 2 * time.Second
	if config.Timeout != 0 {
		timeout = config.Timeout
	}

	manager := &Manager{
		baseBrowser: browser,
		sem:         semaphore.NewWeighted(maxConcurrentTests),
		Timeout:     timeout,
	}

	return manager, nil
}

// Acquire returns a TestBrowser. Resources are automatically cleaned up at the end of the test.
func (m *Manager) Acquire(t testing.TB) *Browser {
	err := m.sem.Acquire(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { m.sem.Release(1) })

	browser := m.baseBrowser.MustIncognito()
	t.Cleanup(browser.MustClose)

	testBrowser := &Browser{
		t:       t,
		Browser: browser,
		Timeout: m.Timeout,
	}

	return testBrowser
}

type Browser struct {
	t testing.TB
	*rod.Browser
	Timeout time.Duration
}

func (b *Browser) Page() *Page {
	page, err := b.Browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		b.t.Fatal(err)
	}

	return &Page{
		t:       b.t,
		Page:    page,
		Timeout: b.Timeout,
	}
}

type Page struct {
	t testing.TB
	*rod.Page
	Timeout time.Duration
}

func (p *Page) ClickOn(jsRegex string) {
	p.t.Helper()

	page := p.Page.Timeout(p.Timeout)

	el, err := page.ElementR(`a, button, input[type="submit"]`, jsRegex)
	if err != nil {
		p.t.Fatalf("failed to find clickable element: %s", jsRegex)
	}

	err = el.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		p.t.Fatalf("failed to click element")
	}
}

func (p *Page) FillIn(labelOrSelector string, content string) {
	p.t.Helper()

	page := p.Page.Timeout(p.Timeout)
	var inputEl *rod.Element
	_, err := page.Race().ElementR("label", labelOrSelector).Handle(func(e *rod.Element) error {
		forAttr, err := e.Attribute("for")
		if err != nil {
			return fmt.Errorf("unable to read label's for attribute: %w", err)
		}

		inputEl, err = page.Element("#" + *forAttr)
		if err != nil {
			return fmt.Errorf("unable to find element from label's for attribute: %q %w", *forAttr, err)
		}

		return nil
	}).Element(labelOrSelector).Handle(func(e *rod.Element) error {
		inputEl = e
		return nil
	}).Do()
	if err != nil {
		p.t.Fatalf("failed to find label or selector for %q: %v", labelOrSelector, err)
	}

	err = inputEl.SelectAllText()
	if err != nil {
		p.t.Fatalf("failed to select all text for %q", labelOrSelector)
	}

	err = inputEl.Input(content)
	if err != nil {
		p.t.Fatalf("failed to input text for %q", labelOrSelector)
	}
}

func (p *Page) AcceptDialog(fn func()) {
	p.t.Helper()

	// The rod documentation for HandleDialog shows fn() as the goroutine instead of the HandleDialog. This is reversed
	// here so any test failures triggered by fn are called from the original test goroutine.
	errChan := make(chan error)
	go func() {
		wait, handle := p.Page.HandleDialog()
		wait()
		err := handle(&proto.PageHandleJavaScriptDialog{Accept: true})
		errChan <- err
	}()

	fn()

	err := <-errChan
	if err != nil {
		p.t.Fatalf("failed to accept dialog: %v", err)
	}
}

func (p *Page) HasContent(selector, jsRegex string) {
	p.t.Helper()

	page := p.Page.Timeout(p.Timeout)
	_, err := page.ElementR(selector, jsRegex)
	if err != nil {
		p.t.Fatalf("failed to find element by selector %q with content matching %q", selector, jsRegex)
	}
}
