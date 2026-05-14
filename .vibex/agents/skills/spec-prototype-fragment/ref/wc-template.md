# Web Component 原型标准模板

> 关联：§4 — 每个 spec 派生原型必须实现为 `<wc-{spec-name}>` 自定义元素。

## 4.1 模板结构

```js
class WcScaffolding extends HTMLElement {
  constructor() {
    super();
    this._phase = 'idle'; // idle | open | preview | loading | done | error
    this.attachShadow({ mode: 'open' });
    this.shadowRoot.innerHTML = `
      <style>/* CSS 变量继承 + 四态样式 */</style>
      <!-- 触发按钮（初态可见） -->
      <button class="trigger-btn" id="trigger">
        ${this.getAttribute('trigger-label') || '初始化 specs'}
      </button>
      <!-- 弹窗遮罩（hidden 初态） -->
      <div class="overlay hidden" id="overlay">
        <div class="modal">
          <!-- ph-open：配置面板 -->
          <div class="phase ph-open" id="ph-open">
            <div class="ph-header">配置脚手架</div>
            <div class="ph-body"><!-- 模块选项 --></div>
            <div class="ph-footer">
              <button class="btn-cancel" id="btn-cancel">取消</button>
              <button class="btn-primary" id="btn-start">开始生成</button>
            </div>
          </div>
          <!-- ph-loading：进度视图 -->
          <div class="phase ph-loading hidden" id="ph-loading">
            <div class="ph-header">生成中...</div>
            <div class="ph-body"><!-- 步骤进度条 --></div>
            <div class="ph-footer">
              <button class="btn-cancel" id="btn-cancel-loading">取消</button>
            </div>
          </div>
          <!-- ph-done：完成视图 -->
          <div class="phase ph-done hidden" id="ph-done">
            <div class="ph-header">✓ 生成完成</div>
            <div class="ph-body"><!-- 文件列表 --></div>
            <div class="ph-footer">
              <button class="btn-primary" id="btn-close-done">完成</button>
            </div>
          </div>
          <!-- ph-error：错误视图 -->
          <div class="phase ph-error hidden" id="ph-error">
            <div class="ph-header">✗ 生成失败</div>
            <div class="ph-body"><!-- 错误信息 --></div>
            <div class="ph-footer">
              <button class="btn-cancel" id="btn-cancel-error">关闭</button>
              <button class="btn-retry" id="btn-retry">重试</button>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  connectedCallback() {
    this._injectCssVars();
    this.shadowRoot.getElementById('trigger').addEventListener('click', () => this._setPhase('open'));
    this.shadowRoot.getElementById('btn-cancel').addEventListener('click', () => this._setPhase('idle'));
    this.shadowRoot.getElementById('btn-cancel-loading').addEventListener('click', () => this._setPhase('idle'));
    this.shadowRoot.getElementById('btn-cancel-error').addEventListener('click', () => this._setPhase('idle'));
    this.shadowRoot.getElementById('btn-close-done').addEventListener('click', () => this._setPhase('idle'));
    this.shadowRoot.getElementById('btn-start').addEventListener('click', () => this._startGeneration());
    this.shadowRoot.getElementById('btn-retry').addEventListener('click', () => this._startGeneration(true));
    this.shadowRoot.getElementById('overlay').addEventListener('click', (e) => {
      if (e.target.id === 'overlay') this._setPhase('idle');
    });
  }

  _injectCssVars() {
    const style = this.shadowRoot.querySelector('style');
    style.textContent = `:host { all: initial; display: contents; } ` + style.textContent;
  }

  _setPhase(phase) {
    this._phase = phase;
    this._syncUI();
  }

  _syncUI() {
    const overlay = this.shadowRoot.getElementById('overlay');
    const panels = ['ph-open', 'ph-loading', 'ph-done', 'ph-error'];
    const p = this._phase;
    overlay.classList.toggle('hidden', p === 'idle');
    panels.forEach(id => {
      this.shadowRoot.getElementById(id).classList.toggle('hidden', id !== `ph-${p}`);
    });
  }

  _startGeneration(isRetry = false) {
    this._setPhase('loading');
    // 模拟异步生成流程...
  }
}
customElements.define('wc-scaffolding', WcScaffolding);
```

## 4.2 必须实现的回调（子类重写）

| 方法 | 何时调用 | 必须做什么 |
|------|---------|-----------|
| `_syncOpen()` | `phase === 'open'` | 显示配置/选择 UI |
| `_syncLoading()` | `phase === 'loading'` | 启动异步操作，显示进度 |
| `_syncDone()` | `phase === 'done'` | 显示成功结果，更新触发按钮状态 |
| `_syncError(msg)` | `phase === 'error'` | 显示错误信息 |

## 4.3 状态 → panel 映射表

| phase 值 | 显示的 panel id | 隐藏的 panel ids |
|---------|----------------|-----------------|
| `idle` | 无（遮罩 hidden） | 全部 |
| `open` | `#ph-open` | `#ph-loading` `#ph-done` `#ph-error` |
| `loading` | `#ph-loading` | `#ph-open` `#ph-done` `#ph-error` |
| `done` | `#ph-done` | `#ph-open` `#ph-loading` `#ph-error` |
| `error` | `#ph-error` | `#ph-open` `#ph-loading` `#ph-done` |

## 4.4 MVP 中使用方式

```html
<!-- 在 MVP 布局中，spec 关联区域用 component 替换 -->
<div class="trail">
  <wc-scaffolding
    trigger-label="初始化 specs"
    class="init-specs-btn"
    data-vibex-annot="L5"
    data-vibex-title="L5 · 初始化 Specs"
    data-vibex-parent-l3="titlebar">
  </wc-scaffolding>
  <span class="run-pill">make validate ✓</span>
</div>
```

## 4.5 CSS 变量透传（两种方式）

### 方式 A（推荐）：shadow DOM 内声明 `var(--xxx)`

shadow DOM 内可直接引用 document 级别的 CSS 自定义属性：

```js
const style = document.createElement('style');
style.textContent = `
  :host { all: initial; display: contents; }
  .overlay { color: var(--wb-text, #e8e8ed); }
  .modal { background: var(--wb-bg-panel, #131314); border: 1px solid var(--wb-border, rgba(255,255,255,.07)); border-radius: 12px; }
  .btn-primary { background: var(--accent-orange, #f09a6a); color: #0d1117; }
  .phase-done { color: var(--accent-green, #87cf8a); }
  .phase-error { color: var(--accent-red, #e16d75); }
`;
```

### 方式 B：显式将 CSS 变量注入到 component 根元素

```js
connectedCallback() {
  const vars = [
    '--wb-bg-panel','--wb-text','--wb-text-sec',
    '--accent-orange','--wb-accent','--accent-green','--accent-red',
    '--font-ui','--font-mono','--wb-border'
  ];
  vars.forEach(v => {
    const val = getComputedStyle(this).getPropertyValue(v).trim();
    if (val) this.style.setProperty(v, val);
  });
}
```

## 4.6 component 专属禁止

- **禁止**在 shadow DOM 外部用 `querySelector` / `getElementById` 操作 component 内部 DOM
- **禁止**在 shadow DOM 外部用普通 `<script>` 定义 component 状态机的核心逻辑
- **禁止**四态 panel 初始不带 `hidden` class
- **禁止** `done` 或 `error` phase 时忘记 `remove('hidden')`（只 add 不 remove）
- **禁止** component 内直接使用绝对颜色值（必须用 `var(--xxx)` 或显式注入变量）
- **禁止** `phase` 值在 `_syncUI()` 中没有对应处理分支
- **禁止** component 的 `_phase` 属性未初始化为 `'idle'`
