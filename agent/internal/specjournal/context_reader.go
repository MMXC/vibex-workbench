// agent/internal/specjournal/context_reader.go — Reads and injects implementation journal context.
// SLICE-agent-context-injection-reader
package specjournal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContextEntry is a parsed checkpoint entry for injection.
type ContextEntry struct {
	ID        string
	Timestamp string
	Agent     string
	Action    string
	Category  string
	Files     []string
	Result    string
	NextAction string
}

// ContextReader reads implementation journals to build agent context.
type ContextReader struct {
	workspaceDir string
}

// NewContextReader creates a new ContextReader.
func NewContextReader(workspaceDir string) *ContextReader {
	return &ContextReader{workspaceDir: workspaceDir}
}

// ReadJournal reads the journal for a spec and returns checkpoint entries.
func (r *ContextReader) ReadJournal(specName string) ([]ContextEntry, error) {
	safeName := sanitizeSpecName(specName)
	journalPath := filepath.Join(r.workspaceDir, "specs", "journal", safeName+".implementation.yaml")

	// Check path is within workspace
	absPath, _ := filepath.Abs(journalPath)
	absRoot, _ := filepath.Abs(r.workspaceDir)
	if !strings.HasPrefix(absPath, absRoot) {
		return nil, fmt.Errorf("path traversal detected")
	}

	data, err := os.ReadFile(journalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no journal yet
		}
		return nil, fmt.Errorf("read journal: %w", err)
	}

	return parseJournal(string(data)), nil
}

// BuildContextSummary creates a human-readable summary for agent injection.
func (r *ContextReader) BuildContextSummary(specName string) (string, error) {
	entries, err := r.ReadJournal(specName)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("## Implementation Journal: %s", specName))
	lines = append(lines, fmt.Sprintf("Total checkpoints: %d", len(entries)))
	lines = append(lines, "")
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("- [%s] %s (%s): %s", e.Timestamp, e.Action, e.Agent, e.Result))
		if e.Category != "" {
			lines = append(lines, fmt.Sprintf("  category: %s", e.Category))
		}
	}
	return strings.Join(lines, "\n"), nil
}

// parseJournal extracts checkpoint entries from YAML-like content.
func parseJournal(content string) []ContextEntry {
	var entries []ContextEntry
	lines := strings.Split(content, "\n")

	var current ContextEntry
	for _, line := range lines {
		if strings.HasPrefix(line, "# id:") {
			if current.ID != "" {
				entries = append(entries, current)
			}
			current = ContextEntry{}
			current.ID = strings.TrimPrefix(strings.TrimSpace(line), "# id:")
		}
		if strings.HasPrefix(line, "# timestamp:") {
			current.Timestamp = strings.TrimPrefix(strings.TrimSpace(line), "# timestamp:")
		}
		if strings.HasPrefix(line, "# agent:") {
			current.Agent = strings.TrimPrefix(strings.TrimSpace(line), "# agent:")
		}
		if strings.HasPrefix(line, "# action:") {
			current.Action = strings.TrimPrefix(strings.TrimSpace(line), "# action:")
		}
		if strings.HasPrefix(line, "# category:") {
			current.Category = strings.TrimPrefix(strings.TrimSpace(line), "# category:")
		}
		if strings.HasPrefix(line, "# files:") {
			files := strings.TrimPrefix(strings.TrimSpace(line), "# files:")
			current.Files = strings.Split(files, ",")
		}
		if strings.HasPrefix(line, "# result:") {
			current.Result = strings.TrimPrefix(strings.TrimSpace(line), "# result:")
		}
		if strings.HasPrefix(line, "# next_action:") {
			current.NextAction = strings.TrimPrefix(strings.TrimSpace(line), "# next_action:")
		}
	}
	if current.ID != "" {
		entries = append(entries, current)
	}
	return entries
}