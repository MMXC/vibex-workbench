# 校验与输出规范

> 关联：§9（输出规范）+ §11（校验表）— 生成 HTML 后、写盘前必须逐行核对。

## 校验表：§11.A–§11.H

### §11.A MVP 全量复制

| 检查项 | 通过标准 | 常见错误 |
|--------|---------|---------|
| 全部区块保留 | wb-root 内所有子元素均保留 | 错误裁剪了 dock 或 statusbar |
| 非关联区域 | 无 `hidden` / `display:none` / `visibility:hidden` | 对无关区域误加了 hidden |

### §11.B MVP CSS 复用

| 检查项 | 通过标准 | 常见错误 |
|--------|---------|---------|
| 背景色 | `var(--wb-bg-panel)` / `var(--wb-bg-base)` 或一致颜色 | 自己发明 `#222` |
| 文字色 | `var(--wb-text)` / `var(--wb-text-sec)` 或一致颜色 | 自己发明 `#eee` |
| accent 色 | `var(--accent-orange)` / `var(--wb-accent)` / `var(--accent-green)` 等 | 自己发明 `#ff6600` |
| 字体 | MVP 的 font-family（`var(--font-ui)` / `var(--font-mono)`） | 自己发明其他字体 |
| 圆角/间距 | 与 MVP 组件一致 | 随意设定 |

### §11.C web component 替换验证

| 检查项 | 通过标准 | 常见错误 |
|--------|---------|---------|
| component 注册 | `<head>` 中有 `customElements.define('wc-{spec-name}', ...)` | 用普通 DOM 替代而非 component |
| shadow DOM | component 内有 `attachShadow({mode:'open'})` | 没有 shadow DOM，UI 在外部 |
| 四态 panels | `#ph-open` `#ph-loading` `#ph-done` `#ph-error` 全部在 shadow DOM 内 | 四态 panel 放在外部 body |
| 标注迁移 | spec 关联区域标注（`data-vibex-annot`）在 `<wc-*>` host 或 shadow DOM 内 | 标注丢失或写在外部 |

### §11.D 初态校验（component idle）

| 检查项 | 通过标准 | 常见错误 |
|--------|---------|---------|
| 触发按钮 | shadow DOM 内可见，无 `hidden` / `disabled` | 按钮被误加 `hidden` |
| `#overlay` | 初始 `class="hidden"` | 初始就展示弹窗 |
| 所有 `.phase-*` panel | 初始 `class="hidden"` | 某个 panel 忘记加 hidden |
| `this._phase` | 初始为 `'idle'` | 初始 phase 不对 |

### §11.E component _syncUI() 逐行核对

核心规则：`_phase` 改变时调用 `_syncUI()`，逐状态核对 panel 显示/隐藏。

| `this._phase` | overlay | `#ph-open` | `#ph-loading` | `#ph-done` | `#ph-error` | 常见错误 |
|--------------|---------|-----------|-------------|-----------|---------|---------|
| `idle` | hidden | hidden | hidden | hidden | hidden | overlay 未 hidden |
| `open` | **可见** | **可见** | hidden | hidden | hidden | `loading`/done/error 被 show |
| `loading` | **可见** | hidden | **可见** | hidden | hidden | `open`/done/error 未 hidden |
| `done` | **可见** | hidden | hidden | **可见** | hidden | **done 忘记 remove hidden** |
| `error` | **可见** | hidden | hidden | hidden | **可见** | **error 忘记 remove hidden** |

### §11.F 弹窗关闭路径（component 内）

每个 phase 都必须有至少一条路径回到 `idle`：

| 触发元素 | 目标 phase | 哪些 phase 支持 |
|---------|-----------|---------------|
| 「取消」按钮 | `idle` | `open` `loading` `error` |
| 「×」按钮 | `idle` | `open` `loading` `done` `error` |
| 遮罩背景点击 | `idle` | `open` `loading` `done` `error` |
| 「完成」按钮 | `idle` | `done`（专用） |
| 「重试」按钮 | `open` 或 `loading` | `error`（专用） |

### §11.G 事件绑定

| 检查项 | 通过标准 |
|--------|---------|
| 触发按钮 | shadow DOM 内绑定 → `_setPhase('open')` |
| 所有关闭按钮 | shadow DOM 内绑定 → `_setPhase('idle')` |
| 外部 script | 无任何外部 `<script>` 操作 component 内部 DOM |

### §11.H CSS 变量继承

| 检查项 | 通过标准 |
|--------|---------|
| shadow DOM 内使用 | `var(--xxx)` 引用 MVP CSS 变量（如 `--wb-bg-panel`） |
| 注入方式 | 或 `connectedCallback()` 中有 `_injectCssVars()` |

---

## 输出规范模板：§9

每次任务分两个阶段输出：

### 阶段一：区块识别确认（必须先完成）

```
### 区块识别确认

**spec**：`{spec.name}`（{level}）
**spec.title**：`{display.title}`
**源原型**：`{顶层 L1 spec prototype.file}`（顶层原型是完整基准）

在**顶层原型**中定位该 spec 对应的标注区域：

| # | 区块 | selector | 置信度 | 理由 |
|---|-----|---------|--------|------|
| 1 | … | … | 高/中/低 | … |

**我的判断**：倾向于 #N，因为……

请确认：回复区块编号，或告诉我正确的选择器。
```

### 阶段二：预览与写盘（用户确认后）

**必须先输出校验清单，确认全部通过后再输出代码。**

```
### 原型构建计划

| 项目 | 内容 |
|------|------|
| spec | {spec.name} |
| 区块 selector | {confirmed selector} |
| 构建路径 | A（完整 MVP 复制 + 关联区域替换为 <wc-{spec-name}> web component）|
| MVP CSS 复用 | ✓ 从 MVP 复制 CSS 变量和样式，透传进 shadow DOM |
| web component | `<wc-{spec-name}>` 四态（idle/open/loading/done/error）均在 shadow DOM 内部 |
| 初态 | MVP 布局正常 + component 触发按钮可见 + 弹窗 hidden |
| 第一步边界态 | 用户点击触发按钮 → `_setPhase('open')` → 配置面板出现 |
| 结束态 | 成功 → `#ph-done` 显示 → 点「完成」回到 idle |
| 错误态 | 失败 → `#ph-error` 显示 → 点「重试」或「取消」回到 idle |
| 弹窗关闭 | 「取消」/「×」/遮罩点击 均回到 idle ✓ |
| 目标文件 | prototypes/{spec-name}.html |

### 写盘前校验清单（详见 ref/validation.md §11.A–§11.H）

#### §11.A MVP 全量复制
- [ ] HTML 包含完整的 MVP 全部区块（wb-root 内所有子元素均保留）
- [ ] 非关联区域没有任何 `hidden` / `display:none` / `visibility:hidden`

#### §11.B MVP CSS 复用
- [ ] 背景色使用 `var(--wb-bg-panel)` / `var(--wb-bg-base)` 或一致颜色
- [ ] 文字色使用 `var(--wb-text)` / `var(--wb-text-sec)` 或一致颜色
- [ ] accent 色使用 `var(--accent-orange)` / `var(--wb-accent)` / `var(--accent-green)` 等

#### §11.C web component 替换
- [ ] spec 关联区域已替换为 `<wc-{spec-name}>` 元素
- [ ] component 在 `<head>` 的 `<script>` 中用 `customElements.define` 注册
- [ ] component 内用 `attachShadow({mode:'open'})` 创建 shadow DOM
- [ ] 所有四态 UI（`#ph-open` `#ph-loading` `#ph-done` `#ph-error`）均在 shadow DOM 内部

#### §11.D 初态检查（component idle）
- [ ] 触发按钮（shadow DOM 内）在 DOM 中可见，无 `hidden` / `disabled`
- [ ] `#overlay`（shadow DOM 内）初始有 `class="hidden"`
- [ ] 所有 `.phase-*` panel（shadow DOM 内）初始有 `class="hidden"`
- [ ] `_phase` 私有属性初始为 `'idle'`

#### §11.E component 状态机 _syncUI() 逐行核对
- [ ] `this._phase === 'open'`: `#overlay` remove `hidden`；`#ph-open` remove `hidden`；其他 panels add `hidden`
- [ ] `this._phase === 'loading'`: `#overlay` remove `hidden`；`#ph-loading` remove `hidden`；其他 panels add `hidden`
- [ ] `this._phase === 'done'`: `#overlay` remove `hidden`；`#ph-done` remove `hidden`；其他 panels add `hidden` ← **done 必须 remove('hidden')**
- [ ] `this._phase === 'error'`: `#overlay` remove `hidden`；`#ph-error` remove `hidden`；其他 panels add `hidden` ← **error 必须 remove('hidden')**
- [ ] `this._phase === 'idle'`: `#overlay` add `hidden`；无任何 phase panel 显示

#### §11.F 弹窗关闭路径（component 内）
- [ ] shadow DOM 内「取消」按钮 → `_setPhase('idle')`
- [ ] shadow DOM 内「×」按钮（或 overlay 自身） → `_setPhase('idle')`
- [ ] overlay 背景点击（`e.target === overlay`） → `_setPhase('idle')`
- [ ] 「完成」按钮（仅 `#ph-done` 内） → `_setPhase('idle')`
- [ ] 「重试」按钮（仅 `#ph-error` 内） → `_setPhase('open')` 或 `_setPhase('loading')`

#### §11.G 事件绑定完整性
- [ ] 触发按钮（shadow DOM 内） → `_setPhase('open')`
- [ ] 所有关闭按钮（shadow DOM 内） → `_setPhase('idle')`
- [ ] 无任何外部 `<script>` 操作 component 内部 DOM

#### §11.H CSS 变量继承（shadow DOM）
- [ ] shadow DOM 样式中使用了 `var(--xxx)` 引用 MVP CSS 变量
- [ ] 或 `connectedCallback()` 中有 `_injectCssVars()` 将变量显式注入

---

[完整 HTML 代码块]

请审阅，确认后我将写盘并更新 spec。
```
