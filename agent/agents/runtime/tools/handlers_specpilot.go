package tools

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vibex/agent/agents/background"
)

// ---------------------------------------------------------------------------
// Path constants — resolved relative to workspace root
// ---------------------------------------------------------------------------

// skillRef returns the absolute path to specpilot skill references
func skillRef(wsRoot string) string {
	return filepath.Join(wsRoot, ".vibex", "agents", "skills", "specpilot", "references")
}

// wsInstall returns the workspace-local .specpilot install dir
func wsInstall(wsRoot string) string {
	return filepath.Join(wsRoot, ".specpilot")
}

// wsMF returns the workspace-local .specpilot-mf install dir
func wsMF(wsRoot string) string {
	return filepath.Join(wsRoot, ".specpilot-mf")
}

// ---------------------------------------------------------------------------
// Port config — env override with sensible defaults
// ---------------------------------------------------------------------------

func dcPort() int {
	if p := os.Getenv("SPECPILOT_DC_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			return n
		}
	}
	return 7890
}

func mfPort() int {
	if p := os.Getenv("SPECPILOT_MF_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			return n
		}
	}
	return 5177
}

// ---------------------------------------------------------------------------
// Install — copy from skill references to workspace-local .specpilot/
// ---------------------------------------------------------------------------

func installSpecPilot(wsRoot string) error {
	srcCLI := filepath.Join(skillRef(wsRoot), "cli")
	dstCLI := wsInstall(wsRoot)
	dstMF := wsMF(wsRoot)

	if _, err := os.Stat(dstCLI); err == nil {
		return nil // already installed
	}

	os.MkdirAll(dstCLI, 0o755)
	os.MkdirAll(dstMF, 0o755)

	// Copy CLI files
	if err := copyDirRecursive(srcCLI, dstCLI); err != nil {
		return fmt.Errorf("copy CLI: %w", err)
	}

	// Copy MF HTML
	srcMFHTML := filepath.Join(skillRef(wsRoot), "mf", "index.html")
	data, err := os.ReadFile(srcMFHTML)
	if err != nil {
		return fmt.Errorf("read MF HTML: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dstMF, "index.html"), data, 0o644); err != nil {
		return fmt.Errorf("write MF HTML: %w", err)
	}

	return nil
}

func copyDirRecursive(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return nil
		}
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

// ---------------------------------------------------------------------------
// Port checking
// ---------------------------------------------------------------------------

func isPortOpen(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		ln.Close()
		return false
	}
	return true
}

func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortOpen(port) {
			// Double-check with actual connection
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
			if err == nil {
				conn.Close()
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}

// ---------------------------------------------------------------------------
// Seed demo data
// ---------------------------------------------------------------------------

func seedData(wsRoot, component string) {
	dp := dcPort()
	mp := mfPort()
	apiBase := fmt.Sprintf("http://127.0.0.1:%d", dp)
	mfBase := fmt.Sprintf("http://127.0.0.1:%d", mp)

	seedKeys := []struct {
		key, val string
	}{
		{"kpi.revenue", "1284500"},
		{"kpi.users", "48291"},
		{"kpi.conversion", "3.8"},
		{"kpi.latency", "142"},
		{"kpi.trend", "12.3"},
		{"alert.status", "healthy"},
		{"alert.count", "0"},
		{"table.users", `[{"id":1,"name":"Alice Chen","email":"alice@example.com","status":"active","score":98},{"id":2,"name":"Bob Kim","email":"bob@example.com","status":"active","score":85},{"id":3,"name":"Carol Wu","email":"carol@example.com","status":"inactive","score":72},{"id":4,"name":"David Lee","email":"david@example.com","status":"active","score":91}]`},
	}
	for _, s := range seedKeys {
		body := fmt.Sprintf(`{"key":%q,"value":%q}`, s.key, s.val)
		http.Post(apiBase+"/api/dc/set", "application/json", strings.NewReader(body))
	}

	mfBody := fmt.Sprintf(`{"name":%q}`, component)
	http.Post(mfBase+"/api/mf/components/register", "application/json", strings.NewReader(mfBody))
}

// ---------------------------------------------------------------------------
// Bootstrap handler
// ---------------------------------------------------------------------------

func mfBootstrapHandler(bgMgr *background.Manager) Handler {
	return func(arguments string) string {
		return mfBootstrapHandlerImpl(arguments, bgMgr)
	}
}

func mfBootstrapHandlerImpl(arguments string, bgMgr *background.Manager) string {
	wsRoot, _ := os.Getwd()

	// Parse optional component name
	var args struct {
		Component string `json:"component"`
	}
	if arguments != "" {
		json.Unmarshal([]byte(arguments), &args)
	}
	component := args.Component
	if component == "" {
		component = "Dashboard"
	}

	dp := dcPort()
	mp := mfPort()

	// 1. Install from skill references to workspace-local dirs
	if err := installSpecPilot(wsRoot); err != nil {
		return fmt.Sprintf(`{"error": "install failed: %v"}`, err)
	}

	// 2. Start DC API server
	dcRunning := isPortOpen(dp)
	if !dcRunning {
		apiScript := filepath.Join(wsInstall(wsRoot), "api_server.py")
		cmd := exec.Command("python", apiScript)
		cmd.Dir = wsInstall(wsRoot)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Sprintf(`{"error": "start DC failed: %v"}`, err)
		}
	}

	// 3. Start MF dev server
	mfRunning := isPortOpen(mp)
	if !mfRunning {
		cmd := exec.Command("python", "-m", "http.server", strconv.Itoa(mp))
		cmd.Dir = wsMF(wsRoot)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Start()
	}

	// 4. Wait for services
	waitForPort(dp, 10*time.Second)
	waitForPort(mp, 10*time.Second)

	// 5. Seed demo data
	seedData(wsRoot, component)

	mfUrl := fmt.Sprintf("http://localhost:%d/#/%s", mp, component)
	return fmt.Sprintf(`{
		"ok": true,
		"dcPort": %d,
		"mfPort": %d,
		"dcUrl": "http://127.0.0.1:%d",
		"mfUrl": "http://localhost:%d",
		"mfRemoteUrl": "%s",
		"installDir": "%s",
		"message": "SpecPilot services bootstrapped in workspace .specpilot/"
	}`, dp, mp, dp, mp, mfUrl, wsInstall(wsRoot))
}

// ---------------------------------------------------------------------------
// Status check — lightweight, safe to call frequently
// ---------------------------------------------------------------------------

func mfStatusHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	dp, mp := dcPort(), mfPort()
	installed := false
	if _, err := os.Stat(wsInstall(wsRoot)); err == nil {
		installed = true
	}
	return fmt.Sprintf(`{
		"installed": %t,
		"dcRunning": %t,
		"mfRunning": %t,
		"dcPort": %d,
		"mfPort": %d,
		"dcUrl": "http://127.0.0.1:%d",
		"mfUrl": "http://localhost:%d"
	}`,
		installed,
		isPortOpen(dp),
		isPortOpen(mp),
		dp, mp, dp, mp,
	)
}

// ---------------------------------------------------------------------------
// Existing CLI wrappers
// ---------------------------------------------------------------------------

func runSP(wsRoot string, args ...string) string {
	cmd := exec.Command("python", append([]string{"-m", "cli"}, args...)...)
	cmd.Dir = wsInstall(wsRoot)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf(`{"error": "specpilot CLI failed: %v"}`, err)
	}
	return strings.TrimSpace(string(out))
}

func dcListHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "dc", "list")
}

func dcGetHandler(arguments string) string {
	var args struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: key required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "dc", "get", args.Key)
}

func dcSetHandler(arguments string) string {
	var args struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: key and value required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "dc", "set", args.Key, fmt.Sprintf("%v", args.Value))
}

func ecHistoryHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "ec", "history")
}

func ecEmitHandler(arguments string) string {
	var args struct {
		Event   string `json:"event"`
		Payload any    `json:"payload"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: event and payload required"}`
	}
	wsRoot, _ := os.Getwd()
	payloadJSON, _ := json.Marshal(args.Payload)
	return runSP(wsRoot, "ec", "emit", args.Event, string(payloadJSON))
}

func ecSubscribeHandler(arguments string) string {
	var args struct {
		Event      string `json:"event"`
		Subscriber string `json:"subscriber"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: event and subscriber required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "ec", "subscribe", args.Event, args.Subscriber)
}

func adListHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "ad", "list")
}

func adSwitchHandler(arguments string) string {
	var args struct {
		Adapter string `json:"adapter"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: adapter name required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "ad", "switch", args.Adapter)
}

func adQueryHandler(arguments string) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: query required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "ad", "query", args.Query)
}

func specListHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "spec", "list")
}

func specGetHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "spec", "get", args.Name)
}

func specBindingHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "spec", "binding", args.Name)
}

func mfListHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "mf", "list")
}

func mfRegisterHandler(arguments string) string {
	var args struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: name and path required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "mf", "register", args.Name, args.Path)
}

func mfResolveHandler(arguments string) string {
	var args struct {
		Spec string `json:"spec"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error": "invalid args: spec name required"}`
	}
	wsRoot, _ := os.Getwd()
	return runSP(wsRoot, "mf", "resolve-from-spec", args.Spec)
}

// ── Exported for HTTP handlers (rtools package) ──────────────────────────────

// SpecpilotDCPort returns the configured DC port
func SpecpilotDCPort() int { return dcPort() }

// SpecpilotMFPort returns the configured MF port
func SpecpilotMFPort() int { return mfPort() }

// SpecpilotServiceStatus returns (installed, dcRunning, mfRunning)
func SpecpilotServiceStatus(wsRoot string) (bool, bool, bool) {
	dp := dcPort()
	mp := mfPort()
	installed := false
	if _, err := os.Stat(wsInstall(wsRoot)); err == nil {
		installed = true
	}
	return installed, isPortOpen(dp), isPortOpen(mp)
}

// SpecpilotBootstrap runs the full bootstrap and returns result map
func SpecpilotBootstrap(wsRoot, component string, dp, mp int) map[string]any {
	component2 := component
	if component2 == "" {
		component2 = "Dashboard"
	}

	// Set env overrides for port config and workspace root
	os.Setenv("SPECPILOT_DC_PORT", strconv.Itoa(dp))
	os.Setenv("SPECPILOT_MF_PORT", strconv.Itoa(mp))
	os.Setenv("SPECPILOT_WORKSPACE_ROOT", wsRoot)

	// 1. Install
	if err := installSpecPilot(wsRoot); err != nil {
		return map[string]any{"ok": false, "error": fmt.Sprintf("install failed: %v", err)}
	}

	// 2. Start DC API
	if !isPortOpen(dp) {
		apiScript := filepath.Join(wsInstall(wsRoot), "api_server.py")
		cmd := exec.Command("python", apiScript)
		cmd.Dir = wsInstall(wsRoot)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Start()
	}

	// 3. Start MF dev server
	if !isPortOpen(mp) {
		cmd := exec.Command("python", "-m", "http.server", strconv.Itoa(mp))
		cmd.Dir = wsMF(wsRoot)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Start()
	}

	// 4. Wait for services
	waitForPort(dp, 10*time.Second)
	waitForPort(mp, 10*time.Second)

	// 5. Write services.json so frontend can read workspace-local ports
	writeServicesJson(wsRoot, dp, mp)

	// 6. Seed demo data
	seedData(wsRoot, component2)

	mfUrl := fmt.Sprintf("http://localhost:%d/#/%s", mp, component2)
	return map[string]any{
		"ok":          true,
		"dcPort":      dp,
		"mfPort":      mp,
		"dcUrl":       fmt.Sprintf("http://127.0.0.1:%d", dp),
		"mfUrl":       fmt.Sprintf("http://localhost:%d", mp),
		"mfRemoteUrl": mfUrl,
		"message":     "SpecPilot bootstrapped",
	}
}

// writeServicesJson writes current service ports to workspace-local .specpilot/services.json
func writeServicesJson(wsRoot string, dcPort, mfPort int) {
	specPilotDir := filepath.Join(wsRoot, ".specpilot")
	os.MkdirAll(specPilotDir, 0o755)
	path := filepath.Join(specPilotDir, "services.json")
	content := fmt.Sprintf(`{"dcPort":%d,"mfPort":%d,"workspace":"%s"}`, dcPort, mfPort, wsRoot)
	os.WriteFile(path, []byte(content), 0o644)
}
