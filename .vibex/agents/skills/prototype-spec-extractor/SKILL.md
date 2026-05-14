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
title: Prototype–Spec Extractor（扩展自主对话）
triggers:
  - zh: 原型拆解, HTML转spec, 原型映射, 提取UI, 提取UX, spec绑定, 区块绑定, 弹窗层级, 抽屉, Proto Spec, 原型提取
  - en: prototype to spec, extract UI spec, HTML prototype spec, bound region, proto-spec extractor
related_skills:
  - spec-designer: 视觉与信息架构意图；本技能负责「层级落位 + 绑定字段 + 与 YAML 图谱一致」。
  - spec-governance: 父子链与治理字段；本技能的 ui_links 不替代 parent/children，仅表达导航与跨层关联。
  - vibex-spec-writing-guide: L1–L5 槽位与文件命名惯例。
---

# Prototype–Spec Extractor（扩展自主对话 Agent 技能）

**规范路径**：与本目录 **`agent.json`** 成对放置于 **`.vibex/agents/skills/prototype-spec-extractor/`** 或 **`.vibex/.agents/skills/prototype-spec-extractor/`**（后者优先）。请求体使用 **`agent_profile=prototype-spec-extractor`** 时，Agent 加载本 `SKILL.md`。

---

## 架构说明

扩展侧栏**是独立的对话窗口**，直接与 Go Agent 交互，**不经过 Workbench 中转**。

```
用户（扩展侧栏） ←→ Agent SSE  ←→ Go Agent
        ↓                              ↓
   DOM / content script         写盘：specs / prototypes
```

- **扩展职责**：原型浏览、DOM 解析、spec 拆解对话、实时高亮/标注。专注原型及 spec 拆解。
- **Agent 职责**：读盘、写盘、推理、补全 YAML。多轮对话在扩展侧栏直接呈现。
- **Workbench 职责**：spec 治理（父子链、门禁、governance consistency）。**不参与扩展对话流**。

Agent 写 spec 完成后，通过 `agent:spec:update` SSE 事件通知 Workbench（可选，Workbench 自身 spec 图谱刷新）。

---

## 0. 入口协议：扩展读取原型路径（第一步硬规则）

**每次任务开始前，扩展必须先完成路径解析，将上下文告知 Agent。**

当用户在 Chrome 扩展地址栏粘贴原型文件路径（格式：`C:/project/repo/.vibex/prototypes/my.html`）时，扩展 content script 自动完成以下分割：

| 路径段 | 示例 | 说明 |
|--------|------|------|
| `workspaceRoot` | `C:/project/repo` | `.vibex/prototypes/` 前半部分 |
| `prototypeRel` | `my.html` | `.vibex/prototypes/` 后的相对路径 |
| `specRoot` | `C:/project/repo/.vibex/specs` | 固定拼接 `.vibex/specs` |
| `level` | `L1` / `L3` / `L4` / `L5` | 从 `prototypeRel` 路径中的目录段推断（L3/L4/L5 通常在对应层级目录下） |

Agent 收到后，**立即在扩展侧栏开启对话**，无需用户手动填写工作区路径。

---

## 1. 层级派生约定：L1 骨架 + L3–L5 覆盖

### 1.1 层级职责

| 原型类型 | 目标层级 | 原型文件约定 | 说明 |
|----------|----------|--------------|------|
| **页面骨架** | L1（L1-goal） | `.vibex/prototypes/{goal-name}.html` | 含完整 HTML 壳（`<html><head><body>`），是所有派生的根 |
| **区域派生** | L3（L3-module） | `.vibex/prototypes/L3-module/{module-name}.html` | 在 L1 骨架的对应区域注入 web components |
| **功能派生** | L4（L4-feature） | `.vibex/prototypes/L4-feature/{feature-name}.html` | 同上，粒度到单个小功能 |
| **组件派生** | L5（L5-slice） | `.vibex/prototypes/L5-slice/{slice-name}.html` | 同上，粒度到实现细节 |

### 1.2 L3–L5 派生原型 HTML 写法

派生原型**复用 L1 骨架**，在功能区域用 **web components** 覆盖：

```html
<!-- L3 派生原型示例：引用骨架（可选） -->
<link rel="skeleton" href="../{L1-goal-name}.html">

<!-- 功能区覆盖：使用自定义标签包裹 -->
<my-module-comp data-level="L3" data-skeleton-placeholder>
  <!-- 这里的内容会叠加到骨架对应区域 -->
</my-module-comp>

<script>
  class MyModuleComp extends HTMLElement {
    connectedCallback() {
      this.style.border = '2px dashed #f59e0b';  // L3 = 橙色
      this.style.outline = '2px solid #f59e0b20';
      this.style.display = 'block';
      this.style.position = 'relative';
    }
  }
  customElements.define('my-module-comp', MyModuleComp);
</script>
```

**层级颜色约定**：

| 层级 | 颜色 |
|------|------|
| L3 | `#f59e0b`（橙色） |
| L4 | `#8b5cf6`（紫色） |
| L5 | `#06b6d4`（青色） |

### 1.3 派生原型与 Spec 的对应关系

每个 L3–L5 spec 对应一个派生原型文件：

```
.spec.yaml  ←  →  .vibex/prototypes/L{N}-*/{spec-name}.html
     ↑                                    ↑
  spec 字段定义                    web components 覆盖骨架区
```

- spec 的 `io_contract.ui` 字段描述该区域的结构和状态
- 派生原型 HTML 中的 `<{spec-name}>` 标签展示 `io_contract.ui` 中定义的内容
- 两者通过相同的 `spec-name`（不含 `.yaml` 后缀）关联

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

## 4. 扩展侧栏对话闭环

扩展侧栏**是唯一的对话界面**，直接接收 Agent 回复，无需 Workbench 中转。

### 4.1 扩展侧栏职责

- 维护 `EventSource("/api/sse/{threadId}")` 订阅，接收 Agent 的所有回复流
- 将回复渲染到侧栏对话线程
- 用户在侧栏输入框继续对话，`POST /api/chat` 驱动下一轮

### 4.2 对话流程

1. **路径解析**（扩展入口）：用户粘贴原型路径 → content script 解析 `.vibex/prototypes/` 分割 → 写入 `browser.storage.local`
2. **发起对话**：`POST /api/chat` 带上 workspace_root 和 prototypeRel，侧栏建立 SSE 订阅
3. **多轮对话**：Agent 在侧栏直接回复；用户继续追问或确认
4. **spec 追问**：Agent 列出「将影响的层级 + 将改的文件路径」，不明确则只问不写
5. **预览确认**：用户口头或 UI「确认」前，仅输出 **草案**
6. **双写**：Agent 先写 `.vibex/prototypes/*.html`，再写 `specs/**/*.yaml`（两处 **同一 `anchor` / `ui_links.id`** 对齐）
7. **完成通知**：Agent 通过 `agent:spec:update` SSE 事件通知 Workbench（Workbench 自身 spec 图谱刷新，不参与对话）

### 4.3 PS 工具（Agent → 扩展执行）

Agent 通过 SSE 推送 `agent:tool:call` 事件，扩展接收后执行：

| 工具名 | 说明 |
|--------|------|
| `ps_highlight` | 高亮 CSS selector 元素 |
| `ps_annotate` | 显示/隐藏节点标注 |
| `ps_parse` | 解析当前页面为 Spec 树 |
| `ps_bind` | 将 Spec 绑定到 DOM selector |
| `ps_onboard` | 执行引导演示 |
| `ps_get_page_context` | 获取当前页面上下文（含 workspaceRoot / prototypeRel / specRoot） |

扩展执行完后通过 `POST /api/extension/tool-result` 回调，Agent 收到后继续 stream 回复。

---

## 5. 闭环 A：自然语言 → spec 追问 → HTML 区块映射 → 行为展示 → 预览确认 → 双写

**阶段**

1. **路径解析**（扩展入口）
2. **意图**：用户一句 NL → Agent 列出「将影响的层级 + 将改的文件路径」
3. **spec 侧追问**：缺 L3 边界、缺验收、缺 `ui_links` 目标 → checklist 追问（最多两轮）
4. **HTML 映射**：补齐 **`data-ps-spec`**、**`data-ps-region`**、**`data-ps-behavior`**
5. **行为展示**：列表输出「用户可见行为 → 对应 spec 路径 → 对应 selector」
6. **预览确认**：用户「确认」前，仅输出 **草案**；**禁止**静默写盘
7. **双写**：先写 `.vibex/prototypes/*.html`，再写 `specs/**/*.yaml`

---

## 6. 闭环 B：任意网页「提取 UI/UX」

1. **定边界**：静态 HTML / SSR 首屏 / 声明不支持的动态壳
2. **多层级产出**：L2 区域划分草案 + L3/L4 候选列表 + L5（仅从页面可观测行为推断）
3. **元数据**：写入 `source_url`、`extracted_at`、`tags[]`
4. **不虚构**：页面不可见的 state/API 不得写成确定事实，标为 `assumption` 或 `open_question`

---

## 7. 双向推导：spec ↔ 页面绑定区块

### 7.1 HTML 侧约定

| 属性 | 用途 |
|------|------|
| `data-ps-spec` | 绑定到 `specs/...` 的稳定 slug 或相对 path |
| `data-ps-region` | 区域锚点，与 YAML `ui_links[].target.anchor` 对齐 |
| `data-ps-behavior` | 可点击/提交等行为锚点，优先 L4/L5 描述引用 |

从 **spec → HTML**：根据 `prototype.file` + `ui_links` 补齐/修正上述属性。

从 **HTML → spec**：解析 DOM → 反推缺失的 L4/L5 与 **ui_links**；若发现 `data-ps-region` 与现有 L3 不一致，触发 **合拆模块** 建议。

---

## 8. 与工具及门禁的衔接

- **写 spec**：仅通过用户确认后的 **specs/write**（或项目等效流程）；`ui_links` 变更属高敏，须在回复中高亮 diff。
- **写 HTML**：仅写入 **`.vibex/prototypes/`**（L1 骨架在根目录，L3-L5 派生在对应层级子目录）；与 `FEAT-workspace-design-prototype-gate` 一致时，走 **confirm** 路径。
- **对话**：扩展侧栏直接 `POST /api/chat` + SSE 订阅；**不通过 Workbench 中转**。
- **Agent**：须能 **read_file / write_file** 于用户 `workspace_root`。

---

## 9. 显式非目标（当前阶段不做）

- 不接 CDP 做完整「环境探针 + 自动 E2E」（后续另技能）。
- 不在本技能内定义 Playwright 用例生成。
- 不替代 **parent/children** 表达分解结构；**ui_links** 只表达 **导航、依赖、弹层归属** 等横切关系。

---

## 10. Agent 自检清单（每次任务结束前）

- [ ] 扩展是否已读取原型路径并解析出 `workspaceRoot / prototypeRel / specRoot / level`？
- [ ] L1 骨架原型是否包含完整 HTML 壳（`<html><head><body>`）？
- [ ] L3-L5 派生原型是否使用 `<{spec-name}>` 自定义标签覆盖骨架区？
- [ ] 每个弹窗/抽屉是否已决定 **L3/L4** 归属并写 **ui_links**？
- [ ] `prototype.file` 与 HTML 头注释中的 **spec 溯源** 是否一致？
- [ ] 是否存在 **未标注 assumption** 的推断？
- [ ] 是否未获确认就写盘？

---

*技能版本：0.3 · 扩展见 `extensions/prototype-extractor`；通道协议见 `.vibex/specs/L4-feature/FEAT-prototype-skill-channel.yaml`；L2 见 `.vibex/specs/L2-skeleton/L2-prototype-extractor.md`。*
