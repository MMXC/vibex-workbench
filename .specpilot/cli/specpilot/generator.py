"""HTML Generator — converts spec YAML to interactive HTML prototype."""
import os
import yaml


def parse_spec(spec_path: str) -> dict:
    with open(spec_path) as f:
        raw = f.read()
    if raw.startswith("---"):
        parts = raw.split("---", 2)
        meta = yaml.safe_load(parts[1]) or {} if len(parts) >= 2 else {}
        content = parts[2].strip() if len(parts) >= 3 else ""
    else:
        meta = yaml.safe_load(raw) or {}
        content = ""
    return {"meta": meta, "content": content}


def spec_to_html(spec: dict) -> str:
    meta = spec.get("meta", {})
    name = meta.get("name", "Prototype")
    spec_type = meta.get("type", "ui")
    theme = meta.get("ui_theme", {})
    components = meta.get("mf_components", [])

    primary = theme.get("primary", "#7c3aed")
    bg = theme.get("bg", "#0f0f1a")
    text = theme.get("text", "#e0e0f0")
    card_bg = theme.get("card_bg", "#1a1a2e")
    border = theme.get("border", "#2a2a4a")

    # Build MF chips
    mf_chips = ""
    if components:
        chips = "".join(
            '<span class="mf-chip">' + (c.get("component", str(c))) + '</span>'
            for c in components
        )
        mf_chips = (
            '<div class="mf-components">'
            '<h3>MF Components</h3>'
            + chips +
            '</div>'
        )

    # Build spec tags
    spec_tags = meta.get("tags", [])
    tag_html = ""
    if spec_tags:
        tag_html = '<div class="spec-tags">' + "".join(
            '<span class="tag">' + t + '</span>' for t in spec_tags
        ) + '</div>'

    html = f"""<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{name} — SpecPilot</title>
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
      padding: 16px 28px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      gap: 14px;
      background: {card_bg};
    }}
    .logo {{ font-size: 18px; }}
    .spec-name {{ font-size: 15px; font-weight: 600; }}
    .spec-type {{
      font-size: 10px;
      padding: 2px 7px;
      border-radius: 10px;
      background: var(--primary);
      color: #fff;
      font-weight: 700;
      letter-spacing: 0.5px;
    }}
    .spec-tags {{ display: flex; gap: 6px; }}
    .tag {{
      font-size: 10px;
      padding: 2px 7px;
      border-radius: 6px;
      background: var(--border);
      color: #aaa;
    }}
    main {{
      flex: 1;
      padding: 28px;
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
      gap: 16px;
      align-content: start;
    }}
    .card {{
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 18px;
      transition: border-color 0.2s;
    }}
    .card:hover {{ border-color: var(--primary); }}
    .card-title {{ font-size: 11px; color: #666; margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.5px; }}
    .card-value {{ font-size: 26px; font-weight: 700; color: var(--primary); }}
    table {{ width: 100%; border-collapse: collapse; font-size: 12px; margin-top: 8px; }}
    th {{ text-align: left; padding: 8px 10px; border-bottom: 1px solid var(--border); color: #888; font-weight: 600; }}
    td {{ padding: 9px 10px; border-bottom: 1px solid {border}22; }}
    tr:hover td {{ background: {border}18; }}
    .badge {{ display: inline-block; padding: 2px 7px; border-radius: 8px; font-size: 10px; font-weight: 700; }}
    .badge-active {{ background: #05966928; color: #34d399; }}
    .badge-away {{ background: #d9770628; color: #fbbf24; }}
    .badge-inactive {{ background: #374151; color: #9ca3af; }}
    .mf-components {{
      padding: 16px 28px;
      border-top: 1px solid var(--border);
      background: {card_bg};
    }}
    .mf-components h3 {{ font-size: 11px; color: #666; margin-bottom: 10px; text-transform: uppercase; letter-spacing: 0.5px; }}
    .mf-chip {{
      display: inline-block;
      padding: 4px 12px;
      border-radius: 14px;
      background: var(--primary);
      color: #fff;
      font-size: 12px;
      font-weight: 600;
      margin: 3px;
    }}
    .actions {{
      padding: 14px 28px;
      display: flex;
      gap: 10px;
      border-top: 1px solid var(--border);
      background: {card_bg};
    }}
    .btn {{
      padding: 7px 14px;
      border-radius: 7px;
      border: none;
      cursor: pointer;
      font-size: 12px;
      font-weight: 600;
      transition: opacity 0.15s;
    }}
    .btn:hover {{ opacity: 0.82; }}
    .btn-primary {{ background: var(--primary); color: #fff; }}
    .btn-outline {{ background: transparent; color: var(--text); border: 1px solid var(--border); }}
    .no-data {{ grid-column: 1 / -1; text-align: center; color: #555; padding: 40px; font-size: 13px; }}
  </style>
</head>
<body>
  <header>
    <span class="logo">🚀</span>
    <span class="spec-name">{name}</span>
    <span class="spec-type">{spec_type.upper()}</span>
    {tag_html}
  </header>
  <main id="main">
    <div class="no-data">加载中...</div>
  </main>
  {mf_chips}
  <div class="actions">
    <button class="btn btn-primary" onclick="render()">🔄 刷新</button>
    <button class="btn btn-outline" id="agent-btn">🤖 Agent Bridge</button>
  </div>
  <script>
    async function render() {{
      if (!window.__dc) {{
        document.getElementById('main').innerHTML = '<div class="no-data">DataCenter 未连接</div>';
        return;
      }}
      const keys = await window.__dc.list();
      const container = document.getElementById('main');
      const kpiKeys = Object.keys(keys).filter(k => k.startsWith('kpi.') || k.startsWith('metric') || k.startsWith('status'));
      const tableKeys = Object.keys(keys).filter(k => k.startsWith('table.') || k.startsWith('data.'));
      let html = '';

      kpiKeys.forEach(function(k) {{
        const v = keys[k];
        const label = k.replace(/\./g, ' / ').replace(/_/g, ' ');
        html += '<div class="card"><div class="card-title">' + label + '</div><div class="card-value">' + v + '</div></div>';
      }});

      tableKeys.forEach(function(k) {{
        const t = keys[k];
        const cols = Array.isArray(t.columns) ? t.columns : ['Key', 'Value'];
        const rows = Array.isArray(t.rows) ? t.rows : [[k, t]];
        const header = '<thead><tr>' + cols.map(function(c) {{ return '<th>' + c + '</th>'; }}).join('') + '</tr></thead>';
        const body = '<tbody>' + rows.map(function(row) {{
          return '<tr>' + row.map(function(cell) {{ return '<td>' + cell + '</td>'; }}).join('') + '</tr>';
        }}).join('') + '</tbody>';
        html += '<div class="card" style="grid-column:1/-1"><div class="card-title">' + k + '</div><table>' + header + body + '</table></div>';
      }});

      if (!html) html = '<div class="no-data">暂无数据 — 运行 <code>specpilot dc list</code> 查看</div>';
      container.innerHTML = html;
    }}

    document.getElementById('agent-btn').onclick = function() {{
      if (window.vibexPrototypeAgentBridge) {{
        window.vibexPrototypeAgentBridge.highlight(['main']);
      }} else {{
        console.log('vibex-prototype-agent-bridge: ready');
      }}
    }};

    render();
  </script>
</body>
</html>"""
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
