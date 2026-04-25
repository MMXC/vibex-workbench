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
	"os"
	"os/exec"
	"path/filepath"
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

// ── Workspace API ────────────────────────────────────────────────

// workspaceRunMakeHandler — P4: POST /api/workspace/run-make
func workspaceRunMakeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var body struct {
		Target    string `json:"target"`
		Workspace string `json:"workspace"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	ws := body.Workspace
	if ws == "" {
		ws = "."
	}
	target := body.Target
	if target == "" {
		target = "validate"
	}
	cmd := exec.Command("make", target)
	cmd.Dir = ws
	out, err := cmd.CombinedOutput()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        err == nil,
		"target":    target,
		"output":    string(out),
		"workspace": ws,
	})
}
func workspaceDetectStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var body struct {
		WorkspaceRoot string `json:"workspace_root"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	ws := body.WorkspaceRoot
	if ws == "" {
		ws = "."
	}

	script := filepath.Join(getBinaryDir(), "generators", "state_detector.py")
	cmd := exec.Command("python3", script, ws)
	cmd.Dir = filepath.Dir(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(w, `{"error":"%v","output":%q}`, err, string(out))
		return
	}
	fmt.Fprint(w, string(out))
}

// workspaceScaffoldHandler — P2: POST /api/workspace/scaffold
func workspaceScaffoldHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var body struct {
		WorkspaceRoot string `json:"workspace_root"`
		Confirm       bool   `json:"confirm"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	ws := body.WorkspaceRoot
	if ws == "" {
		ws = "."
	}

	script := filepath.Join(getBinaryDir(), "generators", "scaffold_generator.py")
	args := []string{script, ws}
	if body.Confirm {
		args = append(args, "--confirm")
	}
	cmd := exec.Command("python3", args...)
	cmd.Dir = ws
	out, err := cmd.CombinedOutput()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     err == nil,
		"output": string(out),
		"error":  err,
	})
}

// getBinaryDir returns the directory containing the backend binary (parent of backend/)
// generators/ and specs/ are at repo root (parent of backend/), not inside backend/
func getBinaryDir() string {
	exe, _ := os.Executable()
	if exe == "" {
		return "."
	}
	// binary is at repo/backend/vibex-backend → go up 2 levels to repo root
	return filepath.Dir(filepath.Dir(exe))
}

func main() {
	http.HandleFunc("/api/sse/threads/", withCORS(sseHandler))
	http.HandleFunc("/api/runs", withCORS(startRunHandler))
	http.HandleFunc("/api/health", withCORS(healthHandler))
	http.HandleFunc("/api/workspace/detect-state", withCORS(workspaceDetectStateHandler))
	http.HandleFunc("/api/workspace/scaffold", withCORS(workspaceScaffoldHandler))
	http.HandleFunc("/api/workspace/run-make", withCORS(workspaceRunMakeHandler))

	log.Println("[VibeX SSE Backend] Listening on http://0.0.0.0:33335")
	log.Println("  Workspace: POST http://localhost:33335/api/workspace/detect-state")
	log.Println("  Workspace: POST http://localhost:33335/api/workspace/scaffold")
	log.Println("  Workspace: POST http://localhost:33335/api/workspace/run-make")
	log.Fatal(http.ListenAndServe(":33335", nil))
}
