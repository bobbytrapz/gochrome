Work in progress.

Package for remote controlling with Chrome DevTools. Requires Google Chrome.

No Windows support at the moment.

## Install

```
go get github.com/bobbytrapz/gochrome
```

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/bobbytrapz/gochrome"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	browser := gochrome.NewBrowser()

	browser.Flags = append(browser.Flags,
		"--blink-settings=imagesEnabled=false")

	tab, err := browser.StartFull(ctx, "", 44144)
	if err != nil {
		panic(err)
	}

	defer browser.Wait()

	_, err = tab.PageNavigate("https://golang.org", "", "", "", "")
	if err != nil {
		panic(err)
	}

	newTab, err := browser.NewTab(ctx)
	if err != nil {
		panic(err)
	}

	url := "https://go.dev"
	_, err = newTab.Goto(url)
	if err != nil {
		panic(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:			
			signal.Stop(sig)
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
```


Check out more [examples](examples)
