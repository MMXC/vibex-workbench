---
name: vibex-qa-entry-points
description: vibex-qa-entry-points — skill for devops category
category: devops
triggers:
- deploy
- CI/CD
- Cloudflare
- webhook
- monitoring
- Next.js QA
- vibex qa entry points
related_skills:
- systematic-debugging
- test-driven-development
- gstack-browse
---
repo_tracked: true


# Vibex QA 入口路径参考

## PRD 详细设计画布（PRD Canvas）✅ 正确入口

```
Dashboard (vibex-app.pages.dev/dashboard)
  → 登录账号
  → 点击任意项目卡片
  → Tab: "详细设计画布" (📐)
  → "打开详细设计画布 →" 链接
  → URL: /project/?id={project_id}
```

**当前状态（2026-04-15）：** 页面存在但内容为 **占位页**，显示：
- 标题："详细设计画布"
- ⚠️ "完整画布待实现" 警告框
- 待实现功能列表：横向多面板 Scroll-Snap Canvas、DDSCanvasStore (Zustand)、三种卡片类型 (BoundedContext/Flow/Component)、AI Draft Flow、Backend CRUD API
- 说明："Epic1 本次交付: 入口 + 路由基础设施"

**重要：别测错入口！**
- ❌ `/canvas/` → 这是 DDS Canvas 工具（上下文/流程/组件/原型 Tab 画布工具），不是 PRD 画布
- ✅ `/project/?id=xxx` → 这是 PRD 详细设计画布（产品需求文档编辑器）

**已知 Bug（2026-04-15）：**
- 🔴 P0：详细设计画布 404 崩溃。点击"打开详细设计画布 →"后页面显示"重试"。根因：`/api/v1/dds/chapters` 返回 404。
  - 验证：`CI=true $B network 2>&1 | grep 404` 可见
  - 触发路径：Dashboard → 任意项目 → 详细设计画布 Tab → "打开详细设计画布 →"
- 🟡 P1：Canvas Tab 刷新后状态残留。切换 Tab 后 reload，Tab 停留在非默认位置。

## DDS Canvas（上下文/流程/组件/原型画布）

```
直接访问: vibex-app.pages.dev/canvas/
```

- 四个主 Tab：🔵 上下文 / 🔀 流程 / 🧩 组件 / 🚀 原型
- 顶部工具栏：导出 / 需求抽屉 / 消息抽屉 / 版本历史 / 模板
- 已知 Bug：刷新后 Tab 双选状态残留

## Dashboard 项目列表

```
vibex-app.pages.dev/dashboard
```

- 左侧导航包含：⊞ 项目 / 🤖 AI原型设计 / 📊领域模型 / 🎨原型预览 / 📝需求列表 / 🔀流程图 等
- 项目列表点击后进入项目详情页

## 认证注意事项

- 账号：`y760283407@outlook.com` / `12345678`
- Playwright session 会在页面导航后**丢失 auth 状态**，需要重新登录
- 建议：每个测试 session 开始时先完成登录再开始 QA

## QA 技巧

### 抽屉/模态框内容隐藏
标准 `snapshot -i` 有时不显示 drawer 内容，用 `-C` 标志：
```bash
$B snapshot -C -o /tmp/drawer.png
```

### 查看页面真实内容（绕过 JS 渲染）
当 accessibility tree 为空但页面明显有内容时，检查 HTML：
```bash
$B html > /tmp/page.html
# 然后分析 HTML 中的 JS payload（如 __next_f.push）
```

### 占位页识别
搜索 HTML 中 `待实现`、`TODO`、`placeholder`、`coming soon` 等关键词判断是否为占位页。

---

_更新于 2026-04-15：PRD 画布入口发现记录_
