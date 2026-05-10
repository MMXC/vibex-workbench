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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func htmlPPTAssetSourceCandidates(wsRoot string) []string {
	return []string{
		filepath.Join(wsRoot, "skills", "html-ppt", "assets"),
		filepath.Join(wsRoot, "skills", "html-ppt", "html-ppt", "assets"),
		filepath.Join(wsRoot, ".agents", "skills", "html-ppt", "assets"),
	}
}

// ensurePPTAssetsCopied copies html-ppt assets into workspace .vibex/assets if missing.
// Existing files are kept to avoid clobbering user customizations.
func ensurePPTAssetsCopied(wsRoot string) error {
	dstRoot := filepath.Join(wsRoot, ".vibex", "assets")
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}

	var srcRoot string
	for _, c := range htmlPPTAssetSourceCandidates(wsRoot) {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			srcRoot = c
			break
		}
	}
	if srcRoot == "" {
		return fmt.Errorf("html-ppt assets source not found")
	}

	return filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(dstRoot, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

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
	WorkspaceRoot string `json:"workspace_root"`
	Confirm      bool   `json:"confirm"`
	Template     string `json:"template"`
	ProjectName  string `json:"projectName"`
	Owner       string `json:"owner"`
	Mode        string `json:"mode"` // "full" or "partial"
}

// workspaceScaffoldHandler POST /api/workspace/scaffold
// Body: { "workspaceRoot": "/path/to/workspace", "confirm": true, "template": "default", "mode": "full" }
// Response: { "ok": true, "state": "ready", "written_files": [...], "skipped_files": [...] }
// AC3 要求：confirm=true 才能写入，否则返回 {ok: false, error: "需确认"}
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

	// AC3: 必须显式 confirm，禁止静默污染磁盘
	if !req.Confirm {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "需确认：前端必须传入 confirm: true",
		})
		return
	}

	// 调用 scaffolder.py
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
	if req.Mode == "partial" {
		args = append(args, "--mode", "partial")
	}
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
			result = map[string]interface{}{
				"ok":          true,
				"writtenFiles": []string{},
				"skippedFiles": []string{},
				"raw":         output,
			}
		}
		// 兼容旧字段名 created → written_files
		if written, ok := result["written_files"].([]any); ok {
			writtenList := make([]string, len(written))
			for i, v := range written {
				if s, ok := v.(string); ok {
					writtenList[i] = s
				}
			}
			result["writtenFiles"] = writtenList
		}
		if skipped, ok := result["skipped_files"].([]any); ok {
			skippedList := make([]string, len(skipped))
			for i, v := range skipped {
				if s, ok := v.(string); ok {
					skippedList[i] = s
				}
			}
			result["skippedFiles"] = skippedList
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

func resolveRepoScriptCandidates(relParts ...string) []string {
	if vx := strings.TrimSpace(os.Getenv("VIBEX_WORKBENCH_ROOT")); vx != "" {
		parts := append([]string{vx}, relParts...)
		return []string{filepath.Join(parts...)}
	}
	exe, err := os.Executable()
	dir := filepath.Dir(os.Args[0])
	if err == nil && exe != "" {
		edir := filepath.Dir(exe)
		baseA := filepath.Join(dir, "..")
		baseB := filepath.Join(edir, "..")
		return []string{
			filepath.Join(append([]string{baseA}, relParts...)...),
			filepath.Join(append([]string{baseB}, relParts...)...),
			filepath.Join(append([]string{edir}, relParts...)...),
			filepath.Join(append([]string{dir}, relParts...)...),
		}
	}
	return []string{
		filepath.Join(append([]string{filepath.Join(dir, "..")}, relParts...)...),
	}
}

func firstExisting(paths []string) string {
	for _, p := range paths {
		ap, err := filepath.Abs(p)
		if err != nil {
			ap = p
		}
		if _, err := os.Stat(ap); err == nil {
			return ap
		}
	}
	return ""
}

type workspaceSpecsBootstrapRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	WorkspaceRootAlt string `json:"workspaceRoot"`
	ProjectSlug   string `json:"project_slug"`
	ProjectName   string `json:"project_name"`
	Owner         string `json:"owner"`
	Confirm       bool   `json:"confirm"`
	Overwrite     bool   `json:"overwrite"`
}

// workspaceSpecsBootstrapHandler POST /api/workspace/spec-bootstrap
// Preferred: skill execute (.agents/skills/workspace-bootstrap/scripts/execute.py)
// Fallback: generators/spec_workspace_bootstrap.py.
func workspaceSpecsBootstrapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req workspaceSpecsBootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	wsRoot := req.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = req.WorkspaceRootAlt
	}
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

	if !req.Confirm {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "需确认：前端必须传入 confirm: true",
		})
		return
	}

	skillScript := firstExisting(resolveRepoScriptCandidates(
		".agents", "skills", "workspace-bootstrap", "scripts", "execute.py",
	))
	legacyScript := firstExisting(resolveRepoScriptCandidates("generators", "spec_workspace_bootstrap.py"))
	scriptPath := skillScript
	useLegacy := false
	if scriptPath == "" {
		scriptPath = legacyScript
		useLegacy = true
	}
	if scriptPath == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "workspace-bootstrap skill execute not found（且 legacy script 缺失）— 设置 VIBEX_WORKBENCH_ROOT 或从仓库根运行 agent",
		})
		return
	}

	projectSlug := strings.TrimSpace(req.ProjectSlug)
	if projectSlug == "" {
		projectSlug = strings.TrimSpace(req.ProjectName)
	}
	owner := strings.TrimSpace(req.Owner)
	if owner == "" {
		owner = "user"
	}

	args := []string{"python3", scriptPath}
	if useLegacy {
		args = append(args, wsRoot, "--owner", owner, "--json")
		if projectSlug != "" {
			args = append(args, "--project-slug", projectSlug)
		}
		if req.Overwrite {
			args = append(args, "--overwrite")
		}
	} else {
		args = append(args, "--workspace-root", wsRoot, "--owner", owner, "--confirm", "--json")
		if projectSlug != "" {
			args = append(args, "--project-slug", projectSlug)
		}
		if req.Overwrite {
			args = append(args, "--overwrite")
		}
	}

	cmdCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	// generators/ 的上一级为 Workbench 源码/安装根（含 generators、spec-templates），供脚本解析默认路径
	cmd.Dir = filepath.Dir(filepath.Dir(scriptPath))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outBytes := stdout.Bytes()
	if len(outBytes) == 0 {
		outBytes = stderr.Bytes()
	}

	var result map[string]interface{}
	if json.Unmarshal(outBytes, &result) == nil && len(result) > 0 {
		if err != nil {
			result["_exec_error"] = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
		return
	}

	fail := map[string]interface{}{
		"ok":     false,
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	}
	if err != nil {
		fail["error"] = err.Error()
	} else {
		fail["error"] = "spec_workspace_bootstrap 输出不可解析"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(fail)
}

// ── spec read ──────────────────────────────────────────────────────

// workspaceSpecReadHandler GET /api/workspace/spec/read
// Query: ?workspaceRoot=/path/to/workspace&path=specs/L1-goal/ENTRY.yaml
// Response: { "ok": true, "path": "...", "content": "..." }
// AC4 要求：能读取 specs/ 目录下的 YAML 文件内容
func workspaceSpecReadHandler(w http.ResponseWriter, r *http.Request) {
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

	specPath := r.URL.Query().Get("path")
	if specPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	// Path traversal protection
	cleanPath := filepath.Clean(specPath)
	absPath := filepath.Join(wsRoot, cleanPath)
	if !strings.HasPrefix(absPath, wsRoot) {
		http.Error(w, "forbidden: path traversal detected", http.StatusForbidden)
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found: "+specPath, http.StatusNotFound)
		} else {
			http.Error(w, "read failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"path":   specPath,
		"content": string(content),
	})
}

// workspaceFileHandler GET /api/workspace/file/<relative-path>
// Serves raw workspace files for iframe-based previews so relative assets work.
func workspaceFileHandler(w http.ResponseWriter, r *http.Request) {
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
	wsRoot = filepath.Clean(wsRoot)

	rawRel := strings.TrimPrefix(r.URL.Path, "/api/workspace/file/")
	if rawRel == "" || rawRel == r.URL.Path {
		http.Error(w, "file path required", http.StatusBadRequest)
		return
	}
	decodedRel, err := url.PathUnescape(rawRel)
	if err != nil {
		http.Error(w, "invalid file path", http.StatusBadRequest)
		return
	}

	relPath := strings.TrimPrefix(decodedRel, "/")
	cleanRel := filepath.Clean(relPath)
	if cleanRel == "." || strings.HasPrefix(cleanRel, "..") {
		http.Error(w, "forbidden: path traversal detected", http.StatusForbidden)
		return
	}

	target := filepath.Join(wsRoot, cleanRel)
	if !strings.HasPrefix(target, wsRoot) {
		http.Error(w, "forbidden: path traversal detected", http.StatusForbidden)
		return
	}

	if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
		http.ServeFile(w, r, target)
		return
	}

	// Compatibility fallback for html-ppt generated links:
	// .vibex/ppt/*.html references ../assets/* → .vibex/assets/*
	// If not present, auto-copy assets into workspace and then serve.
	slashRel := filepath.ToSlash(cleanRel)
	if strings.HasPrefix(slashRel, ".vibex/assets/") {
		_ = ensurePPTAssetsCopied(wsRoot)
		if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
			http.ServeFile(w, r, target)
			return
		}
		sub := strings.TrimPrefix(slashRel, ".vibex/assets/")
		candidates := make([]string, 0, 3)
		for _, root := range htmlPPTAssetSourceCandidates(wsRoot) {
			candidates = append(candidates, filepath.Join(root, filepath.FromSlash(sub)))
		}
		for _, candidate := range candidates {
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				http.ServeFile(w, r, candidate)
				return
			}
		}
	}

	http.Error(w, "not found", http.StatusNotFound)
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

type workspaceTreeNode struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"` // file | dir
	Size  int64  `json:"size,omitempty"`
	Mtime int64  `json:"mtime,omitempty"`
}

type workspaceGitStatusFile struct {
	Path   string `json:"path"`
	Index  string `json:"index"`
	Work   string `json:"worktree"`
	Status string `json:"status"`
}

type workspaceGitCommitRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	SpecPath      string `json:"spec_path"`
	Message       string `json:"message"`
}

// workspaceVerifySpecsRequest is the POST body for /api/workspace/verify-specs.
type workspaceVerifySpecsRequest struct {
	WorkspaceRoot string            `json:"workspace_root"`
	Format        string            `json:"format"`  // summary | json | short (default: summary)
	Checks        string            `json:"checks"`  // comma-separated: file_existence,parent_chain,completeness,behaviors
	Levels        string            `json:"levels"`  // comma-separated: 4_feature,5_slice,etc.
	ShowPass      bool              `json:"show_pass"`
}

// workspaceVerifySpecsHandler POST /api/workspace/verify-specs
// Runs verify_specs CLI and returns the report as JSON.
// Build verify_specs from the repository root: go build -o verify_specs ./cmd/verify_specs/
func workspaceVerifySpecsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req workspaceVerifySpecsRequest
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

	// Find the verify_specs binary next to the agent executable or under cwd (./verify_specs).
	exe, err := os.Executable()
	binPath := "./verify_specs"
	if err == nil {
		// exe might be the agent binary; look for verify_specs in the same dir
		binPath = filepath.Join(filepath.Dir(exe), "verify_specs")
	}

	// Try both relative and absolute paths
	for _, candidate := range []string{binPath, "./verify_specs"} {
		if _, err := os.Stat(candidate); err == nil {
			binPath = candidate
			break
		}
	}

	args := []string{"--workspace", wsRoot, "--format", "json"}
	if req.Checks != "" {
		args = append(args, "--check", req.Checks)
	}
	if req.Levels != "" {
		args = append(args, "--level", req.Levels)
	}
	if req.ShowPass {
		args = append(args, "--show-pass")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output = strings.TrimSpace(output) + "\n[stderr]\n" + stderr.String()
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			http.Error(w, `{"error":"verify-specs timed out after 120s"}`, http.StatusGatewayTimeout)
			return
		}
		// If binary not found, return a helpful error
		if os.IsNotExist(err) {
			http.Error(w, `{"error":"verify_specs binary not found. Build from repo root: go build -o verify_specs ./cmd/verify_specs/"}`, http.StatusServiceUnavailable)
			return
		}
		// Return JSON even on failure (the binary outputs JSON with exit code 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if output != "" {
			w.Write([]byte(output))
		} else {
			fmt.Fprintf(w, `{"error":%q}`, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
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

// workspaceTreeHandler GET /api/workspace/tree?workspaceRoot=...&path=...
func workspaceTreeHandler(w http.ResponseWriter, r *http.Request) {
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
	wsRoot = filepath.Clean(wsRoot)

	rel := strings.TrimSpace(r.URL.Query().Get("path"))
	if rel == "" {
		rel = "."
	}
	cleanRel := filepath.Clean(rel)
	if cleanRel == "." {
		cleanRel = ""
	}
	if strings.HasPrefix(cleanRel, "..") {
		http.Error(w, "forbidden: path traversal detected", http.StatusForbidden)
		return
	}
	target := filepath.Join(wsRoot, cleanRel)
	if !strings.HasPrefix(filepath.Clean(target), wsRoot) {
		http.Error(w, "forbidden: path traversal detected", http.StatusForbidden)
		return
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		http.Error(w, "read dir failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	nodes := make([]workspaceTreeNode, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".git") || strings.HasPrefix(name, ".sessions") {
			continue
		}
		full := filepath.Join(target, name)
		info, err := e.Info()
		if err != nil {
			continue
		}
		relPath, err := filepath.Rel(wsRoot, full)
		if err != nil {
			continue
		}
		node := workspaceTreeNode{
			Name:  name,
			Path:  filepath.ToSlash(relPath),
			Type:  "file",
			Size:  info.Size(),
			Mtime: info.ModTime().Unix(),
		}
		if e.IsDir() {
			node.Type = "dir"
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type == "dir"
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"path":  filepath.ToSlash(cleanRel),
		"nodes": nodes,
	})
}

// workspaceReadFileHandler GET /api/workspace/read-file?workspaceRoot=...&path=...
func workspaceReadFileHandler(w http.ResponseWriter, r *http.Request) {
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
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	absPath := filepath.Join(wsRoot, cleanPath)
	if !strings.HasPrefix(filepath.Clean(absPath), filepath.Clean(wsRoot)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		http.Error(w, "read failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	const max = 512 * 1024
	truncated := false
	if len(data) > max {
		data = data[:max]
		truncated = true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"path":      filepath.ToSlash(cleanPath),
		"content":   string(data),
		"truncated": truncated,
	})
}

// workspaceGitStatusHandler GET /api/workspace/git/status?workspaceRoot=...
func workspaceGitStatusHandler(w http.ResponseWriter, r *http.Request) {
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
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wsRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		http.Error(w, "git status failed: "+out.String(), http.StatusInternalServerError)
		return
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	files := make([]workspaceGitStatusFile, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		idx := string(line[0])
		wt := string(line[1])
		path := strings.TrimSpace(line[3:])
		files = append(files, workspaceGitStatusFile{
			Path:   path,
			Index:  idx,
			Work:   wt,
			Status: strings.TrimSpace(idx + wt),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"files": files,
		"count": len(files),
	})
}

// workspaceGitCommitBySpecHandler POST /api/workspace/git/commit-spec
func workspaceGitCommitBySpecHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req workspaceGitCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		http.Error(w, "message required", http.StatusBadRequest)
		return
	}
	specPath := strings.TrimSpace(req.SpecPath)
	if specPath != "" {
		msg = fmt.Sprintf("%s\n\nspec: %s", msg, specPath)
	}

	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = wsRoot
	var addOut bytes.Buffer
	addCmd.Stdout = &addOut
	addCmd.Stderr = &addOut
	if err := addCmd.Run(); err != nil {
		http.Error(w, "git add failed: "+addOut.String(), http.StatusInternalServerError)
		return
	}

	commitCmd := exec.Command("git", "commit", "-m", msg)
	commitCmd.Dir = wsRoot
	var commitOut bytes.Buffer
	commitCmd.Stdout = &commitOut
	commitCmd.Stderr = &commitOut
	if err := commitCmd.Run(); err != nil {
		text := commitOut.String()
		if strings.Contains(strings.ToLower(text), "nothing to commit") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "nothing to commit", "output": text})
			return
		}
		http.Error(w, "git commit failed: "+text, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"output": commitOut.String(),
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
