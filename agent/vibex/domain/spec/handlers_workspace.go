package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rt "vibex/agent/agents/runtime/tools"
)

// ─────────────────────────────────────────────────────────────
// Detect State Types
// ─────────────────────────────────────────────────────────────

type DetectionSignal struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Reason string `json:"reason,omitempty"`
}

type DetectStateResult struct {
	State       string              `json:"state"` // "empty" | "partial" | "ready"
	Signals     []DetectionSignal  `json:"signals"`
	Suggestions []string           `json:"suggestions"`
	Timestamp   string             `json:"timestamp"`
}

// ─────────────────────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────────────────────

func MakeWorkspaceDetectStateHandler(workspaceDir string, setStepType func(threadID, stepType string)) rt.Handler {
	return func(arguments string) string {
		if setStepType != nil {
			setStepType("", "workspace-detect")
		}

		// Parse optional workspace_dir override
		targetDir := workspaceDir
		if arguments != "" {
			var args struct {
				WorkspaceDir string `json:"workspace_dir"`
			}
			if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.WorkspaceDir != "" {
				targetDir = args.WorkspaceDir
			}
		}

		result := detectState(targetDir)

		// Emit SSE canvas event if broadcaster available
		// (broadcaster not available in this handler factory, emit via return message)

		return formatDetectResult(result)
	}
}

func detectState(dir string) DetectStateResult {
	signals := []DetectionSignal{}
	suggestions := []string{}
	timestamp := time.Now().Format(time.RFC3339)

	// Signal 1: specs/ directory
	specsDir := filepath.Join(dir, "specs")
	specsExists := dirExists(specsDir)
	signals = append(signals, DetectionSignal{
		Path:   "specs/",
		Exists: specsExists,
		Reason: boolToReason(specsExists, "specs 目录存在", "无 specs 目录"),
	})

	// Signal 2: generators/gen.py
	genFile := filepath.Join(dir, "generators", "gen.py")
	genExists := fileExists(genFile)
	signals = append(signals, DetectionSignal{
		Path:   "generators/gen.py",
		Exists: genExists,
		Reason: boolToReason(genExists, "生成器入口存在", "无生成器入口"),
	})

	// Signal 3: Makefile with lint-specs target
	makefile := filepath.Join(dir, "Makefile")
	makefileExists := fileExists(makefile)
	makefileHasLint := false
	if makefileExists {
		makefileHasLint = makefileHasTarget(makefile, "lint-specs")
	}
	signals = append(signals, DetectionSignal{
		Path:   "Makefile (lint-specs)",
		Exists: makefileHasLint,
		Reason: boolToReason(makefileHasLint, "Makefile 含 lint-specs", "Makefile 缺少 lint-specs"),
	})

	// Signal 4: frontend/package.json
	pkgJSON := filepath.Join(dir, "frontend", "package.json")
	pkgExists := fileExists(pkgJSON)
	signals = append(signals, DetectionSignal{
		Path:   "frontend/package.json",
		Exists: pkgExists,
		Reason: boolToReason(pkgExists, "前端 package.json 存在", "无前端配置"),
	})

	// Determine state
	var state string
	specsScore := 0
	if specsExists {
		specsScore++
	}
	if genExists {
		specsScore++
	}
	if makefileHasLint {
		specsScore++
	}
	if pkgExists {
		specsScore++
	}

	switch {
	case specsScore == 0:
		state = "empty"
		suggestions = []string{
			"点击「初始化脚手架」开始搭建项目",
			"或描述你想要的产品，agent 将引导你完成初始化",
		}
	case specsScore < 4:
		state = "partial"
		if !specsExists {
			suggestions = append(suggestions, "specs 目录缺失")
		}
		if !genExists {
			suggestions = append(suggestions, "生成器未配置，运行「生成脚手架」补全")
		}
		if !makefileHasLint {
			suggestions = append(suggestions, "Makefile 缺少校验 target")
		}
		suggestions = append(suggestions, "运行「生成脚手架」补全剩余部分")
	default:
		state = "ready"
		suggestions = []string{
			"在 spec 编辑器中打开或新建规格文件",
			"运行「校验 Spec」检查 spec 质量",
			"编写 L1–L4 spec 后运行「生成代码」",
		}
	}

	return DetectStateResult{
		State:       state,
		Signals:     signals,
		Suggestions: suggestions,
		Timestamp:   timestamp,
	}
}

func formatDetectResult(r DetectStateResult) string {
	stateIcon := map[string]string{
		"empty":   "⚪",
		"partial": "🟡",
		"ready":   "🟢",
	}[r.State]

	stateCN := map[string]string{
		"empty":   "空（未初始化）",
		"partial": "半成品（缺组件）",
		"ready":   "就绪（可开发）",
	}[r.State]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s 状态: %s (%s)\n\n", stateIcon, r.State, stateCN))
	sb.WriteString("检测信号:\n")
	for _, s := range r.Signals {
		status := "❌"
		if s.Exists {
			status = "✅"
		}
		sb.WriteString(fmt.Sprintf("  %s %s — %s\n", status, s.Path, s.Reason))
	}
	sb.WriteString("\n建议:\n")
	for i, suggestion := range r.Suggestions {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
	}
	sb.WriteString(fmt.Sprintf("\n时间: %s", r.Timestamp))

	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func boolToReason(value bool, trueReason, falseReason string) string {
	if value {
		return trueReason
	}
	return falseReason
}

func makefileHasTarget(path, target string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// .PHONY: targets 或 targets: pattern
		if strings.HasPrefix(trimmed, ".PHONY:") {
			if strings.Contains(trimmed, target) {
				return true
			}
		}
		if strings.HasPrefix(trimmed, target+":") {
			return true
		}
	}
	return false
}
