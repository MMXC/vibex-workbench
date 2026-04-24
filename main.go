//go:build !web

package main

import (
	"context"
	"embed"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/build
var assets embed.FS

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
	backendBinary := "./backend/vibex-backend"
	if _, err := os.Stat(backendBinary); os.IsNotExist(err) {
		// 尝试从当前可执行文件所在目录推导
		exe, err := os.Executable()
		if err == nil {
			candidate := filepath.Join(filepath.Dir(exe), "backend", "vibex-backend")
			if _, err := os.Stat(candidate); err == nil {
				backendBinary = candidate
			}
		}
	}

	port := a.backendPort
	if port == 0 {
		port = 33335
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

func main() {
	app := &App{}

	err := wails.Run(
		&options.App{
			Title:  "VibeX Workbench",
			Width:  1280,
			Height: 800,
			AssetServer: &assetserver.Options{
				Assets: assets,
			},
			BackgroundColour: options.NewRGBA(30, 30, 30, 255),
			Bind: []interface{}{app},
			OnStartup: func(ctx context.Context) {
				app.ctx = ctx
			},
			OnDomReady: func(ctx context.Context) {
				// DOM ready, frontend JS is running
			},
			OnBeforeClose: func(ctx context.Context) bool {
				app.KillGoBackend(ctx)
				return false
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
