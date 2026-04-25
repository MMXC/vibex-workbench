//go:build !web

package main

import (
	"context"
	"embed"
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

// indexHTML returns the embedded index.html content.
// Used by the SPA fallback handler.
func getIndexHTML() ([]byte, error) {
	// Try embed first
	f, err := assets.Open("index.html")
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

// appHandler serves SPA static files with a fallback, and proxies /api/*
// requests to the Go backend subprocess.
type appHandler struct {
	backendPort int
	spaFallback spaFallbackHandler
}

func (h *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Proxy /api/* to the Go backend subprocess
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		backendURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", h.backendPort),
			Path:   r.URL.Path,
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
	// All other requests: serve SPA fallback
	h.spaFallback.ServeHTTP(w, r)
}

// spaFallbackHandler serves index.html for any GET request that the
// embed cannot satisfy. This enables SPA client-side routing (e.g. /workbench).
type spaFallbackHandler struct{}

func (h *spaFallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := getIndexHTML()
	if err != nil {
		fmt.Printf("[spaFallback] getIndexHTML error: %v\n", err)
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
var theAppHandler = &appHandler{backendPort: 33338}

func main() {
	app := &App{}

	err := wails.Run(
		&options.App{
			Title:     "VibeX Workbench",
			Width:     1280,
			Height:    800,
			MinWidth:  800,
			MinHeight: 600,
			Frameless: false,
			AssetServer: &assetserver.Options{
				Assets:  assets,
				Handler: theAppHandler,
			},
			BackgroundColour: options.NewRGBA(15, 15, 15, 255),
			Bind: []interface{}{app},
			OnStartup: func(ctx context.Context) {
				app.ctx = ctx
				// 设置原生应用菜单
				appMenu := buildAppMenu(ctx)
				runtime.MenuSetApplicationMenu(ctx, appMenu)
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
