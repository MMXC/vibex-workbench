// agent/cmd/web/workspace_handlers.go — Workspace lifecycle HTTP handlers.
// Implements state detection, scaffolding, spec read/write, and make execution.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── detect-state ──────────────────────────────────────────────────

// workspaceDetectStateRequest is the POST body for /api/workspace/detect-state.
type workspaceDetectStateRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
}

// workspaceDetectStateHandler GET/POST /api/workspace/detect-state
// Body: { "workspaceRoot": "/path/to/workspace" }
// Response: { "state": "empty"|"half"|"ready", "workspaceRoot": "...", "signals": [...], "suggestions": [...] }
func workspaceDetectStateHandler(w http.ResponseWriter, r *http.Request) {
	wsRoot := cfg.WorkspaceDir

	if r.Method == http.MethodPost || r.Method == http.MethodGet {
		var req workspaceDetectStateRequest
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&req)
		}
		// Query param overrides body
		if qs := r.URL.Query().Get("workspaceRoot"); qs != "" {
			wsRoot = qs
		} else if req.WorkspaceRoot != "" {
			wsRoot = req.WorkspaceRoot
		}
	} else {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析 workspace root（支持 WORKSPACE_ROOT 环境变量或请求体）
	resolveRoot := func() string {
		if wsRoot != "" {
			return wsRoot
		}
		if env := os.Getenv("WORKSPACE_ROOT"); env != "" {
			return env
		}
		return ""
	}()

	result := map[string]interface{}{"workspaceRoot": resolveRoot}
	// 调用 state_detector.py — 基于 backend binary 位置推导 generators 路径
	// backend binary 在 backend/vibex-backend 或 backend/vibex-backend.exe
	// generators 在同级的 ../generators/
	scriptPath := filepath.Join(filepath.Dir(os.Args[0]), "..", "generators", "state_detector.py")
	scriptPath, _ = filepath.Abs(scriptPath) // 规范化路径
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		result["state"] = "error"
		result["error"] = fmt.Sprintf("state_detector.py not found at %s", scriptPath)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	cmd := exec.Command("python3", scriptPath, resolveRoot, "--json")
	cmd.Dir = resolveRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// 即使出错也尝试解析已输出内容
		result["state"] = "error"
		result["error"] = err.Error()
		if stderr.Len() > 0 {
			result["stderr"] = stderr.String()
		}
	} else {
		var stateResult map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &stateResult); err == nil {
			result = stateResult
		} else {
			result["state"] = "error"
			result["error"] = "failed to parse state_detector output"
			result["stdout"] = stdout.String()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ── scaffold ───────────────────────────────────────────────────────

// workspaceScaffoldRequest is the POST body for /api/workspace/scaffold.
type workspaceScaffoldRequest struct {
	WorkspaceRoot  string `json:"workspace_root"`
	Template       string `json:"template"`
	ProjectName    string `json:"projectName"`
	Owner          string `json:"owner"`
}

// workspaceScaffoldHandler POST /api/workspace/scaffold
// Body: { "workspaceRoot": "/path/to/workspace", "template": "default", "projectName": "...", "owner": "..." }
// Response: { "ok": true, "created": [...], "errors": [...] }
func workspaceScaffoldHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req workspaceScaffoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	wsRoot := req.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}

	// 调用 scaffolder.py — 基于 backend binary 位置推导 generators 路径
	scriptPath := filepath.Join(filepath.Dir(os.Args[0]), "..", "generators", "scaffolder.py")
	scriptPath, _ = filepath.Abs(scriptPath)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("scaffolder.py not found at %s", scriptPath), http.StatusInternalServerError)
		return
	}

	template := req.Template
	if template == "" {
		template = "default"
	}
	projectName := req.ProjectName
	if projectName == "" {
		projectName = filepath.Base(wsRoot)
	}
	owner := req.Owner
	if owner == "" {
		owner = "user"
	}

	args := []string{scriptPath, wsRoot, "--template", template}
	cmdCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "python3", args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VIBEX_PROJECT_NAME=%s", projectName),
		fmt.Sprintf("VIBEX_OWNER=%s", owner),
	)
	cmd.Dir = wsRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()

	var result map[string]interface{}
	if err == nil {
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			result = map[string]interface{}{"ok": true, "created": []string{}, "raw": output}
		}
	} else {
		result = map[string]interface{}{
			"ok":     false,
			"error":  err.Error(),
			"stderr": stderr.String(),
			"stdout": output,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ── spec write ────────────────────────────────────────────────────

// workspaceSpecWriteRequest is the POST body for /api/workspace/specs/write.
type workspaceSpecWriteRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	Path          string `json:"path"`
	Content       string `json:"content"`
}

// workspaceSpecWriteHandler POST /api/workspace/specs/write
// Body: { "workspaceRoot": "...", "path": "L1-goal/ENTRY.yaml", "content": "..." }
// Response: { "ok": true, "path": "..." }
func workspaceSpecWriteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req workspaceSpecWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	wsRoot := req.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}

	// Path traversal protection
	cleanPath := filepath.Clean(req.Path)
	absPath := filepath.Join(wsRoot, cleanPath)
	if !strings.HasPrefix(absPath, wsRoot) {
		http.Error(w, "forbidden: path traversal detected", http.StatusForbidden)
		return
	}

	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		http.Error(w, "write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "path": req.Path})
}

// ── run-make ─────────────────────────────────────────────────────

// workspaceRunMakeRequest is the POST body for /api/workspace/run-make.
type workspaceRunMakeRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	Target        string `json:"target"`
}

// workspaceRunMakeHandler POST /api/workspace/run-make
// Body: { "workspaceRoot": "...", "target": "lint-specs"|"generate" }
// Response: { "ok": true, "output": "...", "exitCode": 0, "timeout": false }
func workspaceRunMakeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req workspaceRunMakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	wsRoot := req.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}

	target := req.Target
	if target == "" {
		target = "lint-specs"
	}

	// 120s timeout per spec
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", target)
	cmd.Dir = wsRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var timedOut bool
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		timedOut = true
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[stderr]\n" + stderr.String()
	}

	exitCode := 0
	if err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			exitCode = ex.ExitCode()
		} else {
			exitCode = 1
		}
	}

	if timedOut {
		output += "\n[timeout] Command exceeded 120s limit"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       err == nil && !timedOut,
		"output":   output,
		"exitCode": exitCode,
		"timeout":  timedOut,
		"target":   target,
	})
}

// ── specs/list ────────────────────────────────────────────────────

// workspaceSpecsListHandler GET /api/workspace/specs/list
// Query: ?workspaceRoot=/path/to/workspace
// Response: { "paths": ["L1-goal/my-goal.yaml", "specs/L2-feature/feat.yaml", ...] }
func workspaceSpecsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wsRoot := r.URL.Query().Get("workspaceRoot")
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		wsRoot = os.Getenv("WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}

	specsDir := filepath.Join(wsRoot, "specs")
	var paths []string
	err := filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() && (strings.HasSuffix(full, ".yaml") || strings.HasSuffix(full, ".yml")) {
			rel, err := filepath.Rel(wsRoot, full)
			if err == nil {
				paths = append(paths, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, "failed to walk specs dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Strings(paths)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"paths": paths})
}

// ── specs/convention ──────────────────────────────────────────────

// workspaceSpecsConventionHandler GET /api/workspace/specs/convention
// Returns a summary of the VibeX spec naming/structure convention.
func workspaceSpecsConventionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	convention := map[string]interface{}{
		"directory_levels": []map[string]string{
			{"L1-goal": "顶层目标 spec（如 specs/L1-goal/xxx.yaml）"},
			{"L2-feature": "功能级 spec（如 specs/L2-feature/xxx.yaml）"},
			{"L3-module": "模块级 spec（如 specs/L3-module/xxx.yaml）"},
			{"L4-feature": "特性级 spec（如 specs/L4-feature/xxx.yaml）"},
			{"L5-component": "组件级 spec（如 specs/L5-component/xxx.yaml）"},
		},
		"required_frontmatter": []string{"name", "level", "parent"},
		"file_pattern":         "*.yaml 或 *.yml",
		"description":          "VibeX 使用五层 spec 体系（L1–L5），通过 YAML frontmatter 的 level 和 parent 字段建立父子关系，构成规格树。",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convention)
}
