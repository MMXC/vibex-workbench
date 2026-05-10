package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Workbench Inspector WebSocket — CDP-like event stream: UI pushes snapshots/events;
// subscribers (tools, debuggers, future agent listeners) receive copies; GET snapshot for polling.

const inspectorRingCap = 256

var inspectorUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Dev: allow localhost / 127.0.0.1 / ::1 (browser & embed webviews).
		o := r.Header.Get("Origin")
		if o == "" {
			return true
		}
		return true
	},
}

// InspectorEnvelope is JSON-RPC–friendly (kind ~ method notification).
type InspectorEnvelope struct {
	V      int                    `json:"v,omitempty"`
	Kind   string                 `json:"kind"`
	Domain string                 `json:"domain,omitempty"`
	Method string                 `json:"method,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
	Ts     string                 `json:"ts,omitempty"`
	Source string                 `json:"source,omitempty"`
}

type inspectorHub struct {
	mu sync.RWMutex

	conns map[*websocket.Conn]struct{}

	ringMu sync.Mutex
	ring   [][]byte // capped append-only ring (last inspectorRingCap messages)

	lastSnapshot atomic.Pointer[inspectorSnapshotState]
}

func (h *inspectorHub) broadcastServer(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.conns {
		_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[inspector] broadcast server write error: %v", err)
		}
	}
}

type inspectorSnapshotState struct {
	Params map[string]interface{} `json:"params,omitempty"`
	Ts     string                 `json:"ts,omitempty"`
	Source string                 `json:"source,omitempty"`
}

var inspectorHubSingleton = &inspectorHub{
	conns: make(map[*websocket.Conn]struct{}),
	ring:  make([][]byte, inspectorRingCap),
}

func (h *inspectorHub) register(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[c] = struct{}{}
}

func (h *inspectorHub) unregister(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, c)
}

func (h *inspectorHub) recordInbound(raw []byte) {
	var env InspectorEnvelope
	if err := json.Unmarshal(raw, &env); err == nil && env.Kind == "snapshot" && len(env.Params) > 0 {
		st := &inspectorSnapshotState{Params: env.Params, Ts: env.Ts, Source: env.Source}
		h.lastSnapshot.Store(st)
	}

	cp := make([]byte, len(raw))
	copy(cp, raw)
	h.ringMu.Lock()
	if len(h.ring) >= inspectorRingCap {
		h.ring = append(h.ring[1:], cp)
	} else {
		h.ring = append(h.ring, cp)
	}
	h.ringMu.Unlock()
}

func (h *inspectorHub) broadcast(from *websocket.Conn, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.conns {
		if c == from {
			continue
		}
		_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[inspector] broadcast write error: %v", err)
		}
	}
}

func (h *inspectorHub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

func (h *inspectorHub) recentEventsJSON(max int) []json.RawMessage {
	h.ringMu.Lock()
	defer h.ringMu.Unlock()
	n := len(h.ring)
	if max <= 0 || max > n {
		max = n
	}
	start := n - max
	if start < 0 {
		start = 0
	}
	out := make([]json.RawMessage, 0, max)
	for i := start; i < n; i++ {
		out = append(out, json.RawMessage(h.ring[i]))
	}
	return out
}

func inspectorWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	conn, err := inspectorUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[inspector] upgrade failed: %v", err)
		return
	}
	inspectorHubSingleton.register(conn)

	go func() {
		defer func() {
			inspectorHubSingleton.unregister(conn)
			_ = conn.Close()
		}()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt != websocket.TextMessage {
				continue
			}
			inspectorHubSingleton.recordInbound(msg)
			inspectorHubSingleton.broadcast(conn, msg)
		}
	}()
}

func inspectorSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodOptions && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	st := inspectorHubSingleton.lastSnapshot.Load()
	ev := inspectorHubSingleton.recentEventsJSON(64)

	out := map[string]interface{}{
		"clients": inspectorHubSingleton.clientCount(),
		"events":  ev,
	}
	if st != nil {
		out["snapshot"] = st
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func inspectorTraceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodOptions && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	env := InspectorEnvelope{
		V:      1,
		Kind:   "trace",
		Domain: "Agent.Validation",
		Method: "nodeTrace",
		Ts:     time.Now().UTC().Format(time.RFC3339Nano),
		Source: "agent",
	}
	if payload != nil {
		env.Params = payload
	} else {
		env.Params = map[string]interface{}{}
	}
	raw, err := json.Marshal(env)
	if err != nil {
		http.Error(w, "marshal trace failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	inspectorHubSingleton.recordInbound(raw)
	inspectorHubSingleton.broadcastServer(raw)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}
