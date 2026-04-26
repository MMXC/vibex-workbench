# Layer Decision Reference

Use this reference when deciding which spec layer to create or update.

## Layer Meanings

| Layer | Owns | Does Not Own |
|---|---|---|
| L1 goal | product mission, users, constraints, success criteria | framework choice, files, implementation |
| L2 skeleton | tech stack, module matrix, entry points, cross-module contracts | module internals, UI details |
| L3 module | module boundary, public API, state definitions, dependencies | user stories, concrete file edits |
| L4 feature | behavior, button/panel flow, user stories, acceptance/test scenarios | exact code patch boundaries |
| L5 slice | target file, allowed/forbidden edits, generation rules, verification commands | product rationale across many files |

## Routing Rules

### User Gives Product Intent

Examples:

- "做一个可以从空仓库长项目的产品"
- "这个 MVP 的边界是什么"

Action:

1. Create/update L1.
2. If architecture is implied, create/update L2.
3. Do not implement code yet.

### User Gives Architecture or Module Change

Examples:

- "加 workspace lifecycle 模块"
- "需要一套 IDE chrome 模块"

Action:

1. Update L2 `modules_matrix`.
2. Create/update L3 module.
3. Create L4 only for concrete user-visible behavior.

### User Gives UI Button, Panel, or Flow

Examples:

- "顶部导航只保留 WebView 内层"
- "底部 terminal panel 要展示 make 输出"

Action:

1. Create/update L4 feature.
2. If the user expects code change, create/update L5 for each target file.

### User Names a File

Examples:

- "`main.go` 需要改 Wails options"
- "`WorkbenchTitlebar.svelte` 加窗口按钮"

Action:

1. Find existing L5 whose `content.file_path` matches the file.
2. If missing, create L5.
3. Implement only after L5 is present.

### User Asks "Can Agent Implement This?"

Action:

1. Check whether L4 has clear behaviors.
2. Check whether L5 exists for each target file.
3. If L5 has boundaries and verification commands, answer "yes, implementable".
4. If L5 is missing, answer "partially; needs L5".

## Parent Chain Rules

- L1 `parent: null`
- L2 parent points to L1
- L3 parent points to L2
- L4 parent points to L3 or, only for broad MVP lifecycle specs, L2
- L5 parent points to L4

Before committing spec changes, run:

```bash
make validate
```

## Current Canonical Directories

Use these for new specs:

```text
specs/L1-goal/
specs/L2-skeleton/
specs/L3-module/
specs/L4-feature/
specs/L5-slice/
```

Do not add new files to older `specs/feature/` or `specs/module/` directories unless maintaining legacy specs.
