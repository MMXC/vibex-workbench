# Spec → 实现 执行协议

> **本文档回答：agent 如何把 spec 变成可运行代码。**
> 
> 对应约束：C1（全产品自动化）、C3（从属链完整）、C4（自举验证）、C7（变更派生 todo）。

---

## 核心公式

```
spec YAML
    ↓ 【step 1】确认 parent 链存在
    ↓ 【step 2】确定 spec level
    ↓ 【step 3】选择对应生成路径
    ↓ 【step 4】lint + validate
    ↓ 【step 5】运行 make generate
    ↓ 【step 6】填 TODO / 写逻辑
    ↓ 【step 7】跑测试 / 门禁
```

---

## Step 1：确认 parent 链（必做，C3 强制）

每个 spec 必须有 `parent` 字段（除了 L1）。

**规则**：在写入或修改任何 spec 前，先确认其 parent 文件存在。

```
spec:
  name: "my-feature"
  level: "4_feature"
  parent: "workbench-shell"   ← 必须存在！
```

**快速检查**：
```bash
make validate
```
失败 → 先补 parent spec，再继续。

---

## Step 2：确定 spec level

| level | 含义 | 谁写 | 生成什么 |
|--------|------|------|---------|
| `1_project-goal` | L1 目标 | 产品/PM | 无代码 |
| `2_skeleton` | L2 架构 | 架构师 | 无代码 |
| `3_module` | L3 模块 | 开发者 | 无代码 |
| `4_feature` | L4 功能 | 开发者 | 无代码，但决定 L5 切片 |
| `5a_uiux` | L5a 交互 | 前端 | Svelte 组件 Skeleton |
| `5b_service` | L5b 服务 | 后端/全栈 | TypeScript service 类 |
| `5c_data` | L5c 数据 | 全栈 | TypeScript interfaces |
| `5d_test` | L5d 测试 | 开发者 | 测试文件 |

---

## Step 3：选择生成路径

```
L5a uiux spec
    → gen.py → *.Skeleton.svelte（永不覆盖 *.svelte）
    → 输出：frontend/src/lib/components/

L5b service spec
    → gen.py → *_services.ts（永久生成）
    → 输出：frontend/src/lib/services/

L5c data spec
    → gen.py → lib/types.ts（合并）
    → 输出：frontend/src/lib/types.ts

L5d test spec
    → 手动编写测试文件（TODO：生成器支持中）
    → 输出：frontend/src/lib/
```

### 三段生成入口

```bash
# 完整生成（推荐）
make generate

# 只生成 types + services
python3 generators/gen.py specs frontend

# 只验证 spec
make validate
```

---

## Step 4：lint + validate（必做）

```bash
make lint-specs    # YAML 语法 + parent 文件存在
make validate      # 从属链深度校验
```

**失败时的处理**：
1. lint 失败 → 修 YAML 语法错误
2. validate 失败 → 检查 parent 链：
   - parent 文件是否存在
   - parent level 是否正确（子 level = parent level + 1）
   - L4 feature 可以 parent 到 L3 module 或另一个 L4 aggregate（如 `workbench-shell`）

---

## Step 5：运行 make generate

```bash
make generate
```

**幂等性规则**：
- `*.Skeleton.svelte` → 永远覆盖，放心重跑
- `*.svelte` → 开发者私有，gen.py 永不覆盖
- `lib/services/*.ts` → 永久生成，覆盖是预期行为
- `lib/types.ts` → 合并模式，增量追加，不丢手写类型

---

## Step 6：填 TODO / 写逻辑

gen.py 生成的代码骨架包含：
- JSDoc 说明来自哪个 spec
- `// TODO:` 注释标记待实现部分
- `throw new Error('NotImplemented: ...')` 作为未实现的占位

**开发者的工作**：在 `*.svelte` 文件里写 UI 逻辑，在 `lib/services/*.ts` 里填 service 方法体。

**禁止**：在 `*.Skeleton.svelte` 里写逻辑——这文件会被 gen.py 覆盖。

---

## Step 7：测试 / 门禁

```bash
make test        # 单元测试
make drift      # spec 与代码漂移检测
make lint-all   # validate + drift
```

**漂移处理**（drift 报警时）：
1. 确认 spec 是否真的需要更新
2. 如果 spec 需要更新 → 按 C7 派生 todo-spec
3. 重新运行 `make generate` 同步生成物
4. 重新运行 `make drift` 确认修复

---

## 变更纪律（C7）

```
任意 spec 变更
    ↓
追加一条 todo-spec
    ↓
关联门禁：相关 test → validate
    ↓
门禁全部通过
    ↓
变更完成
```

**禁止**：「改了 spec 却无明示后续任务」的静默变更。

---

## 目录速查

| 资源 | 路径 |
|------|------|
| 所有 spec | `specs/**/*.yaml` |
| spec validator | `generators/validate_specs.py` |
| 从属链校验 | `spec-engine/validate_chain.py` |
| 代码生成器 | `generators/gen.py` |
| 漂移检测 | `generators/drift_check.py` |
| 层级契约 | `specs/meta/spec-layer-contract.yaml` |
| 目录约定 | `specs/meta/spec-directory-convention.yaml` |
| 门禁入口 | `Makefile` |

---

## 快速启动命令

```bash
# 第一次：验证 spec 全集
make validate

# 开发循环：修改 spec → 生成 → 验证
make generate && make drift

# 提交前：全量门禁
make lint-all && make test
```
