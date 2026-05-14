---
name: spec-prototype-fragment
description: |
  根据 L3/L4/L5 spec，从关联原型中提取或构建派生原型片段。
  **核心原则：永远从工作区顶层 L1 spec 的 prototype.file 读取完整顶层原型，在其标注区域内用 web component 替换扩展交互。**
  禁止从零构建任何不在 MVP 中的 UI 区域（布局、颜色、字体、间距必须与 MVP 一致）。
  两种路径均需用 web components 实现交互行为和四态（初/边/终/错）。
  **区块识别须与用户确认，不确定时直接询问。**
  **原型必须从 MVP 的初态开始，用户点击触发第一步边界态，弹窗可关闭回到初态。**
  **非此 spec 关联的区域必须正常显示，不得隐藏或移除。**
  结果写入 prototypes/{spec-name}.html，更新 spec.prototype.file。
  触发词：生成原型片段、反推原型、spec 原型片段、原型提取。
---

# spec-prototype-fragment（Spec 原型片段提取器）

**规范路径**：与本目录 `agent.json` 成对放置于 `.vibex/agents/skills/spec-prototype-fragment/`。

**关联文件**：
- `ref/wc-template.md` — Web Component 模板（§4 完整代码 + CSS 透传）
- `ref/path-b.md` — 路径 B 详细说明
- `ref/validation.md` — 校验表（§11）+ 输出规范模板（§9）
- `ref/example.md` — 完整生成示例（§10）
- `scripts/onboard.js` — 可复制的状态机脚手架

---

## 0. 核心原则

### 0.1 顶层原型是完整基准，不得裁剪

- **直接复制完整的顶层原型 HTML**，包括全部 CSS 变量、字体、布局
- spec 对应的 UI 区域通过 `data-vibex-title` 和 `data-vibex-annot` 标注定位
- **非关联区域不得添加 `hidden` / `display:none`** — 其他区块必须与顶层原型初态完全一致
- 不允许在原型中发明任何顶层原型中不存在的 UI 元素
- 如果顶层原型中没有对应区域，**先询问用户**，而不是从零构建

### 0.2 不确定就问用户

出现以下情况时，**必须**先询问用户，不得自行决定：
- 找到多个可能的区块候选
- spec 没有明确的 `data-vibex-title` 可匹配
- spec 在 MVP 中没有明显的视觉对应区域
- 需要新建 UI 元素

### 0.3 四态完整要求

每个派生原型必须包含：**初态、边界态（交互中）、结束态（成功）、错误态（失败）**。

- **原型必须从 MVP 的初态开始**：展示 MVP 中该区域的原始静态 UI，用户点击后触发第一步边界态。禁止直接在第一步就展示弹窗/表单/进度等中间态。
- **弹窗/模态必须能关闭**：用户可以点「取消」或「×」关闭弹窗，最终原型回到初态。禁止任何不可退出的交互路径。

---

## 1. 前置条件

### 1.1 前置检查

1. `read_file` 读取 spec 内容，确认 spec.level 不是 `L1-goal`
2. **读取工作区顶层 L1 spec**（`.vibex/specs/L1-goal/` 下 level 含 `1` 的 spec）
3. 从顶层 L1 spec 的 `prototype.file` 字段获取顶层原型路径
4. **如果顶层 L1 spec 没有 `prototype.file` 或文件不存在**，立即告知用户：
   > 「工作区顶层 L1 spec 缺少 prototype.file 字段或原型文件不存在。
   > 请先完善顶层原型后重试。」
5. **读取顶层原型 HTML**，作为完整基准（复制全部 HTML，不做裁剪）
6. 在顶层原型中搜索 `data-vibex-title` 或 class/id 匹配，定位 spec 关联区域

---

## 2. 区块识别（强制确认步骤）

### 2.1 搜索方式（优先级递减）

1. `data-vibex-title` 匹配 `display.title`
2. `data-vibex-annot` 匹配 spec level（L3 → `L3`，L4 → `L4`，L5 → `L5`）
3. class/id 模糊匹配（如 spec name 含 `statusbar` → `.statusbar`）
4. `structure.parent_selector` → 精确 CSS 选择器

### 2.2 询问格式

```
### 需要确认：区块定位

**spec**：`{spec.name}`（{level}）
**spec.title**：`{display.title}`
**源原型**：`{顶层 L1 spec prototype.file}`（顶层原型是完整基准）

在**顶层原型**中，我找到了以下可能对应的区块：

| # | 区块描述 | 选择器 | 理由 | 置信度 |
|---|---------|--------|------|--------|
| 1 | … | … | … | 高/中/低 |

**我的判断**：倾向于选择 **#N**，因为……

请确认：
- 回复「1」或「2」选择区块
- 或告诉我正确的选择器
- 或告诉我该 spec 没有对应的 UI 区域（我将基于相似区域扩展）
```

用户确认区块后，记录选定的 selector，进入扩展步骤。

---

## 3. 路径 A：复制完整 MVP + 关联区域替换为 web component 原型

**这是默认路径。只要 MVP 中有对应区域，就必须用此路径。**

### 3.1 核心概念：占位替换而非叠加

MVP 中 spec 关联的标注区域（如 `.init-specs-btn`、`.statusbar`）是**静态占位**。
派生原型中，该占位元素应被**完整替换**为一个自包含的 `<wc-{spec-name}>` 自定义元素，
其内部通过 shadow DOM 承载全部四态 UI 和状态机逻辑。

**这是替换，不是叠加。** 外层 MVP 的其他所有区域保持原样不动。

### 3.2 操作步骤

1. **复制完整的顶层原型 HTML**（包含全部区块，不做任何裁剪）
2. 找到 spec 关联的标注占位元素
3. **将占位元素整块替换为 `<wc-{spec-name}>` web component**
4. web component 内部包含：
   - shadow DOM 样式（继承 MVP CSS 变量，详见 `ref/wc-template.md` §4.5）
   - 四态 UI 结构（`.phase-*` panels，均含 `hidden` class 初态）
   - 状态机逻辑（`_setPhase()` + `_syncUI()`）
5. **所有非关联区域保持 MVP 初态**
6. 保留顶层原型全部 CSS 变量

### 3.3 替换示例对比

**❌ 错误方式（在原位用普通 `<script>` 打补丁）**：

```html
<!-- MVP 中的 .init-specs-btn 原位保留，只是绑了个 click -->
<button class="init-specs-btn">初始化 specs</button>
<script>
  // 弹窗在 body 末尾，状态机在外部
  document.querySelector('.init-specs-btn').addEventListener('click', openModal);
</script>
```

**✅ 正确方式（替换为 web component）**：

```html
<!-- MVP 中 .init-specs-btn 占位被替换为 web component -->
<wc-scaffolding
  class="init-specs-btn"
  data-vibex-annot="L5"
  data-vibex-title="L5 · 初始化 Specs">
</wc-scaffolding>
<!-- 四态 panels + 状态机全部在 component 内部（shadow DOM） -->
```

完整模板代码见 `ref/wc-template.md` §4.1。

### 3.4 标注保留

- **保留**关联区域的 `data-vibex-annot` / `data-vibex-title` → 移到 `<wc-*>` host 元素上
- **保留**关联区域内嵌套的子 L4/L5 标注 → 移入 shadow DOM 内部
- **非关联区域的标注不得被移除或遮挡**

---

## 4. 路径 B：基于 MVP 相似区域扩展（谨慎使用）

**仅当 MVP 中完全没有对应区域时使用。必须先询问用户。**

详细说明见 `ref/path-b.md`。

---

## 5. 四态构建规则（从初态出发 + component 内自包含）

四态 UI（初态/边界态/成功态/错误态）必须完整实现在 `<wc-{spec-name}>` web component 的 shadow DOM 内部。
web component 的 `_syncUI()` 方法是唯一的 UI 同步入口，外部 MVP 不操作 component 内部 DOM。

### 5.1 从初态出发

原型打开时：MVP 布局正常渲染 + component 处于 `idle` phase（触发按钮可见，无弹窗）。
用户点击触发按钮 → `_setPhase('open')` → 配置面板出现。

### 5.2 弹窗闭环

| 关闭触发 | 目标 phase | 说明 |
|---------|-----------|------|
| 「取消」按钮 | `idle` | 所有非 idle 状态均有 |
| 「×」按钮 | `idle` | 遮罩层内 |
| 遮罩背景点击 | `idle` | 所有非 idle 状态均有 |
| 「完成」按钮 | `idle` | 仅 `done` 状态 |
| 「重试」按钮 | `open` 或 `loading` | 仅 `error` 状态 |

### 5.3 四态推导优先级

1. `content.ui_spec.states` — spec 中定义的状态列表
2. `content.ui_spec.acceptance` — 成功/失败路径
3. `io_contract.behavior` — 行为序列
4. `io.input` / `io.output` — 输入输出

---

## 6. 写盘与回写

### 6.1 目标路径

`prototypes/{spec-name}.html`

### 6.2 回写到 `prototype.file`

**必须更新 `prototype.file` 字段，不是 `spec.prototype_ref`。**

```yaml
prototype:
  file: ".vibex/prototypes/{spec-name}.html"
  validates: []
  status: derived
```

### 6.3 HTML 文件结构

每个派生原型 HTML 必须严格遵循以下三层结构：

```html
<!doctype html>
<html lang="zh-CN">
<head>
  <!-- ① MVP CSS 变量区：直接从顶层原型复制，全部保留 -->
  <style>/* 完整顶层原型 CSS（不做裁剪） */</style>

  <!-- ② Web Component 定义：放在 <head> 末尾（详见 ref/wc-template.md） -->
  <script>
    class Wc{SpecName} extends HTMLElement {
      constructor() {
        super();
        this._phase = 'idle';
        this.attachShadow({ mode: 'open' });
        // shadow DOM 内容：变量继承 + 四态 panels + 样式
      }
      connectedCallback() { /* 事件绑定（全部在 shadow DOM 内部） */ }
      _injectCssVars() { /* CSS 变量继承（方式 A/B 见 ref/wc-template.md §4.5） */ }
      _setPhase(p) { this._phase = p; this._syncUI(); }
      _syncUI() {
        const ov = this.shadowRoot.getElementById('overlay');
        ov.classList.toggle('hidden', this._phase === 'idle');
        ['ph-open','ph-loading','ph-done','ph-error'].forEach(id =>
          this.shadowRoot.getElementById(id).classList.toggle('hidden',
            !id.endsWith(this._phase)));
      }
    }
    customElements.define('wc-{spec-name}', Wc{SpecName});
  </script>
</head>
<body>
  <!-- ③ MVP 布局区：完整复制，不做裁剪 -->
  <div class="workbench-root">
    <!-- 所有 MVP 区块全部保留 -->

    <!-- ④ spec 关联区域：被 component 替换（见 ref/wc-template.md §4.4） -->
    <div class="trail">
      <wc-{spec-name}
        data-vibex-annot="L3"
        data-vibex-title="L3 · {spec.title}（扩展）">
      </wc-{spec-name}>
    </div>

    <!-- 其他 MVP 区块保持原样 -->
  </div>

  <!-- ⑤ 标注控制栏（VibeX 标注系统） -->
  <div id="vibex-annot-ctrl">...</div>
  <script>/* 标注切换脚本（独立于 component）*/</script>
</body>
</html>
```

---

## 7. 输出规范

每次任务分两个阶段输出。详细模板见 `ref/validation.md`（§9）。

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

完整校验清单见 `ref/validation.md`（§11.A–§11.H），核心项：

- §11.A MVP 全量复制
- §11.B MVP CSS 复用
- §11.C web component 替换（`<wc-{spec-name}>` + `customElements.define` + `attachShadow`）
- §11.D 初态（component idle）
- §11.E `_syncUI()` 逐状态核对（done/error 不得忘记 `remove('hidden')`）
- §11.F 弹窗关闭路径（所有非 idle 状态均可回到 `idle`）
- §11.G 事件绑定（全部在 shadow DOM 内部，无外部 script 操作 component）
- §11.H CSS 变量继承（shadow DOM 内使用 `var(--xxx)`）

---

## 8. 示例

完整生成示例（从输入到写盘）见 `ref/example.md`。

---

## 9. 校验表速查

详细校验表（含逐行核对、常见错误）见 `ref/validation.md`。

### 快速核对：_syncUI() 状态表

| `this._phase` | overlay | `#ph-open` | `#ph-loading` | `#ph-done` | `#ph-error` |
|--------------|---------|-----------|-------------|-----------|-----------|
| `idle` | hidden | hidden | hidden | hidden | hidden |
| `open` | **可见** | **可见** | hidden | hidden | hidden |
| `loading` | **可见** | hidden | **可见** | hidden | hidden |
| `done` | **可见** | hidden | hidden | **可见** | hidden |
| `error` | **可见** | hidden | hidden | hidden | **可见** |

---

## 10. 禁止事项

### 通用禁止（所有原型）

- **禁止在区块未确认前生成 HTML**
- **禁止多个候选时自行选择（必须询问用户）**
- **禁止原型缺少四态之一（idle/open/loading/done/error）**
- **禁止更新 `spec.prototype_ref`（应更新 `prototype.file`）**
- **禁止覆盖已存在的派生原型**
- **禁止在初态就展示弹窗/表单/进度等中间态**（必须从 MVP 静态触发按钮开始）
- **禁止任何不可退出的交互路径**（弹窗必须能关闭回到 idle）
- **禁止从零构建任何不在 MVP 中的 UI 元素**（颜色、字体、布局、组件都必须从 MVP 复用）
- **禁止对非关联区域添加 `hidden` / `display:none`** — 其他区块必须正常显示
- **禁止复制 MVP 后裁剪或移除任何区块** — 必须保留完整布局

### web component 专属禁止

- **禁止在 shadow DOM 外部用 `querySelector` / `getElementById` 操作 component 内部 DOM**
- **禁止在 shadow DOM 外部用普通 `<script>` 定义 component 状态机的核心逻辑**
- **禁止四态 panel 初始不带 `hidden` class**
- **禁止 `done` 或 `error` phase 时忘记 `remove('hidden')`**（只 add 不 remove）
- **禁止 component 内直接使用绝对颜色值**（必须用 `var(--xxx)` 或显式注入变量）
- **禁止 `phase` 值在 `_syncUI()` 中没有对应处理分支**
- **禁止 component 的 `_phase` 属性未初始化为 `'idle'`**
