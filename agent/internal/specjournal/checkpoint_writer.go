// agent/internal/specjournal/checkpoint_writer.go — Append-only checkpoint writer for implementation journals.
// SLICE-agent-checkpoint-writer
package specjournal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckpointEvent represents an agent step event to be logged.
type CheckpointEvent struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Agent     string `json:"agent"`
	Action    string `json:"action"`
	Files     []string `json:"files,omitempty"`
	Result    string `json:"result,omitempty"`
	NextAction string `json:"next_action,omitempty"`
	Category  string `json:"category,omitempty"` // from validation-and-failure.md
}

// CheckpointWriter appends checkpoints to a spec's implementation journal.
type CheckpointWriter struct {
	workspaceDir string
}

// NewCheckpointWriter creates a new CheckpointWriter.
func NewCheckpointWriter(workspaceDir string) *CheckpointWriter {
	return &CheckpointWriter{workspaceDir: workspaceDir}
}

// JournalPath returns the path to the implementation journal for a given spec name.
func (w *CheckpointWriter) JournalPath(specName string) (string, error) {
	safeName := sanitizeSpecName(specName)
	journalDir := filepath.Join(w.workspaceDir, "specs", "journal")
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir journal dir: %w", err)
	}
	return filepath.Join(journalDir, safeName+".implementation.yaml"), nil
}

// WriteCheckpoint appends a checkpoint to the journal for the given spec.
func (w *CheckpointWriter) WriteCheckpoint(specName string, event CheckpointEvent) error {
	path, err := w.JournalPath(specName)
	if err != nil {
		return err
	}

	// Ensure path is within workspaceDir
	absPath, _ := filepath.Abs(path)
	absRoot, _ := filepath.Abs(w.workspaceDir)
	if !strings.HasPrefix(absPath, absRoot) {
		return fmt.Errorf("path traversal detected")
	}

	// Ensure timestamp
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("cp-%d", time.Now().UnixNano())
	}

	// Atomic write: write to temp file then rename
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}

	// Append as YAML comment-style block
	entry := fmt.Sprintf("# checkpoint: %s\n", event.Timestamp)
	entry += fmt.Sprintf("# id: %s | agent: %s | action: %s\n", event.ID, event.Agent, event.Action)
	if event.Category != "" {
		entry += fmt.Sprintf("# category: %s\n", event.Category)
	}
	if len(event.Files) > 0 {
		entry += fmt.Sprintf("# files: %s\n", strings.Join(event.Files, ", "))
	}
	if event.Result != "" {
		entry += fmt.Sprintf("# result: %s\n", event.Result)
	}
	if event.NextAction != "" {
		entry += fmt.Sprintf("# next_action: %s\n", event.NextAction)
	}
	entry += "---\n"

	if _, err := f.WriteString(entry); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write tmp: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// sanitizeSpecName removes characters that could be used for path traversal.
func sanitizeSpecName(name string) string {
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}