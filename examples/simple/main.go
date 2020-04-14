package main

import (
	"context"
	"log"
	"os"
	"os/signal"

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

	// add more flags
	browser.Flags = append(browser.Flags,
		"--blink-settings=imagesEnabled=false")

	// start browser
	// we use StartFull so we can see what the browser is doing
	// normally, we would want to use Start to run chrome headless
	// ctx is passed to exec.CommandContext so cancel will exit the browser
	// profileDir is "" so a temporary directory will be made for the chrome user profile
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

	// navigate with our first tab
	_, err = tab.PageNavigate("https://golang.org", "", "", "")
	if err != nil {
		panic(err)
	}

	// make a new tab
	// tabs work by communicating with the browser over websockets
	// there a few goroutines started in the package to do this
	newTab, err := browser.NewTab(ctx)
	if err != nil {
		panic(err)
	}

	// navigate with that new tab
	// afterwards, there should be two pages open in the browser
	url := "https://go.dev"
	_, err = newTab.Goto(url)
	if err != nil {
		panic(err)
	}

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
