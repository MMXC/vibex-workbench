---
name: coord-short-circuit-phase1
description: coord-short-circuit-phase1 — skill for openclaw category
category: openclaw
triggers:
- openclaw
- team-tasks
- gateway hook
- custom hook
- coord decision
- openclaw hook
- coord short circuit phase1
related_skills:
- archon-workflow-engine
- openclaw-internals-reversed
- darwin-skill-execution
---
repo_tracked: true


# Coord: Short-Circuit Phase1 When Root Cause Already Known

When the user (or a prior analysis session) has already identified the root cause and proposed a solution, running the full `analyze-requirements → create-prd → design-architecture` chain is wasteful. The phase1 agents will just rediscover what the user already told you.

## When to Use

- User provides a root cause analysis with a clear problem/solution
- Problem is a single, well-scoped bug fix or endpoint addition
- No ambiguity about requirements — solution is obvious

## Workflow

### Step 1: Write phase1 docs directly (as Hermes/coord)

Create the three documents yourself:

```bash
# Analysis doc — synthesize user's root cause into structured format
write_file: docs/<project>/analysis.md

# PRD — functional requirements + acceptance criteria
write_file: docs/<project>/prd.md

# Architecture — technical approach + key decisions
write_file: docs/<project>/architecture.md
```

### Step 2: Mark phase1 stages done via task update

```bash
task update <project> analyze-requirements done
# → automatically triggers create-prd notification

task update <project> create-prd done
# → automatically triggers design-architecture notification

task update <project> design-architecture done
# → automatically triggers coord-decision notification
```

### Step 3: Coord self-review (since YOU wrote the docs)

Do a quick sanity check, then:
```bash
task update <project> coord-decision done
```

### Step 4: Allow and create phase2

```bash
task allow <project>
# If error "未检测到 Epic" → mark coord-decision done first

task phase2 <project> -e "<epic-name>" --docs-subdir "<subdir>" --work-dir "<workdir>" --yes
```

## Key task_manager.py Quirks

1. **`task phase1` → project already exists**: The chain was already created. Don't re-run `phase1`. Check status with `task status <project>`.

2. **Project data format**: Stage data is in `<project>.json` under `stages` dict (keys are stage IDs like `analyze-requirements`, not array indices). Tasks list is NOT under `tasks[]` — it's under `stages.<id>`.

3. **`task phase1` `--notify` flag**: Only needed when dispatching `analyze-requirements` to the analyst agent. If writing docs yourself, skip it.

4. **JSON storage**: Team-tasks data at `/root/.openclaw/workspace-coord/team-tasks/<project>.json` (NOT main `vibex.json`). Read directly for debugging but prefer `task` CLI.

5. **`dispatch-unit` requires IMPL_PLAN.md`**: Only works after phase1 is complete and architecture doc has been written.

## Anti-Pattern: Don't Do This

- Don't wait for the full phase1 agent chain when root cause is already known
- Don't re-run `task phase1` if project already exists
- Don't try to `dispatch-unit` before phase1 docs exist

## When NOT to Short-Circuit

- Requirements are ambiguous or have multiple solution options
- Need stakeholder validation (PM sign-off on PRD)
- Complex multi-epic project with uncertain scope
- Security-sensitive changes requiring extra review
