package gochrome

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"
)

// SetUserAgent override
func (t *Tab) SetUserAgent(ua string) {
	t.NetworkSetUserAgentOverride(ua, "", "")
}

// SetRequestHeaders override
func (t *Tab) SetRequestHeaders(ua string, lang string, platform string) {
	t.NetworkSetUserAgentOverride(ua, lang, platform)
}

// GetResponseBody for a request
func (t *Tab) GetResponseBody(id NetworkRequestId) (string, error) {
	res, err := t.NetworkGetResponseBody(id)
	if err != nil {
		return "", fmt.Errorf("Tab.GetResponseBody: %v", err)
	}
	var body string
	if res.Base64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(res.Body)
		if err != nil {
			return "", fmt.Errorf("Tab.GetResponseBody: %v", err)
		}
		body = string(decoded)
	} else {
		body = res.Body
	}
	return body, nil
}

// HTTPResource is a resource the browser has fetched
type HTTPResource struct {
	Type    NetworkResourceType
	Headers NetworkResponse
	Body    string
}

// OnResource passes the http response headers and body
// Document, Stylesheet, Image, Media, Font,
// Script, TextTrack, XHR, Fetch, EventSource,
// WebSocket, Manifest, SignedExchange, Ping,
// CSPViolationReport, Other
func (t *Tab) OnResource(onResource func(res HTTPResource), types ...NetworkResourceType) error {
	if len(types) == 0 {
		types = []NetworkResourceType{
			"Document", "Stylesheet", "Image", "Media", "Font",
			"Script", "TextTrack", "XHR", "Fetch", "EventSource",
			"WebSocket", "Manifest", "SignedExchange", "Ping",
			"CSPViolationReport", "Other",
		}
	}
	_, err := t.NetworkEnable(0, 0, 0)
	if err != nil {
		return nil
	}
	var m sync.Mutex
	resources := make(map[NetworkRequestId]HTTPResource)

	add := func(id NetworkRequestId, res HTTPResource) {
		m.Lock()
		defer m.Unlock()
		resources[id] = res
	}
	del := func(id NetworkRequestId) {
		m.Lock()
		defer m.Unlock()
		delete(resources, id)
	}

	t.Events.OnNetworkResponseReceived = func(ev NetworkResponseReceivedEvent) {
		// fmt.Printf("Response: %s (%s)\n", ev.RequestId, ev.Response["url"])
		for _, tt := range types {
			if ev.Type == tt {
				add(ev.RequestId, HTTPResource{
					Type:    ev.Type,
					Headers: ev.Response,
				})
			}
		}
	}

	t.Events.OnNetworkLoadingFinished = func(ev NetworkLoadingFinishedEvent) {
		// fmt.Printf("Loaded: %s\n", ev.RequestId)
		if r, ok := resources[ev.RequestId]; ok {
			del(ev.RequestId)
			body, err := t.GetResponseBody(ev.RequestId)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Tab.OnResource: %v\n", err)
			}
			onResource(HTTPResource{
				Type:    r.Type,
				Headers: r.Headers,
				Body:    body,
			})
		}
	}

	return nil
}

// WaitForNetworkIdle blocks until network is idle for d seconds
// relies on network events so
// Tab.NetworkEnable should be called first
func (t *Tab) WaitForNetworkIdle(d time.Duration) {
	idle := time.NewTimer(d)
	t.Events.OnNetworkDataReceived = func(ev NetworkDataReceivedEvent) {
		if idle.Stop() {
			idle.Reset(d)
		}
	}
	select {
	case <-idle.C:
		t.Events.OnNetworkDataReceived = nil
	}
}
