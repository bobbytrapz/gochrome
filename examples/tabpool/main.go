package main

import (
	"context"
	"fmt"
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

	// add more flags
	browser.Flags = append(browser.Flags,
		"--blink-settings=imagesEnabled=false")

	// start browser
	// we use StartFull so we can see what the browser is doing
	// normally, we would want to use Start to run chrome headless
	// ctx is passed to exec.CommandContext so cancel will exit the browser
	// userProfileDir is "" so a temporary directory will be made for the chrome user profile
	// the port is used to connect to the Chrome DevTools
	// we are given a *chrome.Tab, which is the first open tab
	_, err := browser.StartFull(ctx, "", 44144)
	if err != nil {
		panic(err)
	}

	// wait for browser to close
	// this makes sure before main returns
	// our goroutines get a chance to finish up
	defer browser.Wait()

	// create tab pool
	// these are tabs we can reuse by grabbing and releasing them
	N := 3
	tabPool, err := browser.NewTabPool(ctx, N)
	if err != nil {
		panic(err)
	}

	// have each tab visit a page
	for i := 0; i < N; i++ {
		tabNumber := i

		// grab a free tab
		// this will block if no tabs are free
		tab := tabPool.Grab()

		go func() {
			defer tabPool.Release(tab)

			// use the tab
			_, err = tab.Goto("https://go.dev")
			tab.WaitForNetworkIdle(5 * time.Second)
			fmt.Printf("Tab %d done\n", tabNumber)
		}()
	}

	// wait for tabs to be released
	tabPool.Wait()

	fmt.Printf("Closing %d tabs\n", N)
	tabPool.Close()

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
