// ─────────────────────────────────────────────────────────────────
//  wc-{{SPEC_NAME}} — 由 wc-gen.py 自动生成
//  四态：idle → open → loading → done | error
// ─────────────────────────────────────────────────────────────────
class Wc{{SAFE_CLASS_NAME}} extends HTMLElement {
  constructor() {
    super();
    this._phase = 'idle';
    this.attachShadow({ mode: 'open' });
    this.shadowRoot.innerHTML = `
      <style>
        /* CSS 变量继承 — 引用 MVP CSS 变量 */
        :root {
{{CSS_VARS}}
        }
        :host {
          display: contents;
        }
        .overlay {
          position: fixed;
          inset: 0;
          background: rgba(0,0,0,.55);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 9999;
        }
        .overlay.hidden { display: none; }
        .modal {
          background: var(--wb-bg-panel, #0f1117);
          border: 1px solid var(--wb-border, rgba(255,255,255,0.07));
          border-radius: 12px;
          min-width: 480px;
          max-width: 640px;
          max-height: 80vh;
          display: flex;
          flex-direction: column;
          box-shadow: 0 24px 64px rgba(0,0,0,.6);
        }
        .ph-header {
          padding: 16px 20px 12px;
          font-size: 14px;
          font-weight: 600;
          color: var(--wb-text, #c0caf5);
          border-bottom: 1px solid var(--wb-border, rgba(255,255,255,0.07));
          display: flex;
          align-items: center;
          justify-content: space-between;
        }
        .ph-header .close-btn {
          background: none;
          border: none;
          color: var(--wb-text-sec, #787c99);
          cursor: pointer;
          font-size: 18px;
          line-height: 1;
          padding: 0;
        }
        .ph-header .close-btn:hover {
          color: var(--wb-text, #c0caf5);
        }
        .ph-body {
          padding: 16px 20px;
          flex: 1;
          overflow-y: auto;
          font-size: 13px;
          color: var(--wb-text-sec, #787c99);
        }
        .ph-footer {
          padding: 12px 20px;
          border-top: 1px solid var(--wb-border, rgba(255,255,255,0.07));
          display: flex;
          gap: 8px;
          justify-content: flex-end;
        }
        .trigger-btn {
          background: var(--accent-orange, #f09a6a);
          color: #0d1117;
          border: none;
          border-radius: 6px;
          padding: 5px 14px;
          font-size: 12px;
          font-weight: 600;
          cursor: pointer;
        }
        .trigger-btn:hover { opacity: 0.85; }
        /* 所有 phase panel 初始 hidden — 约束 */
        .phase { display: none; }
        .phase.hidden { display: none; }
        .phase.active { display: block; }
        .btn {
          padding: 6px 16px;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          border: 1px solid var(--wb-border, rgba(255,255,255,0.1));
          background: transparent;
          color: var(--wb-text, #c0caf5);
        }
        .btn:hover { background: rgba(255,255,255,0.06); }
        .btn-primary {
          background: var(--accent-orange, #f09a6a);
          color: #0d1117;
          border-color: transparent;
          font-weight: 600;
        }
        .btn-primary:hover { opacity: 0.85; }
        .txt-success { color: var(--accent-green, #87cf8a); }
        .txt-error   { color: var(--accent-red, #e16d75); }
        .txt-running { color: var(--accent-yellow, #efc66b); }
        .progress-bar-wrap {
          height: 4px;
          background: rgba(255,255,255,0.08);
          border-radius: 2px;
          overflow: hidden;
          margin: 12px 0;
        }
        .progress-bar {
          height: 100%;
          background: var(--wb-accent, #72d6d0);
          border-radius: 2px;
          transition: width 0.3s ease;
        }
        .step-list {
          list-style: none;
          padding: 0;
          margin: 0;
          font-size: 12px;
          font-family: var(--font-mono, monospace);
        }
        .step-list li {
          padding: 3px 0;
          display: flex;
          align-items: center;
          gap: 6px;
        }
        .step-list .done    { color: var(--accent-green, #87cf8a); }
        .step-list .running { color: var(--wb-accent, #72d6d0); }
        .step-list .pending { color: var(--wb-text-sec, #787c99); }
        .step-list .error   { color: var(--accent-red, #e16d75); }
        .step-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          background: currentColor;
          flex-shrink: 0;
        }
        .step-list .running .step-dot { animation: pulse 1s infinite; }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50%       { opacity: 0.35; }
        }
      </style>

      <!-- 触发按钮（idle 态可见） -->
      <button class="trigger-btn" id="trigger">{{TRIGGER_LABEL}}</button>

      <!-- 弹窗遮罩（idle 态 hidden） -->
      <div class="overlay hidden" id="overlay">
        <div class="modal">
          <!-- ph-open：配置/选择面板 -->
          <div class="phase hidden" id="ph-open">
            <div class="ph-header">
              <span>{{TRIGGER_LABEL}}</span>
              <button class="close-btn" id="btn-x-open">×</button>
            </div>
            <div class="ph-body" id="body-open">
              <!-- 子类重写此区域内容 -->
            </div>
            <div class="ph-footer">
              <button class="btn" id="btn-cancel-open">取消</button>
              <button class="btn btn-primary" id="btn-start">开始</button>
            </div>
          </div>

          <!-- ph-loading：执行进度 -->
          <div class="phase hidden" id="ph-loading">
            <div class="ph-header">
              <span class="txt-running">执行中…</span>
              <button class="close-btn" id="btn-x-loading">×</button>
            </div>
            <div class="ph-body">
              <div class="progress-bar-wrap">
                <div class="progress-bar" id="progress-bar" style="width:0%"></div>
              </div>
              <ul class="step-list" id="step-list"></ul>
            </div>
            <div class="ph-footer">
              <button class="btn" id="btn-cancel-loading">取消</button>
            </div>
          </div>

          <!-- ph-done：成功完成 -->
          <div class="phase hidden" id="ph-done">
            <div class="ph-header">
              <span class="txt-success">✓ 完成</span>
              <button class="close-btn" id="btn-x-done">×</button>
            </div>
            <div class="ph-body" id="body-done">
              操作已完成。
            </div>
            <div class="ph-footer">
              <button class="btn btn-primary" id="btn-close-done">完成</button>
            </div>
          </div>

          <!-- ph-error：执行失败 -->
          <div class="phase hidden" id="ph-error">
            <div class="ph-header">
              <span class="txt-error">✗ 失败</span>
              <button class="close-btn" id="btn-x-error">×</button>
            </div>
            <div class="ph-body" id="body-error">
              发生错误，请检查后重试。
            </div>
            <div class="ph-footer">
              <button class="btn" id="btn-cancel-error">关闭</button>
              <button class="btn btn-primary" id="btn-retry">重试</button>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  connectedCallback() {
    this._bindEvents();
  }

  _bindEvents() {
    // 触发按钮：idle → open
    this.shadowRoot.getElementById('trigger').addEventListener('click', () => this._setPhase('open'));

    // 关闭到 idle
    const closeIdle = () => this._setPhase('idle');
    this.shadowRoot.getElementById('btn-cancel-open').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-cancel-loading').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-cancel-error').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-x-open').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-x-loading').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-x-done').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-x-error').addEventListener('click', closeIdle);
    this.shadowRoot.getElementById('btn-close-done').addEventListener('click', closeIdle);

    // 开始执行：open → loading
    this.shadowRoot.getElementById('btn-start').addEventListener('click', () => this._run());

    // 重试：error → loading
    this.shadowRoot.getElementById('btn-retry').addEventListener('click', () => this._run());

    // 遮罩点击 → idle
    this.shadowRoot.getElementById('overlay').addEventListener('click', (e) => {
      if (e.target.id === 'overlay') this._setPhase('idle');
    });
  }

  _setPhase(phase) {
    this._phase = phase;
    this._syncUI();
  }

  _syncUI() {
    const overlay = this.shadowRoot.getElementById('overlay');
    const phases = ['open', 'loading', 'done', 'error'];
    const p = this._phase;

    // idle：隐藏遮罩；否则显示遮罩
    overlay.classList.toggle('hidden', p === 'idle');

    // 所有 phase panel 初始 hidden → 约束
    phases.forEach(name => {
      const el = this.shadowRoot.getElementById('ph-' + name);
      if (!el) return;
      if (name === p) el.classList.add('active');
      else el.classList.remove('active');
    });
  }

  _run() {
    this._setPhase('loading');
    this._simulateSteps();
  }

  _simulateSteps() {
    const steps = [
      { label: '准备环境…',         cls: 'done'    },
      { label: '读取配置…',         cls: 'running' },
      { label: '执行操作…',         cls: 'pending' },
      { label: '验证结果…',         cls: 'pending' },
      { label: '完成',              cls: 'pending' },
    ];
    const list = this.shadowRoot.getElementById('step-list');
    const bar  = this.shadowRoot.getElementById('progress-bar');

    const draw = () => {
      if (list) {
        list.innerHTML = steps.map(s =>
          `<li class="${s.cls}"><span class="step-dot"></span>${s.label}</li>`
        ).join('');
      }
      if (bar) {
        const pct = Math.round((steps.filter(s => s.cls === 'done').length / steps.length) * 100);
        bar.style.width = pct + '%';
      }
    };

    draw();
    let i = 0;
    const tick = () => {
      if (i < steps.length) {
        if (steps[i].cls === 'running') steps[i].cls = 'done';
        if (i + 1 < steps.length && steps[i + 1].cls === 'pending') {
          steps[i + 1].cls = 'running';
        }
        i++;
        draw();
        if (i < steps.length - 1) {
          setTimeout(tick, 600);
        } else {
          setTimeout(() => this._onDone(), 400);
        }
      }
    };
    setTimeout(tick, 400);
  }

  _onDone() {
    const body = this.shadowRoot.getElementById('body-done');
    if (body) body.textContent = '操作已完成。';
    this._setPhase('done');
  }

  _onError(msg) {
    const body = this.shadowRoot.getElementById('body-error');
    if (body) body.textContent = msg || '发生错误，请检查后重试。';
    this._setPhase('error');
  }
}

customElements.define('wc-{{SPEC_NAME}}', Wc{{SAFE_CLASS_NAME}});
