# Spec — E1: SSE Backend Integration

> PRD: `prd.md` E1 章节
> 规范层级: L4 Feature Spec

## SSE URL 配置

### 当前问题
`frontend/src/lib/sse.ts` 和 `frontend/src/routes/workbench/+page.svelte` 中 SSE URL 硬编码为 `http://localhost:33335`

### 目标行为
```typescript
const SSE_URL = import.meta.env.VITE_SSE_URL ?? 'http://localhost:33335';
```

### 间距/颜色规范
无 UI 变更，纯逻辑修改。

---

## SSE 重连逻辑

### 状态机

| 当前状态 | 触发 | 下一状态 | 行为 |
|----------|------|----------|------|
| connected | onerror | retrying | 记录 attempt=1，3s 后重连 |
| retrying | timeout | connected | 重新 connect |
| retrying | onerror | retrying | attempt++, attempt ≤ 5 → 2^n×1000ms 退避 |
| retrying | attempt > 5 | failed | 停止重连，设置 error state |
| failed | manual_retry | retrying | 重置 attempt=1 |

### UI 状态规范

> SSE 为后台连接，无直接 UI。但连接状态可通过 `uiStore.sseStatus` 暴露。

#### 理想态
- SSE `connected` → 无 UI 指示
- Run 正常执行

#### 加载态
- N/A（SSE 连接在后台）

#### 错误态
- `failed` 状态（5 次重连均失败）→ Console 告警：`[SSE] Max retries exceeded. Manual retry available.`

#### 空状态
- N/A

---

## 验收标准

```typescript
// F1.1
expect(import.meta.env.VITE_SSE_URL).toBeDefined();
// 或 fallback 正确
const url = import.meta.env.VITE_SSE_URL ?? 'http://localhost:33335';
expect(url).toMatch(/^https?:\/\//);

// F1.2 — 重连次数验证（需 mock EventSource）
// attempt=1: 3000ms, attempt=2: 6000ms, attempt=3: 12000ms
expect(reconnectDelay(1)).toBe(3000);
expect(reconnectDelay(2)).toBe(6000);
expect(reconnectDelay(5)).toBe(48000);
// 5 次内重连上限
expect(maxAttempts).toBe(5);
```

### DoD
- [ ] `VITE_SSE_URL` 环境变量读取逻辑已实现
- [ ] 重连逻辑覆盖所有状态转换
- [ ] 最大重试次数限制生效
- [ ] README.md 更新 `.env.example` 文件
