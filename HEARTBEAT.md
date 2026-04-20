# VibeX Workbench 心跳 — 执行准则

> **核心原则：「先契约与验收门禁，再放量 AI 改动」**

VibeX Workbench 的迭代速度必须以 **可重复的机器门禁** 为边界：在 SSE 事件契约、仓库级 validate 与少量 E2E 未统一前，不将「大量 AI 提交/大改」视为健康进展，否则容易形成更快堆技术债。心跳应优先报告**契约与门禁是否变严、变绿**，而不是仅报告功能点数量或 diff 规模。

---

## 钉死什么（定义「完成」）

### SSE / 契约

1. **单一真源**：存在一份 `EVENTS.md` 或 `specs/*_service/service.backend.yaml` 中的事件表，且与 `agent/cmd/web` 中所有 `broadcastSSE` 调用的事件名、JSON 字段**逐条一致**。
2. **Mock 与 Agent 若并存**：
   - 须说明是否已收敛为同一张契约表
   - 或分表并标注哪条路径为「主路径」
   - **禁止长期双写且字段不一致不标红**
3. 任何事件/字段的变更须在本期心跳中列出 diff

### 验收

4. **validate**：`make validate`（或等价 `python3` 命令）在代表该心跳的 commit/PR 上为**必过项**
5. **E2E（1～2 条）**：在 `package.json` 或 Playwright 中**具名**（例如：「接 SSE 能出 run 节点」「发一条 chat 能完成一轮 tool 并落 store」），并写清跑法（`npm run test:e2e` 等）。**无具名用例 = 该条心跳不认定为已钉死**

---

## 每期心跳报告结构

```markdown
## 本期心跳 — YYYY-MM-DD

### 契约与上次相比
[  ] 契约冻结，无 diff
[  ] 有 diff（列出变更）
  - 事件/字段变更：...

### 门禁 validate
- [ ] PASS
- [ ] FAIL（附首行错误）

### E2E
- [ ] 用例名 + PASS / FAIL / SKIP
- [ ] 用例名 + PASS / FAIL / SKIP

### AI 放量条件
- [ ] 允许加速迭代（validate 绿 + E2E 绿）
- [ ] 仅允许契约/门禁类 PR，暂缓铺功能
```

---

## 红线（每期心跳必须明确写出）

若出现以下任一情况，本期结论**必须包含**：

> ⚠️ **当前处于技术债高风险区；下一周期优先还债（契约 + validate + E2E），不推荐合并大范围 AI 生成改动。**

触发条件：
- `make validate` 红
- E2E 具名用例未定义
- Agent（33338）/Mock（33335）SSE 字段与前端 `sse.ts` 不一致且无迁移计划

---

## 给 Hermes 的执行指令模板

```
本期心跳请以 SSE 契约一致性、validate、1～2 条具名 E2E 为首要输出；
次要输出才是功能清单。
若契约未收敛，禁止将『代码行数 / 工具调用次数』列为进展指标。
```
