---
name: spec-governance-completeness
description: |
  VibeX spec 覆盖治理 skill——Coverage Matrix + Consistency Check + 全景视图刷新。
  当 VibeX spec 体系出现"哪些能力还没 spec"、"L4 是否都有 L5"、"parent chain 是否断裂"等
  问题时，运行本 skill 获得完整诊断报告。
triggers:
  - "spec 覆盖率"
  - "L4 缺 L5"
  - "spec 全景视图"
  - "spec 完整性检测"
  - "哪些能力还没 spec"
  - "spec coverage matrix"
---

# Spec-Governance Completeness Skill

## 何时使用

运行本 skill 的场景：
- 想知道"所有产品能力有没有都被 spec 覆盖"
- 发现有 L4 没有对应的 L5 实现切片
- 需要更新 spec 全景视图（每次新建/修改 spec 后）
- 发现 spec 树有不一致（parent 断裂、命名冲突）
- 准备冲刺验收前，想知道哪些 spec 还是空的

## 核心原则

**Coverage Completeness**（覆盖完整性）= 每一层每个节点，要么有子节点，要么显式标记"不需要子节点"。
不能有悬空的能力。

**Consistency**（一致性）= parent chain 完整、同一层无命名冲突、`generates[]` 无跨 spec 文件冲突。

## 工作流

### Step 1：扫描 spec 树（自动）

扫描 `specs/` 目录，构建：
- 完整 spec 列表（name → level/parent/status）
- L3 → L4 映射
- L4 → L5 映射
- parent chain 验证

### Step 2：生成 Coverage Matrix

```
VibeX Spec Coverage Matrix
=========================
Total: N specs | L1×1 | L2×1 | L3×6 | L4×M | L5×K

L3 Module → L4 Features
  MOD-ide-chrome [proposal]
    ├─ FEAT-ide-titlebar [proposal]      ✓ L5×3
    ├─ FEAT-ide-activity-sidebar [proposal] ✓ L5×2
    ├─ FEAT-ide-editor-tabs [proposal]  ✓ L5×1
    ├─ FEAT-ide-agent-panel [proposal]   ✓ L5×1 (Cursor-aligned)
    └─ FEAT-ide-bottom-dock [proposal]  ✓ L5×2

  MOD-spec-editor [proposal]
    ├─ FEAT-spec-editor [proposal]       ⚠ L5=0 ← 缺少
    ├─ FEAT-spec-write [proposal]        ⚠ L5=0 ← 缺少
    └─ ...

L4 Features without L5 (实现空白区)
  ⚠️  FEAT-build-panel        # 最高优先级
  ⚠️  FEAT-spec-editor
  ...
```

优先级定义：
- 🔴 L4=0 L5 且有 Cursor 对标 spec → 最高优先
- 🟡 L4=0 L5 且是 MVP 核心能力 → 高优先
- 🟢 其他 L4=0 L5

### Step 3：一致性检查

- **Parent chain 断裂**：L4.parent 不在 spec 列表中
- **同层命名冲突**：同一目录下两个 spec name 相同
- **generates[] 冲突**：同一 file path 被多个 spec 的 generates[] 声明
- **孤立 L4**：FEAT- 命名的 L4 但 parent 不存在
- **状态不一致**：parent 已 ready 但 child 还是 proposal

### Step 4：输出全景视图

生成 `specs/_governance/coverage-report.md`：

```
# VibeX Workbench Spec Coverage Report
Generated: {timestamp}

## 覆盖率摘要
- L1: 1/1 (100%)
- L2: 1/1 (100%)
- L3: 6/6 (100%)
- L4: M/N 有 L5 (X%)
- L5: K total slices

## L4 实现进度
| L4 Feature | L5 Slices | Status | Priority |
|------------|-----------|--------|----------|
| FEAT-ide-agent-panel | 1 | proposal | 🔴 Cursor对齐 |
| FEAT-spec-code-bidirectional | 2 | proposal | 🔴 超越Cursor |

## 缺失 L5（按优先级）
1. [🔴] FEAT-spec-editor → 需要 SLICE-spec-editor-component
2. [🔴] FEAT-spec-write → 需要 SLICE-spec-editor-monaco
...

## 一致性问题
- 无 parent chain 断裂
- 无命名冲突
```

### Step 5：更新 spec-governance 心跳任务（可选）

如果发现新问题，将诊断结果写入 team-tasks 作为待办：
```
coord-short-circuit-phase1 发现：
- 12 个 L4 缺 L5（实现空白区）
- 最优先：FEAT-spec-editor、FEAT-spec-write
→ 请安排实现
```

## 已知 VibeX Spec 树状态（截至 2026-04-28）

```
L1: vibex-workbench-mvp [proposal]
  └─ L2: vibex-workbench-skeleton [proposal]
        ├─ L3: MOD-ide-chrome [proposal]
        │     ├─ FEAT-ide-titlebar [proposal]          ✓ L5×3
        │     ├─ FEAT-ide-activity-sidebar [proposal]   ✓ L5×2
        │     ├─ FEAT-ide-editor-tabs [proposal]        ✓ L5×1
        │     ├─ FEAT-ide-agent-panel [proposal]        ✓ L5×1 ← Cursor对齐
        │     └─ FEAT-ide-bottom-dock [proposal]       ✓ L5×2
        ├─ L3: MOD-spec-editor [proposal]
        │     ├─ FEAT-spec-editor [proposal]            ⚠ L5=0
        │     ├─ FEAT-spec-write [proposal]             ⚠ L5=0
        │     ├─ FEAT-spec-graph-expansion [proposal]  ✓ L5×3
        │     ├─ FEAT-canvas-expand [proposal]         ⚠ L5=0
        │     ├─ FEAT-new-l1-wizard [proposal]          ⚠ L5=0
        │     ├─ FEAT-spec-code-bidirectional [proposal] ✓ L5×2 ← 超越Cursor
        │     └─ ...
        ├─ L3: MOD-state-detection [proposal]
        ├─ L3: MOD-workspace-root [proposal]
        ├─ L3: MOD-scaffolding [proposal]
        └─ L3: MOD-build-panel [proposal]

缺失 L5（12个）:
  🔴 FEAT-spec-editor, FEAT-spec-write, FEAT-canvas-expand, FEAT-new-l1-wizard
  🟡 FEAT-build-panel, FEAT-make-integration, FEAT-scaffolding, FEAT-state-detection
  🟡 FEAT-state-detection-fix, FEAT-workspace-lifecycle, FEAT-workspace-selector
  🟢 FEAT-mvp-governance
```

## 分析脚本（已实现）

分析脚本路径：`scripts/spec_coverage.py`（在 vibex-workbench repo 根目录）

```bash
# 控制台矩阵（快速）
python3 scripts/spec_coverage.py

# 生成完整 markdown 报告（git push 后自动更新）
python3 scripts/spec_coverage.py --report
# 输出：specs/_governance/coverage-report.md
```

脚本能力：
- 扫描所有 specs/ 下的 YAML，构建 name→{level,parent,status} 映射
- 计算 L3→L4、L4→L5 覆盖矩阵
- 识别无 L5 的 L4，按 CRITICAL/HIGH/MEDIUM 排序
- 检测 parent chain 断裂 + 重复命名
- 生成 coverage-report.md（可提交到 git 作为历史快照）

## YAML 格式陷阱

**已踩坑：多行字符串的 unclosed quote**

症状：`yaml.parser.ParserError: expected <block end>, but found '<scalar>'`
位置：通常在 `content: "..."` 这种字段跨多行时

原因：YAML 多行标量字符串格式错误，例如：
```yaml
# 错误 — content 值跨行但引号在第一行就闭合了，导致后续行被当成新 key
      sections:
        - header: "Drift Summary"
          content: "N drifted, M clean — [Check Now] 按钮
        - list: "Drifted Files"
```
末尾的 `"` 只闭合了 `content` 的第一个字符，`- list:` 被解析为缩进错误的标量。

修复：确保多行内容全在同一行，或用 YAML 块标量 `|`：
```yaml
# 正确：单行字符串
          content: "N drifted, M clean — [Check Now] 按钮"

# 或正确：块标量
          content: |
            N drifted, M clean
            [Check Now] 按钮
```

验证：每次写完 YAML 后运行 `python3 -c "import yaml; yaml.safe_load(open('path'))"` 确保能 parse。

## 相关 Skills

- `spec-chain-audit`：parent chain 修复
- `spec-cross-consistency-verification`：跨 spec 接口/错误码一致性
- `spec-darwin-loop`：spec 质量 Darwin 迭代评分
- `coord-short-circuit-phase1`：coord 任务派发
