package gochrome

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WaitForTabConnect decides how long we wait to connect to a tab
var WaitForTabConnect = 10 * time.Second

// Tab command channel
// For now, a single tab is expected to handle one command at a time
type Tab struct {
	send       chan []byte
	recv       chan []byte
	closed     chan struct{}
	connection tabConnectionInfo
	rw         sync.RWMutex
	nextReqID  int
}

// response from /json and /json/new
type tabConnectionInfo struct {
	Description          string `json:"description"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// WaitForLoad waits for a page to load up to given maximum
// pretty useless right now but we may improve it in the future
func (t *Tab) WaitForLoad(maxWait time.Duration) {
	<-time.After(maxWait)
}

func (b *Browser) connectTab(tci tabConnectionInfo) (*Tab, error) {
	conn, _, err := websocket.DefaultDialer.Dial(tci.WebSocketDebuggerURL, nil)
	if err != nil {
		err = fmt.Errorf("websocket.Dial: %w", err)
		return nil, err
	}

	tab := &Tab{
		send:       make(chan []byte),
		recv:       make(chan []byte),
		closed:     make(chan struct{}),
		connection: tci,
	}

	// response from chrome
	type resChrome struct {
		// call response
		Result json.RawMessage
		// event
		Method string `json:"method"`
		Params json.RawMessage
	}

	type evPageLifecycle struct {
		Name      string  `json:"name"`
		Timestamp float64 `json:"timestamp"`
		FrameID   string  `json:"frameId"`
		LoaderID  string  `json:"loaderId"`
	}

	// when Page.close causes a target to close we get this
	// so we treat it like an event
	type evInspectorDetached struct {
		Params struct {
			Reason string `json:"reason"`
		} `json:"params"`
	}

	// read
	// handle events
	b.wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			Log("got: %s", data)
			if err != nil {
				Log("closed: %s", err)
				return
			}
			var msg resChrome
			err = json.Unmarshal(data, &msg)
			if err != nil {
				Log("json: %s", err)
				return
			}
			switch msg.Method {
			case "Page.lifecycleEvent":
				var ev evPageLifecycle
				err := json.Unmarshal(msg.Params, &ev)
				if err != nil {
					Log("Page.lifecycleEvent: %s", err)
					return
				}
				Log("Page.lifecycleEvent: ev: %+v", ev)
			case "Inspector.detached":
				var ev evInspectorDetached
				err := json.Unmarshal(msg.Params, &ev)
				if err != nil {
					Log("Inspector.detached: %s", err)
					return
				}
				Log("Inspector.detached: ev: %+v", ev)
				close(tab.closed)
			default:
				select {
				case tab.recv <- msg.Result:
					Log("result: %s", msg.Result)
				case <-time.After(500 * time.Millisecond):
					Log("result: timeout")
				}
			}
		}
	}()

	// handle writing/closing
	go func() {
		defer func() {
			conn.Close()
			b.wg.Done()
		}()
		for {
			select {
			case <-tab.closed:
				Log("tab was closed")
				return
			case <-done:
				return
			case <-b.exit:
				return
			case msg := <-tab.send:
				Log("send: %s", msg)
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	return tab, nil
}

// SendCommand builds a command and sends it
// { "id": 0, "method": "Page.navigate", params: {"url": "..."} }
func (t *Tab) SendCommand(args map[string]interface{}) {
	// build command
	id := t.addReq()
	args["id"] = id
	data, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}

	// send command
	select {
	case t.send <- data:
		// Log("send: %s", data)
	case <-time.After(500 * time.Millisecond):
		Log("send: timeout")
	}
}

// ID gives the tab id
func (t *Tab) ID() string {
	return t.connection.ID
}

func (t *Tab) addReq() int {
	t.rw.Lock()
	defer t.rw.Unlock()
	t.nextReqID++
	return t.nextReqID
}
