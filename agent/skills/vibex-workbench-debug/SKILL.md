---
name: vibex-workbench-build-debug
description: vibex-workbench Go build errors, API 404, and coverage report bugs
triggers:
  - "go build error redeclared"
  - "API 404 but handler registered"
  - "coverage report missing L5 count"
  - "workspace_dir not propagating"
---

# vibex-workbench Build Debug Patterns

## Context
vibex-workbench 是一个多文件 Go agent 项目，frontend 用 SvelteKit。构建和调试时有几个常见坑。

---

## Pattern 1: Go 编译错误 — 重复函数声明

### 症状
```
handlers.go:35:6: MakeWorkspaceDetectStateHandler redeclared in this block
  handlers.go:513:6: other declaration of MakeWorkspaceDetectStateHandler
```

### 根因
同一个 `package main` 下两个文件声明了同名的顶级函数。常见于：
- 一个文件是旧版实现（Python 调用），另一个是新版原生实现（纯 Go）
- 两个 agent 在不同分支独立加了同一个功能

### 诊断步骤
```bash
# 1. 找到所有声明位置
grep -rn "func MakeWorkspaceDetectStateHandler" agent/

# 2. 读两个实现的上下文，判断哪个是主版本
#    原生 Go 实现 > Python exec 调用实现
# 3. 删掉旧文件中的重复声明
```

### 修复
保留更完整的版本（通常是原生实现），删掉另一个：
```bash
# 删掉旧实现中的函数体（保留其他函数）
# 注意：只删目标函数，不要误伤相邻函数
```

### 验证
```bash
cd agent && go build ./cmd/web/ 2>&1
```

---

## Pattern 2: 缺少函数实现

### 症状
```
spec_write_protocol.go:53:21: undefined: normalizeWorkspaceRoot
```

### 根因
`handlers.go` 删除了某个函数但没清理所有调用点，或新代码引用了未实现的函数。

### 诊断步骤
```bash
# 找所有调用点
grep -rn "normalizeWorkspaceRoot" agent/

# 找其他同名的实现（可能在其他文件）
grep -rn "func normalize" agent/ --include="*.go"
```

### 修复
在 `spec_write_protocol.go` 顶部补充实现：
```go
// normalizeWorkspaceRoot validates and normalizes a workspace root path.
// Returns (normalizedPath, errorCode). errorCode is empty on success.
func normalizeWorkspaceRoot(ws string) (string, string) {
    if ws == "" {
        return "", "workspace_root_empty"
    }
    clean := filepath.Clean(ws)
    info, err := os.Stat(clean)
    if err != nil {
        if os.IsNotExist(err) {
            return "", "workspace_root_not_found"
        }
        return "", "workspace_root_error"
    }
    if !info.IsDir() {
        return "", "workspace_root_not_directory"
    }
    return clean, ""
}
```

---

## Pattern 3: 旧 binary 缓存导致 API 404

### 症状
API 返回 404，但代码里 handler 已注册。健康检查显示旧 workspace 路径。

### 根因
修改代码后没有重新 `go build`，正在运行的是修改前的 binary。

### 诊断
```bash
# 健康检查看运行的是哪个 workspace
curl -s http://localhost:33338/health > /tmp/health.json
python3 -c "import json; print(json.load(open('/tmp/health.json')).get('workspace_dir', 'N/A'))"

# 看进程
ss -tlnp | grep 33338
# 看 binary 时间戳
ls -la agent/vibex-agent-web
```

### 修复
```bash
cd agent
pkill vibex-agent-web
go build -o vibex-agent-web ./cmd/web/
# 重新启动
./vibex-agent-web > /tmp/vibex-server.log 2>&1 &
sleep 3
curl -s http://localhost:33338/health > /tmp/health2.json
python3 -c "import json; print(json.load(open('/tmp/health2.json')))"
```

---

## Pattern 4: coverage report 漏计数

### 症状
L4 spec 显示 `NO L5` 但实际有对应的 `5_implementation` 文件。

### 根因
`scripts/spec_coverage.py` 只认 `level: 5_slice`，不认 `level: 5_implementation`。

### 修复
```python
# scripts/spec_coverage.py 两处修改：
# 1. l4_to_l5 映射构建
elif info['level'] in ('5_slice', '5_implementation'):

# 2. 总数统计
l5_n = sum(1 for i in specs.values() if i['level'] in ('5_slice', '5_implementation'))
```

### 验证
```bash
python3 scripts/spec_coverage.py
# 检查 L4 without L5 列表是否大幅缩减
```

---

## 调试检查清单

1. `cd agent && go build ./cmd/web/` — 确认 Go 编译干净
2. `pkill vibex-agent-web` — 杀旧进程
3. `go build -o vibex-agent-web ./cmd/web/` — 重新构建
4. `./vibex-agent-web &` — 启动新进程
5. `sleep 3 && curl http://localhost:33338/health` — 确认健康
6. `curl -X POST http://localhost:33338/api/workspace/detect-state -d '{"workspace_root":"/tmp/test"}'` — 测 API

## 相关文件
- `agent/cmd/web/main.go` — 入口，Config 初始化
- `agent/cmd/web/workspace_handlers.go` — workspace HTTP handler 原生实现
- `agent/vibex/domain/spec/handlers.go` — spec domain handler（可能含旧实现）
- `agent/cmd/web/spec_write_protocol.go` — spec 写协议，含 normalizeWorkspaceRoot
- `agent/cmd/web/server.go` — Config.WorkspaceDir 来源
