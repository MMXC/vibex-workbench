# L1 — Project Goal Layer

**回答：做这个项目是为了什么？做成什么样才算成功？**

---

## L1 本质

L1 是项目宪章——定义使命（mission）、不做边界（out-of-scope）、
目标用户（target_users）、成功标准（constraints），
以及 L1/L2 分工（l1_l2_lineage）。

## L1 vs goskill Project

| goskill | VibeX L1 |
|---------|----------|
| Mission statement | content.mission |
| Target users | content.target_users |
| Out of scope | io_contract.boundary |
| Constraints | content.constraints |
| Success criteria | content.constraints（含 Darwin 评分目标）|
| — | content.l1_l2_lineage（L1/L2 分工明确）|
| — | content.product_value_layers（三层价值分离）|

## L1 自举启示

goskill 的 spec.yaml 是 100% YAML，VibeX 用 Markdown + YAML frontmatter。
两种都是自举友好的格式，关键在于：
- YAML 部分必须能被 `safe_load` 解析
- Markdown 部分是人类可读的补充说明
- **io_contract 必须放在 YAML frontmatter**，不要放在 Markdown body

## 创建新的 L1 spec

1. 在 `projects/{project-name}/` 下创建 `SPEC.md`
2. 写 YAML frontmatter（含 spec/io_contract/content）
3. 写 Markdown body（架构图、示例等人类可读内容）
4. 运行 `validate_specs.py` 确认 YAML frontmatter 可解析
