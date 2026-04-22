---
name: spec-first-workflow
description: Spec-First 工作流 — 接到需求后，先找 spec、改 spec、确认 spec，再实现验证迭代。适用于 vibex-workbench 所有需求处理场景。
category: spec-driven
title: Spec-First Workflow
triggers:
- 接到需求
- spec first
- 需求落地
- spec-driven
- 先找 spec
- spec先行
- 需求先写spec
- 不要直接写代码
- 按流程落地
- 需求落地流程
- spec驱动开发
- 需求进来先做什么
- 从spec开始
related_skills:
- spec-designer
- vibex-agent-ops
- darwin-skill
---

# Spec-First Workflow

## TL;DR

```
需求进来
  → 找 spec（session_search + search_files）
  → 改 spec（写 YAML 提案）
  → 确认 spec（四阶段澄清）
  → 实现（generator / 手动）
  → 验证（validator + criteria engine）
  → 迭代直到完成
```

---

## 流程图

```
┌─────────────────────────────────────────────────────┐
│  需求进来                                              │
└──────────────────┬────────────────────────────────────┘
                   ▼
┌──────────────────────────────────────────────────────┐
│  Step 1  需求接收                                      │
│  → 判断类型：新功能 / Bug / 优化 / 未知                   │
│  → 禁止：直接写代码                                      │
└──────────────────┬────────────────────────────────────┘
                   ▼
┌──────────────────────────────────────────────────────┐
│  Step 2  找相关 Spec                                    │
│  → session_search（历史）                               │
│  → search_files（specs/目录）                           │
│  → 确认 parent 链                                       │
└──────────────────┬────────────────────────────────────┘
                   ▼
┌──────────────────────────────────────────────────────┐
│  Step 3  改 Spec                                        │
│  → 新功能：建 feature spec + parent                     │
│  → Bug：追加 issues 条目                               │
│  → UI变更：写 ui_spec 段落                              │
│  → 关键：parent 存在吗？不存在先补 parent               │
└──────────────────┬────────────────────────────────────┘
                   ▼
┌──────────────────────────────────────────────────────┐
│  Step 4  确认 Spec（四阶段澄清）                         │
│  → 简单：clarify 问用户                                 │
│  → 中等：memlace clarification API                     │
│  → 复杂：完整四阶段（①②③④）                              │
│  → 状态：draft → planning → clarified                  │
└──────────────────┬────────────────────────────────────┘
                   ▼
┌──────────────────────────────────────────────────────┐
│  Step 5  实现                                           │
│  → Generator路径（推荐）或 手动路径                       │
│  → 调用 skill：spec-designer / vibex-agent-ops         │
└──────────────────┬────────────────────────────────────┘
                   ▼
┌──────────────────────────────────────────────────────┐
│  Step 6  验证                                           │
│  → spec_code_validator（Critical = 0 才 pass）          │
│  → vibex_criteria_engine（AC 全覆盖）                   │
│  → 失败 → 回 Step 3 → 补 spec → 再 Step 5              │
└──────────────────┬────────────────────────────────────┘
                   ▼
           ┌───────────────┐
           │  ✓ verified   │
           │  需求完成      │
           └───────────────┘
```

---

## 核心原则

1. **永远先找 spec，再动手** — 需求不是代码，需求是 spec
2. **spec 变更即提案** — 任何代码改动必须有对应的 spec 变更，没有例外
3. **确认后才能实现** — 未经确认的 spec 不能进入实现阶段
4. **验证驱动迭代** — 用工具验证，不靠主观判断

---

## 完整工作流（6步）

### Step 1: 需求接收

收到用户需求时，首先判断类型：

| 需求类型 | 入口动作 |
|---------|---------|
| 新功能 | 在 specs/ 下建对应 spec 文件 |
| Bug 修复 | 找到对应 feature spec，在其 `issues` 下追加条目 |
| 优化/重构 | 找到对应 module/feature spec，更新 `implementation_notes` |
| 未知领域 | 先 session_search 历史，再 search_files 扫 specs/ |

**禁止行为**：收到需求后直接写代码，不查 spec。

### Step 2: 找相关 Spec

```
1. session_search(需求关键词) — 查历史会话
2. search_files(关键词, path="specs", target="content") — 扫 spec 文件
3. 读找到的 spec 全文，理解当前状态
```

找 spec 时优先搜索：
- `specs/project-goal/` — 总目标
- `specs/feature/` — 功能 spec
- `specs/module/` — 模块 spec

**找到的结果必须包含 parent 链**：确认该 spec 是从属于哪个 L1/L2 的。

### Step 3: 改 Spec

根据需求类型决定修改范围：

**新增功能**：
```yaml
# 在 specs/feature/{name}/{name}_feature.yaml 新建
parent: "{L1/L2 spec name}"
phase: planning | clarified | implementation | verified
io_contract: ...
behaviors: ...
acceptance_criteria: ...
```

**Bug 修复**：在对应 spec 的 `issues` 数组追加：
```yaml
issues:
  - id: bug-001
    description: "..."
    severity: critical | warning | info
    status: open | fixed
    verified_by: "{validator name}"
```

**更新实现**：在 `implementation_notes` 更新实现策略。

**关键**：写完 spec 变更后，停下来问自己：这个 spec 的 parent 是谁？parent 存在吗？如果 parent 不存在，先补 parent spec。

### Step 4: 确认 Spec（四阶段澄清）

四阶段澄清结构（适用于功能/架构类变更）：

```
① 技术选型确认 → tech-stack spec（L2-①）
② MVP原型确认 → mvp-prototype spec（L2-②）
③ 前后端分层确认 → frontend-backend spec（L2-③）
④ 功能/用户故事确认 → feature specs（L2-④）
```

澄清方式选择：
- **简单变更**（typo/注释/文档）：直接 `clarify` 问用户确认
- **中等复杂度**（接口变更/参数调整）：用 memlace clarification API 走一轮 Q&A
- **高复杂度**（新功能/架构重构）：完整走四阶段澄清

**澄清入口**：点击图谱上对应 L2 spec 节点 → 展开 ClarificationPanel → 详情处可复制/编辑/确认。

确认后：
- 更新 spec 的 `status: clarified`
- 在 `notes` 记录澄清结论摘要

### Step 5: 实现

按 spec 实施，分两种路径：

**Generator 路径**（推荐）：
```bash
# 运行 spec 驱动的生成器
python3 generators/gen.py --spec specs/feature/{name}/{name}_feature.yaml
# 然后验证
python3 generators/spec_self_corrector.py --spec specs/feature/{name}/{name}_feature.yaml
```

**手动路径**（spec 确认但无对应 generator）：
- 直接修改代码
- 每改一处，在 spec 对应条目加 `implementation_notes` 记录

### Step 6: 验证

双重验证：

**Spec 层验证**：
```bash
python3 generators/spec_code_validator.py --spec {path} --mode full
# Critical = 0 才算 pass
```

**Criteria 层验证**：
```bash
python3 generators/vibex_criteria_engine.py --spec {path} --mode derive
# 检查 acceptance_criteria 是否全部被覆盖
```

**验证通过标准**：
- Critical issues = 0
- 所有 acceptance_criteria 有对应实现

验证失败 → 回到 Step 3 更新 spec → Step 5 补实现 → 再验证。

### Step 7: 完成

验证通过后：
1. 更新 spec 的 `phase: verified`
2. 如有必要，更新 memlace 偏好（`memlace_search` 读偏好，`write_pref` 写选择）
3. 通知用户完成

---

## 边界条件

| 情况 | 处理方式 |
|------|---------|
| 找不到相关 spec | 先建 L1 总目标 spec，再往下建子 spec |
| spec 和代码不一致 | 以 spec 为准，更新代码；如代码优于 spec，更新 spec |
| 澄清过程中需求变更 | 记录变更，重新走 Step 3 |
| 实现中 spec 漏了边界条件 | **停下手上的实现**，补 spec → 重新确认 → 对比已写代码是否符合新 spec（符合则保留补充，不符合则回退） |
| 用户催促跳过 spec | 说明 spec-first 的价值，如果用户坚持则标注 `spec_skipped: true` 并记录原因 |
| 验证卡住 | 看 `generators/spec_code_validator.py` 输出，逐条解决 |

---

## 工具依赖

| 工具 | 用途 |
|------|------|
| `session_search` | 查历史会话找相关需求 |
| `search_files` | 扫 specs/ 目录 |
| `read_file` | 读 spec 全文 |
| `patch` | 修改 spec 文件 |
| `memlace clarification API` | 四阶段澄清 |
| `spec_code_validator.py` | 验证代码层 |
| `vibex_criteria_engine.py` | 验证 criteria 层 |
| `clarify` | 简单确认 |

---

## UI Spec 定义（ClarificationPanel + L2 闪烁卡点）

当需求涉及 spec 图谱 UI 变更时，按以下结构定义 UI spec：

### L2 闪烁卡点（GoalSpecCanvas 中的 L2 spec 节点）

| 状态 | 视觉表现 | CSS 动画 |
|------|---------|---------|
| `pending`（澄清中） | 橙/蓝/紫/绿边框 + `@keyframes pulse-ring` 扩散环 | `animation: pulse-ring 2s ease-out infinite` |
| `in_progress`（进行中） | 静态边框 + 标题后缀 `(进行中)` | 无动画 |
| `confirmed` | 绿色边框 `#22c55e` | 无动画 |
| `draft` | 灰色边框 | 无动画 |

阶段色标：
- ① tech_stack → 蓝 `#3b82f6`
- ② mvp_prototype → 紫 `#a855f7`
- ③ frontend_split → 青 `#06b6d4`
- ④ user_stories → 绿 `#22c55e`

### ClarificationPanel 详情面板

组件路径：`frontend/src/lib/components/workbench/ClarificationPanel.svelte`

| 功能区 | 内容 |
|--------|------|
| 标题栏 | `{specName}` + 阶段标签 + 状态徽章 |
| 阶段说明 | 当前阶段的中文描述（如"MVP 原型确认"） |
| Rounds 列表 | 每轮展示：`Q{index} · {phase_label}` + 答案 + derived spec 摘要 |
| Draft 编辑器 | `textarea` 预填当前 draft，可复制修改 |
| 操作按钮 | `Copy Draft`（复制草稿）+ `Confirm & Write Spec`（锁定会话） |

**状态驱动按钮逻辑**：
- `status === 'draft'`：`Copy Draft` 可用，`Confirm` 禁用（等待澄清）
- `status === 'in_progress'`：两者均可用
- `status === 'confirmed'`：两者均禁用，显示 `✅ 已确认` 徽章

### 图谱集成规范

```
总目标节点（点击）
    ↓
L2 spec 卡片列表（按 parent 分组）
    ↓
点击任意 L2 节点 → 展开 ClarificationPanel（右侧抽屉）
    ↓
在 Panel 中完成澄清 → 点 Confirm → 面板关闭，卡片变 confirmed
```

**当在 spec-first 流程中涉及 UI 变更时**，在 Step 3 写 spec 时必须包含本节对应的 `ui_spec` 段落：

```yaml
ui_spec:
  component: ClarificationPanel
  file: frontend/src/lib/components/workbench/ClarificationPanel.svelte
  changes:
    - type: add | modify | remove
      element: "{按钮/卡片/动画}"
      spec: "{具体描述}"
```

---

## Spec 状态生命周期

spec 有 5 种状态，按以下顺序流转：

```
draft → planning → clarified → implementation → verified
                ↳ rejected（任意阶段可拒）
```

| 状态 | 含义 | 可进入的动作 |
|------|------|-------------|
| draft | 刚创建或大幅修改后 | 澄清 → clarified |
| planning | 正在澄清或设计 | 确认 → clarified |
| **clarified** | 用户已确认，可实现 | 实现 → implementation |
| implementation | 实现中 | 验证通过 → verified |
| verified | 验证通过，需求完成 | — |

**关键规则**：
- 实现中如果 spec 有变更，必须回退到 **planning**（重新澄清）
- spec_skipped 的条目在实现笔记中标注 `spec_skipped: true`
- verified 状态是最终状态，只有新需求可以新建 draft

---

## 与其他 Skill 的关系

### spec-designer（澄清问题模板 + UI 设计原则）
**何时调用**：Step 4 需要设计澄清问题时
- 调用方式：`skill_view("spec-designer")` 获取四阶段澄清问题模板
- 场景：L2 澄清进入② MVP原型或③前后端分层时，需要具体问题列表
- 注意：spec-designer 负责"怎么问"，spec-first 负责"问哪个阶段"

### vibex-agent-ops（任务派发）
**何时调用**：Step 5 实现阶段，spec 确认后需要派活给其他 agent 时
- 调用方式：`skill_view("vibex-agent-ops")` 获取任务模板
- 场景：Go agent / Python generator / 前端组件 需要分别派发给不同 agent 时
- 注意：spec-first 负责"哪个 spec 需要实现"，vibex-agent-ops 负责"怎么把 spec 包装成任务"

### darwin-skill（迭代优化）
**何时调用**：spec-first skill 本身需要改进时
- 调用方式：`skill_view("darwin-skill")` 走完整优化流程
- 场景：发现 spec-first 流程有漏洞或不足时

### 实际调用决策表

| 需求场景 | 需要调用的 skill | 调用时机 |
|---------|----------------|---------|
| 新功能澄清 | spec-designer | Step 4 开始时 |
| 实现需要多 agent 协作 | vibex-agent-ops | Step 5 开始时 |
| 优化 spec-first 本身 | darwin-skill | 任何时候 |
| 验证实现质量 | 无需调用，用工具 | Step 6 验证时 |
