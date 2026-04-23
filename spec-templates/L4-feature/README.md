# L4 — Feature Layer

**回答：这个功能具体做什么？有哪些行为？验收标准是什么？**

---

## L4 本质

L4 是功能行为层。定义 behaviors（做什么）、test_scenarios（如何验证）。
这是 spec-first 流水线的核心输入——S4 代码生成依赖 L4 的 behaviors。

## L4 vs goskill

goskill 的 `.spec` 格式基本就是 L4 级别的内容：
- `context` → behaviors
- `validation` → test_scenarios  
- `goa` DSL → behaviors 结构化表达

VibeX L4 比 goskill 更丰富，支持：
- 多 spec 引用（L4 可以引用多个 L3）
- mixin 支持（安全/可观测性等横切 concerns）
- user_stories（从用户视角描述）

## L4 自举启示

spec_bootstrap_pipeline.py 的工作流：
- S1 → S2：生成 L4 io_contract（clarification）
- S3 → 补全 L4 behaviors
- S4 → 从 L4 behaviors 生成代码
- S5 → 用 test_scenarios 验证

**L4 behaviors 是 gen.py 代码生成的直接输入**。
