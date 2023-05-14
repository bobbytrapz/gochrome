package gochrome

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	TemporaryUserProfileDirectory = ""
	DefaultPort                   = 44144
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
	var err error
	if userProfileDir == TemporaryUserProfileDirectory {
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

	if port == 0 {
		port = DefaultPort
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
		"--new-window",
		"--window-size=1280,1696",
		fmt.Sprintf("--user-data-dir=%s", userProfileDir),
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"about:blank",
	)

	switch runtime.GOOS {
	case "darwin":
		path := "/Applications/Google Chrome.app"
		if s, err := os.Stat(path); err == nil && s.IsDir() {
			args := []string{
				"--new", "--fresh", "--wait-apps",
				"-a", path, "--args",
			}
			args = append(args, opts...)
			b.cmd = exec.CommandContext(ctx, "open", args...)
		} else {
			return nil, fmt.Errorf("we checked for chrome at %q and got an error: %w", path, err)
		}
	case "linux":
		names := []string{
			"chromium-browser",
			"chromium",
			"google-chrome",
		}
		var app string
		for _, name := range names {
			if _, err := exec.LookPath(name); err == nil {
				app = name
				break
			}
		}
		b.cmd = exec.CommandContext(ctx, app, opts...)
	case "windows":
		// todo: find chrome on windows; modify flags for windows
		return nil, fmt.Errorf("gochrome does not support Windows.")
	}

	if err = b.cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start chrome: %w", err)
	}
	Log("%s (%d) profile=%s", b.cmd.Path, b.cmd.Process.Pid, userProfileDir)

	b.monitorBrowserProcess()

	// handle exit
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer func() {
			if tmpDir != "" {
				// remove temporary directory
				Log("remove: %s", tmpDir)
				os.RemoveAll(tmpDir)
			}
		}()
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

	go func() {
		for {
			select {
			case <-ctx.Done():
				Log("close browser")
				err := b.Close()
				if err != nil {
					Log("while closing browser: %s", err)
				}
			}
		}
	}()

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
			res, err := b.performRequest(ctx, http.MethodGet, u.String())
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
	res, err := b.http(ctx, http.MethodGet, "/json")
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
