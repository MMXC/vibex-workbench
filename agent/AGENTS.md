# VibeX Agent — Architecture

## Overview

`vibex/agent` is the Go coding agent engine for the VibeX workbench.
Based on [nanoClaudeCode](https://github.com/neyuki778/nanoClaudeCode) (S01–S08),
extended with vibex-specific tools.

## Three-Layer Model

```
Model Layer   → LLM reasoning + tool selection
Harness Layer → Tools + Skills + Observation interfaces
Runtime Layer → Agent loop + persistence + concurrency
```

## Project Structure

```
agent/
├── go.mod / go.sum              # module: vibex/agent
├── .env                         # API key + config
├── cmd/
│   ├── agent/                   # CLI entry (original stdin/stdout)
│   └── web/                     # Web server (SSE + HTTP) ← VibeX uses this
│       └── main.go             # Bridges agent runtime with frontend SSE
├── agents/
│   ├── runtime/
│   │   └── tools/
│   │       ├── specs.go        # Parent tool specs (bash/read/write/todo/skill/subagent)
│   │       ├── specs_vibex.go  # Vibex tool specs (spec_designer/feature/validate/canvas_update/sync/bug_report/result_track/make_validate)
│   │       ├── handlers_base.go
│   │       ├── handlers_vibex.go  # Vibex tool implementations
│   │       ├── handlers_skill.go
│   │       ├── handlers_subagent.go
│   │       └── ...
│   ├── skills/store.go         # SKILL.md loader (loads from ~/.hermes/skills/)
│   ├── sessions/store.go       # Per-thread session persistence
│   ├── subagent/               # Concurrent sub-agent runner
│   ├── compact/                 # Context compression
│   └── background/             # Background bash tasks
└── .sessions/                  # Per-thread session files (created at runtime)
```

## Tools Available

### Base Tools (nanoClaudeCode S01–S03)
- `bash` — shell commands (workspace-relative, 30s timeout, dangerous command blocking)
- `read_file` — read file with size limit
- `write_file` — write file with auto-dir-create
- `todo_set` — maintain TODO state machine

### Skill Tools (S05)
- `skill_list` — list available skills from `~/.hermes/skills/`
- `skill_load` — activate a skill for subsequent turns
- `skill_unload` — deactivate a skill

### Sub-agent Tools (S04)
- `subagent_spawn` — spawn concurrent sub-agent (max 4 parallel, 2 retries)
- `subagent_wait` — wait for sub-agent jobs to complete

### Background Tasks (S08)
- `bash_bg` — start background shell command
- `bg_wait` — wait for background task
- `bg_list` — list background tasks

### Vibex-Specific Tools
- `spec_designer` — create spec YAML draft from user intent
- `spec_feature` — break goal spec into feature specs
- `spec_validate` — validate spec YAML syntax + required fields
- `canvas_update` — update Canvas visualization via SSE
- `spec_sync` — sync spec ↔ prototypes/generated code
- `make_validate` — run `make validate` in vibex-workbench
- `bug_report` — create bug-changelog entry
- `spec_result_track` — mark result confirmed/pending, emit SSE
- `tdd_design` — generate TDD test cases from spec's io_contract (go/python/typescript)
- `tdd_run` — execute tests, return RED/GREEN status + update Canvas
- `tdd_iterate` — run tests + show next behavior step from spec

### TDD Workflow

The agent follows strict TDD for every feature/bug route:

```
RED (write failing tests first)
  → tdd_design: parse spec io_contract → generate test files
  → tdd_run: confirm tests fail (RED)
  → implement feature / fix bug
GREEN (implement until tests pass)
  → tdd_run: confirm tests pass (GREEN)
  → tdd_iterate: show next behavior step
REFACTOR (clean up, no behavior change)
  → tdd_run: ensure tests still pass
```

Test cases are derived from spec's io_contract fields:
- **input** → happy path test case
- **boundary** → edge/boundary condition cases
- **behavior** → one test per numbered behavior step
- **output** → expected result in each test

Canvas nodes emitted: `canvas.tdd_nodes` (initial) + `canvas.tdd_cycle` (per run)
RED = 🔴, GREEN = 🟢, REFACTOR = 🔵

## API Endpoints (port 33338)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/chat` | Send message, returns `{status, threadId}` |
| GET | `/api/sse/<threadId>` | SSE stream for thread events |
| GET | `/api/threads/<threadId>/history` | Get conversation history |
| GET | `/api/skills` | List available skills |

## SSE Events Emitted

- `connected` — SSE connection established
- `agent.thinking` — agent is processing
- `message.delta` — text delta (user/assistant)
- `tool.called` — tool invocation started
- `tool.completed` — tool invocation finished
- `canvas.<event>` — canvas update events
- `result.confirmed` — result confirmation event
- `run.completed` — agent turn finished
- `error` — error occurred

## Configuration (.env)

```env
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o
SUBAGENT_MODEL=gpt-4o-mini
SKILLS_DIR=/root/.hermes/skills
WORKSPACE_DIR=/root/vibex-workbench
DEBUG_HTTP=false
```

## Quick Start

```sh
cd /root/vibex-workbench/agent
cp .env.example .env  # edit with your API key
./vibex-agent-web
```

## Self-Reflection (auto-improvement after each turn)

After each agent turn, `RunSelfReflectionIfWorthy` analyzes the tool calls from that turn and detects automatable patterns:

| Pattern | Detection | Action |
|---------|-----------|--------|
| `validate` → `generate` in same turn | Sequential make calls | Attempt to add auto-chain SSE to handler (conservative: only if `sse` already imported) |
| `bash` called 3+ times in one turn | High bash frequency | Log pattern for manual follow-up |
| Same tool called 3+ times | Repetitive pattern | Suggest creating a skill |

Results are broadcast as `agent.self_reflection` SSE events. The agent modifies its own handler code when safe to do so, without requiring user input.

## Skills

Skills are loaded from `SKILLS_DIR` (`~/.hermes/skills/`).
Each skill is a directory with `SKILL.md` (and optional `scripts/`, `references/`).

The agent can `skill_load` skills at runtime based on context.

**Repo tracking**: Agent-specific skills live in `agent/skills/` (git-tracked).
Generic skills live in `vibex-workbench/skills/`.
To persist a skill change: `cp -r ~/.hermes/skills/<name>/ agent/skills/ && git commit`.
