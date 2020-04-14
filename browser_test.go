package gochrome_test

import (
	"context"
	"testing"

	"github.com/bobbytrapz/gochrome"
)

var debugPort = 40506

func TestBrowserDoesOpen(t *testing.T) {
	browser := gochrome.NewBrowser()

	// build a context so we can cancel
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// start browser
	// profileDir is "" so a temporary directory will be made
	// we ignore the *chrome.Tab it returns
	_, err := browser.Start(ctx, "", debugPort)
	if err != nil {
		t.Errorf("browser.Start: %v", err)
	}

	// wait for browser to close
	// if we do not call this the browser may not shutdown cleanly
	defer browser.Wait()

	// use cancel
	// browser should close all the tabs and close the browser
	cancel()
}
