---
name: specpilot-datalayer
description: |
  SpecPilot 四层数据能力 — 数据中心 / 事件中心 / 适配器切换 / Spec治理 / MF组件注册。
  触发条件：用户提到"数据中心"、"事件中心"、"适配器切换"、"spec治理"、"MF组件"、"四层"、"数据联动"、"状态中心"。
  触发词：dc、event、adapter、spec、mf、datacenter、eventcenter。
  当用户在项目中需要管理多数据源、事件总线、适配器热切换、或 spec 驱动的组件注册时，加载此技能。
---

# SpecPilot — 四层数据能力

SpecPilot 提供五层 CLI 能力，通过 subprocess 调用，返回 JSON 格式结果。

## CLI 路径

```bash
SPECPILOT=/tmp/specpilot/specpilot
# 或直接 python3 -m cli（从 /tmp/specpilot 目录运行）
```

## 五层命令速查

### L2 数据中心（DataCenter）

```bash
# 查询所有状态
$SPECPILOT dc list
{"data":{"cpu_usage":{"value":65,"updated":"..."},"memory_percent":{"value":72,"updated":"..."}}}

# 查询单个 key
$SPECPILOT dc get cpu_usage
{"key":"cpu_usage","value":65,"updated":"..."}

# 设置状态
$SPECPILOT dc set cpu_usage 80

# 订阅变更（watch，5秒超时返回当前值）
$SPECPILOT dc watch cpu_usage
{"key":"cpu_usage","value":65,"version":3}

# SQL 风格查询
$SPECPILOT dc query "SELECT *"
```

### L3 事件中心（EventCenter）

```bash
# 订阅事件
$SPECPILOT ec subscribe kpi:threshold.exceeded KPICard
{"subscriber":"KPICard","event":"kpi:threshold.exceeded"}

# 发布事件
$SPECPILOT ec emit kpi:threshold.exceeded '{"threshold":80}'
{"event":"kpi:threshold.exceeded","payload":{"threshold":80},"subscribers":["KPICard","AlertBadge"]}

# 查看事件历史
$SPECPILOT ec history
{"events":[{"event":"kpi:threshold.exceeded","subscribers":["KPICard","AlertBadge"]}]}
```

### L4 适配器管理器（Adapter）

```bash
# 列出所有适配器
$SPECPILOT ad list
{"adapters":[{"name":"mock","active":true},{"name":"http","active":false}]}

# 切换适配器
$SPECPILOT ad switch ws
{"ok":true,"active":"ws"}

# 查询（通过当前适配器）
$SPECPILOT ad query "SELECT * FROM servers"
{"result":{"cpu_usage":65,"memory_percent":72}}

# 测试连接
$SPECPILOT ad test ws
{"ok":true,"latency_ms":12}
```

### L4 Spec 注册表（SpecRegistry）

```bash
# 列出所有 spec
$SPECPILOT spec list
{"specs":[{"name":"L3-dashboard-kpi","level":"L3","title":"KPI Dashboard","field_count":3,"event_count":2}]}

# 获取单个 spec 详情
$SPECPILOT spec get L3-dashboard-kpi

# 检查字段绑定覆盖率
$SPECPILOT spec binding L3-dashboard-kpi
{"coverage":0.83,"bindings":[...]}

# 检查 spec 完整性
$SPECPILOT spec check L3-dashboard-kpi
```

### L1 MF 组件注册表（MFRegistry）

```bash
# 列出已注册组件（按当前 spec 解析）
$SPECPILOT mf list
{"components":[{"name":"KPICard","path":"prototype/pages/KPICard.svelte"}]}

# 注册新组件
$SPECPILOT mf register KPICard "prototype/pages/KPICard.svelte"

# 按 spec 联动注册
$SPECPILOT mf resolve-from-spec L3-dashboard-kpi
{"components":["KPICard","TrendChart","AlertBadge"],"spec":"L3-dashboard-kpi"}
```

## 四层联动场景

### 场景：用户说"CPU超过80%时KPICard标红"

```
1. 写入 spec field：
   dc set cpu_threshold 80

2. 绑定事件链路：
   ec subscribe kpi:threshold.exceeded KPICard

3. 注册触发组件：
   mf register AlertBadge "prototype/pages/AlertBadge.svelte"

4. 发布事件验证：
   ec emit kpi:threshold.exceeded '{"threshold":80}'

5. 验证数据绑定：
   dc query "SELECT cpu_usage WHERE cpu_usage > 80"
```

### 场景：切换数据源适配器

```
1. 切换到真实 WebSocket 源：
   ad switch ws

2. 验证连接：
   ad test ws

3. 查询新数据：
   ad query "SELECT * FROM servers"

4. 写入 DataCenter 同步：
   dc query "SELECT *"

5. 检查事件订阅是否保持：
   ec history
```

## 集成模式

### 模式 1：Agent 主导（推荐）
Agent 分析用户意图 → 调用 CLI 命令 → 将结果写入 SpecRegistry → 通知用户完成

### 模式 2：前端轮询
HTML/JS 每 5 秒调用 `SPECPILOT dc list` 刷新四层状态（参考 /tmp/specpilot-verify/index.html）

### 模式 3：事件驱动
`ec subscribe` 建立长连接 → 数据变更时事件推送 → 自动更新 DataCenter 状态

## 内置 Spec（开箱即用）

| Name | Level | Fields | Events |
|------|-------|--------|--------|
| L3-dashboard-kpi | L3 | cpu_usage, memory_percent, alert_threshold | kpi:threshold.exceeded, data:updated |
| L3-dashboard-server | L3 | server_name, status, uptime | server:status.changed |
| L3-dashboard-import | L3 | import_status, file_count, rows | import:completed |

## 注意事项

- CLI 命令必须从 `/tmp/specpilot` 目录运行，或设置 `SPECPILOT_BASEDIR=/tmp/specpilot`
- 适配器切换后 DataCenter 不会自动刷新，需要手动 `dc query "SELECT *"` 或 `dc apply-result`
- Spec 绑定覆盖率 < 80% 时会有警告，建议补全字段绑定
- MF 组件注册后需要重新访问原型页面才能热更新
