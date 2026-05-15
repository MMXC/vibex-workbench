# WC 模板参考

> 快速参考：`wc-gen.py` 自动生成完整 WC，参考此文件理解结构。

## 四态结构

```html
<!-- 触发按钮（idle 可见） -->
<button class="trigger-btn" id="trigger">触发</button>

<!-- 弹窗遮罩（idle hidden） -->
<div class="overlay hidden" id="overlay">
  <div class="modal">
    <div class="phase" id="ph-open">配置面板</div>
    <div class="phase" id="ph-loading">执行进度</div>
    <div class="phase" id="ph-done">成功完成</div>
    <div class="phase" id="ph-error">失败提示</div>
  </div>
</div>
```

## CSS 变量（必须引用 MVP 变量）

```css
:host { display: contents; }
.overlay { background: rgba(0,0,0,.55); }
.modal {
  background: var(--wb-bg-panel, #0f1117);
  border: 1px solid var(--wb-border, rgba(255,255,255,.07));
  border-radius: 12px;
}
.btn-primary { background: var(--accent-orange, #f09a6a); }
.txt-success { color: var(--accent-green, #87cf8a); }
.txt-error   { color: var(--accent-red, #e16d75); }
```

## 状态机核心

```js
_setPhase(p) { this._phase = p; this._syncUI(); }

_syncUI() {
  const overlay = this.shadowRoot.getElementById('overlay');
  overlay.classList.toggle('hidden', this._phase === 'idle');
  ['open','loading','done','error'].forEach(name => {
    const el = this.shadowRoot.getElementById('ph-' + name);
    if (!el) return;
    el.classList.toggle('active', name === this._phase);
  });
}
```

## 事件绑定（全部在 shadow DOM 内）

```js
connectedCallback() {
  this.shadowRoot.getElementById('trigger')
    .addEventListener('click', () => this._setPhase('open'));
  // 关闭按钮
  this.shadowRoot.getElementById('btn-cancel')
    .addEventListener('click', () => this._setPhase('idle'));
  // 遮罩点击
  this.shadowRoot.getElementById('overlay')
    .addEventListener('click', e => {
      if (e.target.id === 'overlay') this._setPhase('idle');
    });
}
```

## 子类重写点

| 方法 | 说明 |
|------|------|
| `_run()` | 开始执行（默认模拟步骤进度） |
| `_onDone()` | 执行成功回调 |
| `_onError(msg)` | 执行失败回调 |
