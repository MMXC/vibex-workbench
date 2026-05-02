// agent/vibex/domain/spec/governance_auto_update.go — Auto-update governance view on spec mutations.
// SLICE-spec-governance-auto-update-hook
package spec

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// AutoUpdateHook triggers governance refresh on spec mutation events.
type AutoUpdateHook struct {
	workspaceDir string
	builder      *ViewBuilder
	panorama     *PanoramaWriter
	mu           sync.Mutex
	lastRun      time.Time
	stale        bool
}

// NewAutoUpdateHook creates a new AutoUpdateHook.
func NewAutoUpdateHook(workspaceDir string) *AutoUpdateHook {
	return &AutoUpdateHook{
		workspaceDir: workspaceDir,
		builder:      NewViewBuilder(workspaceDir),
		panorama:     NewPanoramaWriter(workspaceDir),
	}
}

// OnSpecMutated is called when a spec file is created or modified.
func (h *AutoUpdateHook) OnSpecMutated(specPath string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("[governance-auto-update] spec mutated: %s", specPath)

	// Mark governance as stale
	h.stale = true

	// Trigger async refresh
	go h.refresh()

	return nil
}

// Refresh runs the full governance refresh cycle.
func (h *AutoUpdateHook) Refresh() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.refreshLocked()
}

func (h *AutoUpdateHook) refreshLocked() error {
	log.Printf("[governance-auto-update] refreshing governance...")

	// Rebuild panorama
	_, err := h.panorama.WritePanorama()
	if err != nil {
		h.stale = true
		return fmt.Errorf("panorama write failed: %w", err)
	}

	h.stale = false
	h.lastRun = time.Now()
	log.Printf("[governance-auto-update] refresh complete")
	return nil
}

func (h *AutoUpdateHook) refresh() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.refreshLocked()
}

// IsStale returns whether the governance data is stale.
func (h *AutoUpdateHook) IsStale() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stale
}

// LastRun returns the timestamp of the last successful refresh.
func (h *AutoUpdateHook) LastRun() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastRun
}

// WriteStaleMarker writes a stale marker file if governance is stale.
func (h *AutoUpdateHook) WriteStaleMarker() error {
	govDir := os.Args[0]
	if len(os.Args) == 0 {
		govDir = h.workspaceDir
	}
	markerPath := fmt.Sprintf("%s/specs/_governance/.stale_governance", h.workspaceDir)

	if h.IsStale() {
		return os.WriteFile(markerPath, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
	}
	// Remove marker if not stale
	os.Remove(markerPath)
	return nil
}