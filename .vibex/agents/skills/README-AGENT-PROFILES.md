# Agent 配置（与「直接配置 agent」一致）

专用 agent 的**首选**配置方式与默认通用 agent 相同：**`.agents/agents/<slug>.json`** + **`prompt_file`**（Markdown），可选 **`adapters`**、**`required_skills`**、**`allowed_tools`**。结构与 **`general-agent.json`** 一致。

## 解析优先级（`agent_profile=<slug>`）

1. **`.agents/profiles/<slug>.json`** — 旧式扁平字段（`developer_message` 字符串），兼容保留。
2. **`.agents/agents/<slug>.json`** — **声明式**（`prompt_file` + `adapters`，与 `general-agent` 同源）。
3. **`.agents/skills/<slug>/agent.json`** — 仅当不存在上两类时的兜底；或与 **(2)** 叠加：用于补充 `required_skills` / `allowed_tools`（见下）。

## `.agents/agents/*.json` 示例（专用 agent）

新建 `.agents/agents/fix-ci.json`：

```json
{
  "prompt_file": ".agents/prompts/fix-ci-agent.md",
  "adapters": {
    "web": "Stay concise; prefer gh/cli for GitHub."
  },
  "required_skills": ["fix-ci"],
  "allowed_tools": ["bash", "read_file", "write_file", "append_file", "todo_set", "skill_list", "skill_load", "skill_unload", "bash_bg", "bg_wait", "bg_list", "subagent_spawn", "subagent_wait"]
}
```

若省略整段 `allowed_tools`，表示该 profile **不限制工具**（与默认 full registry 相同语义）。需要窄白名单时请显式列出。

若存在 **`.agents/skills/fix-ci/agent.json`**，可与 **(2)** 合并：用于补上 agents 文件里未写的 `required_skills`，或在 agents 未声明 `allowed_tools` 时用 skill 侧补充工具列表。

## 默认通用 agent

全局入口仍为 **`.agents/agents/general-agent.json`** → **`prompt_file`**（启动时 `ResolveDeveloperMessage`，不是通过 `agent_profile`）。

## 技能扫描

Agent 合并 **`skills/`** 与 **`.agents/skills/`** 注册 SKILL；`required_skills` 会在回合开始时预加载。
