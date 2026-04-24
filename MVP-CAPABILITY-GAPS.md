# VibeX Workbench MVP — 能力边界与缺失清单

> 目标：记录 5 项 MVP 能力中哪些已实现、哪些是桩、哪些完全缺失。
> 路径：`/root/v-test`（vibex-workbench 副本，specs/ 已按 L0–L5 模板重建）

---

## 能力一：绑定仓库根

### Spec 状态
- L3 `MOD-workspace-root.yaml` ✅ 起草完成
- L4 `FEAT-workspace-selector.yaml` ✅ 起草完成

### 实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| `frontend/src/lib/server/workspace-root.ts` | ✅ 已有 | 读 `WORKSPACE_ROOT` env，fallback 到 `cwd/..` |
| `agent/cmd/web/main.go` | ✅ 已有 | 通过 `WORKSPACE_ROOT` env 启动 |
| frontend 目录选择器 UI | ❌ 缺失 | 无路径输入框，无 localStorage 持久化 |
| SSE 重连触发 | ❌ 缺失 | 切换路径后 agent 上下文不重置 |
| `/api/workspace/set-root` | ❌ 缺失 | 无 API 保存路径到前端 store 以外的地方 |

### 边界
- MVP 阶段：WORKSPACE_ROOT 只在单次会话内有效，不持久化到文件
- 上限：不支持多 workspace 并行

---

## 能力二：空/半/就绪三态探测

### Spec 状态
- L3 `MOD-state-detection.yaml` ✅ 起草完成
- L4 `FEAT-state-detection.yaml` ✅ 起草完成

### 实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| `generators/state_detector.py` | ❌ **完全缺失** | 无此文件 |
| `/api/workspace/detect-state` | ❌ **完全缺失** | 无此 endpoint |
| frontend 状态指示器 UI | ❌ **完全缺失** | 无状态展示组件 |

### 检测信号（来自 spec 定义）

```
empty:   specs/ 不存在 + generators/gen.py 不存在
partial: specs/ 存在 + generators/gen.py 不存在
ready:   specs/ 存在 + generators/gen.py 存在 + Makefile 含 lint-specs
```

### 边界
- 只做只读探测，不做自动修复
- 上限：不检测 spec 内容合法性（那是 validate 的职责）

---

## 能力三：首次脚手架生成

### Spec 状态
- L3 `MOD-scaffolding.yaml` ✅ 起草完成
- L4 `FEAT-scaffolding.yaml` ✅ 起草完成

### 实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| `spec-templates/` | ✅ 完整 | L0–L5 模板齐全 |
| `generators/scaffolder.py` | ❌ **完全缺失** | 无此文件 |
| `/api/workspace/scaffold` | ❌ **完全缺失** | 无此 endpoint |
| scaffolding wizard UI | ❌ **完全缺失** | 无确认对话框/预览 |

### 生成产物（来自 spec 定义）

```
specs/
  L1-goal/ENTRY.yaml          ← 入口 L1 模板
generators/
  gen.py                        ← 生成器入口
  validate_specs.py             ← 已有
spec-engine/
  validate_chain.py             ← 已有
Makefile                        ← 含 lint-specs / validate / generate
frontend/package.json           ← 可选
agent/.env.example              ← 可选
```

### 边界
- 只做首次生成，不做增量更新
- 上限：不覆盖已有文件（幂等）

---

## 能力四：Spec 可读 + 可编辑

### Spec 状态
- L3 `MOD-spec-editor.yaml` ✅ 起草完成
- L4 `FEAT-spec-editor.yaml` ✅ 起草完成

### 实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| `/api/workspace/specs/list` | ✅ 已有 | 返回 spec 文件列表 |
| `/api/workspace/specs/read` | ✅ 已有 | 读取单个 spec 文件内容 |
| `FileService.readSpec` | ⚠️ 桩 | 有签名，`NotImplemented` |
| `FileService.writeSpec` | ⚠️ 桩 | 有签名，`NotImplemented` |
| `FileService.listSpecs` | ⚠️ 桩 | 有签名，`NotImplemented` |
| `SpecParserService.parseYAML` | ⚠️ 桩 | `NotImplemented` |
| `SpecParserService.serializeYAML` | ⚠️ 桩 | `NotImplemented` |
| `/api/workspace/specs/write` | ❌ **完全缺失** | 无 route |
| 新建 L1 向导 | ❌ **完全缺失** | 无 UI |
| 客户端 YAML 解析 | ❌ **完全缺失** | 保存前无预检验 |

### 51 个 NotImplemented 桩分布

```
spec_editor_services.ts        11桩 ← 核心，spec 读写的全部桩
routing_panel_services.ts     17桩 ← clarification 循环
dsl_canvas_services.ts        10桩 ← canvas 可视化
workbench_shell_services.ts   10桩 ← layout / event-bus
code_gen_panel_services.ts     2桩 ← generate / diff
```

### 边界
- MVP 只支持新建/编辑 L1 goal spec
- 上限：不支持 L2–L5 编辑（后续迭代）

---

## 能力五：校验/生成一触即达

### Spec 状态
- L3 `MOD-build-panel.yaml` ✅ 起草完成
- L4 `FEAT-build-panel.yaml` ✅ 起草完成

### 实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| `Makefile` (lint-specs) | ✅ 已有 | `generators/validate_specs.py` |
| `Makefile` (validate) | ✅ 已有 | `spec-engine/validate_chain.py` |
| `Makefile` (generate) | ⚠️ 桩 | 有 target，gen.py 逻辑待实现 |
| `/api/workspace/run-make` | ❌ **完全缺失** | 无此 endpoint |
| 前端「校验」按钮 | ❌ **完全缺失** | 无 action bar |
| 前端「生成」按钮 | ❌ **完全缺失** | 无 action bar |
| 错误浮层 | ❌ **完全缺失** | 无结果展示组件 |

### 边界
- 只做执行 + 结果展示，不做自动修复
- 上限：60s 超时保护，输出截断到 500 字符

---

## 汇总表

```
┌─────────────────────────┬──────────┬──────────────┬────────────┐
│ 能力                     │ 已实现   │ 桩           │ 完全缺失   │
├─────────────────────────┼──────────┼──────────────┼────────────┤
│ ① 仓库根绑定              │ 读取env  │ —            │ UI选择器   │
│ ② 三态探测                │ —        │ —            │ 全链路缺失 │
│ ③ 脚手架生成              │ 模板     │ —            │ scaffolder │
│ ④ spec 读写              │ 2/3 API  │ 11桩        │ write API  │
│ ⑤ 校验/生成按钮           │ Makefile │ generate桩  │ run-make   │
├─────────────────────────┼──────────┼──────────────┼────────────┤
│ 前端 services 层          │ —        │ 51桩        │ —          │
│ YAML 错误（5文件）         │ —        │ p-metaspec/ │ —          │
└─────────────────────────┴──────────┴──────────────┴────────────┘

已重建 spec 树：L1×1 + L2×1 + L3×5 + L4×5 = 12 个新 spec，全部通过 lint + parent chain 验证
```

---

## 重建原则记录

1. **模板格式**：使用 `spec-templates/L*-template.yaml` 纯 YAML 格式（不加 `---` 结尾）
2. **命名规范**：L1 用 `name: vibex-workbench-mvp`，parent chain 从 L1 指向 null 开始
3. **io_contract 优先**：每个 spec 先写 io_contract，再填 content
4. **不复制原 specs/**：新树在 `specs/L1-goal/` 等新目录，旧文件保留在原位置备查
