// Package vibexpaths — 工作区内 VibeX 约定路径（相对工作区根，POSIX 斜杠用于日志/JSON）。
// Spec 与 Agent 元数据统一落在 .vibex 下，避免依赖 Workbench 安装目录。
package vibexpaths

import (
	"path/filepath"
	"strings"
)

const (
	// SpecsRootRel 为规格树根目录（其下为 L1-goal … L5-slice、_governance）。
	SpecsRootRel = ".vibex/specs"
	// AgentsRootRel 为 agent 配置/流程/模板根。
	AgentsRootRel = ".vibex/agents"
	// AgentsDotAgentsRootRel 为与 AgentsRootRel 并行的可选根（目录名为 `.agents`，常见于 IDE/画布同步技能包）。
	// 技能合并顺序见 WorkspaceMergedSkillsDirs：后者目录覆盖同名 skill。
	AgentsDotAgentsRootRel = ".vibex/.agents"
	// PrototypesRootRel 为可交付 HTML 原型目录（与 design-kit 约定一致）。
	PrototypesRootRel = ".vibex/prototypes"
)

// WorkspaceMergedSkillsDirs 返回待合并的技能根目录（低优先级 → 高优先级；后者覆盖同名 SKILL）。
// 顺序：<workspace>/skills · .vibex/agents/skills · .vibex/.agents/skills
func WorkspaceMergedSkillsDirs(workspaceRoot string) []string {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return nil
	}
	return []string{
		filepath.Join(root, "skills"),
		filepath.Join(root, filepath.FromSlash(AgentsRootRel), "skills"),
		filepath.Join(root, filepath.FromSlash(AgentsDotAgentsRootRel), "skills"),
	}
}
