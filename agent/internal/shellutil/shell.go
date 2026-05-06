// Package shellutil resolves a POSIX shell for bash_bg / bash tool execution.
// On Windows, Git Bash is preferred over zsh (typically unavailable).
package shellutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EnvPOSIXShell overrides auto-detection when set to a shell executable (e.g. MSYS2 bash).
const EnvPOSIXShell = "VIBEX_POSIX_SHELL"

// ResolvePOSIXShell returns argv[0] for a shell that supports `-lc <command>` (bash/zsh).
func ResolvePOSIXShell() (string, error) {
	if v := strings.TrimSpace(os.Getenv(EnvPOSIXShell)); v != "" {
		return v, nil
	}
	if runtime.GOOS == "windows" {
		if p := findGitBash(); p != "" {
			return p, nil
		}
		return "", fmt.Errorf(
			"Git Bash not found (install Git for Windows, or set %s to the path of bash.exe)",
			EnvPOSIXShell,
		)
	}
	if p, err := exec.LookPath("bash"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("zsh"); err == nil {
		return p, nil
	}
	return "/bin/sh", nil
}

// WorkingDir is the directory used for bash / bash_bg (workspace root when configured).
func WorkingDir() string {
	for _, k := range []string{"WORKSPACE_ROOT", "WORKSPACE_DIR"} {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" {
			continue
		}
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	return "."
}

func findGitBash() string {
	bases := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		filepath.Join(os.Getenv("LocalAppData"), "Programs"),
	}
	for _, base := range bases {
		if base == "" {
			continue
		}
		cand := filepath.Join(base, "Git", "bin", "bash.exe")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand
		}
	}
	if p, err := exec.LookPath("bash.exe"); err == nil {
		return p
	}
	if p, err := exec.LookPath("bash"); err == nil {
		return p
	}
	return ""
}
