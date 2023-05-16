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
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := log.New(os.Stderr, "gochrome: ", log.LstdFlags|log.Lshortfile)
	gochrome.Log = logger.Printf

	browser := gochrome.NewBrowser()
	browser.Flags = append(browser.Flags,
		"--blink-settings=imagesEnabled=false")

	_, err := browser.StartFull(ctx, gochrome.TemporaryUserProfileDirectory, gochrome.DefaultPort)
	if err != nil {
		panic(err)
	}
	defer browser.Wait()

	newTab, err := browser.NewTab(ctx)
	if err != nil {
		panic(err)
	}

	js := "document.title"
	title, err := newTab.Extract("https://go.dev", 1*time.Minute, 10*time.Second, js, func(v any) bool {
		title, ok := v.(string)
		if ok {
			return len(title) > 0
		}
		return false
	})

	log.Printf("got title: %q", title)

	// handle keyboard interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			// ctrl+c will shut down the browser
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
