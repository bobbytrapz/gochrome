package gochrome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
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
	// open tabs
	tabs []*Tab
	// released tabs
	released chan *Tab
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
		UserAgent: "Go/gochrome",
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

func (b *Browser) fetch(ctx context.Context, link string) (*http.Response, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		panic("invalid url" + link)
	}

	req, err := b.newRequest(ctx, u.Host, "GET", link)
	if err != nil {
		return nil, err
	}

	res, err := b.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do: %w", err)
	}

	return res, nil
}

// get a response from the browser
func (b *Browser) get(ctx context.Context, path string) (res *http.Response, err error) {
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
			res, err = b.fetch(ctx, u.String())
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
	res, err := b.get(ctx, "/json/new")
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}
	defer res.Body.Close()

	var tci tabConnectionInfo
	err = json.NewDecoder(res.Body).Decode(&tci)
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
	b.tabs = append(b.tabs, tab)

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

// EachTab performs a function on each open tab
// this may not work if tabs are closed
// may want to look into a solution here
// users probably will not want to close tabs at all
// except when exiting the browser
func (b *Browser) EachTab(tabfn func(index int, tab *Tab)) {
	for n, t := range b.tabs {
		tabfn(n, t)
	}
}
