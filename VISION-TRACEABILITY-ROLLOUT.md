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

### 2.3 尚未覆盖的 L4（workbench-shell 子树）

以下文件**尚无** `content.vision_traceability`，按优先级分批补齐：

1. **`workbench-shell`**（聚合 L4，`parent: MOD-workbench-shell`）— 建议先做：为子 feature 提供统一 L1/L2 锚点与边界。
2. **`workbench-layout-resize`** — 与 IDE Chrome、持久化键强相关，验收与 R2 原型对照频繁。
3. **`workbench-conversation`** — 对话/SSE 与 C6、右侧 AI 栏契约交叉。
4. **`canvas-renderer`** — Canvas 渲染与 dsl-canvas / shell 边界。

---

## 3. 推广阶段（建议顺序）

### Phase A — workbench-shell  subtree 收口（当前焦点）

在同一子目录 `specs/feature/workbench-shell/` 内连续改，单次 MR 可含 2–4 个文件，改完跑门禁。

| 顺序 | Spec | 约束锚点建议 |
|------|------|----------------|
| A1 | `workbench-shell_feature.yaml` | C1、C6、C7；聚合叙事对齐 `experience_visualization` |
| A2 | `workbench-layout_resize_feature.yaml` | C1、C5（local-first 持久化）、C6 |
| A3 | `workbench-conversation_feature.yaml` | C1、C6、C7 |
| A4 | `canvas-renderer_feature.yaml` | C6、C7；与 MOD-dsl-visualizer 边界对照 `boundaries.not_here` |

### Phase B — L5 切片（按需）

`spec-layer-contract` 对 L5 的要求侧重 `parent_chain` 与可验收细节；**不强制**每文件冗长 `vision_traceability`。若要为某一主线（例如 spec-editor 测试闭环）增强可追溯性，可在个别代表性的 `*_test.yaml` 或 `*_uiux.yaml` 增加简短 `trace_note`（字段名与生成器约定为准），避免全量 20+ 文件同质化堆砌。

### Phase C — 模板与 CI

- `specs/meta/*template*`：在 feature 模板中预留 `vision_traceability` 占位（可选）。
- CI：保留 `make lint-specs` + `make validate`；后续若接入「段落存在性」抽检，以 `spec-layer-contract.yaml` 为单一真相源。

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
