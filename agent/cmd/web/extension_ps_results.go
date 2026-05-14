// cmd/web/extension_ps_results.go — Prototype-Skill 结果注册表与回调 HTTP handler。
// Extension 执行完 ps_* 工具后，通过 POST /api/extension/tool-result 回调结果。
// 同一 chat turn 中，Go Agent 的 LLM stream 在 ps_* 工具处暂停，
// 等待结果注入后才继续推理。
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type psToolResult struct {
	ToolName string `json:"toolName"`
	CallID   string `json:"callId"`
	Result   any    `json:"result"`
	Error    string `json:"error,omitempty"`
	ThreadID string `json:"threadId"`
}

// pendingPSResults: callID → result channel（单次消费）
var (
	pendingPSMu   sync.RWMutex
	pendingPSResults = make(map[string]chan psToolResult)
)

// psToolTimeout is how long a ps_* tool call waits for extension result before returning.
const psToolTimeout = 30 * time.Second

// RegisterPSToolCall 注册一个等待结果的 channel。
// 返回写入函数（由 extension HTTP handler 调用时写入）和结果读取函数（由 runToolLoop 注入时调用）。
func RegisterPSToolCall(callID string) (write func(psToolResult), waitForResult func() psToolResult) {
	resultCh := make(chan psToolResult, 1)
	pendingPSMu.Lock()
	pendingPSResults[callID] = resultCh
	pendingPSMu.Unlock()

	// 清理：超时后自动移除
	go func() {
		time.Sleep(psToolTimeout)
		pendingPSMu.Lock()
		delete(pendingPSResults, callID)
		pendingPSMu.Unlock()
	}()

	write = func(r psToolResult) {
		pendingPSMu.RLock()
		ch, ok := pendingPSResults[callID]
		pendingPSMu.RUnlock()
		if ok {
			select {
			case ch <- r:
			default:
			}
		}
	}

	waitForResult = func() psToolResult {
		select {
		case r := <-resultCh:
			return r
		case <-time.After(psToolTimeout):
			return psToolResult{
				Error:    "timeout: extension did not respond within " + psToolTimeout.String(),
				ToolName: "",
				CallID:   callID,
			}
		}
	}

	return write, waitForResult
}

// GetPSResult 读取已注册的结果（不阻塞）
func GetPSResult(callID string) (psToolResult, bool) {
	pendingPSMu.RLock()
	ch, ok := pendingPSResults[callID]
	pendingPSMu.RUnlock()
	if !ok {
		return psToolResult{}, false
	}
	select {
	case r := <-ch:
		pendingPSMu.Lock()
		delete(pendingPSResults, callID)
		pendingPSMu.Unlock()
		return r, true
	default:
		return psToolResult{}, false
	}
}

// extensionToolResultHandler: POST /api/extension/tool-result
func extensionToolResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req psToolResult
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.CallID == "" {
		http.Error(w, "callId is required", http.StatusBadRequest)
		return
	}

	log.Printf("[PS-Result] callback: tool=%s callId=%s thread=%s",
		req.ToolName, req.CallID, req.ThreadID)

	pendingPSMu.RLock()
	ch, ok := pendingPSResults[req.CallID]
	pendingPSMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if !ok {
		// callID 不在等待列表中（可能是旧请求或超时已移除）
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "callId not found or already resolved"})
		return
	}

	// 写入结果
	select {
	case ch <- req:
		pendingPSMu.Lock()
		delete(pendingPSResults, req.CallID)
		pendingPSMu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "content": "result delivered"})
	default:
		// channel 已满或已关闭
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "channel full or closed"})
	}
}
