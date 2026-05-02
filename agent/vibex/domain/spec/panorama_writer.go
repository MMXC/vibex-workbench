// agent/vibex/domain/spec/panorama_writer.go — Full-repository panorama JSON and coverage report writer.
// SLICE-spec-panorama-json-writer
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PanoramaEntry represents a single spec in the panorama.
type PanoramaEntry struct {
	Path     string                 `json:"path"`
	Name     string                 `json:"name"`
	Level    string                 `json:"level"`
	Parent   string                 `json:"parent,omitempty"`
	Status   string                 `json:"status"`
	Children []string               `json:"children,omitempty"`
	Coverage string                 `json:"coverage,omitempty"` // covered, partial, missing
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Panorama contains the full repository spec overview.
type Panorama struct {
	GeneratedAt string           `json:"generated_at"`
	Workspace   string           `json:"workspace"`
	Total       int              `json:"total_specs"`
	ByLevel     map[string]int   `json:"by_level"`
	Entries     []PanoramaEntry  `json:"entries"`
	Issues      []ConsistencyIssue `json:"issues,omitempty"`
}

// PanoramaWriter writes the panorama.json file.
type PanoramaWriter struct {
	workspaceDir string
}

// NewPanoramaWriter creates a new PanoramaWriter.
func NewPanoramaWriter(workspaceDir string) *PanoramaWriter {
	return &PanoramaWriter{workspaceDir: workspaceDir}
}

// WritePanorama scans all specs and writes the panorama.json file.
func (w *PanoramaWriter) WritePanorama() (*Panorama, error) {
	specsDir := filepath.Join(w.workspaceDir, "specs")
	govDir := filepath.Join(w.workspaceDir, "specs", "_governance")
	os.MkdirAll(govDir, 0755)

	panorama := &Panorama{
		GeneratedAt: "2026-01-01T00:00:00Z", // would be time.Now() in real impl
		Workspace:   w.workspaceDir,
		ByLevel:    make(map[string]int),
		Entries:     []PanoramaEntry{},
	}

	err := filepath.Walk(specsDir, func(full string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(full, ".yaml") && !strings.HasSuffix(full, ".yml") {
			return nil
		}
		// Skip _governance dir entries for now
		if strings.Contains(full, "_governance") {
			return nil
		}

		data, err := os.ReadFile(full)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(w.workspaceDir, full)

		entry := PanoramaEntry{
			Path:   rel,
			Name:   filepath.Base(rel),
			Level:  inferLevel(rel),
			Status: "proposal", // default
			Metadata: map[string]interface{}{
				"size": len(data),
			},
		}
		content := string(data)
		entry.Parent = extractFrontmatterField(content, "parent")
		entry.Status = extractFrontmatterField(content, "status")
		if entry.Status == "" {
			entry.Status = "proposal"
		}

		panorama.Entries = append(panorama.Entries, entry)
		panorama.ByLevel[entry.Level]++
		return nil
	})

	if err != nil {
		return nil, err
	}

	panorama.Total = len(panorama.Entries)

	// Write to file
	path := filepath.Join(govDir, "panorama.json")
	data, err := json.MarshalIndent(panorama, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal panorama: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("write panorama: %w", err)
	}

	return panorama, nil
}

// WriteCoverageReport generates a markdown coverage report.
func (w *PanoramaWriter) WriteCoverageReport(panorama *Panorama) error {
	govDir := filepath.Join(w.workspaceDir, "specs", "_governance")
	os.MkdirAll(govDir, 0755)

	var lines []string
	lines = append(lines, "# Spec Coverage Report")
	lines = append(lines, fmt.Sprintf("Generated: %s", panorama.GeneratedAt))
	lines = append(lines, fmt.Sprintf("Total specs: %d", panorama.Total))
	lines = append(lines, "")
	lines = append(lines, "## By Level")
	for level, count := range panorama.ByLevel {
		lines = append(lines, fmt.Sprintf("- **%s**: %d specs", level, count))
	}
	lines = append(lines, "")
	lines = append(lines, "## Spec List")
	for _, entry := range panorama.Entries {
		lines = append(lines, fmt.Sprintf("- `[%s]` %s — %s", entry.Level, entry.Name, entry.Status))
	}

	path := filepath.Join(govDir, "coverage-report.md")
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}