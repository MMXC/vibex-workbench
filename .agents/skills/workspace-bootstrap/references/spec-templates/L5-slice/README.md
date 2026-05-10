# L5 — Slice Layer

**回答：这个文件的具体内容是什么？gen.py 如何生成它？**

---

## L5 本质

L5 是 gen.py 生成的最小单位——每个文件/脚本的生成规格。
不描述系统行为，只描述"这个文件长什么样"。

## L5 vs goskill

goskill 的 `generator/generate.go` 里有大量 `generateXxxFile()` 函数，
每个函数对应一个 L5 slice——文件路径 + 内容模板。

## L5 自举启示

vibex-workbench 的 gen.py 对 L5 的处理：
- `*.Skeleton.svelte` = gen.py 生成（覆盖）
- `*.svelte` = 开发者私有（不覆盖）
- `routes/*.ts` = gen.py 生成（类型定义）
- `stores/*.ts` = gen.py 生成（状态管理）

L5 模板本身也是 L5 spec，可以用 gen.py 自举生成。

---

## 与本仓库「产品分层」的关系

产品视角下，L5 对应 L4 的**实现切片**。仅抽屉容器占位与「向抽屉填入可点能力」时的归属，见 [`../L0-meta-convention/L0-layer-contract-template.yaml`](../L0-meta-convention/L0-layer-contract-template.yaml) 内 `content.product_layering_governance.l5_slice_implementation`、`slot_drawer_implementation_note`。
