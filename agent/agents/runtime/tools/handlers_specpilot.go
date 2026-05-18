package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Path constants — resolved relative to workspace root
// ---------------------------------------------------------------------------

// wsInstall returns the workspace-local .specpilot dir
func wsInstall(wsRoot string) string {
	return filepath.Join(wsRoot, ".specpilot")
}

// prototypesDir returns {wsRoot}/.specpilot/prototypes/
func prototypesDir(wsRoot string) string {
	return filepath.Join(wsRoot, ".specpilot", "prototypes")
}

// ---------------------------------------------------------------------------
// Bootstrap — file-based, no Python services
// ---------------------------------------------------------------------------

func mfBootstrapHandler(bgMgr any) Handler {
	return func(arguments string) string {
		return mfBootstrapHandlerImpl(arguments)
	}
}

func mfBootstrapHandlerImpl(arguments string) string {
	wsRoot, _ := os.Getwd()

	// Parse optional spec-id
	var args struct {
		SpecID string `json:"specId"`
	}
	if arguments != "" {
		json.Unmarshal([]byte(arguments), &args)
	}

	// Ensure .specpilot/prototypes/ directory exists (idempotent)
	protoDir := prototypesDir(wsRoot)
	os.MkdirAll(protoDir, 0o755)

	// Write services.json so frontend knows workspace config
	writeServicesJson(wsRoot)

	protoDirRel := filepath.Join(".specpilot", "prototypes")
	return fmt.Sprintf(`{
  "ok": true,
  "protoDir": "%s",
  "protoDirRel": "%s",
  "message": "SpecPilot initialized — no backend services required"
}`, protoDir, protoDirRel)
}

// ---------------------------------------------------------------------------
// Status — file-based: check directory existence
// ---------------------------------------------------------------------------

func mfStatusHandler(arguments string) string {
	wsRoot, _ := os.Getwd()
	protoDir := prototypesDir(wsRoot)
	exists := false
	if fi, err := os.Stat(protoDir); err == nil && fi.IsDir() {
		exists = true
	}

	// Count prototype files
	protoCount := 0
	if exists {
		entries, _ := os.ReadDir(protoDir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".html") {
				protoCount++
			}
		}
	}

	return fmt.Sprintf(`{
  "ok": true,
  "installed": %t,
  "protoDir": "%s",
  "protoCount": %d,
  "mode": "file-based"
}`, exists, protoDir, protoCount)
}

// ---------------------------------------------------------------------------
// Prototype operations (used by Go HTTP server and agent tools)
// ---------------------------------------------------------------------------

// PrototypeEntry describes one prototype file
type PrototypeEntry struct {
	SpecID    string `json:"specId"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updatedAt"`
}

// PrototypeList returns all .html files in prototypes/
func PrototypeList(wsRoot string) []PrototypeEntry {
	dir := prototypesDir(wsRoot)
	var out []PrototypeEntry
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		specID := strings.TrimSuffix(e.Name(), ".html")
		fi, _ := e.Info()
		out = append(out, PrototypeEntry{
			SpecID:    specID,
			Path:      filepath.Join(dir, e.Name()),
			Size:      fi.Size(),
			UpdatedAt: fi.ModTime().Format(time.RFC3339),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

// PrototypeGet reads a single prototype file
func PrototypeGet(wsRoot, specID string) (html string, exists bool) {
	path := filepath.Join(prototypesDir(wsRoot), specID+".html")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// PrototypeWrite writes a prototype HTML file
func PrototypeWrite(wsRoot, specID, html string) error {
	os.MkdirAll(prototypesDir(wsRoot), 0o755)
	path := filepath.Join(prototypesDir(wsRoot), specID+".html")
	return os.WriteFile(path, []byte(html), 0o644)
}

// ---------------------------------------------------------------------------
// Exported for Go HTTP server (rtools package)
// ---------------------------------------------------------------------------

// SpecpilotDCPort returns the configured DC port (still here for API compat)
func SpecpilotDCPort() int { return 7890 }

// SpecpilotMFPort returns the configured MF port (still here for API compat)
func SpecpilotMFPort() int { return 5177 }

// SpecpilotServiceStatus returns (installed, dcRunning, mfRunning)
// For file-based mode: installed = .specpilot/ exists, services always true
func SpecpilotServiceStatus(wsRoot string) (bool, bool, bool) {
	installed := false
	if _, err := os.Stat(wsInstall(wsRoot)); err == nil {
		installed = true
	}
	return installed, true, true // services always "running" in file-based mode
}

// SpecpilotBootstrap runs bootstrap — file-based version
func SpecpilotBootstrap(wsRoot, component string, dp, mp int) map[string]any {
	protoDir := prototypesDir(wsRoot)
	os.MkdirAll(protoDir, 0o755)
	writeServicesJson(wsRoot)

	return map[string]any{
		"ok":        true,
		"mode":      "file-based",
		"protoDir":  protoDir,
		"message":   "SpecPilot file-based mode — no backend services",
	}
}

// writeServicesJson writes workspace service config
func writeServicesJson(wsRoot string) {
	path := filepath.Join(wsInstall(wsRoot), "services.json")
	content := fmt.Sprintf(`{"mode":"file-based","workspace":"%s"}`, wsRoot)
	os.WriteFile(path, []byte(content), 0o644)
}

// ---------------------------------------------------------------------------
// Legacy CLI wrappers — return deprecated message (Python CLI removed)
// ---------------------------------------------------------------------------

func runSP(wsRoot string, args ...string) string {
	return `{"deprecated": true, "message": "Python CLI removed — use file-based prototype workflow"}`
}

func dcListHandler(arguments string) string     { return `{"deprecated": true}` }
func dcGetHandler(arguments string) string       { return `{"deprecated": true}` }
func dcSetHandler(arguments string) string      { return `{"deprecated": true}` }
func ecHistoryHandler(arguments string) string   { return `{"deprecated": true}` }
func ecEmitHandler(arguments string) string      { return `{"deprecated": true}` }
func ecSubscribeHandler(arguments string) string { return `{"deprecated": true}` }
func adListHandler(arguments string) string      { return `{"deprecated": true}` }
func adSwitchHandler(arguments string) string    { return `{"deprecated": true}` }
func adQueryHandler(arguments string) string    { return `{"deprecated": true}` }
func specListHandler(arguments string) string   { return `{"deprecated": true}` }
func specGetHandler(arguments string) string    { return `{"deprecated": true}` }
func specBindingHandler(arguments string) string { return `{"deprecated": true}` }
func mfListHandler(arguments string) string     { return `{"deprecated": true}` }
func mfRegisterHandler(arguments string) string { return `{"deprecated": true}` }
func mfResolveHandler(arguments string) string  { return `{"deprecated": true}` }
