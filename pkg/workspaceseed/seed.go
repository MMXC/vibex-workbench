// Package workspaceseed — 将内嵌默认文件释放到用户工作区根（不覆盖已存在文件）。
// 与 Workbench 安装路径解耦，仅依赖 embed.FS。
package workspaceseed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// 显式列出路径：目录形式的 //go:embed default 会忽略以 `.` 开头的文件/目录（如 .vibex）。
//
//go:embed default/.vibex/agents/ppt-layer-templates.yaml
var defaultSeed embed.FS

// Apply 将 default/ 下树复制到 workspaceRoot（已存在路径跳过，不覆盖用户内容）。
func Apply(workspaceRoot string) error {
	root := filepath.Clean(workspaceRoot)
	if root == "" || root == "." {
		return fmt.Errorf("workspace root required")
	}
	return fs.WalkDir(defaultSeed, "default", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel("default", path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		dst := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
		data, err := defaultSeed.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}
