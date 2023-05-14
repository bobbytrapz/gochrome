package gochrome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Browser is a single running chrome browser
type Browser struct {
	// flags passed into chrome
	Flags []string
	// useragent string passed when using HTTPClient
	UserAgent string
	// closed when browser exits
	exit chan struct{}
	// makes sure browser/tabs close cleanly
	wg sync.WaitGroup
	// chrome process
	cmd *exec.Cmd
	// client to communicate with chrome
	// also used for making other requests
	HTTPClient *http.Client
	// chrome browser address
	addr string
	// mutex so we cannot not add more than one tab at a time
	newTab sync.Mutex
}

// NewBrowser creates a new chrome browser
// does not start a chrome process
func NewBrowser() *Browser {
	return &Browser{
		UserAgent: "",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		Flags: []string{
			// note: I don't know what most of this does yet
			// I lifted it from puppeteer
			"--disable-background-networking",
			"--enable-features=NetworkService,NetworkServiceInProcess",
			"--disable-background-timer-throttling",
			"--disable-backgrounding-occluded-windows",
			"--disable-breakpad",
			"--disable-client-side-phishing-detection",
			"--disable-default-apps",
			"--disable-dev-shm-usage",
			"--disable-extensions",
			"--disable-features=site-per-process,TranslateUI,BlinkGenPropertyTrees",
			"--disable-hang-monitor",
			"--disable-ipc-flooding-protection",
			"--disable-popup-blocking",
			"--disable-prompt-on-repost",
			"--disable-renderer-backgrounding",
			"--disable-sync",
			"--force-color-profile=srgb",
			"--metrics-recording-only",
			"--no-first-run",
			"--safebrowsing-disable-auto-update",
			"--enable-automation",
			"--password-store=basic",
			"--use-mock-keychain",
		},
	}
}

// NewBrowserWithFlags creates a new chrome Browser
// and sets given flags
func NewBrowserWithFlags(flags []string) *Browser {
	return &Browser{
		UserAgent: "",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		Flags: flags,
	}
}

func (b *Browser) newRequest(ctx context.Context, host string, method string, url string) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, nil)
	if err != nil {
		err = fmt.Errorf("http.NewRequest: %w", err)
		return
	}

	// headers
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("DNT", "1")
	req.Header.Add("Host", host)
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("User-Agent", b.UserAgent)

	req = req.WithContext(ctx)

	return
}

func (b *Browser) performRequest(ctx context.Context, method string, link string) (*http.Response, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		panic("invalid url" + link)
	}

	req, err := b.newRequest(ctx, u.Host, method, link)
	if err != nil {
		return nil, err
	}

	res, err := b.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do: %w", err)
	}

	return res, nil
}

// use the browser http-based api
func (b *Browser) http(ctx context.Context, method string, path string) (res *http.Response, err error) {
	u := url.URL{Scheme: "http", Host: b.addr, Path: path}

	timeout := time.After(WaitForTabConnect)
	for {
		select {
		case err := <-ctx.Done():
			return nil, fmt.Errorf("cancel: %s", err)
		case <-timeout:
			Log("timeout")
			return nil, fmt.Errorf("timeout")
		default:
			res, err = b.performRequest(ctx, method, u.String())
			if err == nil {
				goto ok
			} else {
				Log("%s", err)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
ok:

	return
}

// NewTab opens a new tab
func (b *Browser) NewTab(ctx context.Context) (*Tab, error) {
	b.newTab.Lock()
	defer b.newTab.Unlock()

	res, err := b.http(ctx, http.MethodPut, "/json/new")
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	var tci tabConnectionInfo
	err = json.NewDecoder(strings.NewReader(string(data))).Decode(&tci)
	if err != nil {
		return nil, fmt.Errorf("json.NewDecoder: %w", err)
	}

	tab, err := b.addTab(tci)
	if err != nil {
		return nil, err
	}

	return tab, nil
}

func (b *Browser) addTab(tci tabConnectionInfo) (*Tab, error) {
	tab, err := b.connectTab(tci)
	if err != nil {
		return nil, fmt.Errorf("could not connect tab: %w", err)
	}

	Log("adding tab:\n%+v", tci)

	if b.UserAgent != "" {
		tab.SetUserAgent(b.UserAgent)
	}

	return tab, nil
}

// PID returns the chrome process id
func (b *Browser) PID() int {
	if b.cmd == nil {
		panic("browser is not open")
	}
	return b.cmd.Process.Pid
}

// Close the browser.
func (b *Browser) Close() error {
	tab, err := b.NewTab(context.Background())
	if err != nil {
		return fmt.Errorf("Browser.NewTab: %w", err)
	}
	_, err = tab.BrowserClose()
	if err != nil {
		return fmt.Errorf("Tab.BrowserClose: %w", err)
	}
	return nil
}
