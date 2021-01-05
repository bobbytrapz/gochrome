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
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := log.New(os.Stderr, "gochrome: ", log.LstdFlags|log.Lshortfile)
	gochrome.Log = logger.Printf

	browser := gochrome.NewBrowser()

	browser.Flags = append(browser.Flags,
		"--blink-settings=imagesEnabled=false")

	tab, err := browser.Start(ctx, "", 44144)
	if err != nil {
		panic(err)
	}

	defer browser.Wait()

	type netResponse struct {
		Response gochrome.NetworkResponse
	}

	tab.OnResource(func(res gochrome.HTTPResource) {
		// print some information about each document
		// snip the body to 100 chars
		if res.Type == "Document" {
			end := len(res.Body)
			if end > 100 {
				end = 100
			}
			fmt.Printf("%s [Document]\n%s\n", res.Response["url"], res.Body[:end])
		}
	}, "Document")
	tab.SetUserAgent("Go/gochrome-test")

	tab.Goto("http://golang.org/")

	go func() {
		tab.WaitForNetworkIdle(3 * time.Second)
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
