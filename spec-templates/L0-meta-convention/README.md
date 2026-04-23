# L0 — Meta Convention Layer

**定义 spec 系统自身的规范与约定。**

---

## L0 本质

L0 是"宪法的宪法"——它定义的规则约束所有其他 spec（L1–L5），包括它自己。

| 特性 | 说明 |
|------|------|
| 自我指涉 | L0 描述的规则适用于包括 L0 在内的所有层级 |
| 最小化 | 只定义元规则，不定义具体功能 |
| 稳定性 | 变更代价极高，需 Darwin 迭代验证 |
| 自举证明 | L0 自身可由 L0 spec-templates 驱动生成 |

## L0 包含的内容

1. **layer-contract** — L1–L5 每层 must_contain + mixin 合并语义
2. **directory-convention** — 目录 ↔ 层级 ↔ 命名 映射
3. **mixin-system** — mixin 定义/引用/合并/循环检测规则
4. **global-io-contract** — 所有层级共享的 io_contract 字段规范

## 与 L1–L5 的关系

```
L0 (meta-convention)
  ↓ 定义规则
L1 (goal) → L2 (skeleton) → L3 (module) → L4 (feature) → L5 (slice)
```

L0 是横向规范层，跨所有项目；L1–L5 是纵向从属链，服务单个项目。
