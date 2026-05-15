#!/usr/bin/env python3
"""
spec-prototype-fragment · wc-gen.py
────────────────────────────────────────────────────────────────────────────
参数化派生原型生成脚本。

使用方式：
    python wc-gen.py --spec-name MOD-build-panel --selector ".build-group" \
                     --output ../prototypes/MOD-build-panel.html

原理：
    1. 复制完整 MVP HTML（不做裁剪）
    2. 在 </style> 后注入 WC 定义脚本
    3. 替换指定 selector 区域为 <wc-{spec-name}> 元素
    4. 写入目标路径

约束（硬编码，非 markdown 描述）：
    ✓ MVP 全量复制，不裁剪任何区块
    ✓ 所有样式用 var(--xxx) 引用 MVP CSS 变量
    ✓ 四态（idle/open/loading/done/error）在 shadow DOM 内部
    ✓ 所有 phase panel 初始含 class="hidden"
    ✓ customElements.define 在 <head> 中注册
────────────────────────────────────────────────────────────────────────────
"""

import argparse
import os
import sys


MVP_PATH = ".vibex/prototypes/vibex-workbench-mvp.html"
CSS_VARS = {
    "--wb-bg-panel":    "#0f1117",
    "--wb-bg-panel-2": "#151822",
    "--wb-text":        "#c0caf5",
    "--wb-text-sec":    "#787c99",
    "--wb-border":      "rgba(255,255,255,0.07)",
    "--accent-green":    "#87cf8a",
    "--accent-red":      "#e16d75",
    "--accent-orange":   "#f09a6a",
    "--accent-yellow":   "#efc66b",
    "--wb-accent":       "#72d6d0",
}


def css_vars_declaration() -> str:
    """生成 :root CSS 变量声明块，供注入到 shadow DOM 的 <style> 中"""
    lines = [":root {"]
    for name, val in CSS_VARS.items():
        lines.append(f"  {name}: {val};")
    lines.append("}")
    return "\n".join(lines)


def wc_script(spec_name: str, trigger_label: str = "") -> str:
    """
    生成完整的 web component 定义脚本。
    四态：idle（初态） → open（配置） → loading（执行） → done（成功）
                                            └→ error（失败）
    """
    if not trigger_label:
        trigger_label = spec_name.replace("-", " ").replace("_", " ").title()

    # 所有 phase panel 初始含 class="hidden" — 这是约束
    return f"""
<script>
// ─────────────────────────────────────────────────────────────
//  wc-{spec_name} — 自动生成 by wc-gen.py
//  四态：idle → open → loading → done | error
// ─────────────────────────────────────────────────────────────
class Wc{safe_class_name(spec_name)} extends HTMLElement {{
  constructor() {{
    super();
    this._phase = 'idle';
    this.attachShadow({{ mode: 'open' }});
    this.shadowRoot.innerHTML = `
      <style>
        /* CSS 变量继承 — 引用 MVP CSS 变量 */
        :root {{
{css_vars_block()}}}
        :host {{
          display: contents;
        }}
        .overlay {{
          position: fixed;
          inset: 0;
          background: rgba(0,0,0,.55);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 9999;
        }}
        .overlay.hidden {{ display: none; }}
        .modal {{
          background: var(--wb-bg-panel, #0f1117);
          border: 1px solid var(--wb-border, rgba(255,255,255,0.07));
          border-radius: 12px;
          min-width: 480px;
          max-width: 640px;
          max-height: 80vh;
          display: flex;
          flex-direction: column;
          box-shadow: 0 24px 64px rgba(0,0,0,.6);
        }}
        .ph-header {{
          padding: 16px 20px 12px;
          font-size: 14px;
          font-weight: 600;
          color: var(--wb-text, #c0caf5);
          border-bottom: 1px solid var(--wb-border, rgba(255,255,255,0.07));
          display: flex;
          align-items: center;
          justify-content: space-between;
        }}
        .ph-header .close-btn {{
          background: none;
          border: none;
          color: var(--wb-text-sec, #787c99);
          cursor: pointer;
          font-size: 18px;
          line-height: 1;
          padding: 0;
        }}
        .ph-header .close-btn:hover {{
          color: var(--wb-text, #c0caf5);
        }}
        .ph-body {{
          padding: 16px 20px;
          flex: 1;
          overflow-y: auto;
          font-size: 13px;
          color: var(--wb-text-sec, #787c99);
        }}
        .ph-footer {{
          padding: 12px 20px;
          border-top: 1px solid var(--wb-border, rgba(255,255,255,0.07));
          display: flex;
          gap: 8px;
          justify-content: flex-end;
        }}
        /* 触发按钮 */
        .trigger-btn {{
          background: var(--accent-orange, #f09a6a);
          color: #0d1117;
          border: none;
          border-radius: 6px;
          padding: 5px 14px;
          font-size: 12px;
          font-weight: 600;
          cursor: pointer;
        }}
        .trigger-btn:hover {{
          opacity: 0.85;
        }}
        /* 所有 phase panel 初始 hidden — 约束 */
        .phase {{
          display: none;
        }}
        .phase.hidden {{
          display: none;
        }}
        .phase.active {{
          display: block;
        }}
        /* 通用按钮 */
        .btn {{
          padding: 6px 16px;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          border: 1px solid var(--wb-border, rgba(255,255,255,0.1));
          background: transparent;
          color: var(--wb-text, #c0caf5);
        }}
        .btn:hover {{
          background: rgba(255,255,255,0.06);
        }}
        .btn-primary {{
          background: var(--accent-orange, #f09a6a);
          color: #0d1117;
          border-color: transparent;
          font-weight: 600;
        }}
        .btn-primary:hover {{
          opacity: 0.85;
        }}
        /* 状态色 */
        .txt-success {{ color: var(--accent-green, #87cf8a); }}
        .txt-error   {{ color: var(--accent-red, #e16d75); }}
        .txt-running {{ color: var(--accent-yellow, #efc66b); }}
        /* 进度条 */
        .progress-bar-wrap {{
          height: 4px;
          background: rgba(255,255,255,0.08);
          border-radius: 2px;
          overflow: hidden;
          margin: 12px 0;
        }}
        .progress-bar {{
          height: 100%;
          background: var(--wb-accent, #72d6d0);
          border-radius: 2px;
          transition: width 0.3s ease;
        }}
        /* 步骤列表 */
        .step-list {{
          list-style: none;
          padding: 0;
          margin: 0;
          font-size: 12px;
          font-family: var(--font-mono, monospace);
        }}
        .step-list li {{
          padding: 3px 0;
          display: flex;
          align-items: center;
          gap: 6px;
        }}
        .step-list .done  {{ color: var(--accent-green, #87cf8a); }}
        .step-list .running {{ color: var(--wb-accent, #72d6d0); }}
        .step-list .pending {{ color: var(--wb-text-sec, #787c99); }}
        .step-list .error   {{ color: var(--accent-red, #e16d75); }}
        .step-dot {{
          width: 6px;
          height: 6px;
          border-radius: 50%;
          background: currentColor;
          flex-shrink: 0;
        }}
        .step-list .running .step-dot {{
          animation: pulse 1s infinite;
        }}
        @keyframes pulse {{
          0%, 100% {{ opacity: 1; }}
          50%       {{ opacity: 0.35; }}
        }}
      </style>

      <!-- 触发按钮（idle 态可见） -->
      <button class="trigger-btn" id="trigger">{trigger_label}</button>

      <!-- 弹窗遮罩（idle 态 hidden） -->
      <div class="overlay hidden" id="overlay">
        <div class="modal">
          <!-- ph-open：配置/选择面板 -->
          <div class="phase" id="ph-open">
            <div class="ph-header">
              <span>{trigger_label}</span>
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
          <div class="phase" id="ph-loading">
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
          <div class="phase" id="ph-done">
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
          <div class="phase" id="ph-error">
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
  }}

  connectedCallback() {{
    this._bindEvents();
  }}

  _bindEvents() {{
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
    this.shadowRoot.getElementById('overlay').addEventListener('click', (e) => {{
      if (e.target.id === 'overlay') this._setPhase('idle');
    }});
  }}

  _setPhase(phase) {{
    this._phase = phase;
    this._syncUI();
  }}

  _syncUI() {{
    const overlay = this.shadowRoot.getElementById('overlay');
    const phases = ['open', 'loading', 'done', 'error'];
    const p = this._phase;

    // idle：隐藏遮罩；否则显示遮罩
    overlay.classList.toggle('hidden', p === 'idle');

    // 所有 phase panel 初始 hidden → 约束
    phases.forEach(name => {{
      const el = this.shadowRoot.getElementById('ph-' + name);
      if (!el) return;
      if (name === p) el.classList.add('active');
      else el.classList.remove('active');
    }});
  }}

  _run() {{
    this._setPhase('loading');
    this._simulateSteps();
  }}

  _simulateSteps() {{
    // 子类可重写 _renderStep / _onDone / _onError
    const steps = [
      {{ label: '准备环境…',         cls: 'done'    }},
      {{ label: '读取配置…',         cls: 'running' }},
      {{ label: '执行操作…',         cls: 'pending' }},
      {{ label: '验证结果…',         cls: 'pending' }},
      {{ label: '完成',              cls: 'pending' }},
    ];
    const list = this.shadowRoot.getElementById('step-list');
    const bar  = this.shadowRoot.getElementById('progress-bar');

    const draw = () => {{
      if (list) {{
        list.innerHTML = steps.map(s =>
          `<li class="${{s.cls}}"><span class="step-dot"></span>${{s.label}}</li>`
        ).join('');
      }}
      if (bar) {{
        const pct = Math.round((steps.filter(s => s.cls === 'done').length / steps.length) * 100);
        bar.style.width = pct + '%';
      }}
    }};

    draw();
    let i = 0;
    const tick = () => {{
      if (i < steps.length) {{
        if (steps[i].cls === 'running') steps[i].cls = 'done';
        if (i + 1 < steps.length && steps[i + 1].cls === 'pending') {{
          steps[i + 1].cls = 'running';
        }}
        i++;
        draw();
        if (i < steps.length - 1) {{
          setTimeout(tick, 600);
        }} else {{
          setTimeout(() => this._onDone(), 400);
        }}
      }}
    }};
    setTimeout(tick, 400);
  }}

  _onDone() {{
    const body = this.shadowRoot.getElementById('body-done');
    if (body) body.textContent = '操作已完成。';
    this._setPhase('done');
  }}

  _onError(msg) {{
    const body = this.shadowRoot.getElementById('body-error');
    if (body) body.textContent = msg || '发生错误，请检查后重试。';
    this._setPhase('error');
  }}
}}

customElements.define('wc-{spec_name}', Wc{safe_class_name(spec_name)});
</script>
"""


def css_vars_block() -> str:
    """生成 shadow DOM 内联 CSS 变量声明"""
    indent = "        "
    lines = [indent + ":root {"]
    for name, val in CSS_VARS.items():
        lines.append(f"{indent}  {name}: {val};")
    lines.append(indent + "}")
    return "\n".join(lines)


def safe_class_name(spec_name: str) -> str:
    """将 spec 名称转换为合法的 JS 类名"""
    parts = spec_name.replace("-", "_").replace(".", "_").split("_")
    return "".join(p.title() for p in parts)


def generate(spec_name: str, selector: str, output_path: str,
             trigger_label: str = "") -> None:
    """
    核心生成函数。
    1. 读取 MVP HTML
    2. 在 </style> 后注入 WC 脚本
    3. 替换 selector 区域为 <wc-*> 元素
    4. 写入输出文件
    """
    # 读取 MVP
    mvp_file = MVP_PATH
    if not os.path.exists(mvp_file):
        # 尝试相对于脚本目录
        script_dir = os.path.dirname(os.path.abspath(__file__))
        mvp_file = os.path.join(script_dir, "..", "..", "..", MVP_PATH)

    if not os.path.exists(mvp_file):
        print(f"Error: MVP file not found: {MVP_PATH}", file=sys.stderr)
        sys.exit(1)

    with open(mvp_file, "r", encoding="utf-8") as f:
        html = f.read()

    # 注入 WC 脚本
    style_end = html.find("</style>")
    if style_end == -1:
        print("Error: </style> not found in MVP HTML", file=sys.stderr)
        sys.exit(1)

    script = wc_script(spec_name, trigger_label)
    html = html[:style_end + len("</style>")] + "\n" + script + html[style_end + len("</style>"):]

    # 替换 selector 区域
    import re
    # 找到 selector 匹配的元素开始位置
    # 用正则匹配 <div class="xxx" ...> 形式
    escaped_sel = re.escape(selector)
    # 简单处理：找到包含该 selector class 的 div 开始标签
    pattern = re.compile(
        r'(<\w+\s[^>]*class="[^"]*' + escaped_sel.replace(r'\.', r'[."]?') + r'[^"]*"[^>]*>)',
        re.IGNORECASE
    )
    # 简化：直接找 class 属性包含 selector 名
    # selector 形如 ".build-group" 或 ".statusbar"
    sel_name = selector.lstrip(".")
    # 匹配 <... class="...{sel_name}..." ...>
    pattern = re.compile(
        r'(<\w+(?:\s[^>]*)?\sclass="[^"]*' + re.escape(sel_name) + r'[^"]*"\s[^>]*>)',
        re.IGNORECASE
    )
    match = pattern.search(html)

    if match:
        wc_tag = (
            f'<wc-{spec_name} '
            f'data-vibex-annot="L3" '
            f'data-vibex-title="L3 · {spec_name}（派生）"'
            f'></wc-{spec_name}>'
        )
        # 替换整个开始标签到下一个 > 之间的内容（简单处理：替换开始标签）
        # 更安全的做法：找到元素开始和结束位置
        html = _replace_element(html, match.start(), selector, wc_tag)
        print(f"Replaced selector '{selector}' with <wc-{spec_name}>")
    else:
        # 未找到，尝试追加到 body 末尾
        body_end = html.rfind("</body>")
        if body_end == -1:
            print(f"Warning: selector '{selector}' not found and </body> not found", file=sys.stderr)
        else:
            wc_tag = (
                f'\n<wc-{spec_name} '
                f'data-vibex-annot="L3" '
                f'data-vibex-title="L3 · {spec_name}（派生）"'
                f'></wc-{spec_name}>\n'
            )
            html = html[:body_end] + wc_tag + html[body_end:]
            print(f"Selector '{selector}' not found, appended to body")

    # 更新标题
    html = html.replace(
        "<title>VibeX Workbench",
        "<title>原型 · L3 · " + spec_name + " — VibeX Workbench"
    )

    # 确保目录存在
    out_dir = os.path.dirname(output_path)
    if out_dir and not os.path.exists(out_dir):
        os.makedirs(out_dir)

    with open(output_path, "w", encoding="utf-8") as f:
        f.write(html)

    size = len(html)
    print(f"Generated: {output_path} ({size} bytes)")


def _replace_element(html: str, match_start: int, selector: str, wc_tag: str) -> str:
    """将匹配元素替换为 WC 标签"""
    import re
    sel_name = selector.lstrip(".")

    # 找到元素的开始和结束标签
    # 先找到开始标签的 >
    gt = html.find(">", match_start)
    if gt == -1:
        return html

    start_tag_end = gt + 1

    # 找对应的结束标签（简化：找 </div> 或类似闭合标签）
    # 构建开标签名
    tag_match = re.match(r'<(\w+)', html[match_start:])
    if not tag_match:
        return html
    tag_name = tag_match.group(1)

    # 计算嵌套深度找结束标签
    depth = 1
    pos = start_tag_end
    while pos < len(html) and depth > 0:
        if html[pos:pos+2] == "</":
            end_pos = html.find(">", pos)
            if end_pos == -1:
                break
            end_tag = html[pos+2:end_pos].strip().split()[0]
            if end_tag == tag_name:
                depth -= 1
                if depth == 0:
                    break
            pos = end_pos + 1
        elif html[pos:pos+1] == "<":
            # 检查是否是自闭合标签或开始标签
            space = html.find(" ", pos)
            gt2 = html.find(">", pos)
            if gt2 == -1:
                break
            if space != -1 and space < gt2:
                next_tag = html[pos+1:space]
            else:
                next_tag = html[pos+1:gt2]
            if next_tag and not next_tag.startswith("!"):
                depth += 1
            pos = gt2 + 1
        else:
            pos += 1

    if depth == 0:
        # 替换从 match_start 到 pos+len(f"</{tag_name}>") 的内容
        end_len = len(f"</{tag_name}>")
        return html[:match_start] + wc_tag + html[pos + end_len:]

    # 回退：只替换开始标签
    return html[:match_start] + wc_tag + html[start_tag_end:]


def main():
    parser = argparse.ArgumentParser(
        description="spec-prototype-fragment · 参数化派生原型生成"
    )
    parser.add_argument("--spec-name", required=True,
                        help="spec 名称（将用作 wc-* 标签名）")
    parser.add_argument("--selector", required=True,
                        help="MVP 中要替换的区域选择器（如 .build-group）")
    parser.add_argument("--output", required=True,
                        help="输出路径（如 ../prototypes/MOD-xxx.html）")
    parser.add_argument("--trigger-label", default="",
                        help="触发按钮文字（默认从 spec-name 生成）")
    args = parser.parse_args()

    generate(args.spec_name, args.selector, args.output, args.trigger_label)


if __name__ == "__main__":
    main()
