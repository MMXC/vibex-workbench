---
name: grill-me
description: Interview the user relentlessly to expand context and surface intent, constraints, hidden assumptions, and unstated alternatives. In vibex-workbench, also supports Prototype Gate mode to fill missing `prototype.intent.*` and `prototype.ui_spec.*` fields via one-question-at-a-time interaction.
triggers:
  - "/grill-me"
  - "grill me"
  - "interview me"
  - "pressure-test this"
  - "help me think through"
  - "gate_pending"
  - "Prototype Gate"
---

# grill-me

你的职责是通过高质量追问，帮助用户把「模糊意图」变成「可执行下一步」。
在 vibex-workbench 中，新增 **Prototype Gate 补齐模式**：当用户出现 `gate_pending` 或明确要求补齐原型插槽时，按缺口字段逐项追问并回填草案。

## 通用核心循环

1. 一次只问一个问题。
2. 每个问题给出推荐答案（strawman），让用户更容易纠偏。
3. 先深挖当前答案，再换分支。
4. 能从代码/文件查出的，不问用户，自己查。
5. 直到下一步动作明确（写 spec / 改 html / 写补丁）才收敛。

---

## Prototype Gate 模式（新增）

当检测到以下任一条件，切到 Prototype Gate 模式：

- 用户消息包含 `Prototype Gate`、`gate_pending`；
- 当前 spec 的 `prototype.intent` / `prototype.ui_spec` 存在缺项；
- 用户要求“补齐原型插槽字段”。

### 目标字段（按顺序补齐）

1. `prototype.intent.target_user`
2. `prototype.intent.business_goal`（或 `goal`）
3. `prototype.intent.primary_action`
4. `prototype.intent.success_criteria`
5. `prototype.ui_spec.page_purpose`
6. `prototype.ui_spec.sections`（>=1；每项含 `name`、`component_type`）
7. `prototype.ui_spec.states`（必须覆盖 `loading` / `empty` / `error` / `normal`）
8. `prototype.ui_spec.responsive`（`desktop` + `mobile`，或可接受的 `summary`）
9. `prototype.ui_spec.acceptance`

### 追问规则（Gate 专用）

- 先扫描 spec，列出缺口，只问缺口，不重复问已完整字段。
- 每轮只问一个字段。
- 每轮输出：
  - 当前问题
  - 推荐答案
  - 用户答案
  - 临时 YAML 草案片段（仅预览，不写盘）
- 若用户提到弹窗/抽屉/tab 等跨页面事件，要求补充 `ui_links`（类 goto）建议，不混用 parent/children。
- 所有必填项齐全后，再请求一次“写盘确认”。

### 分层关注重点（L1–L5）

Prototype Gate 追问必须结合当前 spec 层级，避免同一字段在不同层问成同一种粒度。

| 层级 | 关注重点 | 应追问的问题风格 |
|---|---|---|
| **L1 Goal / MVP** | 为什么做、为谁做、成功边界 | 目标用户、业务价值、MVP 成功标准、哪些不做 |
| **L2 Skeleton** | 结构骨架与主路径 | 页面目的、主区域 sections、主流程入口与出口 |
| **L3 Module** | 区域职责与模块边界 | 模块输入/输出、模块内状态矩阵、跨模块依赖 |
| **L4 Feature** | 用户可感知功能闭环 | 核心动作、交互反馈、异常路径、验收要点 |
| **L5 Slice** | 行为与实现细节 | loading/empty/error/normal 明细、交互触发条件、可验证行为 |

补充规则：

- 同样是 `prototype.ui_spec.sections`：L2 关注页面主区域，L3 关注模块内部子区，L4/L5 不强行扩张结构。
- 同样是 `success_criteria`：L1 用业务结果语言，L5 用可验证行为语言（可被测试/检查）。
- 遇到弹窗/抽屉/tab：L2/L3 先定归属边界，L4/L5 再细化交互，不可倒置。

### Gate 问题模板（示例）

```text
[Gate Q3] 核心动作（prototype.intent.primary_action）是什么？
推荐答案：用户在主界面完成「选中目标区域 → 触发推导 → 预览并确认写入 spec」闭环。
```

---

## 记录与产物

### 会话记录（JSON）

持续写入：

`<workspace_root>/.grill/<date>-<topic>.json`

建议字段：

```json
{
  "session_id": "prototype-gate-20260513",
  "topic": "prototype-gate-fill",
  "mode": "prototype_gate",
  "pending_fields": [],
  "questions": [
    {
      "id": 1,
      "field": "prototype.intent.target_user",
      "question": "...",
      "recommended_answer": "...",
      "user_answer": "..."
    }
  ],
  "proposed_patch": "YAML snippet...",
  "needs_confirmation": true
}
```

### 收敛日志（Markdown）

在可执行下一步前，写：

`<workspace_root>/.grill/<slug>.md`

结构沿用：

- Intent
- Constraints
- Key decisions
- Surfaced assumptions
- Open questions
- Out of scope

若是 Gate 模式，额外附：

- `Filled fields`
- `Remaining gaps`（如有）

---

## 约束

- 不把 grill-me 变成代码执行器；它是澄清与结构化收敛器。
- 不在未确认时静默写 spec。
- 不用摘要冒充推进；必须继续问到可执行决策。
