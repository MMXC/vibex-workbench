# Spec — E6: Workbench Shell

> PRD: `prd.md` E6 章节
> 规范层级: L4 Feature Spec

## 三栏布局激活

### 当前问题
```css
grid-template-columns: 280px 1fr 0px;  /* 右栏宽度 0 */
```

### 目标行为
```css
grid-template-columns: 280px 1fr 320px;
```

---

## 响应式断点规范

| 断点 | 布局 | 说明 |
|------|------|------|
| ≥ 1200px | 280px \| 1fr \| 320px | 完整三栏 |
| 768px–1199px | 240px \| 1fr \| 280px | 压缩侧栏 |
| < 768px | 1fr（侧栏隐藏） | 单栏，Composer 固定底部 |

### 断点实现
```svelte
<!-- WorkbenchShell.svelte -->
<script>
  let windowWidth = $state(1440);
  
  $effect(() => {
    if (typeof window !== 'undefined') {
      windowWidth = window.innerWidth;
    }
  });
  
  let leftW = $derived(windowWidth >= 1200 ? 280 : 240);
  let rightW = $derived(windowWidth >= 768 ? (windowWidth >= 1200 ? 320 : 280) : 0);
</script>

<div class="shell" style="
  grid-template-columns: {leftW}px 1fr {rightW}px;
">
```

---

## 栏宽动画
- 栏宽变化使用 CSS transition：`transition: grid-template-columns 200ms ease`
- 禁止动画过程中内容抖动

---

## Composer 始终可见
- Composer 固定在底部：`grid-area: B`
- 最小高度：`auto`，最大高度：35vh（输入框可扩展）
- 移动端：`position: fixed; bottom: 0; width: 100%`

---

## 布局降级规范

### 断点 < 768px（移动端）
```
┌────────────────────────────────┐
│ ☰  [Thread/Artifact 切换]     │  ← 顶部 Tab
├────────────────────────────────┤
│                                │
│         主区域                  │
│                                │
├────────────────────────────────┤
│         Composer               │
└────────────────────────────────┘
```
- Tab 切换 ThreadList / ArtifactPanel
- 主区域显示 Canvas

---

## 间距/颜色 Token

```css
--shell-left-width: 280px;   /* ≥1200px */
--shell-left-width-sm: 240px; /* 768-1199px */
--shell-right-width: 320px;   /* ≥1200px */
--shell-right-width-sm: 280px;/* 768-1199px */
--composer-min-height: 80px;
--composer-max-height: 35vh;

--color-shell-bg: #111;
--color-sidebar-bg: #111;
--color-main-bg: #0a0a0a;
--color-panel-bg: #111;
```

---

## 验收标准

```typescript
// F6.1 — 右栏宽度
const rightPanel = document.querySelector('.sidebar-right');
expect(rightPanel.offsetWidth).toBe(320);

// F6.2 — 响应式
// 1440px
window.resizeTo(1440, 900);
expect(shellGridStyle).toContain('280px 1fr 320px');

// 1024px
window.resizeTo(1024, 768);
expect(shellGridStyle).toContain('240px 1fr 280px');

// 768px
window.resizeTo(768, 1024);
expect(shellGridStyle).toContain('240px 1fr 280px');

// 375px（移动端）
window.resizeTo(375, 667);
expect(composer.offsetTop + composer.offsetHeight).toBeCloseTo(window.innerHeight, 0);
expect(ArtifactPanel).not.toBeVisible();  // 隐藏
```

### DoD
- [ ] 右栏 `grid-template-columns` 从 `0px` 改为 `320px`
- [ ] 1440px: 三栏完整 (280 | 1fr | 320)
- [ ] 1024px: 三栏压缩 (240 | 1fr | 280)
- [ ] 768px: 可用布局，Composer 始终可见
- [ ] < 768px: 移动端降级，Tab 切换侧栏
- [ ] 所有间距使用 CSS Token（--shell-*）
- [ ] 布局变化有平滑过渡（200ms ease）
