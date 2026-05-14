---
name: prototype-annotator
description: |
  Chrome 扩展配套的原型标注技能。在以下场景必须加载本技能：
  对原型 HTML 按 L1/L3/L4/L5 层级添加虚线框+颜色+Ln标注，展示 spec 层级结构；
  在原型上叠加视觉层级标记；确认标注效果后写回 HTML 文件。
  触发词：标注原型、添加层级标注、L1/L3/L4/L5 标注、原型加标注、虚线框标注。
category: spec-driven-project
title: Prototype Annotator（原型标注）
triggers:
  - zh: 标注原型, 添加层级标注, L1标注, L3标注, L4标注, L5标注, 虚线框标注, 原型加标注
  - en: annotate prototype, add level annotation, L1 L3 L4 L5 annotation
related_skills:
  - prototype-spec-extractor: 原型拆解为 L1–L5 spec；标注是 spec 后的视觉映射。
---

# Prototype Annotator（原型标注技能）

**规范路径**：与本目录 **`agent.json`** 成对放置于 **`.vibex/agents/skills/prototype-annotator/`**。请求体使用 **`agent_profile=prototype-annotator`** 时，Agent 加载本 `SKILL.md`。

---

## 0. 入口：读取原型 HTML

每次任务开始前，Agent 必须先读取原型 HTML 文件内容：

1. 从 `ps_get_page_context` 或 `ps_parse` 获取当前原型的 `workspaceRoot` 和 `prototypeRel`
2. 拼接完整路径：`{workspaceRoot}/.vibex/prototypes/{prototypeRel}`
3. 使用 `read_file` 读取 HTML 内容
4. 分析 DOM 结构，识别可标注区域

---

## 1. 标注格式约定

### 1.1 标注属性（直接注入 HTML）

在原型 HTML 中，使用 **`data-vibex-annot`** 属性标记区域：

```html
<!-- L1 页面骨架标注 -->
<div class="wb-root" data-vibex-annot="L1" data-vibex-title="L1 · 页面骨架">
  ...
</div>

<!-- L3 视觉大块标注 -->
<div class="titlebar" data-vibex-annot="L3" data-vibex-title="L3 · 顶部标题栏">
  ...
</div>
<div class="wb-left-composite" data-vibex-annot="L3" data-vibex-title="L3 · 左侧栏组合">
  ...
  <!-- L4 必须在 L3 内部 -->
  <div class="activity-bar" data-vibex-annot="L4" data-vibex-title="L4 · 活动图标栏">
    <!-- L5 在 L4 内部，为按钮或内容块 -->
    <button data-vibex-annot="L5" data-vibex-title="L5 · 资源管理器图标"
      data-vibex-parent-l3="wb-left-composite">
      ...
    </button>
  </div>
</div>
```

### 1.2 层级嵌套规则（强制）

```
L1（页面骨架）
 └── L3（视觉大块：顶/底/左/主/右）
      └── L4（功能组件）  ← 必须有父级 L3
           └── L5（按钮/内容切片）  ← 必须在 L4 内部
```

**L4 必须在 L3 内部；L5 必须在 L4 内部。** 严禁跨层级标注。

---

## 2. 标注样式 + 控制脚本（技能脚本区）

**所有标注样式和控制逻辑统一注入原型 HTML，技能脚本区可整体复用。**

### 2.1 标注 CSS（注入 `<head>` 末尾）

```css
/* ── VibeX 原型层级标注样式（prototype-annotator 技能脚本） ── */
/* 纯粹叠加，不改原样式。不使用 !important，不叠加 background，只叠加虚线框和标签。 */
[data-vibex-annot] {
  position: relative;
}

/* 虚线边框叠加层 */
[data-vibex-annot]::before {
  content: '';
  position: absolute;
  inset: 0;
  border: 2px dashed var(--vibex-color, #ccc);
  border-radius: inherit;
  pointer-events: none;
  z-index: 9998;
}

/* 层级标签浮层 */
[data-vibex-annot]::after {
  content: attr(data-vibex-title);
  position: absolute;
  top: -18px;
  left: 0;
  background: var(--vibex-color, #ccc);
  color: #fff;
  font-size: 10px;
  font-family: ui-monospace, monospace;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 3px 3px 3px 0;
  white-space: nowrap;
  z-index: 9999;
  pointer-events: none;
  line-height: 14px;
  box-shadow: 0 2px 6px rgba(0,0,0,.4);
}

/* L1 骨架层 — 蓝灰 */
[data-vibex-annot="L1"] { --vibex-color: #6b7280; }
/* L3 模块层 — 橙色 */
[data-vibex-annot="L3"] { --vibex-color: #f59e0b; }
/* L4 功能层 — 紫色 */
[data-vibex-annot="L4"] { --vibex-color: #8b5cf6; }
/* L5 切片层 — 青色 */
[data-vibex-annot="L5"] { --vibex-color: #06b6d4; }
```

### 2.2 层级颜色对照

| 层级 | 颜色 | CSS 变量 |
|------|------|---------|
| L1 骨架 | `#6b7280`（蓝灰） | `--vibex-L1` |
| L3 模块 | `#f59e0b`（橙色） | `--vibex-L3` |
| L4 功能 | `#8b5cf6`（紫色） | `--vibex-L4` |
| L5 切片 | `#06b6d4`（青色） | `--vibex-L5` |

### 2.3 悬浮控制栏 CSS

```css
/* ── VibeX 标注悬浮控制栏 ── */
#vibex-annot-ctrl {
  position: fixed;
  bottom: 16px;
  right: 16px;
  z-index: 99999;
  display: flex;
  gap: 4px;
  background: rgba(15, 15, 15, 0.88);
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 8px;
  padding: 6px 8px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  font-family: ui-monospace, monospace;
  font-size: 11px;
  backdrop-filter: blur(8px);
}
#vibex-annot-ctrl button {
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: 4px;
  background: transparent;
  color: #9ca3af;
  cursor: pointer;
  padding: 3px 7px;
  font-size: 11px;
  font-family: ui-monospace, monospace;
  transition: all 0.15s;
  line-height: 1;
}
#vibex-annot-ctrl button:hover { background: rgba(255,255,255,0.08); color: #f3f4f6; }
#vibex-annot-ctrl button.active { font-weight: 700; border-color: currentColor; }
#vibex-annot-ctrl button.active-l1 { color: #9ca3af; }
#vibex-annot-ctrl button.active-l3 { color: #f59e0b; }
#vibex-annot-ctrl button.active-l4 { color: #8b5cf6; }
#vibex-annot-ctrl button.active-l5 { color: #06b6d4; }
#vibex-annot-ctrl button.active-all { color: #e5e7eb; background: rgba(255,255,255,0.1); }
#vibex-annot-ctrl button.active-hide { color: #6b7280; }

/* 隐藏时移除相对定位，隐藏伪元素 */
[data-vibex-hide] { position: static !important; }
[data-vibex-hide]::before,
[data-vibex-hide]::after { display: none !important; }
```

### 2.4 悬浮控制栏 HTML

```html
<!-- VibeX 标注控制栏（prototype-annotator 技能脚本注入） -->
<div id="vibex-annot-ctrl">
  <button id="vbtn-all" class="active-all active" data-level="all">全部</button>
  <button id="vbtn-l1" class="active-l1" data-level="L1">L1</button>
  <button id="vbtn-l3" class="active-l3" data-level="L3">L3</button>
  <button id="vbtn-l4" class="active-l4" data-level="L4">L4</button>
  <button id="vbtn-l5" class="active-l5" data-level="L5">L5</button>
  <button id="vbtn-hide" class="active-hide" data-level="hide">隐藏</button>
</div>
```

### 2.5 悬浮控制栏 JS

```javascript
/* VibeX 标注控制栏脚本（prototype-annotator 技能脚本注入） */
(function () {
  var ctrl = document.getElementById('vibex-annot-ctrl');
  if (!ctrl) return;
  var btnAll = document.getElementById('vbtn-all');
  var btns = Array.prototype.slice.call(
    document.querySelectorAll('#vibex-annot-ctrl button[data-level]')
  );

  function activate(level) {
    btns.forEach(function (b) { b.classList.remove('active'); });
    if (level === 'all') {
      btnAll.classList.add('active');
      document.querySelectorAll('[data-vibex-annot]').forEach(function (el) {
        el.removeAttribute('data-vibex-hide');
      });
    } else if (level === 'hide') {
      btnAll.classList.add('active');
      document.querySelectorAll('[data-vibex-annot]').forEach(function (el) {
        el.setAttribute('data-vibex-hide', '');
      });
    } else {
      btns.filter(function (b) { return b.dataset.level === level; })
          .forEach(function (b) { b.classList.add('active'); });
      // 先全部隐藏
      document.querySelectorAll('[data-vibex-annot]').forEach(function (el) {
        el.setAttribute('data-vibex-hide', '');
      });
      // 再显示当前层
      document.querySelectorAll('[data-vibex-annot="' + level + '"]').forEach(function (el) {
        el.removeAttribute('data-vibex-hide');
      });
    }
  }

  btns.forEach(function (b) {
    b.addEventListener('click', function () { activate(b.dataset.level); });
  });
})();
```

### 2.6 技能脚本注入位置

标注样式 → `<head>` 末尾（`</style>` 标签之后）
控制栏 HTML → `</body>` 之前
控制栏 JS → `</body>` 之前（`<script>` 块内）

---

## 3. 标注工作流程（分步递归）

**核心原则：先父后子，先粗后细。每步只做当前层级的标注，完成后请用户确认，再进入下一层级。**

---

### 第一步：标注 L1 + L3（所有视觉大块）

1. **读取原型 HTML**：`read_file` 完整内容
2. **识别 L1**：最外层容器（页面骨架），通常为 `body` 直接子元素
3. **识别所有 L3**：在 L1 下，按传统 IDE 五分区枚举所有视觉大块（顶/底/左/主/右/状态栏），**一个都不能漏**
4. **输出 L1+L3 标注计划**（文字 + 表格）：

   ```
   ### 第一步标注计划

   | 序号 | 区域 | 选择器 | 层级 | 理由 |
   |------|------|---------|------|------|
   | 1 | 页面根容器 | .wb-root | L1 | 整体骨架 |
   | 2 | 顶部标题栏 | .titlebar | L3 | 视觉大块：顶部 |
   | 3 | 左侧栏 | .wb-left-composite | L3 | 视觉大块：左侧 |
   | 4 | 主内容区 | .wb-main | L3 | 视觉大块：主体 |
   | 5 | 右侧 AI 栏 | .wb-right | L3 | 视觉大块：右侧 |
   | 6 | 底部 Dock | .wb-dock-wrap | L3 | 视觉大块：底部 |
   | 7 | 底部状态栏 | .statusbar | L3 | 视觉大块：底部状态栏 |
   ```
5. **用户确认 L1+L3** 后，写入文件（只写 L1 和 L3 属性）
6. **`ps_refresh`** 刷新页面，确认标注效果

---

### 第二步：逐个 L3 细分标注其 L4（每个 L3 单独处理）

对**每一个 L3 区域**，依次执行：

7. **选定当前 L3**：在 HTML 中定位到该 L3 区域的源码
8. **识别 L4**：在当前 L3 内找完整的小功能块（列表见 §4.1）
9. **输出该 L3 的 L4 标注计划**：

   ```
   ### L3「{L3标题}」→ L4 标注计划

   | L3 | 选择器 | L4 | 选择器 | 理由 |
   |----|---------|----|---------|------|
   | 左侧栏 | .wb-left-composite | 活动图标栏 | .activity-bar | 导航入口 |
   | 左侧栏 | .wb-left-composite | Spec 资源管理器 | .spec-explorer | 完整功能 |
   ```
10. **用户确认该 L3 的 L4** 后，写入文件（只写该 L3 内部元素的 L4 属性）
11. **`ps_refresh`** 刷新页面，确认该 L3 → L4 的标注效果

**重复步骤 7-11，直至所有 L3 的 L4 全部标注完成。**

> 🔄 **Agent 自我审查（每完成一个 L3 的 L4 后执行）**：
> 1. 扫描该 L3 的 HTML 结构，确认所有功能子块均已标注为 L4
> 2. 扫描其他 L3，确认没有遗漏的 L4 区域
> 3. 如有遗漏，立即补全后再进入下一步

---

### 第三步：逐个 L4 细分标注其 L5（每个 L4 单独处理）

对**每一个 L4 组件**，依次执行：

12. **选定当前 L4**：在 HTML 中定位到该 L4 区域的源码
13. **识别 L5**：在当前 L4 内找按钮、表单项、内容切片；判断是否有打开新页面级组件的 L5（如弹窗、抽屉触发器），**如有则必须加 `data-vibex-parent-l3`**
14. **输出该 L4 的 L5 标注计划**：

    ```
    ### L4「{L4标题}」→ L5 标注计划

    | L4 | L5 | 选择器 | 理由 | 跨页关联 |
    |----|----|---------|------|---------|
    | 活动图标栏 | 资源管理器图标 | button[title="资源管理器"] | 导航入口 | wb-left-composite |
    | Dock 面板 | Dock Tab | .dock-tabs | 面板切换 | wb-dock-wrap |
    ```
15. **用户确认该 L4 的 L5** 后，写入文件（只写该 L4 内部元素的 L5 属性）
16. **`ps_refresh`** 刷新页面，确认该 L4 → L5 的标注效果

**重复步骤 12-16，直至所有 L4 的 L5 全部标注完成。**

> 🔄 **Agent 自我审查（每完成一个 L4 的 L5 后执行）**：
> 1. 扫描该 L4 的 HTML 结构，确认所有按钮/表单项/内容块均已标注为 L5
> 2. 扫描同级 L4，确认没有遗漏的 L5
> 3. 扫描父 L3 内其他 L4，确认没有遗漏的 L5
> 4. 如有遗漏，立即补全后再进入下一步

---

### 第四步：注入技能脚本

17. **注入标注 CSS**（见 §2.1）、**控制栏 CSS**（见 §2.3）→ 追加到 `<head>` 末尾 `</style>` 之后
18. **注入控制栏 HTML**（见 §2.4）→ 追加到 `</body>` 之前
19. **注入控制栏 JS**（见 §2.5）→ 追加到 `</body>` 之前（独立 `<script>` 块）
20. **`write_file`** 完整 HTML 到原型文件
21. **`ps_refresh`** 最终刷新，确认控制栏可用

---

## 4. 标注决策规则

### 4.1 层级定义

| 层级 | 定义 | 典型选择器 |
|------|------|-----------|
| **L1 骨架** | 页面整体结构，无 L1 则整个页面不可分割 | `body > div.wb-root`，`<body>` |
| **L3 区块** | 视觉大块，IDE 布局级区域（顶/底/左/主/右）；L4/L5 的父容器；**每个 L3 必须有 L4 子级；完成后执行自我审查确认无遗漏** | `.titlebar`（顶部）、`.wb-dock-wrap`（底部）、`.wb-left-composite`（左侧）、`.wb-main`（主体）、`.wb-right`（右侧）、`.statusbar`（底部状态栏） |
| **L4 功能** | 完整的小功能块；在 L3 内部；完成后执行自我审查扫描同级/父级是否有 L5 遗漏 | `.menu-strip`，`.command-center`，`.window-controls`，`.activity-bar`，`.spec-explorer`，`.center-r2`，`.ai-column`，`.dock`，`.dock-tabs`，`.dock-body`，`.sb-left`，`.sb-right` |
| **L5 切片** | L4 内的行为级细节：按钮、表单项、内容块；**若有跨 L3 跳转必须加 `data-vibex-parent-l3`** | `button`（导航/操作），`.hdr`，`.chat-region`，`.composer-region`，`.dock-tab` |

### 4.2 嵌套约束

```
❌ 错误：L4 不在 L3 内
  <div L1>
    <div L4>  ← 缺少 L3 父级
      <div L5>

✅ 正确：L3 → L4 → L5
  <div L1>
    <div L3>
      <div L4>
        <div L5>
```

### 4.3 L5 跨页关联

当 L5 元素点击后会打开一个新的页面级组件时：

1. 给该 L5 元素添加 `data-vibex-parent-l3="{L3类名}"` 属性
2. 控制栏切换 L5 时，自动高亮关联的 L3（通过 `data-vibex-parent-l3` 匹配）

**不要过度标注**：只在关键结构点加标注，不要每个 div 都加。

---

## 5. 扩展 PS 工具用法

| 工具 | 用途 |
|------|------|
| `ps_parse` | 解析当前原型页面，获取 DOM 结构辅助判断 |
| `ps_highlight` | 临时高亮指定选择器，验证区域范围 |
| `ps_refresh` | 刷新页面，查看最新标注效果 |
| `ps_get_page_context` | 获取当前原型路径和 workspace |

---

## 6. 输出规范

每次任务输出：

1. **标注计划**（文字）：列出每个将被标注的区域 + 对应层级 + 理由 + L5 跨页关联声明
2. **HTML 代码块**：完整修改后的 HTML（含标注样式、控制栏、标注属性）
3. **标注清单**：表格形式列出所有标注区域
4. **技能脚本清单**：列出注入的 CSS / HTML / JS 区块

---

## 7. 禁止事项

- 不在运行时 DOM 上标注（只修改 HTML 源文件）
- 不删除原有功能和样式，只叠加标注层
- 不在标注中写业务逻辑
- L4 不能独立于 L3 存在；L5 不能独立于 L4 存在
- 标注样式不得使用 `!important`（除控制栏隐藏逻辑外）

---

## 8. 示例输出

```
### 标注计划

```
L1 · 页面骨架
 └── L3 · 顶部标题栏
      ├── L4 · 菜单栏
      ├── L4 · 命令中心
      └── L4 · 窗口控制
 └── L3 · 左侧栏
      ├── L4 · 活动图标栏
      │    └── L5 · 资源管理器（按钮，跨页关联 wb-left-composite）
      └── L4 · Spec 资源管理器
 └── L3 · 主内容区
      └── L4 · 中心视图
 └── L3 · 右侧 AI 栏
      └── L4 · Agent 对话
           ├── L5 · AI 栏标题
           ├── L5 · 对话历史（跨页关联 wb-right）
           └── L5 · 作曲输入（跨页关联 wb-right）
 └── L3 · 底部 Dock
      └── L4 · Dock 面板
           ├── L4 · Dock Tab 栏
           └── L4 · Dock 内容区
 └── L3 · 底部状态栏
      ├── L4 · 状态栏左
      └── L4 · 状态栏右
```

| L3 | L4 | 选择器 | L5 | 选择器 | 跨页关联 |
|----|----|---------|----|---------|---------|
| 顶部标题栏 | 菜单栏 | .menu-strip | — | — | — |
| 顶部标题栏 | 命令中心 | .command-center | — | — | — |
| 顶部标题栏 | 窗口控制 | .window-controls | — | — | — |
| 左侧栏 | 活动图标栏 | .activity-bar | 资源管理器 | button[title="资源管理器"] | wb-left-composite |
| 左侧栏 | Spec 资源管理器 | .spec-explorer | — | — | — |
| 主内容区 | 中心视图 | .center-r2 | — | — | — |
| 右侧 AI 栏 | Agent 对话 | .ai-column | AI 栏标题 | .hdr | — |
| 右侧 AI 栏 | Agent 对话 | .ai-column | 对话历史 | .chat-region | wb-right |
| 右侧 AI 栏 | Agent 对话 | .ai-column | 作曲输入 | .composer-region | wb-right |
| 底部 Dock | Dock 面板 | .dock | — | — | — |
| 底部 Dock | Dock Tab 栏 | .dock-tabs | — | — | — |
| 底部 Dock | Dock 内容区 | .dock-body | — | — | — |
| 底部状态栏 | 状态栏左 | .sb-left | — | — | — |
| 底部状态栏 | 状态栏右 | .sb-right | — | — | — |

### 技能脚本清单

- §2.1 标注 CSS（注入 <head>）
- §2.3 控制栏 CSS（注入 <head>）
- §2.4 控制栏 HTML（注入 </body> 前）
- §2.5 控制栏 JS（注入 </body> 前）

请确认效果，确认后我将写入文件并刷新页面。
```

---

*技能版本：0.4 · 每个 L3 必有 L4 + 完整树状结构示例 v0.4。*
