package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vibex-workbench/pkg/vibexpaths"
	"vibex/agent/internal/shellutil"
)

func bashHandler(arguments string) string {
	cmd, err := parseCommand(arguments)
	if err != nil {
		return "invalid args: " + err.Error()
	}
	fmt.Printf("Tool use: bash %s\n", cmd)
	return runBash(cmd)
}

func readFileHandler(arguments string) string {
	path, limit, err := parseReadFileArgs(arguments)
	if err != nil {
		return "invalid args: " + err.Error()
	}
	safe, err := resolveReadablePath(path)
	if err != nil {
		return "invalid path: " + err.Error()
	}
	data, err := os.ReadFile(safe)
	if err != nil {
		return "error: " + err.Error()
	}
	if limit <= 0 {
		limit = 10000
	}
	if limit > 50000 {
		limit = 50000
	}
	if len(data) > limit {
		return string(data[:limit]) + fmt.Sprintf("\n\n(truncated: %d/%d bytes)", limit, len(data))
	}
	return string(data)
}

func resolveReadablePath(path string) (string, error) {
	normalized := normalizePathAgainstWorkspace(path)
	if safe, err := safeWorkspacePath(normalized); err == nil {
		return safe, nil
	}
	return resolvePathAgainstSkills(path)
}

// resolvePathAgainstSkills supports relative paths requested by skill instructions,
// e.g. "html-ppt/references/themes.md" -> "<skills-dir>/html-ppt/references/themes.md".
func resolvePathAgainstSkills(path string) (string, error) {
	p := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if p == "." || p == "" || strings.HasPrefix(p, "../") || strings.Contains(p, "/../") {
		return "", fmt.Errorf("invalid path")
	}
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("path must stay inside workspace")
	}
	skillName := parts[0]
	rel := filepath.FromSlash(parts[1])

	roots := skillsRootCandidates()
	if len(roots) == 0 {
		return "", fmt.Errorf("path must stay inside workspace")
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(root, skillName, rel))
		rootClean := filepath.Clean(root)
		relToRoot, err := filepath.Rel(rootClean, candidate)
		if err != nil {
			continue
		}
		if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("path must stay inside workspace")
}

func writeFileHandler(arguments string) string {
	path, content, err := parseWriteFileArgs(arguments)
	if err != nil {
		return "invalid args: " + err.Error()
	}
	return writeOrAppendFile(path, content, false)
}

func appendFileHandler(arguments string) string {
	path, content, err := parseAppendFileArgs(arguments)
	if err != nil {
		return "invalid args: " + err.Error()
	}
	return writeOrAppendFile(path, content, true)
}

func writeOrAppendFile(path, content string, appendMode bool) string {
	path = normalizePathAgainstWorkspace(path)
	safe, err := safeWorkspacePath(path)
	if err != nil {
		return "invalid path: " + err.Error()
	}
	dir := filepath.Dir(safe)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "error: " + err.Error()
		}
	}
	if appendMode {
		f, err := os.OpenFile(safe, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return "error: " + err.Error()
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return "error: " + err.Error()
		}
	} else {
		if err := os.WriteFile(safe, []byte(content), 0o644); err != nil {
			return "error: " + err.Error()
		}
	}
	// Read-back verification to detect partial/truncated writes immediately.
	info, statErr := os.Stat(safe)
	if statErr != nil {
		return "error: verify failed after write: " + statErr.Error()
	}
	mode := "write"
	if appendMode {
		mode = "append"
	}
	return fmt.Sprintf("ok: %s %d bytes to %s (file_size=%d)", mode, len(content), path, info.Size())
}

// normalizePathAgainstWorkspace strips workspace root from absolute paths (models often pass drive paths on Windows).
func normalizePathAgainstWorkspace(path string) string {
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(filepath.Clean(path))
	}
	cp := filepath.Clean(path)
	for _, root := range workspaceRootCandidates() {
		r := filepath.Clean(root)
		rel, err := filepath.Rel(r, cp)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return filepath.ToSlash(rel)
	}
	return path
}

func workspaceRootCandidates() []string {
	var out []string
	for _, k := range []string{"WORKSPACE_ROOT", "WORKSPACE_DIR"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func skillsRootCandidates() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		v = filepath.Clean(v)
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	add(os.Getenv("SKILLS_DIR"))
	for _, root := range workspaceRootCandidates() {
		add(filepath.Join(root, "skills"))
		add(filepath.Join(root, filepath.FromSlash(vibexpaths.AgentsRootRel), "skills"))
		add(filepath.Join(root, filepath.FromSlash(vibexpaths.AgentsDotAgentsRootRel), "skills"))
	}
	return out
}

func safeWorkspacePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute path is not allowed")
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid path")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return clean, nil
}

func runBash(command string) string {
	if err := validateBashCommand(command); err != nil {
		return "blocked: " + err.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sh, err := shellutil.ResolvePOSIXShell()
	if err != nil {
		return "error: " + err.Error()
	}
	cmd := exec.CommandContext(ctx, sh, "-lc", command)
	cmd.Dir = shellutil.WorkingDir()
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "error: command timeout (30s)"
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		text = "(no output)"
	}
	if err != nil {
		return "error: " + err.Error() + "\n" + text
	}
	return truncateOutput(text)
}

func validateBashCommand(command string) error {
	blocked := []string{"rm -rf /", "shutdown", "reboot", "mkfs", ":(){:|:&};:"}
	for _, banned := range blocked {
		if strings.Contains(command, banned) {
			return fmt.Errorf("dangerous command")
		}
	}
	return nil
}

func truncateOutput(text string) string {
	if len(text) > 50000 {
		return text[:50000]
	}
	return text
}
