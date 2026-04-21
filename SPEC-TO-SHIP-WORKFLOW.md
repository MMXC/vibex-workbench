# Spec → 落地操作卡

把「改规格」到「可合并的实现」收成一条可重复的默认路径。细节仍以各 YAML 正文为准；门禁语义见 `specs/project-goal/vibex-workbench-goal.yaml`（C7、`spec_workflow_units`）。

**边界**：本卡覆盖 **门禁 + `make generate` 已实现的产物**（主要为 L5 data/uiux → TS 类型与 Svelte Skeleton）。**L5 service/test 生成、agent 统一执行协议、C2 工具化**等落地层缺口见 **`SPEC-LANDING-GAPS.md`**（与治理层完整度对照）。

---

## 1. 默认流水线（每条变更至少走一遍）

| Step | 做什么 | 命令 / 产出 |
|:----:|--------|-------------|
| **1** | 改 `specs/**`：`spec.parent` 合法；层级与该层 `must_contain` 对齐（见下方引用） | 手工编辑 YAML |
| **2** | **聚合门禁**：语法 + 父文件存在；再从「名字 → parent」深度查一层级 | `make lint-specs` → `make validate`<br>（或直接 `make validate`，已隐含 lint-specs） |
| **3** | 读本能力 **L4/L5** 里的 `content.io_contract`、`implementation` / `changelog`：**判定**本次是「生成物更新」还是「手写维护路径」 | 无统一命令——读 spec |
| **4a** | 若生成器消费该改动（类型、枚举、图谱等）：更新前端生成代码 | `make generate`（**已内含** `lint-specs`） |
| **4b** | 若标明「开发者维护」（如壳层组件、bridge）：直接改源码，不要求本步 generate | IDE / 编辑器 |
| **5** | 契约与手写 TS/Svelte **是否漂移**（启用 drift 时） | `make drift` |
| **6** | 变更影响 L5 test 或里程碑时：**登记 todo / 关闭关联门禁**（C7） | backlog / PR 描述 / todo-spec |

合并前推荐的「一遍过」：

```bash
make lint-all    # validate + drift；validate 已含 lint-specs
```

自举或发版前可用：

```bash
make self-generate   # validate + generate；适合证明 specs 仍可驱动生成
```

---

## 2. 与「层级契约 / 愿景追溯」的对照（不必每一步都敲）

| 目的 | 文档 |
|------|------|
| 治理 vs 落地缺口（诚实登记） | `SPEC-LANDING-GAPS.md` |
| Agent/人执行协议（骨架） | `SPEC-EXECUTION-PROTOCOL.md` |
| 每层最少写啥 | `specs/meta/spec-layer-contract.yaml` |
| L1 约束与门禁哲学 | `specs/project-goal/vibex-workbench-goal.yaml`（C1–C8、`spec_workflow_units`） |
| vision_traceability 推广状态与约束速查 | `VISION-TRACEABILITY-ROLLOUT.md` |
| 目标拆解与阅读顺序 | `docs/spec-goal-decomposition-traceability.md` |

新建 L4 时可粘贴：`specs/meta/snippets/l4-vision-traceability.stub.yaml`；生成器模板见 `specs/feature/feature-template/`。

---

## 3. `make` 目标速查（仓库根执行）

| 目标 | 含义 |
|------|------|
| `make validate` | `lint-specs` + `spec-engine/validate_chain.py`（**推荐作为合并前 spec 侧门槛**） |
| `make generate` | `lint-specs` + `generators/gen.py`（更新 `frontend` 下生成物） |
| `make drift` | `generators/drift_check.py`（spec 与代码约定是否一致） |
| `make lint-all` | `validate` + `drift` |
| `make self-generate` | `validate` + `generate` |
| `make dev` / `make build` | 依赖路径上会先 **generate**（故会先 lint-specs） |

---

## 4. 常见分叉（避免「一律 generate」）

- **生成管线**：`make generate` 覆盖 `generators/gen.py` 所实现产物；改 spec 后若属于生成范围，应在 MR 中带生成 diff。
- **手写维护**：许多 feature 写明「不手改某生成文件 / 由某路径维护」——以对应 **L4 `io_contract` / `implementation`** 为准；此时 Step 4 走 **4b**，仍须 **Step 2** 绿。
- **最小可验单元**：局部行为以对应 **`5d_test`** 或仓内测试为准；整仓 spec 树一致性以 **`validate`** 为准（与 goal 中「最小单元 test / 总单元 validate」一致）。

---

## 5. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-04-22 | 初版：默认步骤 + make 对照 + 与契约文档互链 |
| 2026-04-22 | 增加 § 边界 + 指向 `SPEC-LANDING-GAPS.md` |
