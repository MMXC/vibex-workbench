// main.go — VibeX Workbench Minimal SSE Backend
// 从 frontend/*_service.yaml 接口契约派生，实现 mock 事件流
//
// 端口: 33335
// SSE端点: GET /api/sse/threads/:threadId
// Run API:  POST /api/runs
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func uuid8() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ── SSE Client Registry ────────────────────────────────────────

type SSEClient struct {
	threadID string
	ch       chan []byte
	closed   bool
}

var clients = make(map[string]map[*SSEClient]bool)

func registerClient(threadID string, c *SSEClient) {
	if clients[threadID] == nil {
		clients[threadID] = make(map[*SSEClient]bool)
	}
	clients[threadID][c] = true
}

func unregisterClient(threadID string, c *SSEClient) {
	if clients[threadID] != nil {
		delete(clients[threadID], c)
		if len(clients[threadID]) == 0 {
			delete(clients, threadID)
		}
	}
}

func broadcast(threadID, event string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, payload)
	if clients[threadID] == nil {
		return
	}
	for c := range clients[threadID] {
		if !c.closed {
			select {
			case c.ch <- []byte(msg):
			default:
			}
		}
	}
}

// ── Mock Run Executor ──────────────────────────────────────────

func runMockRun(threadID string) {
	runID := uuid8()
	goal := "用户目标"

	time.Sleep(300 * time.Millisecond)
	broadcast(threadID, "run.started", map[string]interface{}{
		"runId":    runID,
		"threadId": threadID,
		"status":   "queued",
		"stage":    "planning",
		"goal":     goal,
	})

	time.Sleep(500 * time.Millisecond)
	broadcast(threadID, "run.stage_changed", map[string]interface{}{
		"runId": runID,
		"stage": "executing",
	})

	tools := []struct {
		name string
		args map[string]string
	}{
		{"read_file", map[string]string{"path": "/workspace/main.py"}},
		{"terminal", map[string]string{"command": "ls -la"}},
		{"search_files", map[string]string{"pattern": "*.py", "path": "/workspace"}},
	}

	for i, t := range tools {
		invID := fmt.Sprintf("%s-t%d", runID, i)
		time.Sleep(600 * time.Millisecond)
		broadcast(threadID, "tool.called", map[string]interface{}{
			"invocationId": invID,
			"runId":        runID,
			"toolName":     t.name,
			"args":         t.args,
		})

		time.Sleep(300 * time.Millisecond)
		broadcast(threadID, "tool.completed", map[string]interface{}{
			"invocationId": invID,
			"result":       map[string]string{"output": fmt.Sprintf("[mock] %s done", t.name), "status": "success"},
		})

		time.Sleep(200 * time.Millisecond)
		broadcast(threadID, "message.delta", map[string]interface{}{
			"runId":    runID,
			"role":     "assistant",
			"delta":    fmt.Sprintf("[%s] 执行完成，输出：[mock] %s done\n", t.name, t.name),
			"is_final": false,
		})
	}

	time.Sleep(300 * time.Millisecond)
	artifactID := uuid8()
	broadcast(threadID, "artifact.created", map[string]interface{}{
		"artifact": map[string]interface{}{
			"id":        artifactID,
			"threadId":  threadID,
			"runId":     runID,
			"type":      "code",
			"name":      "output.py",
			"content":   "print('hello from Go SSE backend!')\n",
			"createdAt": time.Now().Format(time.RFC3339),
		},
	})

	time.Sleep(300 * time.Millisecond)
	broadcast(threadID, "run.completed", map[string]interface{}{
		"runId":   runID,
		"summary": fmt.Sprintf("Run %s completed. 3 tools executed.", runID),
	})
}

// ── HTTP Handler ──────────────────────────────────────────────

func sseHandler(w http.ResponseWriter, r *http.Request) {
	threadID := r.URL.Path[len("/api/sse/threads/"):]
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
	registerClient(threadID, c)
	defer unregisterClient(threadID, c)

	connected, _ := json.Marshal(map[string]string{"threadId": threadID, "status": "connected"})
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", connected)
	flusher.Flush()

	go runMockRun(threadID)

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok","port":33335}`)
}

// withCORS 允许前端（如 localhost:5173）跨域访问 POST /api/runs（含 OPTIONS 预检）
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

type startRunReq struct {
	ThreadID string `json:"threadId"`
	Goal     string `json:"goal"`
}

func startRunHandler(w http.ResponseWriter, r *http.Request) {
	var req startRunReq
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	if req.ThreadID == "" {
		req.ThreadID = "default"
	}
	if req.Goal == "" {
		req.Goal = "用户目标"
	}

	runID := uuid8()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"runId":    runID,
		"threadId": req.ThreadID,
		"status":   "queued",
	})

	go runMockRun(req.ThreadID)
}

func main() {
	http.HandleFunc("/api/sse/threads/", withCORS(sseHandler))
	http.HandleFunc("/api/runs", withCORS(startRunHandler))
	http.HandleFunc("/api/health", withCORS(healthHandler))

	log.Println("[VibeX SSE Backend] Listening on http://0.0.0.0:33335")
	log.Println("  SSE:  GET  http://localhost:33335/api/sse/threads/<threadId>")
	log.Println("  Runs: POST http://localhost:33335/api/runs")
	log.Fatal(http.ListenAndServe(":33335", nil))
}
