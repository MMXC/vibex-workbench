# Spec `vision_traceability` 推广路线

面向 **L3 模块** 与 **L4 feature**：在 `content` 下增加与 `specs/meta/spec-layer-contract.yaml` 对齐的 `vision_traceability` 段落，把 L1 约束与 L2 骨架锚点写清，便于评审与 drift/抽检扩展。

---

## 1. 契约与范本

| 产物 | 路径 |
|------|------|
| 层级最低集 | `specs/meta/spec-layer-contract.yaml`（`layers.*.must_contain`） |
| 拆解与追溯阅读 | `docs/spec-goal-decomposition-traceability.md` §9–10 |
| L1 约束定义 | `specs/project-goal/vibex-workbench-goal.yaml`（C1–C8） |

**范本（结构照抄，改路径/约束编号/摘要）**：见同文件 `rollout.pilot_m0.exemplar_paths`。

---

## 2. 当前完成度（快照）

### 2.1 五 MOD（L3）— 已全部具备 `vision_traceability`

- `MOD-workbench-shell`
- `MOD-spec-engine`
- `MOD-router`
- `MOD-dsl-visualizer`
- `MOD-code-generator`

### 2.2 与各 MOD 主路径对齐的 L4 — 已具备

| MOD | L4 feature |
|-----|------------|
| MOD-workbench-shell | `workbench-ide-chrome`（pilot） |
| MOD-spec-engine | `spec-editor` |
| MOD-router | `routing-panel` |
| MOD-dsl-visualizer | `dsl-canvas` |
| MOD-code-generator | `code-gen-panel` |

### 2.3 workbench-shell 子树 L4（Phase A 已完成）

已写入 `content.vision_traceability`：

| Spec | 路径 |
|------|------|
| 聚合 L4 | `specs/feature/workbench-shell/workbench-shell_feature.yaml` |
| 布局 | `specs/feature/workbench-shell/workbench-layout_resize_feature.yaml` |
| 对话 | `specs/feature/workbench-shell/workbench-conversation_feature.yaml` |
| Canvas 渲染 | `specs/feature/canvas-renderer/canvas-renderer_feature.yaml`（`parent: MOD-workbench-shell`，独立目录） |

### 2.4 Phase B：L5 代表性 `trace_note`（已于本仓库试点）

在 **`content.trace_note`** 增加 2–4 行可追溯说明（不复制 L4 `vision_traceability` 全文）。试点文件：

| 层级 | 路径 |
|------|------|
| L5d | `specs/feature/spec-editor/spec-editor_test.yaml` |
| L5a | `specs/feature/spec-editor/spec-editor_uiux.yaml` |
| L5d | `specs/feature/workbench-shell/workbench-shell_test.yaml` |
| L5d | `specs/feature/routing-panel/routing-panel_test.yaml` |
| L5d | `specs/feature/code-gen-panel/code-gen-panel_test.yaml`（仍为 draft） |

其余 L5 切片按需增补，避免全仓库同质化堆砌。

---

## 3. 推广阶段（建议顺序）

### Phase A — workbench-shell  subtree 收口（已完成）

| 顺序 | Spec | 约束锚点（落地） |
|------|------|------------------|
| A1 | `workbench-shell_feature.yaml` | C1、C6、C7 |
| A2 | `workbench-layout_resize_feature.yaml` | C1、C5、C6 |
| A3 | `workbench-conversation_feature.yaml` | C1、C6、C7 |
| A4 | `canvas-renderer_feature.yaml` | C6、C7；附与 `dsl-canvas` 分工 |

### Phase B — L5 切片（试点已完成；其余按需）

`spec-layer-contract` 对 L5 的要求侧重 `parent_chain` 与可验收细节；**不强制**每文件冗长 `vision_traceability`。已在代表性 `*_test.yaml` / `*_uiux.yaml` 增加 **`content.trace_note`**（见 §2.4）；其它 L5 可按主线（如 dsl-canvas_test、canvas-renderer_uiux）逐个补一行引用 parent L4 即可。

### Phase C — 模板与 CI（已完成占位）

| 产物 | 说明 |
|------|------|
| `specs/feature/feature-template/feature_template_feature.yaml` | L4 生成用 `template:` 内嵌 **`vision_traceability`** 占位（占位约束 C1/C7，`l3.module_spec_path` 随 `${PARENT_ID}`；生成非 MOD parent 时需人工改路径） |
| `specs/feature/feature-template/feature_template_uiux.yaml` | L5a `template:` 内含可选 **`trace_note`** 占位 |
| `specs/meta/snippets/l4-vision-traceability.stub.yaml` | 纯手工新建 L4 时可粘贴的片段（无顶层 `spec`，不参与命名 spec 索引） |
| Makefile | `# --- Spec validation ---` 段注释标明 **lint-specs → validate** 为一阶门禁 |

**CI**：仍以 `make lint-specs`、`make validate`（及按需 `make drift`）为准；段落存在性机器抽检待定，单一真相源保持 **`spec-layer-contract.yaml`**。

---

## 4. 约束编号速查（选用时对照 goal 正文）

| ID | 名称 | 典型挂载模块 / 场景 |
|----|------|---------------------|
| C1 | 全产品自动化 | 路由、编辑器、生成面板 |
| C2 | 输入输出确认制 | Router、澄清与 ConfirmedIO |
| C3 | 从属链完整 | spec-engine、图与生成输入 |
| C4 | 自举验证 | code-generator、generate 管线 |
| C5 | Local-first | layout 持久化、IndexedDB |
| C6 | 行为展示驱动 | dsl-visualizer、conversation、canvas |
| C7 | 分层门禁与 todo | 全体；validate 聚合 |

同一 L4 可选 **3–5 个**约束，避免堆砌；module 层通常 **2–4 个**。

---

## 5. 纪律

1. 每批改动后：`make lint-specs` && `make validate`。
2. `io_contract.changelog` 保持**列表项**缩进，禁止重复嵌套键名 `changelog:`。
3. 版本号：`spec.version` 与 changelog 同步递增；`meta.updated` 填实际修订日。
4. 聚合 L4（如 `workbench-shell`）的 `parent` 仍为 `MOD-workbench-shell`；子能力挂其下的 feature 继续用 `parent: workbench-shell`（与现有从属链一致）。

---

## 6. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-04-22 | 初版：五 MOD + 主 L4 已完成；Phase A–C 路线与约束速查 |
| 2026-04-22 | Phase A 完成：`workbench-shell` 聚合 + layout-resize + conversation + `canvas-renderer` |
| 2026-04-22 | Phase B 试点：`content.trace_note` 见于 spec-editor / shell / routing / codegen 等 5 个 L5 文件 |
| 2026-04-22 | Phase C：`feature_template_*` 模板 + `snippets/l4-vision-traceability.stub.yaml`；Makefile 注释门禁 |
