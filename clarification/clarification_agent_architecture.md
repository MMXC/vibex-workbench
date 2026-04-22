# spec-first 澄清自举架构

vibex-workbench 的 spec-first 自举中，L1→L2 阶段需要澄清 agent 与用户对话，从意图生成 L2 spec。

## 澄清轮次设计

```
用户意图 (L1)
    ↓
轮次① 技术选型确认      → tech-stack spec（L2-①）
轮次② MVP原型确认      → mvp-prototype spec（L2-②）
    可交互HTML先出 → 用户确认形态 → 生成可下钻子spec关联原型
轮次③ 前后端分层确认    → frontend/backend spec（L2-③）
    各层边界 + 胶水代码引入规则
轮次④ 功能/用户故事确认 → feature specs（L2-④）
    按阶段优先级，思维导图式确认
    ↓
人工确认 L2 完整性
    ↓
L2 → L3（engine 派生） + 生成器落地
```

## memlace 集成

**澄清对话存储：项目本地 YAML，不混用 Hermes 系统记忆**

```
vibex-workbench/
  clarification/
    vibex-workbench-skeleton_clf.yaml  ← 澄清历史（项目私有）
    vibex-workbench-frontend_clf.yaml
    vibex-workbench-backend_clf.yaml
    vibex-workbench-features_clf.yaml
  lib/
    memlace_client.py                  ← 读 ~/.hermes/memories/ 偏好
```

**memlace_client.py**（轻量，无 Hermes 依赖）：
```python
# 读偏好
memory = Path.home() / ".hermes/memories/MEMORY.md"
user = Path.home() / ".hermes/memories/USER.md"

# 写澄清结果 → 项目本地 YAML，不动 Hermes 原生 memory
clarification_block → specs/clarification/*.clf.yaml

# 人工确认后才回写 USER.md（显式偏好）
```

## clarification YAML 结构

```yaml
spec:
  name: "vibex-workbench-frontend"
  parent: "vibex-workbench-skeleton"
  clarification_round: 2
  status: "in_progress"  # draft | in_progress | confirmed | rejected

rounds:
  - round: 1
    at: "2026-04-22T10:00:00"
    agent_question: "你想要的是哪种交互形态？拖拽/连线/面板式？"
    user_answer: "拖拽节点 + 连线，类似 Figma 的节点图"
    confirmed: true

  - round: 2
    at: "2026-04-22T10:05:00"
    agent_question: "节点需要支持哪些操作？"
    user_answer: "增删改 + 分组，缩放不需要"
    confirmed: true

final:
  confirmed_at: "2026-04-22T10:10:00"
  confirmed_by: "user"
  derived_spec: "specs/architecture/vibex-workbench-frontend.yaml"
```

## spec 图谱集成

```
[spec 图谱]
  点击总目标卡片
  → 展示 L2 spec 卡片列表（4类）
  → 点击任意 L2 卡片
  → 展开澄清对话面板
  → 对话结束 → L2 spec 写入 + 标记 confirmed
  → 回到图谱：L2 卡片变为 ✅confirmed 状态
```

澄清 agent = Hermes skill + vibex-workbench service：

- 输入：memlace read → 读偏好 + 当天上下文
- 触发：用户点击 L2 spec 卡片
- 对话循环：agent 生成问题 → 写 clarification block → 用户回复 → 继续或结束
- 输出：clarification/*.clf.yaml + specs/architecture/*.yaml

## 缺失组件（待实现）

- `clarification_agent.py` — 澄清 agent 主逻辑（读 memlace，生成问题，写 clf.yaml）
- `lib/memlace_client.py` — memlace 读写接口
- frontend 侧澄清对话面板路由

## 关键设计原则

1. **澄清结果隔离**：项目本地 clf.yaml ≠ Hermes 系统记忆
2. **先原型后细节**：轮次②先给可交互 HTML，用户有参照物再聊细节
3. **分离形态确认 vs 功能确认**：②是"像不像"，④是"对不对"
4. **人工终态**：澄清完成后必须用户确认，才触发 engine 派生
