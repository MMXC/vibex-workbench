"""HTML Generator — converts spec YAML to interactive HTML prototype.

Key improvements over v1:
- MF components generate real <iframe> elements (not chip labels)
- Injects window.__dc bridge pointing to DataCenter HTTP API
- Injects window.__ec for EventCenter pub/sub
- Hot reload on spec file change (via fetch polling)
- Shows spec metadata (name, description, level, status)
- DC data is live-updated via /api/dc/list
"""
import os
import yaml
import json


def parse_spec(spec_path: str) -> dict:
    """Parse spec YAML with optional YAML frontmatter."""
    with open(spec_path) as f:
        raw = f.read()
    if raw.startswith("---"):
        parts = raw.split("---", 2)
        meta = yaml.safe_load(parts[1]) or {} if len(parts) >= 2 else {}
        content = parts[2].strip() if len(parts) >= 3 else ""
    else:
        try:
            meta = yaml.safe_load(raw) or {}
        except Exception:
            meta = {}
        content = ""
    return {"meta": meta, "content": content, "path": spec_path}


def _esc(s):
    """Escape a string for embedding in JS double-quoted strings."""
    return str(s).replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")


def spec_to_html(spec: dict) -> str:
    """Convert parsed spec dict to HTML prototype."""
    meta = spec.get("meta", {})
    name = meta.get("name", "Prototype")
    spec_type = meta.get("type", "ui")
    level = meta.get("level", "")
    status = meta.get("status", "")
    description = meta.get("description", "")
    tags = meta.get("tags", [])
    components = meta.get("mf_components", [])
    theme = meta.get("ui_theme", {})

    primary = theme.get("primary", "#7c3aed")
    bg = theme.get("bg", "#0f0f1a")
    text = theme.get("text", "#e0e0f0")
    card_bg = theme.get("card_bg", "#1a1a2e")
    border = theme.get("border", "#2a2a4a")

    # Python-side port vars (used in footer HTML outside JS block)
    DC_PORT = os.environ.get("SPECPILOT_DC_PORT", "7890")
    MF_PORT = os.environ.get("SPECPILOT_MF_PORT", "5177")

    # ── Spec tags ──────────────────────────────────────────────────────────────
    tag_html = ""
    if tags:
        tag_html = '<div class="spec-tags">' + "".join(
            f'<span class="tag">{_esc(t)}</span>' for t in tags
        ) + "</div>"

    # ── MF Component iframes ─────────────────────────────────────────────────
    # Each mf_component generates:
    #   1. A section in #mf-slots grid
    #   2. An iframe loading the MF URL (which renders the component)
    #   3. The iframe gets window.__specpilotDcBase injected before load
    mf_slots_html = ""
    mf_iframe_init = ""

    for i, comp in enumerate(components):
        comp_name = comp.get("component", f"Component{i+1}")
        mf_url = comp.get("mf_url", "")
        dc_key = comp.get("dc_key", "")
        description_c = comp.get("description", "")

        slot_id = f"mf-slot-{i}"
        # Use preview server /components/<name> route — lets preview server inject DC base
        comp_url = f"/components/{comp_name}?__key={dc_key}"
        mf_slots_html += f"""
      <div class="mf-slot">
        <div class="mf-slot-header">
          <span class="mf-slot-name">{_esc(comp_name)}</span>
          <span class="mf-slot-key">{'key: ' + dc_key if dc_key else ''}</span>
        </div>
        <div class="mf-slot-body">
          <iframe
            id="{slot_id}"
            src="{comp_url}"
            class="mf-iframe"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
            title="{_esc(comp_name)}"
          ></iframe>
        </div>
        <div class="mf-slot-desc">{_esc(description_c)}</div>
      </div>"""
        mf_iframe_init += f"""
      injectDcBase(document.getElementById('{slot_id}'));"""

    if not mf_slots_html:
        mf_slots_html = '<div class="no-mf">无 MF 组件声明</div>'

    # ── DC Data display ───────────────────────────────────────────────────────
    # Render DC keys as KPI cards + tables
    dc_display = ""
    if components:
        # Show DC keys from mf_components in a collapsible DC panel
        dc_keys = [c.get("dc_key", "") for c in components if c.get("dc_key")]
        if dc_keys:
            dc_display = f"""
      <div class="dc-panel" id="dc-panel">
        <div class="dc-panel-header" onclick="toggleDcPanel()">
          <span>DataCenter</span>
          <span id="dc-toggle">▼</span>
        </div>
        <div class="dc-panel-body" id="dc-panel-body">
          <div id="dc-values">Loading...</div>
        </div>
      </div>"""

    # ── Tag level/status chips ────────────────────────────────────────────────
    meta_chips = ""
    if level:
        meta_chips += f'<span class="meta-chip level-chip">L{level}</span>'
    if status:
        cls = "status-pass" if status in ("done", "implemented") else \
              "status-wip" if status in ("wip", "partial") else "status-missing"
        meta_chips += f'<span class="meta-chip {cls}">{status}</span>'

    # ── JS: DC bridge + EC bridge + hot reload ────────────────────────────────
    js = f"""
  // ── Bridge Setup ─────────────────────────────────────────────────────────
  var DC_PORT = {{SPECPILOT_DC_PORT}};
  var MF_PORT = {{SPECPILOT_MF_PORT}};
  var DC_BASE = 'http://localhost:' + DC_PORT + '/api';
  var MF_BASE = 'http://localhost:' + MF_PORT;

  window.__dc = {{
    list: async function() {{
      try {{ var r = await fetch(DC_BASE + '/dc/list'); return r.ok ? await r.json() : {{}}; }} catch {{ return {{}}; }}
    }},
    get: async function(key) {{
      try {{ var r = await fetch(DC_BASE + '/dc/' + encodeURIComponent(key)); return r.ok ? await r.json() : null; }} catch {{ return null; }}
    }},
    set: async function(key, value) {{
      try {{ await fetch(DC_BASE + '/dc/' + encodeURIComponent(key), {{method:'PUT',headers:{{'Content-Type':'application/json'}},body:JSON.stringify({{value}})}}); }} catch {{}}
    }},
    keys: async function() {{
      try {{ var r = await fetch(DC_BASE + '/dc/list'); return r.ok ? Object.keys(await r.json()) : []; }} catch {{ return []; }}
    }}
  }};

  window.__ec = {{
    publish: async function(event, data) {{
      try {{ await fetch(DC_BASE + '/ec/publish', {{method:'POST',headers:{{'Content-Type':'application/json'}},body:JSON.stringify({{event, data}})}}); }} catch {{}}
    }}
  }};

  // ── Inject DC base into MF iframes ────────────────────────────────────────
  function injectDcBase(iframe) {{
    if (!iframe || !iframe.contentWindow) return;
    iframe.contentWindow.postMessage({{type:'__specpilot_init', dcBase: DC_BASE, mfBase: MF_BASE}}, '*');
  }}

  // ── MF Component iframes ─────────────────────────────────────────────────
  {mf_iframe_init}

  // ── Render DC values ─────────────────────────────────────────────────────
  async function renderDcValues() {{
    var el = document.getElementById('dc-values');
    if (!el) return;
    var keys = await window.__dc.keys();
    var html = '<div class="dc-grid">';
    for (var ki = 0; ki < keys.length; ki++) {{
      var k = keys[ki];
      var d = await window.__dc.get(k);
      var v = (d && d.value !== undefined) ? d.value : d;
      var vs = typeof v === 'object' ? JSON.stringify(v).slice(0,80) : String(v);
      html += '<div class="dc-kv"><span class="dc-k">' + k + '</span><input class="dc-v" data-key="' + k + '" value="' + vs + '" onblur="dcSet(this)"/></div>';
    }}
    html += '</div>';
    el.innerHTML = html;
  }}

  async function dcSet(input) {{
    var key = input.getAttribute('data-key');
    var val = input.value;
    try {{ var v = JSON.parse(val); }} catch {{ var v = val; }}
    await window.__dc.set(key, v);
  }}

  // ── Toggle DC panel ───────────────────────────────────────────────────────
  function toggleDcPanel() {{
    var b = document.getElementById('dc-panel-body');
    var t = document.getElementById('dc-toggle');
    var open = b.style.display !== 'none';
    b.style.display = open ? 'none' : 'block';
    t.textContent = open ? '▶' : '▼';
    if (!open) renderDcValues();
  }}

  // ── Hot reload: poll spec file mtime ────────────────────────────────────
  var SPEC_PATH = '{_esc(spec.get("path", ""))}';
  var SPEC_MTIME = null;
  async function checkHotReload() {{
    if (!SPEC_PATH) return;
    try {{
      var r = await fetch('/api/spec/mtime?path=' + encodeURIComponent(SPEC_PATH));
      if (r.ok) {{
        var d = await r.json();
        if (SPEC_MTIME === null) {{ SPEC_MTIME = d.mtime; return; }}
        if (d.mtime !== SPEC_MTIME) {{ SPEC_MTIME = d.mtime; location.reload(); }}
      }}
    }} catch {{}}
  }}
  setInterval(checkHotReload, 3000);

  // Initial load
  renderDcValues();
"""

    html = f"""<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{_esc(name)} — SpecPilot</title>
  <style>
    * {{ box-sizing: border-box; margin: 0; padding: 0; }}
    :root {{
      --primary: {primary};
      --bg: {bg};
      --text: {text};
      --card-bg: {card_bg};
      --border: {border};
    }}
    body {{
      font-family: system-ui, -apple-system, sans-serif;
      background: var(--bg);
      color: var(--text);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }}
    header {{
      padding: 14px 24px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      gap: 12px;
      background: {card_bg};
      flex-wrap: wrap;
    }}
    .logo {{ font-size: 18px; }}
    .spec-name {{ font-size: 15px; font-weight: 700; }}
    .spec-type {{
      font-size: 10px; padding: 2px 8px; border-radius: 10px;
      background: var(--primary); color: #fff; font-weight: 800; letter-spacing: 0.5px;
    }}
    .meta-chip {{
      font-size: 10px; padding: 2px 7px; border-radius: 6px;
      font-weight: 700; letter-spacing: 0.5px;
    }}
    .level-chip {{ background: rgba(114,214,208,0.2); color: #72d6d0; }}
    .status-pass {{ background: rgba(52,211,153,0.15); color: #34d399; }}
    .status-wip {{ background: rgba(251,191,36,0.15); color: #fbbf24; }}
    .status-missing {{ background: rgba(248,113,113,0.15); color: #f87171; }}
    .spec-tags {{ display: flex; gap: 5px; flex-wrap: wrap; }}
    .tag {{ font-size: 10px; padding: 2px 7px; border-radius: 6px; background: var(--border); color: #aaa; }}
    .spec-desc {{ font-size: 11px; color: #556; flex-basis: 100%; margin-top: 4px; }}

    /* Main: MF slots + DC panel side by side */
    main {{
      flex: 1;
      display: grid;
      grid-template-columns: 1fr 280px;
      gap: 0;
      overflow: hidden;
      height: calc(100vh - 56px - 36px);
    }}
    #mf-slots {{
      overflow-y: auto;
      padding: 20px;
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 16px;
      align-content: start;
    }}

    /* MF Slot */
    .mf-slot {{
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 10px;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }}
    .mf-slot-header {{
      padding: 10px 14px;
      border-bottom: 1px solid var(--border);
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: rgba(124,58,237,0.08);
    }}
    .mf-slot-name {{ font-size: 12px; font-weight: 700; color: #a78bfa; }}
    .mf-slot-key {{ font-size: 10px; color: #72d6d0; font-family: monospace; }}
    .mf-slot-body {{
      flex: 1;
      min-height: 160px;
      position: relative;
      background: #0a0b10;
    }}
    .mf-iframe {{
      width: 100%;
      height: 100%;
      min-height: 160px;
      border: none;
      position: absolute;
      top: 0; left: 0; right: 0; bottom: 0;
    }}
    .mf-slot-desc {{ font-size: 10px; color: #445; padding: 6px 14px; border-top: 1px solid var(--border); }}
    .no-mf {{ grid-column: 1/-1; text-align: center; color: #445; padding: 60px; font-size: 13px; }}

    /* DC Panel */
    .dc-panel {{
      border-left: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      overflow: hidden;
      background: {card_bg};
    }}
    .dc-panel-header {{
      padding: 10px 14px;
      border-bottom: 1px solid var(--border);
      display: flex;
      justify-content: space-between;
      font-size: 11px;
      font-weight: 700;
      color: #72d6d0;
      cursor: pointer;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      user-select: none;
    }}
    .dc-panel-body {{
      flex: 1;
      overflow-y: auto;
      padding: 10px;
    }}
    .dc-grid {{ display: flex; flex-direction: column; gap: 4px; }}
    .dc-kv {{ display: flex; flex-direction: column; gap: 2px; }}
    .dc-k {{ font-size: 10px; color: #72d6d0; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }}
    .dc-v {{ background: #0a0b10; border: 1px solid #1e2030; border-radius: 4px; color: #99a; font-size: 11px; font-family: monospace; padding: 3px 6px; width: 100%; }}
    .dc-v:focus {{ border-color: #7aa2ff; outline: none; }}

    footer {{
      height: 36px;
      display: flex;
      align-items: center;
      gap: 0;
      border-top: 1px solid var(--border);
      background: #10131a;
      padding: 0;
      font-size: 11px;
      overflow: hidden;
      flex-shrink: 0;
    }}
    .sb-item {{
      display: flex; align-items: center; gap: 5px;
      padding: 0 12px; height: 100%;
      border-right: 1px solid #1e2030;
      color: #556; white-space: nowrap;
    }}
    .sb-dot {{ width: 6px; height: 6px; border-radius: 50%; }}
    .sb-dot.green {{ background: #4ade80; }}
    .sb-dot.yellow {{ background: #fbbf24; }}
    .sb-dot.red {{ background: #f87171; }}
  </style>
</head>
<body>
  <header>
    <span class="logo">🚀</span>
    <span class="spec-name">{_esc(name)}</span>
    <span class="spec-type">{spec_type.upper()}</span>
    {meta_chips}
    {tag_html}
    {f'<div class="spec-desc">{_esc(description)}</div>' if description else ''}
  </header>

  <main>
    <div id="mf-slots">
{mf_slots_html}
    </div>
    <aside id="dc-sidebar">
      {dc_display}
    </aside>
  </main>

  <footer>
    <div class="sb-item"><span class="sb-dot green"></span>SpecPilot</div>
    <div class="sb-item">DC:{DC_PORT}</div>
    <div class="sb-item">MF:{MF_PORT}</div>
    <div class="sb-item">🔄 热刷新 3s</div>
  </footer>

  <script>
{js}
  </script>
</body>
</html>"""

    # Replace port placeholders with actual env values (0 = use defaults)
    html = html.replace("{SPECPILOT_DC_PORT}", os.environ.get("SPECPILOT_DC_PORT", "7890"))
    html = html.replace("{SPECPILOT_MF_PORT}", os.environ.get("SPECPILOT_MF_PORT", "5177"))

    return html


class Generator:
    def __init__(self, workspace: str = "."):
        self.workspace = workspace

    def generate(self, spec_path: str, output_path: str) -> str:
        spec = parse_spec(spec_path)
        html = spec_to_html(spec)
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(html)
        return output_path
