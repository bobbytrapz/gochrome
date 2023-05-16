package gochrome

import (
	"context"
	"testing"
	"time"
)

func TestExtract(t *testing.T) {
	t.Run("simple extract", func(t *testing.T) {
		// build a context so we can cancel
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		browser := NewBrowser()

		tab, err := browser.Start(ctx, TemporaryUserProfileDirectory, DefaultPort)
		if err != nil {
			t.Fatal(err)
		}
		defer browser.Wait()

		js := "document.title"

		title, err := tab.Extract("https://go.dev", 1*time.Minute, 10*time.Second, js, func(v any) bool {
			title, ok := v.(string)
			if ok {
				return len(title) > 0
			}
			return false
		})

		if err != nil {
			t.Fatal(err)
		}

		t.Logf("got title: %q", title)

		cancel()
	})
}
