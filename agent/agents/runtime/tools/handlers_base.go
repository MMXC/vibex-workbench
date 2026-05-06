package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	path = normalizePathAgainstWorkspace(path)
	safe, err := safeWorkspacePath(path)
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

func writeFileHandler(arguments string) string {
	path, content, err := parseWriteFileArgs(arguments)
	if err != nil {
		return "invalid args: " + err.Error()
	}
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
	if err := os.WriteFile(safe, []byte(content), 0o644); err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("ok: wrote %d bytes to %s", len(content), path)
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
