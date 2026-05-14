# 示例：MOD-scaffolding 完整生成过程

> 关联：§10 — 展示从输入到写盘的完整流程。

## 输入

`spec_path = .vibex/specs/L3-module/MOD-scaffolding.yaml`

## Agent 分析

1. 读取顶层 L1 spec，确认 `prototype.file` = `.vibex/prototypes/vibex-workbench-mvp.html`
2. 读取顶层原型 HTML，在 `.trail` 区块内找到 `.init-specs-btn`（`data-vibex-title="L5 · 初始化 Specs"`）
3. 确认为 `.trail` 区块（L3 titlebar 上下文）

## 区块识别确认请求

```
### 区块识别确认

**spec**：`MOD-scaffolding`（L3-module）
**spec.title**：`scaffolding`
**源原型**：`.vibex/prototypes/vibex-workbench-mvp.html`

在**顶层原型**中找到：

| # | 区块描述 | selector | 置信度 | 理由 |
|---|---------|---------|--------|------|
| 1 | 初始化 Specs 触发按钮 | `.init-specs-btn` | 高 | data-vibex-title="L5 · 初始化 Specs" 精确匹配 |

请确认：回复「1」
```

## 用户确认：「1」

## Agent 构建

1. **复制完整 MVP HTML**（不做裁剪）
2. **替换占位**：`.init-specs-btn` → `<wc-scaffolding>`
3. **实现 web component**（四态 + 状态机全部在 shadow DOM 内部，详见 ref/wc-template.md）
4. **回写 spec prototype.file**

### 生成的 MVP 布局片段

```html
<!-- MVP 布局中，.init-specs-btn 被替换为 component -->
<div class="trail">
  <wc-scaffolding
    class="init-specs-btn"
    data-vibex-annot="L5"
    data-vibex-title="L5 · 初始化 Specs"
    data-vibex-parent-l3="titlebar">
  </wc-scaffolding>
  <span class="run-pill">make validate ✓</span>
</div>
```

### 注册的 Web Component

```js
// <head> 中注册 web component
class WcScaffolding extends HTMLElement {
  constructor() {
    super();
    this._phase = 'idle';
    this.attachShadow({ mode: 'open' });
    this.shadowRoot.innerHTML = `
      <style>/* 变量继承 + 四态样式 */</style>
      <button class="trigger-btn">初始化 specs</button>
      <div class="overlay hidden">
        <div class="modal">
          <div class="phase ph-open" id="ph-open">...</div>
          <div class="phase ph-loading hidden" id="ph-loading">...</div>
          <div class="phase ph-done hidden" id="ph-done">...</div>
          <div class="phase ph-error hidden" id="ph-error">...</div>
        </div>
      </div>
    `;
  }
  connectedCallback() {
    this.shadowRoot.querySelector('.trigger-btn').onclick = () => this._setPhase('open');
    this.shadowRoot.querySelectorAll('.btn-cancel').forEach(b => b.onclick = () => this._setPhase('idle'));
    this.shadowRoot.querySelector('.overlay').onclick = e => {
      if (e.target.classList.contains('overlay')) this._setPhase('idle');
    };
    this.shadowRoot.querySelector('#btn-start').onclick = () => this._startGeneration();
    this._injectCssVars();
  }
  _injectCssVars() {
    const s = this.shadowRoot.querySelector('style');
    s.textContent = `:host { all: initial; display: contents; } ` + s.textContent;
  }
  _setPhase(p) { this._phase = p; this._syncUI(); }
  _syncUI() {
    const ov = this.shadowRoot.querySelector('.overlay');
    ov.classList.toggle('hidden', this._phase === 'idle');
    ['ph-open','ph-loading','ph-done','ph-error'].forEach(id => {
      this.shadowRoot.getElementById(id).classList.toggle('hidden', !id.endsWith(this._phase));
    });
  }
  _startGeneration() {
    this._setPhase('loading');
    // 模拟异步...
  }
}
customElements.define('wc-scaffolding', WcScaffolding);
```

## 回写的 Spec YAML

```yaml
prototype:
  file: ".vibex/prototypes/MOD-scaffolding.html"
  validates: []
  status: derived
```
