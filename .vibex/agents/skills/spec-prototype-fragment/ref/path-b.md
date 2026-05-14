# 路径 B：基于 MVP 相似区域扩展

> 关联：§5 — 仅当 MVP 中完全没有对应区域时使用。必须先询问用户。

## 5.1 何时使用路径 B

- spec 描述的功能在 MVP 中完全不存在（如独立的向导页面）
- 用户明确指定使用路径 B

## 5.2 询问格式

```
### 需要确认：原型 UI 位置

**spec**：`{spec.name}`（{level}）
**spec.title**：`{display.title}`

该 spec 在 MVP 中没有直接对应的 UI 区域。

请描述该功能的 UI 位置：
1. 作为顶部 titlebar trail 区的一个按钮？
2. 作为左侧 activity bar 的一个图标？
3. 作为中央视图区的独立 tab？
4. 作为右下角弹窗？
5. 其他（请描述）：________
```

## 5.3 基于相似区域扩展

即使从零构建，也要基于 MVP 的设计语言：

1. **复用 MVP 的 CSS 变量**（颜色、字体、圆角、间距）
2. **参考 MVP 中相似组件的样式**（按钮、弹窗、输入框、卡片）
3. **布局必须与 MVP 风格一致**（深色背景、等宽字体、vscode 风格）

## 5.4 MVP 样式基准

从 MVP 中提取可复用的样式基准：

```css
/* 弹窗 */
background: #0f1117;
border: 1px solid #1e2235;
border-radius: 10px;
box-shadow: 0 24px 64px rgba(0,0,0,.6);

/* 按钮主色调 */
--accent-orange: #f09a6a;
--wb-accent: #72d6d0;
--accent-green: #87cf8a;
--accent-red: #e16d75;

/* 文字色 */
--wb-text: #c0caf5;
--wb-text-sec: #787c99;
```
