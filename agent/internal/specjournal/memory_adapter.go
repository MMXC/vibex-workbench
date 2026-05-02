// agent/internal/specjournal/memory_adapter.go — Bridge between journal checkpoints and memplace/memlace.
// SLICE-memplace-implementation-memory-adapter
package specjournal

import (
	"fmt"
	"log"
)

// MemoryAdapter bridges spec journal checkpoints to memplace/memlace memory stores.
type MemoryAdapter struct {
	workspaceDir string
}

// NewMemoryAdapter creates a new MemoryAdapter.
func NewMemoryAdapter(workspaceDir string) *MemoryAdapter {
	return &MemoryAdapter{workspaceDir: workspaceDir}
}

// NotifyCheckpoint sends a checkpoint summary to memory after a journal write.
func (a *MemoryAdapter) NotifyCheckpoint(specName string, event CheckpointEvent) error {
	// Build a concise summary for the memory store
	summary := fmt.Sprintf("[%s] %s implementation checkpoint: %s (agent: %s)",
		event.Timestamp, specName, event.Action, event.Agent)
	if event.Category != "" {
		summary += fmt.Sprintf(" | category: %s", event.Category)
	}
	if event.Result != "" {
		summary += fmt.Sprintf(" | result: %s", event.Result)
	}

	log.Printf("[memory-adapter] checkpoint summary: %s", summary)

	// In a full implementation, this would call memplace/memlace store API.
	// For now, log the intent. The actual store integration is deferred.
	return nil
}