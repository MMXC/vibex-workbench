# Spec 执行协议（占位）

**状态：骨架** — 统一描述「人或 agent 如何从 spec 走到实现」的行为契约；细节待与 `skills/`、`generators/gen.py` 边界对齐后填充。

当前请以 **`SPEC-TO-SHIP-WORKFLOW.md`** 为默认步骤；**已知缺口**见 **`SPEC-LANDING-GAPS.md`**。

---

## 草案目录（待补全）

1. **输入**：工作区根、`specs/` 入口、当前任务涉及的 L4/L5 路径。  
2. **读序**：goal → skeleton → MOD → feature → L5 切片（与 `spec_workflow_units` 一致）。  
3. **门禁**：何时必须 `make validate` / `make generate` / `make drift`。  
4. **生成 vs 手写**：以各 L4 `io_contract` / `implementation` 为准；与 `gen.py` 覆盖范围对照 `SPEC-LANDING-GAPS.md`。  
5. **Todo / C7**：变更后 backlog 条目最小字段。  
6. **与外部 skill**：`spec-frontend` 等与仓库边界的引用关系（不重复写长文）。

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-04-22 | 创建占位；回应落地层「agent 行为未固化」缺口 |
