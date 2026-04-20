# VibeX Workbench 心跳 — 可验证清单

> **核心原则：先契约与验收门禁，再放量AI改动。禁止用主观形容词，只填 PASS/FAIL/YES/NO。**

---

## A. 仓库门禁（必须通过，exit code 0）

在 `vibex-workbench` 根目录执行，任一成功即可：

| ID | 检查项 | 命令 |
|----|--------|------|
| A1 | Spec 语法 | `make lint-specs` 或 `PYTHONUTF8=1 python3 generators/validate_specs.py specs` |
| A2 | 从属链 | `make validate` |
| A3 | 前端可构建 | `cd frontend && npm run build` |

**结论**：A1–A3 若任一失败 → 本节标 FAIL，粘贴第一条报错行（不要求全文）。

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
| A1–A3 PASS **且** B3 均为 YES（或声明「本期仅 mock 路径」）**且** C1–C2 PASS | **允许加速迭代（契约已收敛）** |
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
