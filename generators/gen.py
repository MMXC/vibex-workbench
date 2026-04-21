#!/usr/bin/env python3
"""
VibeX Workbench — spec-to-code generator
从 L5 data/uiux/service spec 生成 TypeScript types + 组件骨架 + service 类

双文件模式 (B):
  *.Skeleton.svelte  ← gen.py 生成（可重跑覆盖）
  *.svelte           ← 开发者写（永不覆盖）

三段生成:
  1. types.ts         ← data spec → TypeScript interfaces
  2. Skeleton.svelte ← uiux spec → Svelte 组件骨架
  3. lib/services/   ← service spec → TypeScript service 类

用法:
  python3 gen.py <spec_dir> <output_dir>
"""

import sys
import yaml
from pathlib import Path
from datetime import date

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")
OUT_DIR  = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("frontend")
SRC_DIR  = OUT_DIR / "src"
GEN_TEMPLATES_DIR = Path("generators/templates")

GEN_HEADER_TS = f"""// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: {SPEC_DIR}
//     生成时间: {date.today()}
// ============================================================

"""

GEN_HEADER_SVELTE = f"""<!-- ============================================================
⚠️  此文件由 spec-to-code 自动生成
来自: {SPEC_DIR}
生成时间: {date.today()}
⚠️  不要直接编辑此文件 — 改 *.svelte
============================================================ -->

"""

DEV_NOTICE_SVELTE = """<!-- ============================================================
此文件由开发者维护，gen.py 永不覆盖
============================================================ -->

"""


# ── helpers ──────────────────────────────────────────────────────
def load_yaml(path: Path) -> dict:
    with open(path, encoding="utf-8") as f:
        return yaml.safe_load(f)

def all_specs(root: Path):
    for f in root.rglob("*.yaml"):
        if "node_modules" in str(f):
            continue
        try:
            data = load_yaml(f)
            if data:
                yield f, data
        except Exception:
            pass

def spec_by_name(root: Path, name: str):
    for f, data in all_specs(root):
        if data.get("spec", {}).get("name") == name:
            return f, data
        if data.get("spec", {}).get("name", "").replace("-", "_") == name.replace("-", "_"):
            return f, data
    return None

def ts_type(yaml_type: str) -> str:
    if yaml_type is None:
        return "unknown"
    t = yaml_type.strip()
    if t.endswith("[]"):
        return f"{ts_type(t[:-2])}[]"
    if t in ("ISO8601 string", "ISO8601 string | null", "datetime"):
        return "string"
    return {
        "string": "string", "number": "number", "boolean": "boolean",
        "array": "unknown[]", "object": "Record<string, unknown>",
        "json": "Record<string, unknown>",
        "string | null": "string | null",
        "number | null": "number | null",
        "boolean | null": "boolean | null",
        "enum": "string",
    }.get(t, t)


# ── 1. TypeScript Service 类（从 L5 service spec）────────────────────
def _parse_param(param_str: str) -> tuple[str, str]:
    """解析 'name: type' → (name, ts_type)。"""
    if ':' in param_str:
        name, typ = param_str.split(':', 1)
        return name.strip(), ts_type(typ.strip())
    return param_str, 'unknown'


def _parse_method(m) -> tuple[str, list, str]:
    """
    解析单个 method，支持两种格式：
      A. dict:  {"name": "...", "params": ["p: t", ...], "returns": "T"}
      B. str:   "methodName(param: type): ReturnType"
               或 "methodName()"
    返回 (m_name, [(p_name, p_type), ...], return_type)
    """
    import re
    if isinstance(m, str):
        # 格式 B：解析方法签名字符串
        sig = m.strip()
        # 去掉结尾的 () 外的空白
        match = re.match(r"^(\w+)\s*(?:\((.*)\))?\s*(?::\s*(\w+))?$", sig)
        if not match:
            return sig, [], "void"
        m_name = match.group(1) or sig
        params_str = match.group(2) or ""
        returns = match.group(3) or "void"
        # 解析参数列表
        param_list = []
        if params_str.strip():
            for p in params_str.split(","):
                p = p.strip()
                if ":" in p:
                    p_name, p_type = p.split(":", 1)
                    param_list.append((p_name.strip(), ts_type(p_type.strip())))
                else:
                    param_list.append((p, "unknown"))
        return m_name, param_list, ts_type(returns)
    else:
        # 格式 A：dict
        m_name = m.get("name", "unknown")
        raw_params = m.get("params", [])
        raw_returns = m.get("returns", "void")
        param_list = [_parse_param(p) for p in raw_params]
        return m_name, param_list, ts_type(raw_returns)


def gen_services() -> dict[str, str]:
    """
    扫描所有 *_service.yaml，返回 {rel_path: ts_code}。
    每个 service spec 生成一个文件：lib/services/<feature>_services.ts
    """
    import re
    result = {}
    services_dir = SRC_DIR / "lib" / "services"

    for f, data in all_specs(SPEC_DIR):
        spec_name = data.get("spec", {}).get("name", "")
        if not re.match(r".+_service$", spec_name):
            continue

        # feature slug 从 parent 取（如 routing-panel_service → parent=routing-panel）
        parent = data.get("spec", {}).get("parent") or ""
        slug = parent.replace("-", "_") if parent else spec_name.replace("_service", "")
        out_path = services_dir / f"{slug}_services.ts"

        io = data.get("content", {}).get("io_contract", {})
        services_list = data.get("content", {}).get("services", [])

        lines = [
            GEN_HEADER_TS.rstrip(),
            f"// 来源: {f}",
            f"// parent: {parent}",
            "",
            f"/**",
            f" * {slug}_services — {io.get('description', spec_name)}",
            f" * input : {io.get('input', '—')}",
            f" * output: {io.get('output', '—')}",
            f" * boundary: {io.get('boundary', '—')}",
            f" */",
            "",
        ]

        for svc in services_list:
            name = svc.get("name", "UnknownService")
            desc = svc.get("description", "")
            methods = svc.get("methods", [])

            lines.append(f"// ── {name} ──────────────────────────────────────────")
            lines.append(f"/** {desc} */")
            lines.append(f"export class {name} {{")
            lines.append(f"  constructor() {{}}")
            lines.append("")

            for m in methods:
                # 容错：methods 可能是 [{"Name": "...", ...}] 或 ["name_only"]
                if isinstance(m, str):
                    m_name, m_params, m_returns = _parse_method(m)
                else:
                    m_name, m_params, m_returns = _parse_method(m)
                param_str = ", ".join(f"{n}: {t}" for n, t in m_params)

                # 生成 JSDoc
                lines.append(f"  /**")
                lines.append(f"   * {desc or m_name}")
                for pn, pt in m_params:
                    lines.append(f"   * @param {pn} — {pt}")
                lines.append(f"   * @returns {m_returns}")
                lines.append(f"   */")

                # 方法签名（async，真实逻辑需开发者补）
                lines.append(f"  async {m_name}({param_str}): Promise<{ts_type(m_returns)}> {{")
                lines.append(f"    // TODO: 实现 {m_name} — {desc}")
                lines.append(f"    throw new Error('NotImplemented: {name}.{m_name}');")
                lines.append(f"  }}")
                lines.append("")

            lines.append("}")
            lines.append("")

        result[str(out_path.relative_to(SRC_DIR))] = "\n".join(lines)
        print(f"  📦 Service: {out_path.relative_to(SRC_DIR)}  ({len(services_list)} class(es))")

    return result


# ── 2. TypeScript 类型 ──────────────────────────────────────────
def gen_types() -> str:
    out = [GEN_HEADER_TS]
    out.append("// ── Generated Types ──────────────────────────────────────────\n")

    # type_map: spec.name (from YAML spec.spec.name) → comma-sep entity names to emit
    # 2026-04-21 修复：旧名称（thread-manager-data 等）已不存在，改用实际 spec name
    type_map = {
        "code-gen-panel_data":   "GenerationJob, GenerationStep, GenerationEvent",
        "dsl-canvas_data":        "SpecNode, SpecEdge, MermaidGraph",
        "routing-panel_data":     "Change, SpecLocation",
        "spec-editor_data":       "SpecFile, Tab, Violation, LayoutState",
        "workbench-shell_data":   "WorkbenchLayout",
    }

    # ── 动态发现：自动包含所有 *_data spec（兜底，避免手动维护）────────────
    import re
    data_specs = {}
    for f, data in all_specs(SPEC_DIR):
        spec_name = data.get("spec", {}).get("name", "")
        if re.match(r".+_data$", spec_name) and spec_name not in type_map:
            entities_data = data.get("content", {}).get("entities", [])
            if isinstance(entities_data, list):
                names = ", ".join(e.get("entity", "") for e in entities_data if e.get("entity"))
                if names:
                    data_specs[spec_name] = names
    # 合并动态发现（不覆盖显式 type_map）
    for k, v in data_specs.items():
        if k not in type_map:
            type_map[k] = v
    seen = set()

    for spec_name, _ in type_map.items():
        result = spec_by_name(SPEC_DIR, spec_name)
        if not result:
            continue
        f, data = result
        content = data.get("content", {})
        entities = content.get("entities", {})

        if isinstance(entities, dict):
            items = entities.items()
        elif isinstance(entities, list):
            items = [(e.get("name") or e.get("entity"), e) for e in entities]
        else:
            items = []

        for name, defn in items:
            if not name or name in seen:
                continue
            seen.add(name)
            fields = defn.get("fields", []) if isinstance(defn, dict) else []
            out.append(f"export interface {name} {{\n")
            for field in fields:
                fname  = field.get("name", "field")
                raw    = field.get("type", "string")
                ftype  = ts_type(raw)
                # required 逻辑：
                # 1. 显式 required:true → 必填
                # 2. 显式 required:false / required:null → 可选
                # 3. 无 required 字段 → 用启发式（id/name/status 等视为必填）
                explicit = field.get("required")
                if explicit is True:
                    opt = ""
                elif explicit is False or explicit is None:
                    # 启发式：常见必填字段即使 YAML 没标 required 也视为必填
                    required_by_default = {"id", "name", "status", "level", "type", "path",
                                           "content", "startedAt", "createdAt", "updatedAt"}
                    opt = "" if fname in required_by_default else "?"
                else:
                    opt = "?"
                desc   = field.get("description", "")
                out.append(f"  {fname}{opt}: {ftype};  // {desc}\n")
            out.append("}\n\n")

    return "".join(out)


def gen_base_types() -> str:
    lines = [
        GEN_HEADER_TS,
        "// ── Base Types (enum/union) ───────────────────────────────\n",
        "export type RunStatus    = 'pending'|'planning'|'executing'|'completed'|'failed'|'cancelled';\n",
        "export type TaskStatus   = 'pending'|'running'|'completed'|'failed';\n",
        "export type ArtifactType = 'code'|'markdown'|'image'|'json'|'diagram'|'text';\n",
        "export type NodeType     = 'thread'|'run'|'task'|'tool'|'artifact'|'message';\n",
    ]
    return "".join(lines)


# ── 2. 组件 Skeleton 生成 ───────────────────────────────────────
# 每个组件生成:
#   Foo.Skeleton.svelte  ← gen.py 写（可覆盖）
#   Foo.svelte           ← 开发者写（永不覆盖，由 gen.py 创建 stub）
#
# 开发者文件结构:
#   import Skeleton from './Foo.Skeleton.svelte';
#   let { ...props } = $props();
#   // 自定义 snippet / 交互逻辑
#   <Skeleton {sidebar} {main} {panel} {composer} />
#
# Skeleton 结构:
#   interface Props { sidebar?: Snippet; main?: Snippet; panel?: Snippet; composer?: Snippet; }
#   let { ... } = $props();
#   // grid layout, {@render ...?.()}


# ── UIUX spec 读取 ──────────────────────────────────────────────
def load_uiux_registry() -> dict:
    """
    扫描所有 *_uiux.yaml，建立 component_id → spec_data 的 registry。
    返回 {(feature_name, component_id): spec_data}
    """
    registry = {}
    for f, data in all_specs(SPEC_DIR):
        spec_name = data.get("spec", {}).get("name", "")
        # 只处理 uiux spec：名字以 _uiux 结尾
        if not spec_name.endswith("_uiux"):
            continue
        content = data.get("content", {})
        components = content.get("components", [])
        feature = spec_name.removesuffix("_uiux")
        for comp in components:
            cid = comp.get("id", "")
            if cid:
                registry[(feature, cid)] = comp
    return registry


def component_file_path(cid: str) -> str:
    """component ID → frontend 相对路径"""
    return f"lib/components/workbench/{cid}.svelte"


def gen_skeleton_from_grid(cid: str, grid_cols: str, grid_rows: str,
                             grid_areas: list, regions: list,
                             spec_source: str = "") -> str:
    """
    从 grid layout 参数生成 Skeleton.svelte。
    grid_areas: [["L","M","R"], ["B","B","B"]] 等
    regions: [{snippet, grid_area, components}, ...]
    """
    # 构建 grid-template-areas 字符串（无续行符，CSS 不支持 \）
    areas_str = "\n    ".join(
        "'" + " ".join(row) + "'" for row in grid_areas
    )

    # 渲染区域 div（按 grid_area 映射，重复 area 只渲染一次）
    area_to_region = {r["grid_area"]: r for r in regions}
    grid_area_tag = {
        "L": ("aside", "sidebar-left"),
        "M": ("main", "main-canvas"),
        "R": ("aside", "sidebar-right"),
        "B": ("footer", "composer-bar"),
    }
    rendered_areas = set()
    region_divs = []
    for row in grid_areas:
        for area in row:
            if area in rendered_areas:
                continue  # 同一 grid-area 只渲染一次
            if area not in area_to_region:
                continue
            rendered_areas.add(area)
            tag, cls = grid_area_tag.get(area, ("div", "region"))
            reg = area_to_region[area]
            region_divs.append(
                f'  <{tag} class="{cls}" style="grid-area: {area};">\n'
                f'    {{@render {reg["snippet"]}?.()}}\n'
                f'  </{tag}>'
            )

    # Props interface：字段间用换行+缩进分隔，最后一个字段带分号
    prop_fields = ";\n    ".join(f"{r['snippet']}?: Snippet" for r in regions)
    if prop_fields:
        prop_fields += ";"

    skeleton = GEN_HEADER_SVELTE + f"""<script lang="ts">
  import type {{ Snippet }} from 'svelte';
  // {spec_source}
  interface Props {{
    {prop_fields}
  }}
  let {{ {", ".join(r["snippet"] for r in regions)} }}: Props = $props();
</script>

<div class="shell" style="
  display: grid;
  grid-template-columns: {grid_cols};
  grid-template-rows: {grid_rows};
  grid-template-areas:
    {areas_str};
  height: 100vh; overflow: hidden;
">
{chr(10).join(region_divs)}
</div>
"""
    return skeleton


def gen_canvas_skeleton(spec_data: dict) -> str:
    """
    从 canvas-renderer_uiux.yaml 生成 CanvasRenderer Skeleton。
    生成 SvelteFlow 包装层骨架：
    - SvelteFlow + Controls + Background
    - canvasStore reactive 绑定
    - 工具栏（Zoom In/Out/Fit View/Toggle）
    - 节点详情浮层
    """
    uiux = spec_data.get("content", {})
    regions = uiux.get("regions", [])
    stores = uiux.get("state_management", {}).get("stores", [])
    behaviors = uiux.get("behaviors", [])
    tokens = uiux.get("design_tokens", {}).get("colors", {})

    bg_color = tokens.get("bg_canvas", "#0a0a0a")
    primary = tokens.get("primary", "#6366f1")

    # 从 toolbar region 提取按钮
    toolbar_region = next((r for r in regions if r.get("id") == "canvas-toolbar"), None)
    toolbar_components = []
    if toolbar_region:
        for comp in toolbar_region.get("components", []):
            toolbar_components.append(comp)

    # 从 behaviors 提取事件映射
    node_click_behavior = next((b for b in behaviors if b.get("trigger") == "onnodeclick"), {})
    node_dblclick_behavior = next((b for b in behaviors if b.get("trigger") == "onnodedoubleclick"), {})
    drag_stop_behavior = next((b for b in behaviors if b.get("trigger") == "onnodedragstop"), {})

    # canvasStore store name
    store_name = next((s["store"] for s in stores if "nodes" in str(s.get("state", []))), "canvasStore")

    # ── Detail panel sections (避免 f-string 内 {expr??default} 语法问题) ──
    # 状态 badge
    status_section = """
          {#if detailNode.data?.status}
            <div class="detail-section">
              <span class="detail-key">status:</span>
              <span class="detail-status"
                class:running={detailNode.data?.status === 'running'}
                class:completed={detailNode.data?.status === 'completed'}
                class:failed={detailNode.data?.status === 'failed'}>
                {detailNode.data?.status}
              </span>
            </div>
          {/if}"""

    args_section = """
          {#if detailNode.data?.args}
            <div class="detail-section">
              <span class="detail-key">args:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data?.args, null, 2)}</pre>
            </div>
          {/if}"""

    result_section = """
          {#if detailNode.data?.result}
            <div class="detail-section">
              <span class="detail-key">result:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data?.result, null, 2)}</pre>
            </div>
          {/if}"""

    # detail header 表达式（Svelte 模板语法，避免 f-string 内 ?? 解析问题）
    detail_type_expr = "{detailNode.type ?? 'node'}"
    detail_label_expr = "{detailNode.data?.label ?? detailNode.id}"

    # 工具栏按钮生成
    toolbar_btns = ""
    for comp in toolbar_components:
        icon_map = {
            "ZoomIn": "+",
            "ZoomOut": "−",
            "FitView": "⊡",
            "ToggleInteractivity": "⊙",
        }
        label_map = {
            "ZoomIn": "Zoom In",
            "ZoomOut": "Zoom Out",
            "FitView": "Fit View",
            "ToggleInteractivity": "Toggle",
        }
        icon = icon_map.get(comp, "●")
        label = label_map.get(comp, comp)
        tooltip_esc = label.replace('"', '&quot;')
        # 注意：这里不是 f-string，{} 即为字面的 Svelte 模板语法
        toolbar_btns += (
            f'    <button title="{tooltip_esc}" class="tool-btn" '
            f'onclick={{() => {{ console.log("[Canvas] {comp}") }}}}>{icon}</button>\n'
        )

    skeleton = GEN_HEADER_SVELTE + f"""<script lang="ts">
  import {{
    SvelteFlow,
    Controls,
    Background,
    type Node,
    type Edge,
  }} from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import {{ {store_name} }} from '$lib/stores/canvas-store';
  // CanvasRenderer 骨架 — Generated from canvas-renderer_uiux.yaml
  // Reactive state driven by {store_name}
  let storeNodes = $state<Node[]>([]);
  let storeEdges = $state<Edge[]>([]);
  let selectedNodeId = $state<string | null>(null);
  let detailNode = $state<Node | null>(null);

  $effect(() => {{
    const unsub = {store_name}.subscribe(s => {{
      storeNodes = (s.nodes as unknown as Node[]);
      storeEdges = (s.edges as unknown as Edge[]);
    }});
    return unsub;
  }});

  // onnodeclick → selectedNodeId
  function handleNodeClick(_: MouseEvent, node: Node) {{
    selectedNodeId = node.id!;
  }}

  // onnodedoubleclick → detailNode
  function handleNodeDoubleClick(_: MouseEvent, node: Node) {{
    detailNode = node;
    selectedNodeId = node.id!;
  }}

  // onnodedragstop → persist position
  function handleNodeDragStop(_: MouseEvent, node: Node) {{
    {store_name}.updateNode(node.id!, {{ position: node.position }} as any);
  }}

  function closeDetail() {{
    detailNode = null;
    selectedNodeId = null;
  }}
</script>

<div class="canvas-renderer">
  <SvelteFlow
    nodes={{{{storeNodes}}}}
    edges={{{{storeEdges}}}}
    fitView
    onnodeclick={{handleNodeClick}}
    onnodedoubleclick={{handleNodeDoubleClick}}
    onnodedragstop={{handleNodeDragStop}}
  >
    <Controls />
    <Background />
  </SvelteFlow>

  <!-- 工具栏浮层 -->
  <div class="canvas-toolbar">
{toolbar_btns}  </div>

  <!-- 节点详情浮层 -->
  {{#if detailNode}}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="detail-overlay" onclick={{closeDetail}}>
      <div class="detail-panel" onclick={{(e) => e.stopPropagation()}}>
        <div class="detail-header">
          <span class="detail-type">[{detail_type_expr}]</span>
          <span class="detail-label">{detail_label_expr}</span>
          <button onclick={{closeDetail}}>×</button>
        </div>
        <div class="detail-body">
          {{#if detailNode.data?.status}}
            <div class="detail-section">
              <span class="detail-key">status:</span>
              <span class="detail-status"
                class:running={{detailNode.data?.status === 'running'}}
                class:completed={{detailNode.data?.status === 'completed'}}
                class:failed={{detailNode.data?.status === 'failed'}}>
                {{detailNode.data?.status}}
              </span>
            </div>
          {{/if}}
          {{#if detailNode.data?.args}}
            <div class="detail-section">
              <span class="detail-key">args:</span>
              <pre class="detail-code">{{JSON.stringify(detailNode.data?.args, null, 2)}}</pre>
            </div>
          {{/if}}
          {{#if detailNode.data?.result}}
            <div class="detail-section">
              <span class="detail-key">result:</span>
              <pre class="detail-code">{{JSON.stringify(detailNode.data?.result, null, 2)}}</pre>
            </div>
          {{/if}}
        </div>
      </div>
    </div>
  {{/if}}
</div>

<style>
  .canvas-renderer {{ width: 100%; height: 100%; position: relative; }}
  :global(.svelte-flow) {{ background: {bg_color}; }}
  :global(.svelte-flow .node) {{ border-radius: 6px; }}

  /* 工具栏 */
  .canvas-toolbar {{
    position: absolute;
    bottom: 16px;
    left: 16px;
    display: flex;
    gap: 4px;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 4px;
    z-index: 10;
  }}
  .tool-btn {{
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    width: 28px;
    height: 28px;
    border-radius: 4px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
  }}
  .tool-btn:hover {{ background: #333; color: #fff; }}

  /* 详情面板 */
  .detail-overlay {{
    position: absolute;
    inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }}
  .detail-panel {{
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 10px;
    width: 480px;
    max-height: 70vh;
    overflow: auto;
  }}
  .detail-header {{
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 14px;
    border-bottom: 1px solid #333;
  }}
  .detail-type {{ color: {primary}; font-size: 12px; }}
  .detail-label {{ color: #e2e8f0; font-size: 13px; flex: 1; font-weight: 500; }}
  .detail-header button {{
    background: none;
    border: none;
    color: #666;
    font-size: 18px;
    cursor: pointer;
  }}
  .detail-header button:hover {{ color: #fff; }}
  .detail-body {{ padding: 14px; display: flex; flex-direction: column; gap: 10px; }}
  .detail-section {{ display: flex; flex-direction: column; gap: 4px; }}
  .detail-key {{ color: #888; font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; }}
  .detail-code {{
    background: #111;
    border: 1px solid #222;
    border-radius: 6px;
    padding: 8px;
    font-size: 11px;
    color: #ccc;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 200px;
    overflow: auto;
    margin: 0;
  }}
  .detail-status {{ font-size: 12px; padding: 2px 8px; border-radius: 10px; }}
  .detail-status.running {{ background: #1e3a5f; color: #60a5fa; }}
  .detail-status.completed {{ background: #1a3a2a; color: #4ade80; }}
  .detail-status.failed {{ background: #3a1a1a; color: #f87171; }}
</style>
"""

    return skeleton


def gen_canvas_stub() -> str:
    """
    CanvasRenderer stub：开发者可在此扩展自动布局、节点渲染器等。
    默认渲染 CanvasRendererSkeleton，行为与 Skeleton 完全相同。
    """
    return DEV_NOTICE_SVELTE + """<script lang="ts">
  import CanvasRendererSkeleton from './CanvasRenderer.Skeleton.svelte';
  // CanvasRenderer — 开发者维护
  // gen.py 永不覆盖此文件
</script>

<CanvasRendererSkeleton />
"""


def gen_stub_with_children(cid: str, regions: list,
                             registry: dict,
                             fallback_children: list) -> str:
    """
    生成 stub：导入子组件并填充 snippet。
    regions 的 components[] 映射到文件路径，找不到时用 fallback_children。
    """
    # 子组件 import 列表（去重）
    child_imports = []
    seen_children = set()
    for reg in regions:
        for child_cid in reg.get("components", []):
            if child_cid in seen_children:
                continue
            seen_children.add(child_cid)
            child_imports.append(
                f"  import {child_cid} from './{child_cid}.svelte';"
            )
    if not child_imports and fallback_children:
        for fc in fallback_children:
            child_imports.append(f"  import {fc} from './{fc}.svelte';")

    snippet_blocks = []
    for reg in regions:
        children = reg.get("components", [])
        if not children:
            snippet_blocks.append(
                f"  #{{snippet {reg['snippet']}()}}\n"
                f"    <!-- {reg['snippet']}: 空区域 -->\n"
                f"  {{/snippet}}"
            )
        else:
            child_tags = "".join(f"    <{c} />\n" for c in children)
            snippet_blocks.append(
                f"  #{{snippet {reg['snippet']}()}}\n"
                f"{child_tags}  {{/snippet}}"
            )

    return DEV_NOTICE_SVELTE + f"""<script lang="ts">
  import type {{ Snippet }} from 'svelte';
  import {cid}Skeleton from './{cid}.Skeleton.svelte';
{chr(10).join(child_imports)}
  interface Props {{}}
  let {{}}: Props = $props();
</script>

<{cid}Skeleton>
{chr(10).join(snippet_blocks)}
</{cid}Skeleton>
"""


def gen_components() -> dict[str, tuple[str, str]]:
    """
    Returns dict: {{ "path": (skeleton_content, stub_content) }}
    - skeleton: written to path + ".Skeleton.svelte"  (gen.py owns)
    - stub:      written to path                      (developer owns, only if file doesn't exist)

    动态读取 uiux spec：
    - workbench-shell: 从 shell_layout 读 grid 参数
    - 其他组件: fallback 到 hardcoded 模板（向后兼容）
    """
    files = {}
    registry = load_uiux_registry()

    # 硬编码 fallback（完全匹配旧模板，保持向后兼容）
    HARDCODED_WORKBENCHSHELL_SKELETON = GEN_HEADER_SVELTE + """<script lang="ts">
  import type { Snippet } from 'svelte';
  // 三栏布局骨架 — Generated from workbench-shell_feature.yaml
  interface Props {
    sidebar?:  Snippet;
    main?:     Snippet;
    panel?:    Snippet;
    composer?: Snippet;
  }
  let { sidebar, main, panel, composer }: Props = $props();
</script>

<div class="shell" style="
  display: grid;
  grid-template-columns: 280px 1fr 0px;
  grid-template-rows: 1fr auto;
  grid-template-areas:
    'L M R'
    'B B B';
  height: 100vh; overflow: hidden;
">
  <aside class="sidebar-left" style="grid-area: L;">
    {@render sidebar?.()}
  </aside>
  <main class="main-canvas" style="grid-area: M;">
    {@render main?.()}
  </main>
  <aside class="sidebar-right" style="grid-area: R;">
    {@render panel?.()}
  </aside>
  <footer class="composer-bar" style="grid-area: B;">
    {@render composer?.()}
  </footer>
</div>
"""
    HARDCODED_WORKBENCHSHELL_STUB = DEV_NOTICE_SVELTE + """<script lang="ts">
  import type { Snippet } from 'svelte';
  import WorkbenchShellSkeleton from './WorkbenchShell.Skeleton.svelte';
  // ── 填充 snippet 内容 ──────────────────────────────────────────
  import ThreadList    from './ThreadList.svelte';
  import ArtifactPanel from './ArtifactPanel.svelte';
  import Composer      from './Composer.svelte';
  interface Props {}
  let {}: Props = $props();
</script>

<WorkbenchShellSkeleton>
  {#snippet sidebar()}
    <ThreadList />
  {/snippet}
  {#snippet main()}
    <div class="canvas-area">
      <p class="placeholder">Canvas Orchestration</p>
    </div>
  {/snippet}
  {#snippet panel()}
    <ArtifactPanel />
  {/snippet}
  {#snippet composer()}
    <Composer />
  {/snippet}
</WorkbenchShellSkeleton>

<style>
  .canvas-area { flex: 1; overflow: hidden; }
  .placeholder { color: #555; font-size: 13px; padding: 16px; }
</style>
"""
    registry = load_uiux_registry()

    # ── WorkbenchShell: 动态读取 shell_layout ────────────────────
    uiux_data = None
    for f, data in all_specs(SPEC_DIR):
        if data.get("spec", {}).get("name") == "workbench-shell_uiux":
            uiux_data = data.get("content", {})
            break

    shell_layout = uiux_data.get("shell_layout") if uiux_data else None

    if shell_layout:
        # ✅ 从 uiux spec 动态生成
        grid_cols = shell_layout.get("grid_template_columns", "280px 1fr 0px")
        grid_rows = shell_layout.get("grid_template_rows", "1fr auto")
        grid_areas = shell_layout.get("grid_template_areas", [["L","M","R"],["B","B","B"]])
        regions = shell_layout.get("regions", [])

        skeleton = gen_skeleton_from_grid(
            "WorkbenchShell", grid_cols, grid_rows, grid_areas, regions,
            spec_source="Generated from workbench-shell_uiux.yaml (shell_layout)"
        )
        stub = gen_stub_with_children(
            "WorkbenchShell", regions, registry,
            fallback_children=["ThreadList", "Composer", "ArtifactPanel"]
        )
        files["lib/components/workbench/WorkbenchShell.svelte"] = (skeleton, stub)
    else:
        # ❌ fallback 到硬编码
        files["lib/components/workbench/WorkbenchShell.svelte"] = (
            HARDCODED_WORKBENCHSHELL_SKELETON,
            HARDCODED_WORKBENCHSHELL_STUB,
        )

    # ── 其他组件: 保持 hardcoded 模板（向后兼容）────────────────
    # 以下硬编码模板保持不变，仅修复 Thread import path
    # (旧的 ThreadList stub 引用了 $lib/types/generated → 已在上轮修复)

    # ── Composer ───────────────────────────────────────────────
    skeleton = GEN_HEADER_SVELTE + """<script lang="ts">
  // Composer 骨架 — Generated from workbench-shell_uiux.yaml
  // mode: text | image | file | url
  interface Props {
    onsubmit?: (content: string, mode: string) => void;
  }
  let { onsubmit }: Props = $props();
  let content = $state('');
  let mode    = $state<'text'|'image'|'file'|'url'>('text');

  function submit() {
    if (!content.trim()) return;
    onsubmit?.(content, mode);
    content = '';
  }
</script>

<div class="composer">
  <div class="mode-tabs">
    <button class:active={mode==='text'}   onclick={() => mode='text'}>文本</button>
    <button class:active={mode==='image'}  onclick={() => mode='image'}>图片</button>
    <button class:active={mode==='file'}   onclick={() => mode='file'}>文件</button>
    <button class:active={mode==='url'}     onclick={() => mode='url'}>URL</button>
  </div>
  <textarea
    bind:value={content}
    placeholder="输入消息，或 @ 引用 Artifact..."
    rows={3}
    onkeydown={(e) => { if (e.key === 'Enter' && e.ctrlKey) submit(); }}
  ></textarea>
  <div class="actions">
    <button class="submit-btn" onclick={submit}>发送 ⌘↵</button>
  </div>
</div>

<style>
  .composer { padding: 8px 16px; background: #1a1a1a; border-top: 1px solid #333; }
  textarea  { width: 100%; background: #222; border: 1px solid #444; border-radius: 8px; color: #eee; padding: 8px; resize: none; }
  .mode-tabs { display: flex; gap: 4px; margin-bottom: 6px; }
  .mode-tabs button { background: transparent; border: none; color: #888; cursor: pointer; padding: 4px 8px; border-radius: 4px; }
  .mode-tabs button.active { background: #333; color: #fff; }
  .actions  { display: flex; justify-content: flex-end; margin-top: 6px; }
  .submit-btn { background: #4f46e5; color: white; border: none; padding: 6px 16px; border-radius: 6px; cursor: pointer; }
</style>
"""
    stub = DEV_NOTICE_SVELTE + """<script lang="ts">
  import ComposerSkeleton from './Composer.Skeleton.svelte';
  // ── 开发者自定义 Composer 行为 ────────────────────────────────
  interface Props {}
  let {}: Props = $props();

  function handleSubmit(content: string, mode: string) {
    console.log('[Composer] Submit:', { content, mode });
  }
</script>

<ComposerSkeleton onsubmit={handleSubmit} />
"""
    files["lib/components/workbench/Composer.svelte"] = (skeleton, stub)

    # ── ThreadList ─────────────────────────────────────────────
    skeleton = GEN_HEADER_SVELTE + """<script lang="ts">
  import { threadStore, currentThread, threadCount } from '$lib/stores/thread-store';
  // ThreadList 骨架 — Generated from thread-manager_uiux.yaml
  interface Props {
    onNewThread?: () => void;
  }
  let { onNewThread }: Props = $props();
  let threads = $state($threadStore.threads);
  let current = $state($currentThread);
  let count   = $state($threadCount);

  $effect(() => {
    const unsub = threadStore.subscribe(s => { threads = s.threads; });
    return unsub;
  });
</script>

<div class="thread-list">
  <div class="header">
    <span>线程 ({count})</span>
    <button onclick={() => onNewThread?.()}>+ 新建</button>
  </div>
  <div class="items">
    {#each threads as thread (thread.id)}
      <div
        class="thread-item"
        class:active={current?.id === thread.id}
        onclick={() => threadStore.setCurrentThread(thread.id)}
        onkeydown={(e) => e.key === 'Enter' && threadStore.setCurrentThread(thread.id)}
        role="button"
        tabindex="0"
      >
        <span class="name">{thread.title ?? thread.goal?.slice(0, 20) ?? '新线程'}</span>
        <span class="meta">{thread.status ?? 'draft'}</span>
      </div>
    {/each}
    {#if threads.length === 0}
      <p class="empty">暂无线程，点击「+ 新建」创建</p>
    {/if}
  </div>
</div>

<style>
  .thread-list  { height: 100%; display: flex; flex-direction: column; background: #111; }
  .header       { display: flex; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #222; color: #ccc; font-size: 13px; }
  .header button { background: #4f46e5; border: none; color: white; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; }
  .items        { flex: 1; overflow-y: auto; }
  .thread-item  { padding: 10px 16px; cursor: pointer; border-bottom: 1px solid #1a1a1a; display: flex; justify-content: space-between; font-size: 13px; }
  .thread-item:hover   { background: #1a1a1a; }
  .thread-item.active  { background: #1e293b; border-left: 3px solid #4f46e5; }
  .name         { color: #e2e8f0; }
  .meta         { color: #666; font-size: 11px; }
  .empty        { color: #555; font-size: 12px; padding: 16px; text-align: center; }
</style>
"""
    stub = DEV_NOTICE_SVELTE + """<script lang="ts">
  import ThreadListSkeleton from './ThreadList.Skeleton.svelte';
  import type { Thread } from '$lib/types/generated';
  // ── 开发者自定义 ThreadList ────────────────────────────────
  interface Props {}
  let {}: Props = $props();

  function newThread() {
    const t: Thread = {
      id: crypto.randomUUID(),
      goal: '新线程',
      title: '新线程',
      createdAt: new Date().toISOString(),
    };
    import { threadStore } from '$lib/stores/thread-store';
    threadStore.addThread(t);
    threadStore.setCurrentThread(t.id);
  }
</script>

<ThreadListSkeleton onNewThread={newThread} />
"""
    files["lib/components/workbench/ThreadList.svelte"] = (skeleton, stub)

    # ── ArtifactPanel ──────────────────────────────────────────
    skeleton = GEN_HEADER_SVELTE + """<script lang="ts">
  import { artifactStore, filteredArtifacts } from '$lib/stores/artifact-store';
  // ArtifactPanel 骨架 — Generated from artifact-registry_uiux.yaml
  let artifacts = $state($filteredArtifacts);

  $effect(() => {
    const unsub = artifactStore.subscribe(() => { artifacts = $filteredArtifacts; });
    return unsub;
  });
</script>

<div class="artifact-panel">
  <div class="header">Artifacts ({artifacts.length})</div>
  <div class="search">
    <input
      placeholder="搜索 artifacts..."
      oninput={(e) => artifactStore.setSearch((e.target as HTMLInputElement).value)}
    />
  </div>
  <div class="items">
    {#each artifacts as a (a.id)}
      <div
        class="artifact-item"
        role="button"
        tabindex="0"
        onclick={() => artifactStore.select(a.id ?? null)}
        onkeydown={(e) => e.key === 'Enter' && artifactStore.select(a.id ?? null)}
      >
        <span class="type">[{a.type}]</span>
        <span class="name">{a.name}</span>
      </div>
    {/each}
    {#if artifacts.length === 0}
      <p class="empty">暂无 Artifact</p>
    {/if}
  </div>
</div>

<style>
  .artifact-panel  { background: #111; height: 100%; overflow: hidden; display: flex; flex-direction: column; }
  .header          { padding: 12px 16px; border-bottom: 1px solid #222; color: #ccc; font-size: 13px; }
  .search          { padding: 8px; }
  .search input    { width: 100%; background: #222; border: 1px solid #444; color: #eee; padding: 6px 10px; border-radius: 6px; font-size: 12px; }
  .items           { flex: 1; overflow-y: auto; }
  .artifact-item   { padding: 8px 16px; cursor: pointer; display: flex; gap: 8px; font-size: 12px; }
  .artifact-item:hover { background: #1a1a1a; }
  .type            { color: #4f46e5; }
  .name            { color: #e2e8f0; }
  .empty           { color: #555; font-size: 12px; padding: 16px; text-align: center; }
</style>
"""
    stub = DEV_NOTICE_SVELTE + """<script lang="ts">
  import ArtifactPanelSkeleton from './ArtifactPanel.Skeleton.svelte';
  // ── 开发者自定义 ArtifactPanel ────────────────────────────
  interface Props {}
  let {}: Props = $props();
</script>

<ArtifactPanelSkeleton />
"""
    files["lib/components/workbench/ArtifactPanel.svelte"] = (skeleton, stub)

    # ── CanvasRenderer: 从 canvas-renderer_uiux.yaml 动态生成 ────────
    canvas_spec_data = None
    for f, data in all_specs(SPEC_DIR):
        if data.get("spec", {}).get("name") == "canvas-renderer_uiux":
            canvas_spec_data = data
            break

    if canvas_spec_data:
        skeleton = gen_canvas_skeleton(canvas_spec_data)
        stub = gen_canvas_stub()
        files["lib/components/workbench/CanvasRenderer.svelte"] = (skeleton, stub)

    return files


# ── 3. 路由生成 ─────────────────────────────────────────────────
def gen_routes() -> dict[str, tuple[str, str]]:
    files = {}

    # +layout.svelte — 纯结构，写覆盖
    skeleton = GEN_HEADER_SVELTE + """<script lang="ts">
  import '../app.css';
  import type { Snippet } from 'svelte';
  interface Props { children?: Snippet; }
  let { children }: Props = $props();
</script>

{@render children?.()}
"""
    files["routes/+layout.svelte"] = (skeleton, None)  # 无 stub

    # +page.svelte — redirect stub，不覆盖
    stub = DEV_NOTICE_SVELTE + """<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  onMount(() => goto('/workbench'));
</script>

<div class="splash">
  <p>⚡ VibeX Workbench — 加载中...</p>
</div>

<style>
  .splash { display: flex; align-items: center; justify-content: center; height: 100vh; color: #888; }
</style>
"""
    files["routes/+page.svelte"] = (None, stub)

    # workbench/+page.svelte — 开发者写，不覆盖
    stub = DEV_NOTICE_SVELTE + """<script lang="ts">
  // VibeX Workbench — 主工作台页面
  import WorkbenchShell from '$lib/components/workbench/WorkbenchShell.svelte';
</script>

<WorkbenchShell />
"""
    files["routes/workbench/+page.svelte"] = (None, stub)

    return files


# ── types merge ────────────────────────────────────────────────
def merge_types(existing_path: Path, new_content: str) -> str:
    """
    将 gen.py 生成的 spec 类型合并到现有 lib/types.ts：
    - gen.py spec 类型优先（正确，必填字段由启发式决定）
    - 保留现有 lib/types.ts 中不在 spec 内的类型（如 Create/Update 变体）
    - 追加 Thread = ConversationThread 别名
    """
    import re

    if not existing_path.exists():
        return new_content + "\n\nexport type Thread = ConversationThread;\n"

    existing = existing_path.read_text(encoding="utf-8")

    # 从 new_content 提取 interface/type 名称集合
    new_names = set(re.findall(r'^export (?:interface|type) (\w+)', new_content, re.MULTILINE))

    # 保留 existing 中不在 new_names 里的定义（Create/Update 变体等）
    kept = []
    for m in re.finditer(r'^(export (?:interface|type) \w+.*?)(?=\n(?:export[type ]|$))',
                         existing, re.DOTALL | re.MULTILINE):
        name_m = re.search(r'export (?:interface|type) (\w+)', m.group(0))
        if name_m and name_m.group(1) not in new_names:
            kept.append(m.group(1).rstrip())

    merged = new_content + "\n\n" + "\n\n".join(kept) + "\n"

    # Thread 别名（修复 ConversationThread vs Thread 命名不一致）
    if "export type Thread = ConversationThread" not in merged:
        merged += "\n\nexport type Thread = ConversationThread;\n"

    return merged


# ── main ─────────────────────────────────────────────────────────
def main():
    print("VibeX Workbench — Code Generator (B 双文件模式)")
    print(f"  Spec dir:   {SPEC_DIR}")
    print(f"  Output dir: {OUT_DIR}")

    # ── Types ────────────────────────────────────────────────
    # 输出到 lib/types.ts（stores 的实际导入路径），合并而非覆盖
    types_path = SRC_DIR / "lib/types.ts"
    new_types = gen_types() + gen_base_types()
    merged = merge_types(types_path, new_types)
    types_path.write_text(merged, encoding="utf-8")
    print(f"  ✅ Types:   {types_path.relative_to(OUT_DIR)} (merged)")

    # ── Services ────────────────────────────────────────────
    # 输出到 lib/services/<feature>_services.ts（永不覆盖，幂等写入）
    for rel_path, code in gen_services().items():
        svc_path = SRC_DIR / rel_path
        svc_path.parent.mkdir(parents=True, exist_ok=True)
        svc_path.write_text(code, encoding="utf-8")

    # 清理孤立的 lib/generated/ 目录（已合并到 lib/types.ts）
    old_dir = SRC_DIR / "lib/generated"
    if old_dir.exists():
        import shutil
        shutil.rmtree(old_dir)
        print(f"  🗑️  Removed: lib/generated/ (merged into lib/types.ts)")

    # ── Components ───────────────────────────────────────────
    for rel_path, (skeleton, stub) in gen_components().items():
        skeleton_path = SRC_DIR / rel_path
        skeleton_path.parent.mkdir(parents=True, exist_ok=True)

        # Skeleton: 永远覆盖
        skeleton_out = skeleton_path.with_name(skeleton_path.name + ".Skeleton.svelte")
        skeleton_out.write_text(skeleton, encoding="utf-8")
        print(f"  ✅ Skeleton: {skeleton_out.relative_to(OUT_DIR)}")

        # Stub: 仅当文件不存在时创建
        if stub and not skeleton_path.exists():
            skeleton_path.write_text(stub, encoding="utf-8")
            print(f"  ✅ Stub:     {skeleton_path.relative_to(OUT_DIR)}  ← 首次创建")
        elif stub:
            print(f"  ⏭️  Stub:     {skeleton_path.relative_to(OUT_DIR)}  ← 已存在，跳过")

    # ── Routes ───────────────────────────────────────────────
    for rel_path, (skeleton, stub) in gen_routes().items():
        route_path = SRC_DIR / rel_path
        route_path.parent.mkdir(parents=True, exist_ok=True)

        if skeleton and not route_path.exists():
            route_path.write_text(skeleton, encoding="utf-8")
            print(f"  ✅ Route:    {route_path.relative_to(OUT_DIR)}  ← 首次创建")
        elif skeleton:
            print(f"  ⏭️  Route:   {route_path.relative_to(OUT_DIR)}  ← 已存在，跳过")
        elif stub:
            if not route_path.exists():
                route_path.write_text(stub, encoding="utf-8")
                print(f"  ✅ Stub:     {route_path.relative_to(OUT_DIR)}  ← 首次创建")
            else:
                print(f"  ⏭️  Stub:    {route_path.relative_to(OUT_DIR)}  ← 已存在，跳过")

    print(f"\n✅ 生成完成")
    print(f"  运行: make dev")

    # ── E3: Sync handler templates from meta-spec ──────────────────
    # Reads top-level 'template' field from specs with level: meta_template
    # Writes to generators/templates/<name>.yaml.tpl
    # This closes the spec self-bootstrapping loop:
    #   feature-template spec (level: meta_template, template field)
    #     → make generate syncs to generators/templates/*.tpl
    #     → handler reads *.tpl and substitutes ${PLACEHOLDER} tokens
    for f, data in all_specs(SPEC_DIR):
        if data.get("spec", {}).get("level") == "meta_template":
            spec_name = data.get("spec", {}).get("name", "")
            # template may be at top level or inside content:
            template_content = data.get("template") or data.get("content", {}).get("template", "")
            if not template_content:
                continue
            tpl_out = GEN_TEMPLATES_DIR / f"{spec_name}.yaml.tpl"
            tpl_out.parent.mkdir(parents=True, exist_ok=True)
            tpl_out.write_text(template_content, encoding="utf-8")
            print(f"  ✅ Template: {tpl_out}  (from spec: {spec_name})")


if __name__ == "__main__":
    main()
