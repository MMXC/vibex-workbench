// agent/cmd/web/workspace_handlers.go — Workspace lifecycle HTTP handlers.
// Implements state detection, scaffolding, spec read/write, and make execution.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
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

// ── path safety helpers ─────────────────────────────────────────────

// pathError codes for structured error responses
const (
	errWorkspaceRootMissing     = "workspace_root_missing"
	errWorkspaceRootNotDir      = "workspace_root_not_directory"
	errPathTraversal            = "path_traversal"
	errInvalidMakeTarget        = "invalid_make_target"
)

// allowedMakeTargets is the allowlist for run-make security
var allowedMakeTargets = map[string]bool{
	"lint-specs": true,
	"validate":    true,
	"generate":    true,
	"mvp-gate":    true,
}

// normalizeWorkspaceRoot resolves a workspace root path to absolute form,
// handling Windows drive letters and Git Bash /c/ paths.
func normalizeWorkspaceRoot(wsRoot string) (string, string) {
	// "" or "." → empty string
	if wsRoot == "" || wsRoot == "." {
		return "", errWorkspaceRootMissing
	}

	// Clean the path first to resolve . and ..
	clean := filepath.Clean(wsRoot)

	// Handle Windows-style /c/ or /d/ paths from Git Bash
	if len(clean) >= 3 && clean[0] == '/' && clean[2] == '/' && clean[1] >= 'a' && clean[1] <= 'z' {
		clean = string(clean[1]-'a'+'A') + ":" + clean[2:]
		clean = filepath.Clean(clean)
	}

	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", errWorkspaceRootNotDir
	}

	// Must be a directory
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", errWorkspaceRootNotDir
	}

	return abs, ""
}

// isSpecPathSafe checks that a relative spec path doesn't escape the workspace.
// Returns ("", nil) if safe, or ("error_code", error_message) if unsafe.
func isSpecPathSafe(relPath, wsRoot string) (string, string) {
	if relPath == "" {
		return errPathTraversal, "spec path is empty"
	}

	// Must start with "specs/" or be a known safe prefix
	safePrefixes := []string{"specs/", "generators/", "Makefile", "README.md"}
	isSafe := false
	for _, prefix := range safePrefixes {
		if strings.HasPrefix(relPath, prefix) {
			isSafe = true
			break
		}
	}
	if !isSafe {
		return errPathTraversal, fmt.Sprintf("spec path must start with one of: %v", safePrefixes)
	}

	// Clean and check traversal
	clean := filepath.Clean(relPath)
	abs := filepath.Join(wsRoot, clean)
	absClean := filepath.Clean(abs)
	if !strings.HasPrefix(absClean, wsRoot) {
		return errPathTraversal, "path traversal detected"
	}
	return "", ""
}

// isTargetAllowed checks if a make target is in the security allowlist.
func isTargetAllowed(target string) bool {
	return allowedMakeTargets[target]
}

// ── detect-state ──────────────────────────────────────────────────

// workspaceDetectStateRequest is the POST body for /api/workspace/detect-state.
type workspaceDetectStateRequest struct {
	WorkspaceRoot string `json:"workspaceRoot"`
}

// workspaceDetectStateHandler GET/POST /api/workspace/detect-state
// Body: { "workspaceRoot": "/path/to/workspace" }
// Response: { "state": "empty"|"partial"|"ready", "workspaceRoot": "...", "signals": [...], "suggestions": [...] }
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
	// 优先用 WORKSPACE_ROOT，次选 cwd
	var scriptPath string
	if resolveRoot != "" {
		scriptPath = filepath.Join(resolveRoot, "generators", "state_detector.py")
	} else {
		cwd, _ := os.Getwd()
		scriptPath = filepath.Join(cwd, "generators", "state_detector.py")
	}
	scriptPath, _ = filepath.Abs(scriptPath)
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
	WorkspaceRoot  string `json:"workspaceRoot"`
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

	// 优先用 wsRoot（已从 req.WorkspaceRoot / cfg / env 解析），次选 cwd
	var scriptPath string
	if wsRoot != "" {
		scriptPath = filepath.Join(wsRoot, "generators", "scaffolder.py")
	} else {
		cwd, _ := os.Getwd()
		scriptPath = filepath.Join(cwd, "generators", "scaffolder.py")
	}
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
	WorkspaceRoot string `json:"workspaceRoot"`
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

	// Normalize workspace root
	wsRootNorm, errCode := normalizeWorkspaceRoot(req.WorkspaceRoot)
	if errCode != "" {
		http.Error(w, errCode+": workspace root", http.StatusBadRequest)
		return
	}

	// Security: spec path guard
	if errCode2, msg := isSpecPathSafe(req.Path, wsRootNorm); errCode2 != "" {
		http.Error(w, errCode2+": "+msg, http.StatusForbidden)
		return
	}

	// Path traversal protection (defense in depth)
	cleanPath := filepath.Clean(req.Path)
	absPath := filepath.Join(wsRootNorm, cleanPath)
	if !strings.HasPrefix(filepath.Clean(absPath), wsRootNorm) {
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
	WorkspaceRoot string `json:"workspaceRoot"`
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

	// Security: target allowlist
	if !isTargetAllowed(target) {
		http.Error(w, fmt.Sprintf("%s: %q not in allowlist", errInvalidMakeTarget, target), http.StatusForbidden)
		return
	}

	// Normalize workspace root
	wsRootNorm, errCode := normalizeWorkspaceRoot(wsRoot)
	if errCode != "" {
		http.Error(w, errCode+": workspace root", http.StatusBadRequest)
		return
	}

	// 120s timeout per spec
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", target)
	cmd.Dir = wsRootNorm
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

// scaffoldPreviewHandler GET /api/workspace/scaffold/preview
// Returns a preview of what scaffold would create, with a UUID confirmation token.
// The token is stored in .vibex/scaffold_preview_token.json for confirm-time validation.
// This is a dry-run: it does NOT write any files.
func scaffoldPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wsRoot := r.URL.Query().Get("workspaceRoot")
	if wsRoot == "" {
		wsRoot = cfg.WorkspaceDir
	}
	if wsRoot == "" {
		http.Error(w, "workspaceRoot required", http.StatusBadRequest)
		return
	}

	// Check current state first
	detector := filepath.Join(cfg.WorkspaceDir, "generators", "state_detector.py")
	cmd := exec.Command("python3", detector, wsRoot, "--json")
	cmd.Dir = cfg.WorkspaceDir
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.Run()

	var currentState string = "unknown"
	if m := map[string]interface{}{}; json.Unmarshal(stdout.Bytes(), &m) == nil {
		if s, ok := m["state"].(string); ok {
			currentState = s
		}
	}

	// Always-ok files that scaffold would create
	previewFiles := []map[string]string{
		{"path": "specs/.gitkeep", "description": "spec 文件目录"},
		{"path": "generators/.gitkeep", "description": "生成器目录"},
		{"path": "spec-templates/.gitkeep", "description": "spec 模板目录"},
		{"path": "Makefile", "description": "构建入口（make validate / make generate）"},
		{"path": "frontend/package.json", "description": "前端依赖配置"},
		{"path": ".vibex/scaffold_preview_token.json", "description": "脚手架预览确认 token（确认后写入）"},
	}

	// If state is empty, also preview the minimal L1 spec
	suggestions := []string{}
	if currentState == "empty" {
		suggestions = append(suggestions, "当前目录为空，建议执行脚手架初始化")
		previewFiles = append(previewFiles,
			map[string]string{"path": "specs/L1-goal/PLACEHOLDER.yaml", "description": "首个 L1 目标规格（占位）"},
		)
	} else if currentState == "partial" {
		suggestions = append(suggestions, "目录已有部分结构，可直接运行 make validate")
	} else if currentState == "ready" {
		suggestions = append(suggestions, "脚手架已就绪，无需重新初始化")
	}

	// Generate UUID token and store
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	tokenFile := filepath.Join(wsRoot, ".vibex", "scaffold_preview_token.json")
	os.MkdirAll(filepath.Dir(tokenFile), 0755)
	tokenData := map[string]interface{}{
		"token":         token,
		"workspace":   wsRoot,
		"created_at":   time.Now().Format(time.RFC3339),
		"preview_files": previewFiles,
		"current_state": currentState,
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"preview_token":  token,
		"current_state":  currentState,
		"preview_files":  previewFiles,
		"suggestions":    suggestions,
		"workspace_root": wsRoot,
	})

	// Store token server-side (for confirm-time validation)
	tokenData["token"] = token
	if data, err := json.Marshal(tokenData); err == nil {
		os.WriteFile(tokenFile, data, 0644)
	}
}
