package gochrome

import "context"

// NewTabPool create a new pool of N tabs
func (b *Browser) NewTabPool(ctx context.Context, N int) error {
	b.released = make(chan *Tab, N)
	for i := 0; i < N; i++ {
		tab, err := b.NewTab(ctx)
		if err != nil {
			return err
		}
		b.released <- tab
	}
	return nil
}

// GrabTab from pool
// blocks if no tabs are available
func (b *Browser) GrabTab() *Tab {
	tab := <-b.released
	return tab
}

// ReleaseTab to pool
func (b *Browser) ReleaseTab(tab *Tab) {
	if tab == nil {
		panic("browser.ReleaseTab: cannot release nil Tab")
	}
	b.released <- tab
}
