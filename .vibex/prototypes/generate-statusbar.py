#!/usr/bin/env python3
"""将 MVP 原型复制并替换 statusbar 区域为 web component"""
import re

# 读取 MVP 原型
with open('.vibex/prototypes/vibex-workbench-mvp.html', 'r', encoding='utf-8') as f:
    mvp_html = f.read()

# 修复 1: 将 --statusbar-h 提升到 :root 级别（让 shadow DOM 能继承）
# 在 :root 的 color-scheme 后面添加 height 变量
mvp_html = mvp_html.replace(
    '      --wb-splitter: #0f1117;',
    '      --wb-splitter: #0f1117;\n      --statusbar-h: 24px;'
)

# web component 定义
wc_script = '''  <!-- Web Component 定义 -->
  <script>
    class WcBottomStatusbar extends HTMLElement {
      constructor() {
        super();
        this._phase = 'idle';
        this._canvasVisible = true;
        this.attachShadow({ mode: 'open' });
        this.shadowRoot.innerHTML = `
          <style>
            :host {
              display: flex;
              align-items: center;
              height: var(--statusbar-h, 24px);
              padding: 0 10px;
              background: #10131a;
              border-top: 1px solid var(--wb-border, rgba(255,255,255,0.07));
              flex-shrink: 0;
              font-size: 12px;
              color: var(--wb-text-sec, #8a8a8e);
              user-select: none;
            }
            .statusbar {
              display: flex;
              align-items: center;
              justify-content: space-between;
              width: 100%;
              height: 100%;
            }
            .sb-left, .sb-right {
              display: flex;
              align-items: center;
              gap: 6px;
              min-width: 0;
            }
            .sb-right { margin-left: auto; }
            .sb-sep { color: #465064; }
            .sb-path {
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
              color: #eef0f5;
            }
            .sb-badge { font-weight: 600; }
            /* 状态徽标 */
            .status-badge {
              display: inline-flex;
              align-items: center;
              gap: 4px;
              padding: 1px 6px;
              border-radius: 3px;
              font-size: 11px;
              font-weight: 500;
            }
            .status-badge.idle {
              background: rgba(135, 207, 138, 0.12);
              color: #87cf8a;
              border: 1px solid rgba(135, 207, 138, 0.25);
            }
            .status-badge.running {
              background: rgba(135, 171, 255, 0.12);
              color: #87abff;
              border: 1px solid rgba(135, 171, 255, 0.25);
            }
            .status-badge.error {
              background: rgba(225, 109, 117, 0.12);
              color: #e16d75;
              border: 1px solid rgba(225, 109, 117, 0.25);
            }
            .status-badge.building {
              background: rgba(239, 198, 107, 0.12);
              color: #efc66b;
              border: 1px solid rgba(239, 198, 107, 0.25);
            }
            .status-badge .dot {
              width: 6px;
              height: 6px;
              border-radius: 50%;
              background: currentColor;
            }
            .status-badge.running .dot {
              animation: pulse 1s infinite;
            }
            @keyframes pulse {
              0%, 100% { opacity: 1; }
              50% { opacity: 0.4; }
            }
            /* Canvas 开关按钮 */
            .canvas-toggle {
              border: 1px solid #303746;
              background: #11141b;
              color: #a3abb9;
              border-radius: 6px;
              padding: 2px 8px;
              font-size: 11px;
              cursor: pointer;
              display: inline-flex;
              align-items: center;
              gap: 4px;
              position: relative;
            }
            .canvas-toggle:hover {
              background: var(--wb-bg-panel-2, #1c202a);
              color: var(--wb-text, #eef0f5);
            }
            /* 下拉菜单 */
            .dropdown {
              position: absolute;
              bottom: calc(100% + 4px);
              right: 0;
              min-width: 160px;
              background: #1c1c1e;
              border: 1px solid var(--wb-border, rgba(255,255,255,0.1));
              border-radius: 6px;
              box-shadow: 0 8px 24px rgba(0,0,0,0.4);
              overflow: hidden;
              z-index: 1000;
            }
            .dropdown.hidden { display: none; }
            .dropdown-item {
              display: flex;
              align-items: center;
              gap: 8px;
              padding: 8px 12px;
              cursor: pointer;
              font-size: 12px;
              color: var(--wb-text, #e8e8ed);
            }
            .dropdown-item:hover {
              background: rgba(255,255,255,0.05);
            }
            .dropdown-item .check {
              width: 14px;
              color: var(--wb-accent, #72d6d0);
              visibility: visible;
            }
            .dropdown-item .check.hidden {
              visibility: hidden;
            }
            .dropdown-item .label { flex: 1; }
          </style>
          <div class="statusbar">
            <!-- 左：状态徽标 + 路径 -->
            <div class="sb-left">
              <span class="status-badge idle" id="status-badge">
                <span class="dot"></span>
                <span class="label">Idle</span>
              </span>
              <span class="sb-sep">|</span>
              <span class="sb-path" title="C:\\project\\vibex-workbench">…/vibex-workbench</span>
            </div>
            <!-- 右：Canvas 开关 + 后端状态 -->
            <div class="sb-right">
              <button class="canvas-toggle" id="canvas-btn">
                <span>Canvas Auto UI</span>
                <span class="canvas-state">On</span>
                <svg width="8" height="8" viewBox="0 0 8 8" fill="currentColor" style="margin-left:2px">
                  <path d="M2 3l2 2 2-2"/>
                </svg>
              </button>
              <!-- 下拉菜单 -->
              <div class="dropdown hidden" id="canvas-dropdown">
                <div class="dropdown-item" id="toggle-canvas">
                  <span class="check" id="canvas-check">✓</span>
                  <span class="label" id="canvas-label">隐藏 Canvas</span>
                </div>
                <div class="dropdown-item" id="fullscreen-canvas">
                  <span class="check hidden">✓</span>
                  <span class="label">全屏模式</span>
                </div>
              </div>
              <span class="sb-sep">|</span>
              <span class="sb-badge" id="backend-badge" style="color:var(--accent-green)">后端就绪</span>
              <span class="sb-sep">|</span>
              <span style="color:#6f7888">VibeX Workbench</span>
            </div>
          </div>
        `;
      }

      connectedCallback() {
        this._bindEvents();
      }

      _bindEvents() {
        const btn = this.shadowRoot.getElementById('canvas-btn');
        const dropdown = this.shadowRoot.getElementById('canvas-dropdown');
        const self = this;

        btn.addEventListener('click', (e) => {
          e.stopPropagation();
          const isHidden = dropdown.classList.contains('hidden');
          document.querySelectorAll('wc-bottom-statusbar').forEach(wc => {
            if (wc !== self) wc._closeDropdown();
          });
          dropdown.classList.toggle('hidden', !isHidden);
        });

        dropdown.querySelectorAll('.dropdown-item').forEach(item => {
          item.addEventListener('click', () => {
            const id = item.id;
            if (id === 'toggle-canvas') {
              const check = self.shadowRoot.getElementById('canvas-check');
              const label = self.shadowRoot.getElementById('canvas-label');
              const isVisible = check.style.visibility !== 'hidden';
              check.style.visibility = isVisible ? 'hidden' : 'visible';
              label.textContent = isVisible ? '显示 Canvas' : '隐藏 Canvas';
              self.shadowRoot.querySelector('.canvas-state').textContent = isVisible ? 'Off' : 'On';
            }
            self._closeDropdown();
          });
        });

        document.addEventListener('click', () => this._closeDropdown());
      }

      _closeDropdown() {
        const dropdown = this.shadowRoot.getElementById('canvas-dropdown');
        if (dropdown) dropdown.classList.add('hidden');
      }

      setState(state) {
        const states = ['idle', 'running', 'error', 'building'];
        if (!states.includes(state)) return;
        this._phase = state;
        this._syncStateUI();
      }

      _syncStateUI() {
        const badge = this.shadowRoot.getElementById('status-badge');
        const labels = { idle: 'Idle', running: 'Running', error: 'Error', building: 'Building' };
        badge.className = 'status-badge ' + this._phase;
        badge.querySelector('.label').textContent = labels[this._phase];
      }
    }
    customElements.define('wc-bottom-statusbar', WcBottomStatusbar);
  </script>
'''

# 查找 </style> 标签位置，在其后插入 web component 定义
style_close_pos = mvp_html.find('</style>')
if style_close_pos == -1:
    print("Error: </style> not found")
    exit(1)

# 在 </style> 后插入 web component
new_html = mvp_html[:style_close_pos + len('</style>')] + '\n' + wc_script + mvp_html[style_close_pos + len('</style>'):]

# 替换 statusbar div 为 web component
statusbar_pattern = r'<div class="statusbar"[^>]*>'
statusbar_match = re.search(statusbar_pattern, new_html)
if statusbar_match:
    wc_element = '<wc-bottom-statusbar data-vibex-annot="L3" data-vibex-title="L3 · 底部状态栏（派生）"></wc-bottom-statusbar>'
    new_html = new_html[:statusbar_match.start()] + wc_element + new_html[statusbar_match.end():]
    print("Replaced .statusbar with web component")
else:
    print("Warning: .statusbar not found")

# 更新标题
new_html = new_html.replace(
    '<title>VibeX Workbench — L1 MVP 静态原型（源码对齐）</title>',
    '<title>原型 · L3 · 底部状态栏</title>'
)

# 写入派生原型
output_path = '.vibex/prototypes/MOD-bottom-statusbar.html'
with open(output_path, 'w', encoding='utf-8') as f:
    f.write(new_html)

print(f"Generated: {output_path}")
print(f"Size: {len(new_html)} bytes")
