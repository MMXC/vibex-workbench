package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vibex/agent/agents/background"
)

// Repo relative path to bundled SpecPilot assets
const specpilotRepoCLI   = "cmd/specpilot"
const specpilotRepoMF    = "cmd/specpilot-mf"
const specpilotInstallCLI = "/tmp/specpilot"
const specpilotInstallMF  = "/tmp/specpilot-mf"

// workingDir returns the current working directory (workspace root)
func workingDir() string {
	d, _ := os.Getwd()
	return d
}

// installDir copies src dir to dest if dest doesn't exist
func installDir(src, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil // already installed
	}
	os.MkdirAll(filepath.Dir(dest), 0o755)
	return copyDir(src, dest)
}

// copyDir recursively copies src to dst
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}

// waitForPort polls a TCP port until it's open (max 30s)
func waitForPort(host string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("sh", "-c", fmt.Sprintf("nc -z %s 2>/dev/null || curl -s %s > /dev/null 2>&1 || true", host, host))
		if err := cmd.Run(); err == nil {
			// Try actual connection check
			check := exec.Command("sh", "-c", fmt.Sprintf("curl -s --connect-timeout 1 %s > /dev/null 2>&1 && echo ok || echo fail", host))
			out, _ := check.CombinedOutput()
			if strings.Contains(string(out), "ok") {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", host)
}

// mfBootstrapHandler wraps mfBootstrapHandlerImpl to match the Handler signature
func mfBootstrapHandler(bgMgr *background.Manager) Handler {
	return func(arguments string) string {
		return mfBootstrapHandlerImpl(arguments, bgMgr)
	}
}

// mfBootstrapHandlerImpl — installs and starts SpecPilot services, returns URLs
func mfBootstrapHandlerImpl(arguments string, bgMgr *background.Manager) string {
	wsRoot := workingDir()
	cliSrc := filepath.Join(wsRoot, specpilotRepoCLI)
	mfSrc := filepath.Join(wsRoot, specpilotRepoMF)

	// 1. Install CLI to /tmp/specpilot
	if err := installDir(cliSrc, specpilotInstallCLI); err != nil {
		return fmt.Sprintf(`{"error": "failed to install specpilot CLI: %v"}`, err)
	}
	// 2. Install MF app to /tmp/specpilot-mf
	if err := installDir(mfSrc, specpilotInstallMF); err != nil {
		return fmt.Sprintf(`{"error": "failed to install specpilot MF: %v"}`, err)
	}

	// 3. Check if services already running
	dcRunning := isPortOpen("127.0.0.1:7890")
	mfRunning := isPortOpen("127.0.0.1:5177")

	// 4. Start DC API server (7890)
	if !dcRunning {
		apiScript := filepath.Join(specpilotInstallCLI, "api_server.py")
		cmd := exec.Command("python3", apiScript)
		cmd.Dir = specpilotInstallCLI
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Sprintf(`{"error": "failed to start api_server: %v"}`, err)
		}
	}

	// 5. Start MF dev server (5177)
	if !mfRunning {
		cmd := exec.Command("python3", "-m", "http.server", "5177")
		cmd.Dir = specpilotInstallMF
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Sprintf(`{"error": "failed to start MF server: %v"}`, err)
		}
	}

	// 6. Wait for ports
	waitForPort("http://127.0.0.1:7890", 10*time.Second)
	waitForPort("http://127.0.0.1:5177", 10*time.Second)

	// 7. Seed demo data
	seedData()

	return fmt.Sprintf(`{
		"ok": true,
		"dcUrl": "http://127.0.0.1:7890",
		"mfUrl": "http://localhost:5177",
		"mfRemoteUrl": "http://localhost:5177/#/%s",
		"message": "SpecPilot services bootstrapped successfully"
	}`, "{component}")
}

// isPortOpen checks if a TCP port is listening
func isPortOpen(addr string) bool {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("nc -z 127.0.0.1 %s 2>/dev/null && echo open || echo closed", strings.Split(addr, ":")[1]))
	out, _ := cmd.CombinedOutput()
	return strings.Contains(string(out), "open")
}

// seedData populates demo data into the DataCenter
func seedData() {
	cmds := [][]string{
		{"python3", "-m", "cli", "dc", "set", "kpi.revenue", "1284500"},
		{"python3", "-m", "cli", "dc", "set", "kpi.users", "48291"},
		{"python3", "-m", "cli", "dc", "set", "kpi.conversion", "3.8"},
		{"python3", "-m", "cli", "dc", "set", "kpi.latency", "142"},
		{"python3", "-m", "cli", "dc", "set", "kpi.trend", "12.3"},
		{"python3", "-m", "cli", "dc", "set", "alert.status", "healthy"},
		{"python3", "-m", "cli", "dc", "set", "alert.count", "0"},
		{"python3", "-m", "cli", "dc", "set", "table.users", `[{"id":1,"name":"Alice Chen","email":"alice@example.com","status":"active","score":98},{"id":2,"name":"Bob Kim","email":"bob@example.com","status":"active","score":85},{"id":3,"name":"Carol Wu","email":"carol@example.com","status":"inactive","score":72},{"id":4,"name":"David Lee","email":"david@example.com","status":"active","score":91}]`},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = specpilotInstallCLI
		cmd.Run()
	}
}

// ---------------------------------------------------------------------------
// Existing handlers (unchanged)
// ---------------------------------------------------------------------------

// runSP runs a specpilot CLI command and returns the output
func runSP(args ...string) string {
	cmd := exec.Command("python3", append([]string{"-m", "cli"}, args...)...)
	cmd.Dir = specpilotInstallCLI
	out, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf(`{"error": "specpilot CLI failed: %v", "stderr": "%v"}`, err, err)
	}
	return strings.TrimSpace(string(out))
}

func dcListHandler(arguments string) string {
	return runSP("dc", "list")
}

func dcGetHandler(arguments string) string {
	var args struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: key required"}`
	}
	return runSP("dc", "get", args.Key)
}

func dcSetHandler(arguments string) string {
	var args struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: key and value required"}`
	}
	return runSP("dc", "set", args.Key, fmt.Sprintf("%v", args.Value))
}

func ecHistoryHandler(arguments string) string {
	return runSP("ec", "history")
}

func ecEmitHandler(arguments string) string {
	var args struct {
		Event   string `json:"event"`
		Payload any    `json:"payload"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: event and payload required"}`
	}
	payloadJSON, _ := json.Marshal(args.Payload)
	return runSP("ec", "emit", args.Event, string(payloadJSON))
}

func ecSubscribeHandler(arguments string) string {
	var args struct {
		Event      string `json:"event"`
		Subscriber string `json:"subscriber"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: event and subscriber required"}`
	}
	return runSP("ec", "subscribe", args.Event, args.Subscriber)
}

func adListHandler(arguments string) string {
	return runSP("ad", "list")
}

func adSwitchHandler(arguments string) string {
	var args struct {
		Adapter string `json:"adapter"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: adapter name required"}`
	}
	return runSP("ad", "switch", args.Adapter)
}

func adQueryHandler(arguments string) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: query required"}`
	}
	return runSP("ad", "query", args.Query)
}

func specListHandler(arguments string) string {
	return runSP("spec", "list")
}

func specGetHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	return runSP("spec", "get", args.Name)
}

func specBindingHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	return runSP("spec", "binding", args.Name)
}

func mfListHandler(arguments string) string {
	return runSP("mf", "list")
}

func mfRegisterHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: name and path required"}`
	}
	return runSP("mf", "register", args.Name, args.Path)
}

func mfResolveHandler(arguments string) string {
	var args struct {
		Spec string `json:"spec"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	return runSP("mf", "resolve-from-spec", args.Spec)
}
