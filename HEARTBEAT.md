# VibeX Workbench 心跳 — 可验证清单

> **核心原则：先契约与验收门禁，再放量AI改动。禁止用主观形容词，只填 PASS/FAIL/YES/NO。**

---

## A. 仓库门禁（必须通过，exit code 0）

在 `vibex-workbench` 根目录执行，任一成功即可：

| ID | 检查项 | 命令 |
|----|--------|------|
| A1 | Spec 语法 | `make lint-specs` 或 `PYTHONUTF8=1 python3 generators/validate_specs.py specs` |
| A2 | 从属链 | `make validate` |
| A3 | 前端可构建 | `cd /root/vibex-workbench/frontend && npm run build` |

**A4 — Agent 自循环可触达**（强化：模拟 Agent `make_validate` 工具的调用路径）
```bash
cd /root/vibex-workbench && make validate
```
判定：exit code 0 = PASS（与 A2 同一命令，但此行明确标注「模拟 Agent make_validate 工具的 workspaceDir 路径」，用于确认 Agent 自迭代可触达门禁）。

**结论**：A1–A4 若任一失败 → 本节标 FAIL，粘贴第一条报错行（不要求全文）。

---

## B. SSE 契约 — Agent 路径（必须可核对字段）

约定：心跳默认检查 Agent（`agent/cmd/web`，端口见 `agent` 的 README，常见 `33338`），与前端 `frontend/src/lib/sse.ts` 中 `HANDLERS` 的 JSON 字段是否一致。

### B1 — 枚举：Agent 实际发出的 event 名

```bash
cd /root/vibex-workbench
rg 'broadcastSSE\([^,]+,\s*"[^"]+"' agent/cmd/web/server.go agent -g'*.go'
```

### B2 — 枚举：前端已订阅的 event 名

```bash
rg "^\s*'[a-z.]+':" frontend/src/lib/sse.ts
```

### B3 — 硬约束（每期必须填写）

| 事件 | Agent 当前 payload 键（server.go 约 128–141 行） | sse.ts handler 读取的键 | 一致？ |
|------|---|---|---|
| `tool.called` | `tool`, `call_id`, `args` | `toolName`, `invocationId`, `runId`, `args` | **YES / NO** |
| `tool.completed` | `tool`, `call_id`, `result` | `invocationId`, `result` | **YES / NO** |

核对方式：读取 `agent/cmd/web/server.go` 约 128–141 行，对照 `frontend/src/lib/sse.ts` 中 `tool.called` / `tool.completed` 两处 handler，逐键填写 YES/NO。

**结论**：B3 任一为 NO → B 节 FAIL，**禁止**在报告写「SSE 已与前端对齐」。

### B4 — 契约真源（钉死后启用）

在仓库增加单一真源（任选其一即可算「已钉死」）：

- 新增 `docs/sse-contract-agent.md`，内含表格：事件名 | JSON 字段 | 类型 | 生产者文件:行；或
- 在 `specs/feature/workbench-shell/` 的 `behavior` 里写同一表格

---

## C. E2E（1～2 条具名用例，必须通过）

```bash
cd frontend && npm run test:e2e:run
```

心跳必须写明以下两条（名称须与 `frontend/tests/e2e/*.spec.ts` 里 `test('...')` 字符串完全一致）：

| ID | 固定用例（示例，以仓库为准可复制） | 判定 |
|----|------|------|
| C1 | `CanvasRenderer mounts — .canvas-renderer is visible`（文件：`tests/e2e/canvas-orchestration.spec.ts`） | PASS / FAIL / SKIP |
| C2 | 任选同目录下另一条稳定用例（不依赖真实 SSE 的） | PASS / FAIL / SKIP |

**判定**：exit code 0 + 日志可见对应测试 passed → PASS；任一失败或未运行 → FAIL。

**说明**：当前 Playwright 多为 preview + 静态结构，不等于「SSE 端到端」；若要心跳验证「真 SSE」，须另增一条 E2E（mock 后端或测试桩）。在未增加前，须在报告写一句：「C2 仅覆盖 UI 挂载，未覆盖 SSE 字节级契约。」

---

## D. AI 放量条件（每期二选一）

| 同时满足条件 | 结果 |
|------|------|
| A1–A4 PASS 且 B3 均为 YES 且 C1–C2 PASS | **允许加速迭代（契约已收敛）** |
| 任一不满足 | **只允许契约 / 门禁 / E2E 类 PR；暂缓铺新功能** |

---

## 红线（触发时报告必须包含）

> ⚠️ **当前处于技术债高风险区；下一周期优先还债（契约 + validate + E2E），不推荐合并大范围 AI 生成改动。**

触发条件：
- A1/A2/A3 任一 FAIL
- B3 任一 NO 且无迁移计划
- E1/E2 任一 FAIL 且无修复计划

---

## 给 Hermes 的一行指令（每期复制）

> 运行 A1–A3；对照 B3 表格填 YES/NO；运行 `cd frontend && npm run test:e2e:run` 确认两条例名通过；输出 PASS/FAIL 表，勿用主观形容词。

---

## E. 闭环状态（spec → 代码生成，已验证的路径）

> 此节记录 spec → 代码 流水线已关闭的路径。新发现断点加在此处。

### E1 — lib/types.ts 生成 ✅ 已关闭（2026-04-21 第一圈）

```
spec YAML (*_data.yaml)
  → make generate
  → lib/types.ts（gen.py 合并版）
  → stores/*.ts 实际使用
```

**已验证：**
- `type_map` 正确映射实际 spec 名 ✅
- 动态发现所有 `*_data` spec ✅
- `required` 启发式（`id`/`name`/`status`/`startedAt` 等 → 必填）✅
- `lib/generated/` 已删除（合并入 `lib/types.ts`）✅
- 所有 stores import resolve ✅
- `Thread` = `ConversationThread` 别名（thread-store.ts）✅

### E1.5 — uiux spec → Skeleton 动态生成 ✅ 已关闭（2026-04-21 第二圈）

```
*_uiux.yaml (shell_layout)
  → make generate
  → WorkbenchShell.Skeleton.svelte
```

**已验证：**
- `shell_layout` 字段 → `grid-template-columns/rows/areas` 直接对应 ✅
- `grid-template-columns` 从 spec 改值 → Skeleton 响应变化 ✅
- 重复 grid-area 自动去重（B B B → 只渲染一次 footer）✅
- 硬编码 fallback 保留（spec 缺失时走旧模板）✅

### E1.6 — canvas-uiux spec → SvelteFlow Skeleton ✅ 已关闭（2026-04-21 第三圈）

```
canvas-renderer_uiux.yaml
  → make generate
  → CanvasRenderer.Skeleton.svelte
```

**已验证：**
- `components[]: ZoomIn/ZoomOut/FitView/ToggleInteractivity` → 工具栏 4 按钮 ✅
- `state_management: canvasStore` → reactive nodes/edges 绑定 ✅
- `behaviors[]: onnodeclick/onnodedoubleclick/onnodedragstop` → 事件处理函数 ✅
- `design_tokens.bg_canvas: #0a0a0a` → CSS background ✅
- `sse_events[]` 声明存在（前端监听未实现，缺口同 B3）✅

### E1.7 — Agent 内置 spec驱动循环 ✅ 已关闭（2026-04-21 第四圈）

```
用户 → spec_feature(name="...")
  → 创建 specs/feature/<name>/<name>_feature.yaml  (L4)
  → 创建 specs/feature/<name>/<name>_uiux.yaml      (L5a)
  → 输出 SPEC-DRIVEN LOOP 三步提示:
      (1) spec_validate
      (2) make_generate     ← 新增内置工具
      (3) canvas_update
```

**已验证：**
- `make_generate` tool 注册到 `agent/vibex/domain/spec/specs.go` ✅
- handler: `MakeMakeGenerateHandler` → `make generate` → gen.py → types/components ✅
- `spec_feature` 描述内嵌 SPEC-DRIVEN LOOP 指南 ✅
- `spec_feature` handler 自动创建 feature + uiux 同目录子规格 ✅

**闭环覆盖：**
```
spec_designer → spec_feature → spec_validate → make_generate → canvas_update
                                      ↓
                               make_validate
```

### E3 — spec自举（待验证）

`spec_designer` 工具 → spec YAML → `make generate` → `make validate` → `spec_designer` 更新

**当前缺口：**
- `spec_feature` 的 template 仍是硬编码 Go string，未从 spec 驱动
- uiux sub-spec template（`canvas_layout.type: flow-canvas` 等）是默认值
- 下一步：`spec_feature` handler 的 content template 应从某 L3/L4 spec 读取，或由 `make_generate` 反向生成

### E2 — spec-designer → spec YAML → make validate ✅ 门禁已闭环

见 A2（`make validate` = `spec_designer` 工具的终端路径）。
