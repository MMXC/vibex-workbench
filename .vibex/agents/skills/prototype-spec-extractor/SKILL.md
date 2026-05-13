---
name: prototype-spec-extractor
description: |
  Chrome 扩展「Proto Spec / 原型提取器」配套的专业 Agent 技能。
  在以下场景必须加载本技能：HTML 原型拆解为 L1–L5 spec 图谱；自然语言驱动下对本地 .vibex/prototypes/*.html
  与 specs/**/*.yaml 做实时追问、映射、预览确认后双写；从任意网页抽取多层级 UI/UX spec；
  spec ↔ 页面绑定区块 ↔ spec 双向推导。与 prototype.file、Design Kit 门禁、specs/write 确认流配合。
  触发词：原型拆解、proto spec、prototype 映射、HTML 转 spec、spec 绑定区块、提取 UI/UX、抽屉弹窗层级、
  prototype-spec、扩展侧栏、.vibex/prototypes。
category: spec-driven-project
title: Prototype–Spec Extractor（扩展配套）
triggers:
  - zh: 原型拆解, HTML转spec, 原型映射, 提取UI, 提取UX, spec绑定, 区块绑定, 弹窗层级, 抽屉, Proto Spec, 原型提取
  - en: prototype to spec, extract UI spec, HTML prototype spec, bound region, proto-spec extractor
related_skills:
  - spec-designer: 视觉与信息架构意图；本技能负责「层级落位 + 绑定字段 + 与 YAML 图谱一致」。
  - spec-governance: 父子链与治理字段；本技能的 ui_links 不替代 parent/children，仅表达导航与跨层关联。
  - vibex-spec-writing-guide: L1–L5 槽位与文件命名惯例。
---

# Prototype–Spec Extractor（扩展配套 Agent 技能）

**规范路径**：与本目录 **`agent.json`** 成对放置于 **`.vibex/agents/skills/prototype-spec-extractor/`** 或 **`.vibex/.agents/skills/prototype-spec-extractor/`**（二者等价参与合并，同名 SKILL 时 **`.vibex/.agents/skills` 覆盖** `.vibex/agents/skills` 与根目录 `skills/`）。请求体使用 **`agent_profile=prototype-spec-extractor`** 时，Agent 会加载本 `SKILL.md`（`required_skills` 与 frontmatter `name` 一致）。合并目录为：`<workspace>/skills` → `.vibex/agents/skills` → `.vibex/.agents/skills`（后者优先）。

本技能定义 **Agent**（本机 Go Agent + 扩展侧栏会话）在处理 **HTML 原型与 VibeX L1–L5 YAML** 时的**专业分工、层级规则、字段约定与闭环流程**。扩展负责 DOM 上下文、侧栏 UI、可选消息桥；Agent 负责读写作业区文件、多轮澄清与 specs/write 前的草案。

---

## 1. 能力总览（与你划的四条一一对应）

| # | 能力 | Agent 职责 | 扩展 / 页面 职责 |
|---|------|--------------|------------------|
| 1 | **HTML 原型 → spec 拆解** | 按 §2 层级表生成/补全 YAML；处理 **层级抬升** 与 **ui_links**（§3） | 提供选中节点、截图语义、data- 属性提示；可选注入 `proto-spec-runtime` 的 hub 数据 |
| 2 | **实时读写本地原型** | NL → 结构化追问 → 草案；确认后 **双写** `spec` + `.vibex/prototypes/*.html`（须门禁） | 展示映射表、diff、预览 URL；配置 `workspace_root` 与 Agent 基址 |
| 3 | **任意网页抽 UI/UX** | 多层级 spec 草案 + 提取范围元数据（URL、时间、标签） | 静态 DOM 树或可访问的片段；声明「仅静态 / 含动态」边界 |
| 4 | **spec ↔ 绑定区块 ↔ spec 双向** | 改 YAML 同步改 HTML `data-ps-*`；改 HTML 反推槽位与 **ui_links** | 高亮 selector、postMessage 事件日志 |

---

## 2. HTML 原型拆解：层级映射（硬规则）

从 **信息架构责任** 映射到 **VibeX 层级**，不是从「文件大小」猜。

| 原型粒度 | 目标层级 | 典型落点 | 说明 |
|----------|----------|----------|------|
| **页面级**（整页 shell、路由壳、全局布局） | **L1 MVP 目标** 与/或 **L2 骨架** | `L1-goal/*.yaml`、`L2-skeleton/*.yaml` | L1 写边界与验收；L2 写区域划分、主导航、与模块边界 |
| **区域级**（主内容区、侧栏、顶栏功能区、业务「一块屏」） | **L3 模块** | `L3-module/*.yaml` | 一个区域对应一个模块或子域；**弹窗/抽屉/tab 若承载独立业务语义，默认归区域级讨论** |
| **组件级**（卡片、表单段、列表单元、按钮组） | **L4 功能** | `L4-feature/*.yaml` | 可用户可见的完整小功能 |
| **行为级**（点击后请求、校验、动画、快捷键、仅技术步骤） | **L5 切片** | `L5-slice/*.yaml` | 实现细节、API、状态机步骤 |

### 2.1 层级抬升（弹窗 / 抽屉 / Tab 等）

当 UI 在视觉上「附在」某页内，但 **改变全局任务上下文** 或 **跨多个 L4 组件编排** 时，Agent **不得**把其仅写在 L5：

- **抽屉、全屏弹层、多步向导**：至少抬到 **L3 区域** 或独立 **L4**（二选一原则：若主要服务某一区域 → 挂在该区域 L3 下并在 L4 列子功能；若全局设置/多模块入口 → 单独 L4 + **ui_links**）。
- **Tab 切换若切换「业务子域」**（如设置里「账号 vs 计费」）：拆成多个 L4 或在 L3 写 **io_contract** 分区，并用 **ui_links** 链到各 L4。
- **纯局部反馈**（toast、行内 error）：可留在 L5 或 L4 的 `io_contract.behavior`。

---

## 3. 非父子关联：`ui_links`（类 goto，不替代 parent/children）

**禁止**用 `parent`/`children` 表达「打开哪个面」「从哪跳到哪」；使用并列字段 **`ui_links`**（可放在 `prototype` 下或 `io_contract` 下，按现有 spec 风格二选一，**全仓统一一种**）。

推荐形态（示例）：

```yaml
prototype:
  file: .vibex/prototypes/checkout-flow.html
  ui_links:
    - id: open_cart_drawer
      rel: opens_overlay        # opens_overlay | navigates_view | spawns_flow | depends_on
      from:
        layer_hint: L4
        summary: 头部购物车图标
      target:
        spec_path: specs/L3-module/MOD-cart-panel.yaml
        anchor: region-cart-drawer
      selector_hints:
        - '[data-ps-region="cart-drawer"]'
        - 'header .cart-icon'
      notes: 抽屉内 Tab 切换见 MOD-cart-panel 内 ui_links；不在本 L4 重复展开。
    - id: goto_payment
      rel: navigates_view
      from:
        spec_path: specs/L4-feature/FEAT-checkout-form.yaml
      target:
        spec_path: specs/L4-feature/FEAT-payment.yaml
      selector_hints:
        - '[data-ps-behavior="submit-checkout"]'
```

**字段含义简述**

- `rel`：关系类型（可扩展枚举，Agent 与用户确认后写入）。
- `from` / `target`：至少一端有 `spec_path`；另一端可用 `layer_hint` + `summary` 占位待落文件。
- `anchor`：与 HTML 内 **`data-ps-anchor`** 或 **`id`** 对齐，供双向推导。
- `selector_hints`：便于扩展高亮与回归；**非唯一真理**，以 `prototype.file` 内 DOM 为准迭代。

---

## 4. 闭环 A：自然语言 → spec 追问 → HTML 区块映射 → 行为展示 → 预览确认 → 双写

**阶段**

1. **意图**：用户一句 NL → Agent 列出「将影响的层级 + 将改的文件路径」，不明确则只问不写。
2. **spec 侧追问**：缺 L3 边界、缺验收、缺 `ui_links` 目标 → 用 checklist 追问（最多两轮结构化问题）。
3. **HTML 映射**：为将改区块分配/更新 **`data-ps-spec`**（可选）、**`data-ps-region`**、**`data-ps-behavior`**（见 §6）；自包含样式原则不变。
4. **行为展示**：用列表或表格输出「用户可见行为 → 对应 spec 路径 → 对应 selector」。
5. **预览确认**：用户口头或 UI「确认」前，仅输出 **草案**（diff 块或附件路径）；**禁止**静默 specs/write。
6. **双写**：先写 `.vibex/prototypes/*.html`，再写 `specs/**/*.yaml`（或按项目门禁顺序）；两处 **同一 `anchor` / `ui_links.id`** 对齐。

Agent 在扩展会话中若未带 `workspace_root`，必须先提示用户在侧栏配置（与扩展 `agent-client` 一致）。

---

## 5. 闭环 B：任意网页「提取 UI/UX」

当用户明确要 **抽整页/抽组件库/抽设计系统** 时：

1. **定边界**：静态 HTML / SSR 首屏 / 声明不支持的动态壳（见 L2 prototype-extractor `constraints`）。
2. **多层级产出**：同时给出 L2 区域划分草案 + L3/L4 候选列表 + L5「仅当已从页面可观测行为推断」。
3. **元数据**：在 HTML 文件头注释或 YAML `meta` 写入 `source_url`、`extracted_at`、`tags[]`。
4. **不虚构**：页面不可见的 state/API 不得写成确定事实，标为 `assumption` 或 `open_question`。

---

## 6. 双向推导：spec ↔ 页面绑定区块

### 6.1 HTML 侧约定（建议在原型内统一）

| 属性 | 用途 |
|------|------|
| `data-ps-spec` | 绑定到 `specs/...` 的稳定 slug 或相对 path（团队选一种规范） |
| `data-ps-region` | 区域锚点，与 YAML `ui_links[].target.anchor` 对齐 |
| `data-ps-behavior` | 可点击/提交等行为锚点，优先 L4/L5 描述引用 |

从 **spec → HTML**：根据 `prototype.file` + `ui_links` 在 Agent 改 HTML 时补齐/修正上述属性。

从 **HTML → spec**：解析 DOM → 反推缺失的 L4/L5 与 **ui_links**；若发现 `data-ps-region` 与现有 L3 不一致，触发 **合拆模块** 建议而非静默合并。

---

## 7. 与工具及门禁的衔接

- **写 spec**：仅通过用户确认后的 **specs/write**（或项目等效流程）；`ui_links` 变更属易误触高敏，须在回复中高亮 diff。
- **写 HTML**：仅写入 **`.vibex/prototypes/`**（或项目 Design Kit 约定的可交付目录）；与 `FEAT-workspace-design-prototype-gate` 一致时，走 **confirm** 路径。
- **扩展**：扩展侧可投递 `POST /api/chat`；流式 SSE 在 Workbench 查看；Agent 须能 **read_file / write_file** 于用户 `workspace_root`。

---

## 8. 显式非目标（当前阶段不做）

- 不接 CDP 做完整「环境探针 + 自动 E2E」（后续另技能）。
- 不在本技能内定义 Playwright 用例生成（可引用 `SLICE-qa-playwright-scenario-runner` 另开任务）。
- 不替代 **parent/children** 表达分解结构；**ui_links** 只表达 **导航、依赖、弹层归属** 等横切关系。

---

## 9. Agent 自检清单（每次任务结束前）

- [ ] 每个弹窗/抽屉是否已决定 **L3/L4** 归属并写 **ui_links**？
- [ ] `prototype.file` 与 HTML 头注释中的 **spec 溯源** 是否一致？
- [ ] 是否存在 **未标注 assumption** 的推断？
- [ ] 是否未获确认就写盘？

---

*技能版本：0.1 · 扩展见 `extensions/prototype-extractor`；L2 见 `.vibex/specs/L2-skeleton/L2-prototype-extractor.md`。*
