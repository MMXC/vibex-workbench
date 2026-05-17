---
name: specpilot-agent
description: |
  SpecPilot Agent — Agent-accessible prototype workspace manager.
  触发条件：用户提到"原型"、"preview"、"specpilot"、"MF组件"、"数据中心"、"生成原型"、"看原型"、"启动服务"。
  触发词：prototype、preview、specpilot、mf、datacenter、start、generate。
  当用户在项目中需要生成/预览原型、管理 MF 组件、或通过 AI Agent 驱动原型迭代时，加载此技能。
---

# SpecPilot Agent

SpecPilot 是一个 workspace 级别的 prototype workspace manager，每个 workspace 独立运行自己的 SpecPilot CLI 实例（init → start 后台常驻）。

## 核心架构

```
workspace/
  .specpilot/
    cli/                   # SpecPilot CLI（python -m specpilot）
    components/            # 已注册的 MF 组件
    previews/              # 原型 HTML 文件
    static/                # 静态资源
    .meta.json             # 端口等元信息
    .pids.json             # 运行中的服务进程
```

## CLI 路径

```bash
cd <workspace>
python3 -m specpilot          # 从 workspace 根运行（自动找 .specpilot/）
# 或
specpilot                     # 需 pip install -e .specpilot/cli
```

## 命令速查

### 初始化（首次打开 workspace）
```bash
specpilot init
# 输出: [specpilot] Initialized at <workspace>/.specpilot
#       [specpilot] Run 'specpilot start' to launch services.
```

### 启动服务（init 后执行一次，之后每次打开 workspace 复用）
```bash
specpilot start
# 前台阻塞，直到 Ctrl+C
# 启动 DataCenter (:7890) + Preview (:5177) 两个 HTTP 服务
# 幂等：检测到端口已监听则跳过
```

### 查看状态
```bash
specpilot status
# Workspace : /path/to/workspace
# SpecPilot : initialized
# DataCenter: 🟢 running  (port 7890)
# Preview   : 🟢 running  (port 5177)
# DC keys   : 7
```

### 停止服务
```bash
specpilot stop
```

### 生成原型（从 spec YAML 生成 HTML）
```bash
specpilot generate <spec.yaml> [-o <output.html>]
# 输出: [specpilot] Generated: .specpilot/previews/index.html
#       [specpilot] Preview:   http://127.0.0.1:5177/preview
```

### DataCenter 读写
```bash
specpilot dc list
# {"ok":true,"keys":{"kpi.revenue":"1.2M","table.users":{...},...}}

specpilot dc get <key>
# {"ok":true,"key":"kpi.revenue","value":"1.2M"}

specpilot dc set <key> <value>
# {"ok":true,"key":"kpi.revenue","value":"2.0M"}
```

### MF 组件注册
```bash
specpilot register <component> --mf-url "/#/KPICard" --dc-key "kpi.revenue"
# [specpilot] Registered: KPICard → /#/KPICard
```

### 打开预览 URL
```bash
specpilot preview
```

## 端到端流程（Agent 驱动）

### 流程 1：首次打开项目，原型预览
```
1. specpilot init                        # 初始化 workspace
2. specpilot start                       # 启动 DC + Preview
3. specpilot generate .specpilot/specs/L3-dashboard.yaml
4. → iframe 加载 http://127.0.0.1:5177/preview
```

### 流程 2：用户说"把收入改成 2.0M"
```
1. specpilot dc set kpi.revenue 2.0M
2. → iframe 自动热刷新（WebSocket reload）
```

### 流程 3：用户说"做一个新的按钮"
```
1. Agent 生成新的 HTML 内容
2. 写入 .specpilot/previews/index.html
3. → iframe 自动热刷新
```

### 流程 4：注册新组件
```
1. specpilot register KPICard --mf-url "/#/KPICard" --dc-key "kpi.revenue"
2. curl http://127.0.0.1:5177/api/components
```

## 前端集成（iframe 预览）

- 预览 URL：`http://127.0.0.1:5177/preview`
- DataCenter API：`http://127.0.0.1:7890/api/dc/<key>`
- 端口可通过环境变量覆盖：`SPECPILOT_DC_PORT=18090 SPECPILOT_MF_PORT=18077`

## 幂等性保证

- **目录层**：每个 workspace 有独立的 `.specpilot/`，多 workspace 并行
- **安装层**：`init` 发现已存在则跳过
- **进程层**：`start` 检测端口已监听则跳过
- **Meta 层**：`init` 写入 `.meta.json`，`start` 读取决定端口

## 注意事项

- CLI 从 workspace 根目录运行，自动找 `.specpilot/` 子目录
- `start` 是前台阻塞命令，Go Agent 需要后台 fork subprocess
- 预览 iframe 的 HTML 会自动注入 DataCenter API helper（`window.__dc.get/set/list`）
- 热加载通过 WebSocket `ws://127.0.0.1:5177/__live` 触发页面 reload
