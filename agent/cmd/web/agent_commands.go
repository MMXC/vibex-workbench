// agent_commands.go — list workspace-defined agent profiles for slash-command routing.
package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vibex/agent/internal/common"
)

type agentCommandDTO struct {
	Command string `json:"command"`
	LabelZh string `json:"label_zh"`
	Source  string `json:"source"`
}

// profileMetaJSON reads optional label_zh from `.agents/profiles/*.json` or `.agents/skills/<name>/agent.json`.
type profileMetaJSON struct {
	LabelZH string `json:"label_zh"`
}

func readJSONLabelZh(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var meta profileMetaJSON
	if json.Unmarshal(data, &meta) != nil {
		return ""
	}
	return strings.TrimSpace(meta.LabelZH)
}

func readAgentJSONLabelZh(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg common.AgentPromptConfig
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return strings.TrimSpace(cfg.LabelZH)
}

func listWorkspaceAgentCommands(wsRoot string) []agentCommandDTO {
	m := make(map[string]*agentCommandDTO)

	upsert := func(slug, label, source string) {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" || slug == "general-agent" {
			return
		}
		e, ok := m[slug]
		if !ok {
			m[slug] = &agentCommandDTO{Command: slug, LabelZh: label, Source: source}
			return
		}
		if e.LabelZh == "" && label != "" {
			e.LabelZh = label
		}
	}

	profDir := filepath.Join(wsRoot, ".agents", "profiles")
	if entries, err := os.ReadDir(profDir); err == nil {
		for _, ent := range entries {
			if ent.IsDir() || !strings.HasSuffix(strings.ToLower(ent.Name()), ".json") {
				continue
			}
			slug := strings.TrimSuffix(strings.ToLower(ent.Name()), ".json")
			path := filepath.Join(profDir, ent.Name())
			label := readJSONLabelZh(path)
			upsert(slug, label, "profile")
		}
	}

	agDir := filepath.Join(wsRoot, ".agents", "agents")
	if entries, err := os.ReadDir(agDir); err == nil {
		for _, ent := range entries {
			if ent.IsDir() || !strings.HasSuffix(strings.ToLower(ent.Name()), ".json") {
				continue
			}
			slug := strings.TrimSuffix(strings.ToLower(ent.Name()), ".json")
			path := filepath.Join(agDir, ent.Name())
			label := readAgentJSONLabelZh(path)
			upsert(slug, label, "agent")
		}
	}

	skillsRoot := filepath.Join(wsRoot, ".agents", "skills")
	if dirs, err := os.ReadDir(skillsRoot); err == nil {
		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			path := filepath.Join(skillsRoot, dir.Name(), "agent.json")
			if _, err := os.Stat(path); err != nil {
				continue
			}
			slug := strings.ToLower(dir.Name())
			label := readJSONLabelZh(path)
			upsert(slug, label, "skill")
		}
	}

	out := make([]agentCommandDTO, 0, len(m))
	for _, v := range m {
		if v.LabelZh == "" {
			v.LabelZh = v.Command
		}
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Command < out[j].Command })
	return out
}

// workspaceAgentCommandsHandler GET /api/workspace/agent-commands?workspaceRoot=...
func workspaceAgentCommandsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsArg := strings.TrimSpace(r.URL.Query().Get("workspaceRoot"))
	root, err := effectiveWorkspaceRoot(wsArg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cmds := listWorkspaceAgentCommands(root)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"commands":       cmds,
		"workspace_root": root,
	})
}
