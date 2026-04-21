# Spec 落地层缺口登记（与治理层对照）

本文档记录仓库内已有共识：**治理层（写什么、如何校验、愿景追溯）已公式化；落地执行层（L5 service/test → 代码、agent 行为、review 工具化）仍有系统性缺口。**  
与 `SPEC-TO-SHIP-WORKFLOW.md` 互补：后者描述**当前能做的默认步骤**；本文描述**步骤之外仍缺什么**。  
Agent 行为统一入口的**占位骨架**见 `SPEC-EXECUTION-PROTOCOL.md`。

---

## 1. 治理层——已相对公式化

下列命令在项目根执行，顺序与含义在 `Makefile` 与 `generators/validate_specs.py`、`spec-engine/validate_chain.py` 中实现：

| 步骤 | 命令 | 作用 |
|:----:|------|------|
| 1 | `make lint-specs` | YAML 语法 + parent 引用文件存在性 |
| 2 | `make validate` | 从属链 / 层级一致性（含上一步） |
| 3 | `make generate` | `generators/gen.py`：spec → 前端生成物（见 §3 覆盖范围） |
| 4 | `make drift` | spec 与约定生成物的漂移检测 |

L1 **C7**（变更 → todo-spec）与 **C4**（自举证明）在 `specs/project-goal/vibex-workbench-goal.yaml` 中定义；**vision_traceability** 推广见 `VISION-TRACEABILITY-ROLLOUT.md`。

---

## 2. 落地层——缺口陈述

### 缺口 1：`gen.py` 覆盖面前端为主，未覆盖 L5 service / test 的实现生成

`generators/gen.py` 文档写明：从 **L5 data / uiux** 生成 TypeScript 类型与 Svelte Skeleton（双文件模式）；**不包含**把 **L5b service**（`io_contract`、接口签名、边界逻辑）机械地变成 Service 层可运行代码，也**不包含**从 **L5d test** 生成可执行测试套件。

因此「data → 类型」「uiux → 骨架」路径存在；「service → 后端/服务实现」「test → 自动化测试」**没有对应的生成器管线**。

### 缺口 2：L4 feature → 可运行系统的「全公式」尚未闭合

理想分解（示意）：

```text
L4 feature spec
    ├─ L5a uiux   → gen.py → Svelte Skeleton（覆盖度因 feature 而异）
    ├─ L5c data   → gen.py → TS types（+ 动态发现 *_data）
    ├─ L5b service → （缺）统一生成器
    └─ L5d test   → （缺）统一生成器 / 或固定脚手架
```

外部 skill 中的「Step 7：手动填 20%」——在仓库语境里正是 **service + test（及 router/agent  glue）** 最常落入的手工区；**这 20% 尚未被仓库内公式化为可重复的生成步骤或检查清单**。

### 缺口 3：Agent 工作流未固化为单一「执行协议」文档

C4 要求「spec 驱动生成」，但 **谁**按何顺序读哪些 spec、**何时**调用 `make generate`、**何时**必须手写、与 **spec-frontend** 等 skill 如何对齐——规则分散在多个 skill 与各 feature 的 `io_contract` / `implementation` 中，**缺少一份统一的 spec→实现 执行手册（execution protocol）**，便于人和 agent 共用。

### 缺口 4：C2（IO 确认）工具化不足

L1 定义了输入输出确认制；**合并前 review 节点**（diff 门禁、必填字段检查）尚未与 CI 全面绑定，更多依赖人工与惯例。

---

## 3. 完成度快照（主观、用于对齐优先级）

| 层次 | 大致完成度 | 代表产出 |
|------|------------|----------|
| Spec 层级设计 + parent chain | 高 | `spec-layer-contract.yaml`、`directory-convention` |
| CI 门禁管道（lint / validate / drift） | 高 | `Makefile`、`validate_specs.py`、`validate_chain.py` |
| L4 写法规范 + vision_traceability | 高 | `VISION-TRACEABILITY-ROLLOUT.md` |
| L5 → UI（types + Skeleton） | 中–高 | `generators/gen.py` |
| L5 → service / test 实现生成 | 低 | 几乎空白 |
| Agent 读 spec → 落地的行为契约（单一文档） | 低 | 待 `SPEC-EXECUTION-PROTOCOL.md`（或等价物） |
| Spec 先行 → Review / C2 工具化 | 中 | 依赖流程，缺统一工具 |

**与里程碑的关系**：自举闭环（如 goal 中 M7 / C4）要验收「规格能驱动可运行产物」，**缺口 1–3 是当前主要堵点**——不是否定现有门禁，而是明确**最后一公里尚未自动化**。

---

## 4. 建议的后续产物（非承诺路线图）

以下名称可在后续 PR 中逐项落地：

1. **`SPEC-EXECUTION-PROTOCOL.md`**（仓库根，**已为占位骨架**）  
   面向 agent/人：读 spec 的顺序、何时 `generate`、何时手写、如何登记 todo（C7）、与外部 skill 的边界。

2. **Service / Test 生成或脚手架策略**  
   小型迭代可考虑：仅生成接口桩 + 测试模板文件名约定；完整「spec-to-service」需单独设计范围（前端 SvelteKit vs 后端目录）。

3. **CI 可选增强**  
   在现有 `lint-all` 之上，按需增加「段落存在性」抽检（以 `spec-layer-contract.yaml` 为源），与缺口登记无关但降低漂移。

---

## 5. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-04-22 | 初版：治理 vs 落地缺口对照，与 gen.py 实际覆盖一致 |
| 2026-04-22 | 指向 `SPEC-EXECUTION-PROTOCOL.md` 占位 |
