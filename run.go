package gochrome

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// WaitForOpen decides how long we wait for chrome to open
var WaitForOpen = 20 * time.Second

// Wait for chrome to close
func (b *Browser) Wait() {
	b.wg.Wait()
}

// Start finds chrome and runs it headless
func (b *Browser) Start(ctx context.Context, userProfileDir string, port int) (*Tab, error) {
	return b.start(ctx, userProfileDir, port, true)
}

// StartFull finds chrome and runs it but full (non-headless)
func (b *Browser) StartFull(ctx context.Context, userProfileDir string, port int) (*Tab, error) {
	return b.start(ctx, userProfileDir, port, false)
}

func (b *Browser) start(ctx context.Context, userProfileDir string, port int, shouldHeadless bool) (*Tab, error) {
	var tmpDir string
	var app string
	var err error
	if userProfileDir == "" {
		tmpDir, err = ioutil.TempDir("", "gochrome-chrome-profile")
		if err != nil {
			panic(err)
		}
		userProfileDir = tmpDir
	} else if userProfileDir[:2] == "~/" {
		var home string
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("os.UserHomeDir: %w", err)
		}
		userProfileDir = filepath.Join(home, userProfileDir[2:])
	}
	switch runtime.GOOS {
	case "darwin":
		path := "/Applications/Google Chrome.app"
		if s, err := os.Stat(path); err == nil && s.IsDir() {
			app = fmt.Sprintf("open %s --args", path)
		}
	case "linux":
		names := []string{
			"chromium-browser",
			"chromium",
			"google-chrome",
		}
		for _, name := range names {
			if _, err := exec.LookPath(name); err == nil {
				app = name
				break
			}
		}
	case "windows":
		// todo: find chrome on windows
	}

	var opts []string

	// optional
	if shouldHeadless {
		opts = []string{"--headless", "--hide-scrollbars", "--mute-audio"}
	}

	if runtime.GOOS == "windows" {
		opts = append(opts,
			"--disable-gpu",
		)
	}

	// defaults
	opts = append(opts, b.Flags...)
	opts = append(opts,
		"--window-size=1280,1696",
		fmt.Sprintf("--user-data-dir=%s", userProfileDir),
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"about:blank",
	)

	if app == "" {
		return nil, fmt.Errorf("could not find chrome")
	}
	b.cmd = exec.CommandContext(ctx, app, opts...)

	if err = b.cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start chrome: %w", err)
	}
	Log("%s (%d) profile=%s", b.cmd.Path, b.cmd.Process.Pid, userProfileDir)

	// monitor process
	b.wg.Add(1)
	b.exit = make(chan struct{}, 1)
	go func() {
		defer b.wg.Done()
		b.cmd.Wait()
		close(b.exit)
		if tmpDir != "" {
			Log("remove: %s", tmpDir)
			os.RemoveAll(tmpDir)
		}
	}()

	// handle exit
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		select {
		case <-ctx.Done():
			Log("cancel: %s", ctx.Err())
			return
		case <-b.exit:
			Log("exited")
			return
		}
	}()

	// connect to running chrome process
	err = b.connect(ctx, fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, err
	}

	// connect to first tab
	tab, err := b.connectFirstTab(ctx)
	if err != nil {
		return nil, err
	}

	return tab, nil
}

func (b *Browser) connect(ctx context.Context, addr string) error {
	b.addr = addr
	u := url.URL{Scheme: "http", Host: b.addr, Path: "/"}

	// wait for connection
	Log("wait for connection to browser...")
	timeout := time.After(WaitForOpen)
	for {
		select {
		case err := <-ctx.Done():
			return fmt.Errorf("cancel: %s", err)
		case <-timeout:
			Log("timeout")
			return fmt.Errorf("timeout")
		default:
			res, err := b.fetch(ctx, u.String())
			if err == nil {
				res.Body.Close()
				goto connected
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
connected:
	Log("connected: %s", b.addr)

	return nil
}

func (b *Browser) connectFirstTab(ctx context.Context) (*Tab, error) {
	res, err := b.get(ctx, "/json")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response []tabConnectionInfo
	err = json.NewDecoder(res.Body).Decode(&response)

	// return the first page we find as the first tab
	var tab *Tab
	for _, tci := range response {
		if tci.Type == "page" {
			tab, err = b.addTab(tci)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	return tab, err
}
