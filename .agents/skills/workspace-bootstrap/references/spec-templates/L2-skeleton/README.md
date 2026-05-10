# L2 — Skeleton Layer

**回答：用什么技术栈？模块有哪些？模块间如何协作？**

---

## L2 本质

L2 把 L1 goal 的意图"翻译"成技术架构骨架。
核心产物：`modules_matrix`（模块划分 + 依赖关系）和 `tech_stack`（技术选型）。

## L2 vs goskill

| goskill | VibeX L2 |
|---------|----------|
| Tech stack | content.tech_stack |
| Module system | content.modules_matrix |
| Entry points | content.entry_points |
| API design | content.api_design |
| — | content.l2_l3_lineage（L2/L3 分工）|

## L2 自举启示

**gen.py** 本身就是 L2 skeleton 的实现：
- 读取 L2 YAML → 解析 modules_matrix
- 选择对应模板 → 生成 Skeleton 文件
- Skeleton 是"半成品"，开发者填入业务逻辑

自举路径：
1. 手工写第一个 gen.py（Stage 1）
2. gen.py 生成了自己的 Skeleton（Stage 2 自举）
3. 用生成的 Skeleton 重写 gen.py（Stage 3 验证）

## 创建新的 L2 spec

1. 在 `projects/{project-name}/` 下创建 `L2-{short-name}.md`
2. 填写 YAML frontmatter（spec.level=2_skeleton, spec.parent=L1-name）
3. 填写 modules_matrix（列出所有模块 + 依赖）
4. 填写 tech_stack（语言/框架/关键库）
5. 运行 `validate_specs.py`
