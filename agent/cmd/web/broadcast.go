// cmd/web/broadcast.go — SSE client registry and broadcast.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// SSEClient represents one SSE connection.
type SSEClient struct {
	threadID string
	ch       chan []byte
	closed   bool
}

var sseClients = sync.Map{} // threadID → map[*SSEClient]bool

// registerSSEClient adds a new SSE client for a thread.
func registerSSEClient(threadID string, c *SSEClient) {
	v, _ := sseClients.LoadOrStore(threadID, &sync.Map{})
	m := v.(*sync.Map)
	m.Store(c, true)
}

// unregisterSSEClient removes an SSE client.
func unregisterSSEClient(threadID string, c *SSEClient) {
	v, _ := sseClients.Load(threadID)
	if m, ok := v.(*sync.Map); ok {
		m.Delete(c)
	}
}

// broadcastSSE sends an SSE event to all clients for a thread.
func broadcastSSE(threadID, event string, data interface{}) {
	v, _ := sseClients.Load(threadID)
	if m, ok := v.(*sync.Map); ok {
		payload, _ := json.Marshal(data)
		msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, payload)
		m.Range(func(k, _ any) bool {
			if c, ok := k.(*SSEClient); ok && !c.closed {
				select {
				case c.ch <- []byte(msg):
				default:
				}
			}
			return true
		})
	}
}

// sseHandler handles SSE connection requests.
func sseHandler(w http.ResponseWriter, r *http.Request) {
	threadID := strings.TrimPrefix(r.URL.Path, "/api/sse/")
	if threadID == "" {
		http.Error(w, "missing threadId", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 64)
	c := &SSEClient{threadID: threadID, ch: ch}
	registerSSEClient(threadID, c)
	defer unregisterSSEClient(threadID, c)

	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", mustMarshal(map[string]string{"threadId": threadID, "status": "connected"}))
	flusher.Flush()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			io.WriteString(w, string(msg))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
