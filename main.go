//go:build !web

package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/build
var assets embed.FS

// mimeType returns the MIME type for a given file extension.
func mimeType(path string) string {
	switch ext := filepath.Ext(path); ext {
	case ".js":
		return "application/javascript"
	case ".mjs":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}

// serveFile tries to serve a file from the embedded FS (prefix "frontend/build"),
// then falls back to the disk directory. Sets correct MIME type.
// Returns true if the file was served, false if not found.
func serveFile(w http.ResponseWriter, r *http.Request, diskDir string) bool {
	path := r.URL.Path

	// Strip leading slash for embed lookups
	embedPath := filepath.Join("frontend/build", path[1:])
	if filepath.Ext(embedPath) == "" && !strings.HasSuffix(path, "/") {
		// Try as-is first
	}
	f, err := assets.Open(embedPath)
	if err == nil {
		defer f.Close()
		data, err := io.ReadAll(f)
		if err == nil {
			w.Header().Set("Content-Type", mimeType(path))
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return true
		}
	}

	// Disk fallback
	diskPath := filepath.Join(diskDir, path)
	data, err := os.ReadFile(diskPath)
	if err == nil {
		w.Header().Set("Content-Type", mimeType(path))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return true
	}

	return false
}

// indexHTML returns the embedded index.html content.
// Used by the SPA fallback handler.
func getIndexHTML() ([]byte, error) {
	// Try embed first
	f, err := assets.Open("frontend/build/index.html")
	if err == nil {
		defer f.Close()
		return io.ReadAll(f)
	}
	// Fallback: read directly from disk (dev mode)
	diskPath := "frontend/build/index.html"
	data, err := os.ReadFile(diskPath)
	if err != nil {
		return nil, fmt.Errorf("embed index.html: %w; disk fallback %s: %v", err, diskPath, err)
	}
	return data, nil
}

// appHandler serves static files from embedded assets (with disk fallback)
// and proxies /api/* requests to the Go backend subprocess.
// This replaces the default Wails AssetServer behavior so that .js/.css
// chunks are served with correct MIME types instead of falling through to
// index.html.
type appHandler struct {
	backendPort int
	diskDir     string
}

func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Proxy /api/* to the Go backend subprocess
	if strings.HasPrefix(path, "/api/") {
		backendURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", h.backendPort),
			Path:   path,
		}
		if r.URL.RawQuery != "" {
			backendURL.RawQuery = r.URL.RawQuery
		}
		proxy := httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL = backendURL
				req.Host = backendURL.Host
				req.Header.Set("X-Forwarded-Host", r.Host)
			},
		}
		proxy.ServeHTTP(w, r)
		return
	}

	// Try to serve a static file (.js, .css, .svg, etc.)
	diskDir := h.diskDir
	if diskDir == "" {
		diskDir = "frontend/build"
	}
	if serveFile(w, r, diskDir) {
		return
	}

	// Not found: for SPA routes, serve index.html
	// For other missing resources, return 404
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := getIndexHTML()
	if err != nil {
		fmt.Printf("[appHandler] getIndexHTML error: %v\n", err)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error: %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// App — 所有 Wails binding methods 都定义在此 struct 上
type App struct {
	ctx           context.Context
	workspaceRoot string
	backendCmd    *exec.Cmd
	backendPort   int
}

// ── Binding Methods ─────────────────────────────────────────

// OpenDirectoryDialog 打开系统原生目录选择器
func (a *App) OpenDirectoryDialog(ctx context.Context) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title:            "选择工作区目录",
		DefaultDirectory: a.workspaceRoot,
	})
	if err != nil {
		return "", fmt.Errorf("OpenDirectoryDialog failed: %w", err)
	}
	if dir == "" {
		return "", nil // 用户取消
	}
	a.workspaceRoot = dir
	runtime.EventsEmit(ctx, "workspace:selected", dir)
	return dir, nil
}

// GetWorkspaceRoot 返回当前工作区根路径
func (a *App) GetWorkspaceRoot() string {
	return a.workspaceRoot
}

// SetWorkspaceRoot 前端设置工作区路径（目录选择后或用户手动输入）
func (a *App) SetWorkspaceRoot(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("workspace root cannot be empty")
	}
	a.workspaceRoot = path
	return nil
}

// SpawnGoBackend 启动 Go backend 子进程
func (a *App) SpawnGoBackend(ctx context.Context) (map[string]any, error) {
	if a.backendCmd != nil && a.backendCmd.Process != nil {
		return map[string]any{
			"ok":   true,
			"port": a.backendPort,
			"pid":  a.backendCmd.Process.Pid,
		}, nil
	}

	// 找 backend binary：优先 ./backend/vibex-backend
	// Windows 编译后是 vibex-backend.exe，Go 也会报告带 .exe 的路径
	candidates := []string{
		"./backend/vibex-backend.exe",
		"./backend/vibex-backend",
	}
	backendBinary := "./backend/vibex-backend"
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			backendBinary = cand
			break
		}
	}
	if _, err := os.Stat(backendBinary); os.IsNotExist(err) {
		// 尝试从当前可执行文件所在目录推导
		exe, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exe)
			for _, suffix := range []string{".exe", ""} {
				candidate := filepath.Join(exeDir, "backend", "vibex-backend"+suffix)
				if _, err := os.Stat(candidate); err == nil {
					backendBinary = candidate
					break
				}
			}
		}
	}

	port := a.backendPort
	if port == 0 {
		port = 33338
	}
	a.backendPort = port

	cmd := exec.CommandContext(ctx, backendBinary, "-port", strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start backend: %w", err)
	}
	a.backendCmd = cmd

	// 等待 backend 真正监听
	go func() {
		for i := 0; i < 30; i++ {
			if isPortAvailable(port) {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "backend:ready", map[string]any{
					"port": port,
					"pid":  cmd.Process.Pid,
				})
			}
			return
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "backend:error", "backend failed to start within timeout")
		}
	}()

	return map[string]any{
		"ok":   true,
		"port": port,
		"pid":  cmd.Process.Pid,
	}, nil
}

// KillGoBackend 终止 Go backend 子进程
func (a *App) KillGoBackend(ctx context.Context) error {
	if a.backendCmd != nil && a.backendCmd.Process != nil {
		if err := a.backendCmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to kill backend: %w", err)
		}
		a.backendCmd = nil
	}
	return nil
}

// RunMake 在 workspace 执行 make target
func (a *App) RunMake(ctx context.Context, target string, workspace string) (map[string]any, error) {
	if workspace == "" {
		workspace = a.workspaceRoot
	}
	if target == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}
	cmd := exec.CommandContext(ctx, "make", target)
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return map[string]any{
		"ok":    err == nil,
		"output": string(out),
	}, err
}

// ── Filesystem Bridge ───────────────────────────────────────

// SpecFile 单个规格文件的元信息
type SpecFile struct {
	Path   string `json:"path"`   // 相对路径，如 specs/L1-goal/xxx.yaml
	Level  int    `json:"level"`  // 1-5，从目录名推导
	Name   string `json:"name"`   // frontmatter.name 或文件名
	Status string `json:"status"` // frontmatter.status，默认为 "active"
}

// levelFromDir 根据目录名推导 level（L1-L5）
func levelFromDir(name string) int {
	switch {
	case strings.HasPrefix(name, "L5"):
		return 5
	case strings.HasPrefix(name, "L4"):
		return 4
	case strings.HasPrefix(name, "L3"):
		return 3
	case strings.HasPrefix(name, "L2"):
		return 2
	case strings.HasPrefix(name, "L1"):
		return 1
	default:
		return 0
	}
}

// parseSpecFrontmatter 提取 frontmatter 中的 name 和 status
func parseSpecFrontmatter(content string) (name, status string) {
	lines := strings.SplitN(content, "\n", 20)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, "\"")
		}
		if strings.HasPrefix(line, "status:") {
			status = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			status = strings.Trim(status, "\"")
		}
	}
	if status == "" {
		status = "active"
	}
	return
}

// ListSpecs 扫描 {root}/specs/ 下所有 .yaml/.yml 文件，返回元信息列表
func (a *App) ListSpecs(root string) []SpecFile {
	if root == "" {
		root = a.workspaceRoot
	}
	specsDir := filepath.Join(root, "specs")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return []SpecFile{}
	}

	var result []SpecFile
	filepath.Walk(specsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		// 从相对路径提取 level
		parts := strings.Split(filepath.ToSlash(rel), "/")
		level := 0
		if len(parts) >= 2 {
			level = levelFromDir(parts[1])
		}

		// 提取 frontmatter
		data, _ := os.ReadFile(path)
		name, status := parseSpecFrontmatter(string(data))
		if name == "" {
			name = info.Name()
		}

		result = append(result, SpecFile{
			Path:   rel,
			Level:  level,
			Name:   name,
			Status: status,
		})
		return nil
	})
	if result == nil {
		result = []SpecFile{}
	}
	return result
}

// ReadSpecFile 读取 {root}/{path} 文件内容
func (a *App) ReadSpecFile(root, path string) (string, error) {
	if root == "" {
		root = a.workspaceRoot
	}
	full := filepath.Join(root, filepath.Clean(path))
	data, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("ReadSpecFile %s: %w", path, err)
	}
	return string(data), nil
}

// WriteSpecFile 写入 {root}/{path} 文件（自动创建中间目录）
func (a *App) WriteSpecFile(root, path, content string) error {
	if root == "" {
		root = a.workspaceRoot
	}
	full := filepath.Join(root, filepath.Clean(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("WriteSpecFile mkdir: %w", err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		return fmt.Errorf("WriteSpecFile write: %w", err)
	}
	return nil
}

// WorkspaceState 工作区状态检测结果
type WorkspaceState struct {
	State       string   `json:"state"` // "empty" | "half" | "ready" | "error"
	Signals     []Signal `json:"signals"`
	Suggestions []string `json:"suggestions"`
}

// Signal 单个检测信号
type Signal struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Reason  string `json:"reason"`
}

// DetectWorkspaceState 调用 generators/state_detector.py，返回状态结构
func (a *App) DetectWorkspaceState(root string) (WorkspaceState, error) {
	if root == "" {
		root = a.workspaceRoot
	}
	script := filepath.Join(root, "generators", "state_detector.py")
	cmd := exec.Command("python3", script, root, "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// state_detector 在目录不存在时返回 error state（exit 1 不代表真正错误）
		// 尝试解析已有的 stdout
	}
	var result WorkspaceState
	if out := stdout.String(); out != "" {
		json.Unmarshal([]byte(out), &result)
	}
	if result.State == "" {
		result.State = "error"
		result.Suggestions = []string{"无法检测工作区状态"}
	}
	return result, nil
}

// ── Agent Spawn ─────────────────────────────────────────────

// agentProcess 跟踪当前 agent subprocess
var agentProcess *exec.Cmd

// findAgentBinary 查找 agent binary 路径
func findAgentBinary() string {
	candidates := []string{
		"./agent/vibex-agent.exe",
		"./agent/vibex-agent",
		"vibex-agent",
	}
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	for _, suffix := range []string{".exe", ""} {
		cand := filepath.Join(exeDir, "agent", "vibex-agent"+suffix)
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return "./agent/vibex-agent"
}

// RunAgent spawn agent subprocess（JSON goal payload），通过 Wails event 把 stdout lines 发给前端
func (a *App) RunAgent(ctx context.Context, goalJSON string) (map[string]any, error) {
	var req struct {
		Goal          string `json:"goal"`
		WorkspaceRoot string `json:"workspaceRoot"`
	}
	if err := json.Unmarshal([]byte(goalJSON), &req); err != nil {
		return nil, fmt.Errorf("invalid goal JSON: %w", err)
	}
	workspace := req.WorkspaceRoot
	if workspace == "" {
		workspace = a.workspaceRoot
	}

	// 杀掉已有的 agent 进程
	if agentProcess != nil && agentProcess.Process != nil {
		agentProcess.Process.Signal(syscall.SIGTERM)
		agentProcess = nil
	}

	bin := findAgentBinary()
	cmd := exec.CommandContext(ctx, bin, "-goal", req.Goal, "-workspace", workspace)
	cmd.Dir = workspace
	cmd.Stderr = os.Stderr
	// 用 pipe 读取 stdout 并转发到 Wails event
	stdoutPipe, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}
	agentProcess = cmd

	// 在后台读取 stdout 并通过 Wails event 转发给前端，同时打印到本进程 stdout
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line) // 透传到 Wails 进程 stdout
			if a.ctx != nil && line != "" {
				runtime.EventsEmit(a.ctx, "agent:stdout", line)
			}
		}
		cmd.Wait()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "agent:done", map[string]any{
				"pid":    cmd.Process.Pid,
				"exited": true,
			})
		}
		agentProcess = nil
	}()

	return map[string]any{
		"ok":   true,
		"pid":  cmd.Process.Pid,
		"goal": req.Goal,
	}, nil
}

// KillAgent 终止当前 agent subprocess
func (a *App) KillAgent(ctx context.Context, pid int) error {
	if agentProcess != nil && agentProcess.Process != nil && agentProcess.Process.Pid == pid {
		if err := agentProcess.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("KillAgent: %w", err)
		}
		agentProcess = nil
	}
	return nil
}

// ── App Menu ──────────────────────────────────────────────

func buildAppMenu(ctx context.Context) *menu.Menu {
	appMenu := menu.NewMenu()

	// 文件
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("新建项目", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:new-project")
	})
	fileMenu.AddText("打开项目…", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:open-project")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("保存", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:save")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("退出", nil, func(_ *menu.CallbackData) {
		runtime.Quit(ctx)
	})

	// 编辑
	editMenu := appMenu.AddSubmenu("编辑")
	editMenu.AddText("撤销", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:undo")
	})
	editMenu.AddText("重做", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:redo")
	})
	editMenu.AddSeparator()
	editMenu.AddText("剪切", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:cut")
	})
	editMenu.AddText("复制", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:copy")
	})
	editMenu.AddText("粘贴", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:paste")
	})

	// 视图
	viewMenu := appMenu.AddSubmenu("视图")
	viewMenu.AddText("侧边栏", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:toggle-sidebar")
	})
	viewMenu.AddText("底部面板", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:toggle-dock")
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("清除缓存并刷新", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:clear-cache")
	})
	viewMenu.AddText("开发者工具", nil, func(_ *menu.CallbackData) {
		runtime.WindowReload(ctx)
	})

	// 终端
	termMenu := appMenu.AddSubmenu("终端")
	termMenu.AddText("新建终端", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:new-terminal")
	})
	termMenu.AddText("运行 make generate", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:run-generate")
	})
	termMenu.AddText("运行 make lint-specs", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:run-lint")
	})

	// 帮助
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("关于 VibeX Workbench", nil, func(_ *menu.CallbackData) {
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "关于 VibeX Workbench",
			Message: "VibeX Workbench\n\n规格驱动的 AI 辅助开发工作台。\n\n版本: dev",
		})
	})

	return appMenu
}

// ── Helper Functions ───────────────────────────────────────

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// ── Wails Lifecycle ───────────────────────────────────────

// global appHandler instance used by AssetServer
var theAppHandler = &appHandler{backendPort: 33338, diskDir: "frontend/build"}

func main() {
	app := &App{}

	err := wails.Run(
		&options.App{
			Title:     "VibeX Workbench",
			Width:     1280,
			Height:    800,
			MinWidth:  800,
			MinHeight: 600,
			Frameless: true,
			Menu:      menu.NewMenu(),
			CSSDragProperty: "--wails-draggable",
			CSSDragValue:   "drag",
			AssetServer: &assetserver.Options{
				Handler: theAppHandler,
			},
			BackgroundColour: options.NewRGBA(15, 15, 15, 255),
			Bind: []interface{}{app},
			OnStartup: func(ctx context.Context) {
				app.ctx = ctx
			},
			OnDomReady: func(ctx context.Context) {
				// 首次启动：清除 WebView2 缓存以避免旧 JS chunks 残留
				go func() {
					time.Sleep(500 * time.Millisecond)
					runtime.WindowReload(ctx)
				}()
				// 启动时自动 spawn Go backend
				go func() {
					_, err := app.SpawnGoBackend(ctx)
					if err != nil {
						runtime.LogError(ctx, "Auto-spawn backend failed: "+err.Error())
						return
					}
					// 通知 frontend 实际端口，同时更新 appHandler 的代理目标
					theAppHandler.backendPort = app.backendPort
					runtime.EventsEmit(ctx, "backend:port", app.backendPort)
				}()
			},
			OnBeforeClose: func(ctx context.Context) bool {
				app.KillGoBackend(ctx)
				return true // 允许窗口关闭
			},
			OnShutdown: func(ctx context.Context) {
				// cleanup
			},
		},
	)
	if err != nil {
		panic(err)
	}
}
