---
name: grill-me
description: 需求澄清 skill — 逐个追问用户设计决策，附推荐答案，达成共识后输出结构化 spec 草稿。触发条件：用户提出模糊需求、提到"帮我理清"、"grill me"、或启动新项目澄清流程。
triggers:
  - "帮我做一个"
  - "grill me"
  - "需求澄清"
  - "问清楚再开始"
  - 启动新项目时
inputs:
  user_intent: 用户原始需求描述（自然语言，越模糊越好）
  workspace_root: 当前项目根目录（用于查询现有代码结构）
outputs:
  clarification_record: JSON 记录，包含所有问答对 + 最终共识
  spec_draft_path: 生成的 spec 草稿路径（写入 specs/{date}-{hash}.md）
  l2_spec_path: 确认后生成的 L2 spec 路径（写入 specs/L2-skeleton/）
visualization:
  frontend_component: ClarificationPanel（grill-me 模式）
  display: 决策树形式，每个节点是一个问题+用户答案，叶节点是最终决策
  state_stores:
    - $grillSession: 当前追问进度
    - $grillHistory: 全部问答记录
    - $grillConsensus: 最终共识快照
---

# grill-me — 需求澄清 skill（vibex-workbench 版）

## 核心指令

沿着设计决策树的一个个分支追问用户，直到达成完全共识。每个问题附带推荐答案。

**逐个提问，不要一次抛出多个问题。**

如果答案在代码库里，自己去查，不要浪费用户时间。

## 追问流程

### 第一层：Scope 确认
- 这个项目的核心用户是谁？
- 最关键的三个功能是什么？
- 哪些功能**绝对不能砍**？

### 第二层：技术约束
- 有技术栈偏好吗？（语言/框架/数据库）
- 有部署环境限制吗？（本地/云/边缘）
- 对性能有具体要求吗？

### 第三层：交互与体验
- 有参考产品吗？（"像 X 那样"）
- 有品牌/设计偏好？
- 需要支持哪些端？（Web / App / 多端）

### 第四层：数据与持久化
- 需要存哪些数据？
- 有导入/导出需求？
- 对数据隐私有要求？

### 第五层：验收标准
- 怎么算"完成了"？
- 有没有必须通过的测试场景？

## 每个问题的标准格式

```
[问题 N] {具体问题}

推荐答案：{你建议的方案}

A. {方案A}  B. {方案B}  C. {其他}  D. {补充说明}
```

## 存储格式

每轮追问结果存入 `workspace_root/.grill/YYYY-MM-DD-{hash}.json`：
{
  "session_id": "uuid",
  "started_at": "ISO8601",
  "questions": [
    {
      "id": 1,
      "question": "...",
      "recommended_answer": "...",
      "user_answer": "...",
      "branch": "scope|tech|ux|data|acceptance"
    }
  ],
  "consensus": {
    "summary": "一句话共识",
    "key_decisions": [...]
  },
  "spec_draft_path": "specs/YYYY-MM-DD-{hash}.md"
}
```

## 与前端可视化联动

- 每次用户回答，前端 `ClarificationTree` 组件高亮当前节点
- 全部问答完成后，前端显示完整决策树（可折叠/展开）
- `spec_draft_path` 指向的草稿可直接在 SpecExplorer 中打开

## 与现有 ClarificationPanel 的关系

vibex-workbench 已有 **Phase-based 澄清系统**（`ClarificationPanel.svelte` + Go backend `/api/clarifications/*`），基于 4 个固定阶段：tech_stack / mvp_prototype / frontend_split / user_stories。

`grill-me` 是 **另一种澄清范式**（轻量追问范式），两者关系：

| 维度 | ClarificationPanel（Phase-based） | grill-me（追问范式） |
|------|------|------|
| 触发方式 | 点击 L2 spec 卡打开面板 | 用户说"grill me"或模糊需求 |
| 问题来源 | 固定阶段，每个阶段预设问题 | 动态追问，按决策树分支走 |
| 认知负荷 | 用户需填充内容 | 用户只需确认/纠正推荐答案 |
| 存储 | `.memlace/clarifications/*.clf.json` | `.grill/{session-id}.json` |
| UI | ClarificationPanel.svelte（已实现） | ClarificationTree（待实现） |

**推荐路径**：用户说模糊需求时用 grill-me；澄清完成后产出的 spec 草稿可通过现有 ClarificationPanel 继续细化。

## 参考文件

## Meta-check 环节（HeavySkill 增强）

grill-me 追问完成后，执行并行自检，发现遗漏的决策分支：

```
并行阶段（K=3）：
  轨迹1（安全视角）    → 认证/权限/数据备份
  轨迹2（扩展视角）    → 多语言/多租户/插件体系
  轨迹3（运维视角）    → 监控/备份/迁移/成本

汇总结论：
  → 报告遗漏的决策（如"是否需要用户认证"）
  → 记录为"未来扩展项"而非阻断项
  → 用户决定是否补问
```

注意：meta-check 不是替代追问，而是追问完成后的**补充扫描**。

## 局限性

- 不替代 spec-first-workflow，输出是 spec 的**上游输入**
- 不做代码级别的技术评估，那是 trellis 和后续执行阶段的事
- 追问深度由用户控制：说"够了"就停止
