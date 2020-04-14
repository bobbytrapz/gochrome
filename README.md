Work in progress.

Package for remote controlling with Chrome DevTools. Requires Google Chrome.

No Windows support at the moment.

## Install

```
go get github.com/bobbytrapz/gochrome
```

## Navigation (example.go)

```go
func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := log.New(os.Stderr, "chrome: ", log.LstdFlags|log.Lshortfile)
	gochrome.Log = logger.Printf

	browser := gochrome.NewBrowser()

	browser.Flags = append(browser.Flags,
		"--blink-settings=imagesEnabled=false")

  // start non-headless so we can see it
	tab, err := browser.StartFull(ctx, "", 44144)
	if err != nil {
		panic(err)
	}

	defer func() {
		browser.Wait()
	}()

	_, err = tab.PageNavigate("https://golang.org", "", "", "")
	if err != nil {
		panic(err)
	}

	newTab, err := browser.NewTab(ctx)
	if err != nil {
		panic(err)
	}

	url := "https://go.dev"
	_, err = newTab.PageNavigate(url, "", "", "")
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
