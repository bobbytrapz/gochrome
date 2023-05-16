package gochrome

import (
	"errors"
	"time"
)

// ExtractCheck should return true if the extraction was ok.
type ExtractCheck func(any) bool

var (
	ErrExtractTimeout = errors.New("timeout")
)

// Extract data from a page.
// url: page url, leave empty to stay on the same page
// timeout: how long before we give up
// delay: how long between attempts
// js: code to evaluate on the page
// check: if true, extraction is a success
// Returns the value extracted from the js code.
func (t *Tab) Extract(url string, timeout time.Duration, delay time.Duration, js string, check ExtractCheck) (any, error) {
	if len(url) > 0 {
		_, err := t.Goto(url)
		if err != nil {
			return nil, err
		}
		t.WaitForNetworkIdle(1 * time.Second)
	}

	to := time.NewTimer(timeout)
	for {
		select {
		case <-to.C:
			return nil, ErrExtractTimeout
		default:
			res, err := t.Evaluate(js)
			if err != nil {
				return nil, err
			}

			v := res.Result["value"]
			if check(v) {
				Log("Tab.Extract: check ok: %+v", v)
				return v, nil
			} else {
				// try again after the given delay
				Log("Tab.Extract: check failed. try again after %s", delay)
				<-time.After(delay)
			}
		}
	}
}
