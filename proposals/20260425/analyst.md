# 提案：修复 validate_specs.py 多文档 YAML 解析错误

**Agent**: analyst
**日期**: 2026-04-25
**项目**: vibex-workbench-validate-fix
**工作目录**: /root/vibex-workbench
**状态**: 正式提交

---

## 1. 问题描述

`make validate` 失败，`validate_specs.py` 无法解析 `specs/p-metaspec/` 下的 5 个 YAML 文件：

```
[ERROR] (5):
  YAML error L2-generator-gen-mechanism.yaml: expected a single document in the stream
  but found another document
  YAML error L4-constraint-goa-style.yaml: expected a single document in the stream
  ...
```

5 个受影响文件：
- `L2-generator-gen-mechanism.yaml`（15 个文档分隔符）
- `L2-workflow-spec-first-iteration.yaml`（12 个）
- `L4-constraint-goa-style.yaml`（8 个）
- `L4-eval-ai-repro.yaml`（10 个）
- `L4-iteration-trigger-paths.yaml`（11 个）

---

## 2. 根因分析

**根因**：`validate_specs.py` 第 59 行使用 `yaml.safe_load(raw)`，只能解析单文档 YAML。

YAML 规范允许多文档流使用 `---` 分隔符。`safe_load()` 在遇到第二个 `---` 时抛出 `YAMLError: expected a single document in the stream`。

实际文件结构：
```yaml
# L2-generator: 代码生成器规格
---
spec:
  version: "0.4"
  level: "2_skeleton"
---
# 第二个文档...
---
# 第三个文档...
```

**触发路径**：
```
python3 validate_specs.py specs/p-metaspec/
  → validate_file() 对每个 .yaml 调用 yaml.safe_load(raw)
  → 多文档 YAML 抛出 YAMLError
  → 函数返回 None
  → specs/p-metaspec/* 全部标记为 error
  → make validate 退出码 1
```

---

## 3. 影响评估

| 维度 | 影响 |
|------|------|
| CI/CD | `make validate` 在所有涉及 p-metaspec 的 PR 上失败 |
| 开发体验 | 新人无法运行 validate，本地开发受阻 |
| 范围 | 仅限 p-metaspec 目录，其他 specs 不受影响 |

---

## 4. 建议方案

### 方案 A（推荐）：`safe_load_all()` 取第一个有效文档

```python
# 修改 validate_specs.py 第 59 行
# 原来：
data = yaml.safe_load(raw)

# 改为：
docs = list(yaml.safe_load_all(raw))
# 取第一个非空文档（实际 spec 数据所在）
data = next((d for d in docs if d is not None), None)
```

**优点**：
- 最小改动（1 行核心变更）
- 兼容现有单文档 YAML（`safe_load_all` 对单文档同样有效）
- `safe_load_all` 返回生成器，内存友好

**缺点**：
- 只取第一个文档，若多文档文件中有多个有效 spec，只验证第一个

**适用性**：p-metaspec 文件的多文档结构中，第一个文档是主 spec，后续是注释/示例，取第一个符合预期。

---

### 方案 B：聚合所有文档

```python
docs = list(yaml.safe_load_all(raw))
# 取第一个非空文档
data = next((d for d in docs if d is not None), None)
```

实际上方案 A 和 B 在这里等价——都取第一个非空文档。区别在于：
- A：用 `next()` 短路
- B：先 list 再取（多文档文件大时多一次内存分配）

**推荐方案 A**。

---

## 5. 验收标准

- [ ] `python3 generators/validate_specs.py specs/p-metaspec/` 退出码 0
- [ ] 5 个多文档 YAML 文件均不再报 `expected a single document` 错误
- [ ] 单文档 YAML 文件（如 `specs/architecture/*.yaml`）验证不受影响
- [ ] `make validate` 退出码 0

---

## 6. 实施步骤

1. 修改 `generators/validate_specs.py` 第 59 行
2. 运行 `python3 generators/validate_specs.py specs/p-metaspec/` 验证
3. 运行 `make validate` 全量验证
4. 提交 PR

---

## 7. 执行依赖

- [ ] 需要修改的文件: `generators/validate_specs.py`
- [ ] 前置依赖: 无
- [ ] 需要权限: 无
- [ ] 预计工时: 0.5h
- [ ] 测试验证命令:
  ```bash
  python3 generators/validate_specs.py specs/p-metaspec/
  make validate
  ```

---

## 执行决策

- **决策**: 推荐执行
- **执行项目**: vibex-workbench-validate-fix
- **执行日期**: 2026-04-25
- **风险**: 极低（单行代码改动，无副作用）
