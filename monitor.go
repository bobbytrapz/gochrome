package gochrome

import (
	"runtime"
)

// handles exit for the browser process
// should be called after the browser process begins
func (b *Browser) monitorBrowserProcess() {
	b.exit = make(chan struct{}, 1)

	switch runtime.GOOS {
	case "darwin":
	case "linux":
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.cmd.Wait()
			close(b.exit)
		}()
	case "windows":
	}
}
