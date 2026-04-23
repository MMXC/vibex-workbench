# L3 — Module Layer

**回答：这个模块的公开 API 是什么？内部状态机如何？**

---

## L3 本质

L3 是模块边界 + 公开 API 签名定义。
不写实现，只写"模块对外部提供什么"和"内部状态是什么"。

## L3 vs goskill

goskill 没有明确的 Module 层，但 goskill 的 package design 大致对应 L3。
VibeX 强调模块边界清晰、公开 API 有版本意识。

## L3 自举启示

gen.py 生成 `*.Skeleton.svelte` 时：
1. 从 L2 modules_matrix 找到模块
2. 从 L3 spec 读取 public_api 签名
3. 生成带占位符的 Skeleton 文件
4. 开发者填入业务逻辑
