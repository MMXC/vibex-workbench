---
name: spec-first-workflow
description: Operates VibeX spec-first workflows. Use when handling product requirements, UI/panel/button changes, self-bootstrap work, L1-L5 specs, or implementation from specs. The agent must find or create specs before code changes, require L5 file-level specs before implementation, and verify with project commands.
category: spec-driven
title: Spec-First Workflow
triggers:
  - 接到需求
  - spec first
  - spec-first
  - spec-driven
  - 自举
  - L1
  - L2
  - L3
  - L4
  - L5
  - 从原型反推规格
  - 按 spec 实现
  - 不要直接写代码
related_skills:
  - spec-designer
  - vibex-agent-ops
  - darwin-skill
---

# Spec-First Workflow

This skill is the entry point for VibeX spec-first work. It teaches the agent how to discover the right spec, choose the right layer, implement only from file-level boundaries, and interpret validation failures.

## Required Loop

1. **Discover** relevant specs under `specs/`.
2. **Decide layer** using the L1-L5 routing rules.
3. **Update the smallest valid spec** before implementation.
4. **Validate specs** with `make validate`.
5. **Require L5** before changing code files.
6. **Implement only inside L5 boundaries**.
7. **Run verification commands** listed in L5.
8. **Classify failures** and either fix spec, fix implementation, or ask user.

## Layer Routing

Use this quick routing table. For details, read [layer-decision.md](layer-decision.md).

| User intent | Layer |
|---|---|
| Product goal, success criteria, target users | L1 |
| Architecture, module matrix, tech stack, entry points | L2 |
| Module boundary, public API, state definitions | L3 |
| Feature behavior, button, panel, user flow, acceptance tests | L4 |
| Concrete file implementation, allowed/forbidden edits, verification commands | L5 |

Rules:

- New product direction: start at L1 or L2.
- New module capability: update L2 and create/update L3.
- New UI button/panel/flow: create/update L4.
- Any code implementation request: require L5 first.
- If L5 is missing or too vague, create/update L5 before touching code.

## Discovery Rules

When a request arrives:

1. Search `specs/` for keywords from the user request.
2. Prefer current L5 if the user names a file.
3. Prefer L4 if the user describes behavior but not a file.
4. Check `spec.parent` chain before editing.
5. If parent does not exist, create or fix parent first.

Current canonical directories:

- `specs/L1-goal/`
- `specs/L2-skeleton/`
- `specs/L3-module/`
- `specs/L4-feature/`
- `specs/L5-slice/`

Older directories may still exist; do not migrate them unless explicitly requested.

## L5 Before Code

Before editing implementation files, read the relevant L5 slice. If none exists, create one.

Each L5 must include:

- `content.file_path`
- `content.implementation_boundary.allowed`
- `content.implementation_boundary.forbidden`
- `content.generation_rules`
- `content.verification.commands`
- `content.verification.failure_conditions`
- `content.dependencies`

For exact implementation protocol, read [l5-implementation-protocol.md](l5-implementation-protocol.md).

## Command Map

Use non-interactive commands. Do not rely on prompts that ask the agent to choose.

| Command | Purpose |
|---|---|
| `make validate` | YAML parse + parent chain validation |
| `make generate` | Spec-to-code generation |
| `make gen-graph` | Dependency graph output |
| `npm --prefix frontend run build` | Frontend compile validation |
| `go build .` | Wails app compile validation |
| `make wails-dev` | Manual desktop runtime check |

Prefer commands listed in the relevant L5 `verification.commands`.

## Failure Handling

Never treat a long terminal log as the source of truth. Classify the failure first.

Common categories:

- `yaml_parse_error`
- `parent_not_found`
- `level_mismatch`
- `missing_l5_boundary`
- `build_error`
- `typecheck_error`
- `runtime_manual_check`
- `generated_drift`

For classification and response rules, read [validation-and-failure.md](validation-and-failure.md).

## Spec Write Protocol (create-child-spec)

When drafting a new child spec (L2-L5), the agent MUST follow the confirmation-first protocol:

**Before user confirmation:**
- Generate draft summary and clarifying questions
- Present draft to user via ClarificationPanel
- Do NOT call `POST /api/workspace/specs/write`
- Do NOT write any file to `specs/`

**After user confirmation:**
- Assemble confirmed payload: `{workspace_root, path, parent_name, target_level, yaml_text, confirmation_id}`
- Call `POST /api/workspace/specs/write`
- Call `make validate` and classify result
- Display validation result category and next action

**Path rules:**
- Spec file paths must be under `specs/{L2-skeleton|L3-module|L4-feature|L5-slice}/`
- Do not allow `..` path traversal
- `spec.parent` in yaml_text must match the parent_name in the payload

**Validation feedback categories:**
- `yaml_parse_error` — YAML syntax issue
- `parent_not_found` — spec.parent not in parent chain
- `level_mismatch` — level field conflicts with path
- `missing_l5_boundary` — L5 missing required fields
- See [validation-and-failure.md](validation-and-failure.md) for full reference

## Implementation Rules

- Do not modify code directly from L4 unless the user explicitly asks for a quick prototype.
- Do not edit outside `implementation_boundary.allowed`.
- Do not bypass `implementation_boundary.forbidden`.
- If implementation reveals a missing boundary, stop and update the spec first.
- If code and spec disagree, prefer the spec unless the user confirms the implementation should become the new source of truth.
- Preserve unrelated user changes in the working tree.

## Completion Criteria

A spec-first task is complete only when:

1. Relevant specs exist and pass `make validate`.
2. Implementation matches L5 boundaries.
3. L5 verification commands have been run or explicitly marked manual.
4. Remaining risks are reported by failure category.

## Related References

- [layer-decision.md](layer-decision.md): how to choose L1-L5.
- [l5-implementation-protocol.md](l5-implementation-protocol.md): file-level implementation contract.
- [validation-and-failure.md](validation-and-failure.md): command results and failure response rules.
