package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bobbytrapz/gochrome"
)

func main() {
	// context is needed to start the browser and open new tabs
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// build logger
	// by default, gochrome does not log
	logger := log.New(os.Stderr, "gochrome: ", log.LstdFlags|log.Lshortfile)
	gochrome.Log = logger.Printf

	// create new browser window
	browser := gochrome.NewBrowser()

	// start browser
	// we use StartFull so we can see what the browser is doing
	// normally, we would want to use Start to run chrome headless
	// ctx is passed to exec.CommandContext so cancel will exit the browser
	// userProfileDir is "" so a temporary directory will be made for the chrome user profile
	// the port is used to connect to the Chrome DevTools
	// we are given a *chrome.Tab, which is the first open tab
	tab, err := browser.StartFull(ctx, "", 44144)
	if err != nil {
		panic(err)
	}

	// wait for browser to close
	// this makes sure before main returns
	// our goroutines get a chance to finish up
	defer browser.Wait()

	_, err = tab.Goto("https://golang.org")
	if err != nil {
		panic(err)
	}

	go func() {
		tab.WaitForNetworkIdle(1 * time.Second)
		err := tab.Screenshot("examples/screenshot/example.png")
		if err != nil {
			panic(err)
		}
		cancel()
	}()

	// handle keyboard interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			// ctrl+c will shut down the browser
			signal.Stop(sig)
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
