package gochrome

import (
	"context"
	"sync"
)

// TabPool is a collection of tabs
type TabPool struct {
	// open tabs
	tabs []*Tab
	// released tabs
	released chan *Tab
	wg       *sync.WaitGroup
}

// NewTabPool create a new pool of N tabs
func (b *Browser) NewTabPool(ctx context.Context, N int) (tabPool *TabPool, err error) {
	tabPool = &TabPool{
		tabs:     make([]*Tab, N),
		released: make(chan *Tab, N),
		wg:       &sync.WaitGroup{},
	}
	for i := 0; i < N; i++ {
		tab, err := b.NewTab(ctx)
		if err != nil {
			return tabPool, err
		}
		tabPool.tabs[i] = tab
		tabPool.released <- tab
	}
	return
}

// Grab from pool
// blocks if no tabs are available
func (tp *TabPool) Grab() *Tab {
	tp.wg.Add(1)
	tab := <-tp.released
	return tab
}

// Release to pool
// you should only release tabs that are members of the pool to avoid confusion
// you can release nil to cancel a Grab request
func (tp *TabPool) Release(tab *Tab) {
	tp.wg.Done()
	if tab != nil {
		tp.released <- tab
	}
}

// Wait blocks until all Grab requests are met and the tabs have been released
func (tp *TabPool) Wait() {
	tp.wg.Wait()
}

// Close all tabs in a pool
func (tp *TabPool) Close() {
	for _, tab := range tp.tabs {
		_, err := tab.PageClose()
		if err != nil {
			Log("TabPool.Close: %s", err)
		}
	}
	close(tp.released)
}
