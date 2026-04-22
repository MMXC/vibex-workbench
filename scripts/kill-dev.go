// scripts/kill-dev.go — 跨平台清理 dev 进程（Linux & macOS）
// 用法: go run scripts/kill-dev.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var ports = []int{5173, 33338}

func main() {
	// 1. 按端口杀
	fmt.Println("[kill-dev] Killing processes on ports...")
	for _, port := range ports {
		killByPort(port)
	}

	// 2. 按进程名杀（兜底）
	patterns := []string{"vite", "cmd/web"}
	fmt.Println("[kill-dev] Killing by process name...")
	for _, pat := range patterns {
		killByName(pat)
	}

	fmt.Println("[kill-dev] Done.")
}

func killByPort(port int) {
	// lsof -ti:PORT 返回该端口的 PID 列表
	cmd := exec.Command("lsof", "-ti", strconv.Itoa(port))
	out, err := cmd.Output()
	if err != nil {
		return
	}
	pids := strings.Fields(string(out))
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid == os.Getpid() {
			continue
		}
		exec.Command("kill", "-9", pidStr).Run()
		fmt.Printf("  killed PID %d (port %d)\n", pid, port)
	}
}

func killByName(pattern string) {
	// pkill -f 匹配整个命令行
	cmd := exec.Command("pkill", "-f", pattern)
	if err := cmd.Run(); err != nil {
		// pkill 没找到进程返回非0，不算错
	}
}
