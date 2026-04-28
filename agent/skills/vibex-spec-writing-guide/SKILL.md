---
name: vibex-spec-writing-guide
description: VibeX spec YAML 写作指南 — 常见语法陷阱、层级约定、validator 快速检查
category: spec-driven
title: VibeX Spec YAML Writing Guide
---

# VibeX Spec YAML Writing Guide

Skill for writing valid VibeX spec YAML files (L1–L5).
Covers common pitfalls that cause `validate_specs.py` to fail.

## Trigger Conditions
- Writing a new spec YAML file
- Editing an existing spec and adding new field values
- Using backticks, quotes, or special characters in YAML values
- The validator reports `YAML error` at a specific line

## YAML Safety Rules

### Rule 1: Single-quoted strings — double single quotes inside

In YAML, single-quoted strings use `''` to represent a literal single quote.
Never use an unescaped `'` inside a single-quoted value.

```yaml
# BAD — unescaped single quote inside single-quoted string
bug: 'half' 作为 key 在 suggestions dict 中不存在

# GOOD — use double quotes instead
bug: "'half' 作为 key 在 suggestions dict 中不存在"
```

### Rule 2: Block scalars (| or >) — no standalone `---` inside

Block scalars (`|`, `>`, `|-`) run until indentation drops.
A standalone `---` inside a block scalar terminates the YAML document.

```yaml
# BAD — --- inside block scalar terminates document
add: |
  return `---
spec:
  version: "0.1"
  ...

# GOOD — use array.join('\n') instead
add: |
  const tmpl = [
    '---',
    'spec:',
    '  version: "0.1"',
    ...
  ].join('\n');
```

### Rule 3: Special characters — double-quote the outer string

When a value contains `:`, `#`, `{`, `}`, `[`, `]` or starts with special chars,
wrap the value in double quotes.

```yaml
# BAD — : triggers map parsing
state: empty: partial: ready

# GOOD
state: "empty | partial | ready"
```

### Rule 5: Line-ending `+` is YAML continuation — keep mapping values on one line

YAML interprets a bare `+` at end of line as a block scalar continuation indicator.
Never put `+` at the end of a mapping value line.

```yaml
# BAD — line-ending + is YAML continuation indicator
        - pass: "绿色徽章 Valid" + 耗时
        - fail: "红色徽章 Invalid" + 错误数 + 摘要
        - warning: "黄色徽章 Warning" + 警告数

# GOOD — simple strings, no operators at line end
        - pass: "绿色徽章 Valid 耗时"
        - fail: "红色徽章 Invalid 错误数 摘要"
        - warning: "黄色徽章 Warning 警告数"
```

### Rule 6: Emoji / special chars in strings

```yaml
# RISKY
acceptance: "state=partial 时，✅校验 ⚙️生成 → 进入 Workbench"

# SAFER
acceptance: "state=partial 时，校验和生成按钮可见，ready=true"
```

## Validator Quick-Check

```bash
# Single file
python3 generators/validate_specs.py specs/L5-slice/YOUR-FILE.yaml

# Full tree
python3 generators/validate_specs.py specs/

# Expected: [OK] All specs passed.
```

## Common Error Patterns

| Error | Cause | Fix |
|---|---|---|
| `expected <block end>, but found '<scalar>'` at column N | `+` at end of line inside a block mapping (YAML continuation indicator) | Use plain strings without `+` operators |
| `expected <block end>, but found '<scalar>'` | Single quote inside single-quoted string | Outer to double quotes |
| `could not find expected ':'` | `---` inside block scalar | Use array.join or indented continuation |
| `mapping values are not allowed` | Bare value with `:` not quoted | Double-quote the whole value |
| Parser error at column X | Special char not escaped | Wrap in double quotes |
| Missing closing quote | Multiline string not closed | Check line where error reported |

## Spec Level Conventions

- **L1**: `parent: null`, `level: "1_project-goal"`
- **L2**: `parent: <L1 name>`, `level: "2_skeleton"`
- **L3**: `parent: <L2 name>`, `level: "3_module"`
- **L4**: `parent: <L3 name>`, `level: "4_feature"`
- **L5**: `parent: <L4 name>`, `level: 5_implementation`

Directories must match levels: `specs/L1-goal/`, `specs/L2-skeleton/`, etc.

## Status Values

- `proposal` — spec written, no code
- `implementing` — code in progress
- `done` — code complete and verified
