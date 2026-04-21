// cmd/web/broadcast.go — SSE client registry and broadcast.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	sseClientChanBuf   = 256
	maxPendingPerThread = 128
)

// 无订阅者时先把事件放进队列；首个 SSE 连上后按序回放，避免 POST /api/chat
// 早于 GET /api/sse 注册导致的「只看到 connected」竞态。
var (
	pendingMu sync.Mutex
	pendingQ  = make(map[string][][]byte) // threadID → 完整 SSE 帧（含 event/data）
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
	payload, _ := json.Marshal(data)
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, payload)
	frame := []byte(msg)

	v, ok := sseClients.Load(threadID)
	if !ok {
		enqueuePending(threadID, frame)
		return
	}
	m := v.(*sync.Map)
	hasSub := false
	m.Range(func(k, _ any) bool {
		hasSub = true
		return false
	})
	if !hasSub {
		enqueuePending(threadID, frame)
		return
	}

	m.Range(func(k, _ any) bool {
		if c, ok := k.(*SSEClient); ok && !c.closed {
			select {
			case c.ch <- frame:
			default:
				log.Printf("[SSE] client channel full, drop event thread=%s event=%s", threadID, event)
			}
		}
		return true
	})
}

func enqueuePending(threadID string, frame []byte) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	q := pendingQ[threadID]
	if len(q) >= maxPendingPerThread {
		log.Printf("[SSE] pending queue full (%d), drop oldest thread=%s", maxPendingPerThread, threadID)
		q = q[1:]
	}
	pendingQ[threadID] = append(q, frame)
}

func dequeuePending(threadID string) [][]byte {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	q := pendingQ[threadID]
	pendingQ[threadID] = nil
	return q
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

	ch := make(chan []byte, sseClientChanBuf)
	c := &SSEClient{threadID: threadID, ch: ch}
	registerSSEClient(threadID, c)
	defer unregisterSSEClient(threadID, c)

	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", mustMarshal(map[string]string{"threadId": threadID, "status": "connected"}))
	flusher.Flush()

	// 回放竞态期间缓冲的事件（run.started / tool.* / message.delta 等）
	for _, frame := range dequeuePending(threadID) {
		io.WriteString(w, string(frame))
		flusher.Flush()
	}

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
