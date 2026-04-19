#!/usr/bin/env python3
"""
spec-to-sveltekit v0.1
从 L5a (uiux) + L5c (data) spec 生成 SvelteKit 代码

用法:
  python3 gen.py <spec_dir> <output_dir> [--mode local|backend]

生成产物:
  src/lib/types.ts          — TypeScript 类型（从 data.yaml）
  src/lib/api.ts           — API client（从 service.yaml）
  src/lib/stores/          — Zustand stores（从 uiux.yaml）
  src/routes/               — 页面骨架（从 uiux.yaml）
  src/components/           — 组件骨架（从 uiux.yaml）
  src/generated/           — 全部生成物（可安全覆盖）
"""

import sys
import json
import yaml
import re
import os
from pathlib import Path
from typing import Any

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")
OUT_DIR = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("frontend")
MODE = "backend"  # default: includes API layer
i = 3
while i < len(sys.argv):
    arg = sys.argv[i]
    if arg == "--mode" and i + 1 < len(sys.argv):
        MODE = sys.argv[i + 1]
        i += 2
    elif arg.startswith("--mode="):
        MODE = arg.split("=", 1)[1]
        i += 1
    else:
        i += 1

GEN_HEADER = f"""// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: {SPEC_DIR}
//     生成时间: {__import__('datetime').date.today()}
//     模式: {MODE}
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

"""


# ── 工具函数 ──────────────────────────────────────────────
def load_yaml(path: Path) -> dict:
    with open(path, encoding="utf-8") as f:
        return yaml.safe_load(f)


def snake_to_camel(s: str) -> str:
    """snake_case → camelCase"""
    parts = s.split("_")
    return parts[0] + "".join(p.title() for p in parts[1:])


def kebab_to_camel(s: str) -> str:
    """kebab-case → camelCase"""
    parts = s.split("-")
    return parts[0] + "".join(p.title() for p in parts[1:])


def plural(s: str) -> str:
    """简单复数化"""
    if s.endswith("s"):
        return s + "es"
    if s.endswith("y"):
        return s[:-1] + "ies"
    return s + "s"


def ts_type(yaml_type: str) -> str:
    """YAML 类型 → TypeScript 类型"""
    mapping = {
        "string": "string",
        "number": "number",
        "boolean": "boolean",
        "datetime": "string",
        "json": "Record<string, unknown>",
    }
    return mapping.get(yaml_type, yaml_type)


def ensure_dir(path: Path):
    path.mkdir(parents=True, exist_ok=True)

def spec_root(spec_arg: Path) -> Path:
    """spec_arg may be a YAML file or a directory — return the parent dir for rglob."""
    return spec_arg.parent if spec_arg.is_file() else spec_arg


# ── 从 data.yaml 生成 TypeScript 类型 ────────────────────
def gen_types(spec_dir: Path) -> str:
    """从所有 data.yaml 生成 TypeScript interfaces"""
    search_root = spec_root(spec_dir)
    entities = []
    for data_file in search_root.rglob("canvas_data.yaml"):
        data = load_yaml(data_file)
        for entity in data.get("content", {}).get("entities", []):
            name = entity["name"]
            fields = entity.get("fields", [])
            imports = set()
            field_lines = []
            for f in fields:
                fname = f["name"]
                ftype = ts_type(f["type"])
                if ftype == "string" and f.get("fk"):
                    imports.add(f["fk"])
                required = f.get("required", False)
                opt = "" if required else "?"
                default = f.get("default")
                comment = f" // {f.get('description', '')}"
                if default is not None:
                    field_lines.append(
                        f"  {fname}{opt}: {ftype};  // default: {default}{comment}"
                    )
                else:
                    field_lines.append(f"  {fname}{opt}: {ftype};{comment}")
            entities.append((name, field_lines))

    lines = [GEN_HEADER, "// ── Generated Types ──────────────────────────────────────────\n"]
    lines.append("export interface CanvasSnapshot {\n  id: string;\n  canvas_id: string;\n  data: string; // gzip压缩的JSON\n  created_at: string;\n  is_auto: boolean;\n}\n")

    for name, field_lines in entities:
        lines.append(f"export interface {name} {{\n")
        lines.extend("".join(f + "\n" for f in field_lines))
        lines.append("}\n\n")

    return "".join(lines)


# ── 从 service.yaml 生成 API client ───────────────────────
def gen_api(spec_dir: Path) -> str:
    """从所有 service.yaml 生成 API client"""
    lines = [GEN_HEADER, "import type {\n"]

    # 收集所有 entities
    entities = set()
    for svc_file in spec_root(spec_dir).rglob("canvas_service.yaml"):
        data = load_yaml(svc_file)
        for api in data.get("content", {}).get("apis", []):
            for m in api.get("methods", []):
                belongs = m.get("belongs_to", "")
                if belongs:
                    entities.add(belongs)

    for ent in sorted(entities):
        lines.append(f"  {ent},\n")
        lines.append(f"  {ent}Create,\n")
        lines.append(f"  {ent}Update,\n")
    lines.append("} from './types';\n\n")

    # BASE URL
    if MODE == "backend":
        lines.append("const API_BASE = '/api';\n\n")
        lines.append("async function request<T>(path: string, options: RequestInit = {}): Promise<T> {\n")
        lines.append("  const res = await fetch(API_BASE + path, {\n")
        lines.append("    headers: { 'Content-Type': 'application/json', ...options.headers },\n")
        lines.append("    ...options,\n")
        lines.append("  });\n")
        lines.append("  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);\n")
        lines.append("  return res.json();\n")
        lines.append("}\n\n")
    else:
        lines.append("// LOCAL 模式：使用 Dexie.js\n")
        lines.append("import { db } from './db';\n\n")

    lines.append("// ── Canvas API ──────────────────────────────────────────────\n\n")

    for svc_file in spec_root(spec_dir).rglob("canvas_service.yaml"):
        data = load_yaml(svc_file)
        svc_name = data.get("spec", {}).get("name", "unknown")
        for api in data.get("content", {}).get("apis", []):
            svc_name = api.get("name", svc_name)
            methods = api.get("methods", [])

            for m in methods:
                mname = m["name"]
                http = m["http_method"]
                endpoint = m["endpoint"]
                belongs = m.get("belongs_to", "")
                req = m.get("request", {})
                resp = m.get("response", {})

                # 生成函数签名
                params = []
                body_fields = []
                param_fields = req.get("params", [])
                query_fields = req.get("query", [])
                body_fields_raw = req.get("body", [])

                for pf in param_fields:
                    params.append(f"{pf['field']}: string")
                for qf in query_fields:
                    dflt = f" = {qf.get('default', 'undefined')}" if qf.get("default") is not None else ""
                    params.append(f"{qf['field']}?: number{dflt}")

                params_str = ", ".join(params)

                # 生成 endpoint（处理 :param）
                ep = endpoint
                ep_params = re.findall(r":(\w+)", ep)
                ep_args = ", ".join(ep_params)
                ep_impl = re.sub(r":(\w+)", r"${\1}", ep)

                # body 构建
                if body_fields_raw:
                    body_args = [f"['{f['field']}']" for f in body_fields_raw]
                    body_map = "body: {" + ", ".join(body_fields_raw[0]["field"] for _ in body_fields_raw) + "}"

                comment = f"  // {svc_name}.{mname} ({http} {endpoint})"

                if MODE == "backend":
                    if body_fields_raw:
                        lines.append(f"export async function {mname}({params_str}): Promise<{belongs} | void> {{\n")
                        body_keys = ", ".join(f"['{f['field']}']: {f['field']}" if f.get('required') else f"['{f['field']}']: {f['field']}" for f in body_fields_raw)
                        lines.append(f"  return request<{belongs}>(`{ep_impl}`, {{\n")
                        lines.append(f"    method: '{http}',\n")
                        lines.append(f"    body: JSON.stringify({{{body_keys}}}),\n")
                        lines.append(f"  }});\n")
                    else:
                        lines.append(f"export async function {mname}({params_str}): Promise<{belongs} | void> {{\n")
                        lines.append(f"  return request<{belongs}>(`{ep_impl}`, {{ method: '{http}' }});\n")
                else:
                    # LOCAL 模式：操作 Dexie
                    entity_map = {
                        "CreateCanvas": "db.canvas.add",
                        "GetCanvas": "db.canvas.get",
                        "ListCanvases": "db.canvas.toArray",
                        "UpdateCanvas": "db.canvas.update",
                        "DeleteCanvas": "db.canvas.update",
                        "AddNode": "db.node.add",
                        "UpdateNode": "db.node.update",
                        "DeleteNode": "db.node.update",
                        "AddEdge": "db.edge.add",
                        "DeleteEdge": "db.edge.update",
                    }
                    db_op = entity_map.get(mname, "// TODO: implement")

                    if mname == "CreateCanvas":
                        lines.append(f"export async function {mname}({params_str}): Promise<Canvas> {{\n")
                        lines.append(f"  const id = crypto.randomUUID();\n")
                        lines.append(f"  const record = {{ id, name, description: description ?? '', viewport_x: 0, viewport_y: 0, zoom: 1.0, created_at: new Date().toISOString(), updated_at: new Date().toISOString(), is_deleted: false }};\n")
                        lines.append(f"  await db.canvas.add(record);\n")
                        lines.append(f"  return record;\n")
                    elif mname == "GetCanvas":
                        lines.append(f"export async function {mname}(id: string): Promise<Canvas | undefined> {{\n")
                        lines.append(f"  return db.canvas.get(id) ?? undefined;\n")
                    elif mname == "ListCanvases":
                        lines.append(f"export async function {mname}(limit = 20, offset = 0): Promise<{{items: Canvas[], total: number}}> {{\n")
                        lines.append(f"  const all = await db.canvas.where('is_deleted').equals(0).toArray();\n")
                        lines.append(f"  const items = all.slice(offset, offset + limit);\n")
                        lines.append(f"  return {{ items, total: all.length }};\n")
                    elif mname == "DeleteCanvas":
                        lines.append(f"export async function {mname}(id: string): Promise<void> {{\n")
                        lines.append(f"  await db.canvas.update(id, {{ is_deleted: true, updated_at: new Date().toISOString() }});\n")
                    elif mname == "AddNode":
                        lines.append(f"export async function {mname}(canvasId: string, type: string, label: string, positionX: number, positionY: number, config?: Record<string, unknown>): Promise<Node> {{\n")
                        lines.append(f"  const id = crypto.randomUUID();\n")
                        lines.append(f"  const record = {{ id, canvas_id: canvasId, type, label, position_x: positionX, position_y: positionY, config: config ?? {{}}, created_at: new Date().toISOString(), is_deleted: false }};\n")
                        lines.append(f"  await db.node.add(record);\n")
                        lines.append(f"  return record;\n")
                    elif mname == "AddEdge":
                        lines.append(f"export async function {mname}(canvasId: string, sourceNodeId: string, sourcePort: string, targetNodeId: string, targetPort: string): Promise<Edge> {{\n")
                        lines.append(f"  const id = crypto.randomUUID();\n")
                        lines.append(f"  const record = {{ id, canvas_id: canvasId, source_node_id: sourceNodeId, source_port: sourcePort, target_node_id: targetNodeId, target_port: targetPort, edge_type: 'default', condition_expression: '', is_deleted: false }};\n")
                        lines.append(f"  await db.edge.add(record);\n")
                        lines.append(f"  return record;\n")
                    else:
                        lines.append(f"export async function {mname}({params_str}): Promise<void> {{\n")
                        lines.append(f"  // TODO: implement {mname}\n")
                        lines.append("  console.warn('Not implemented:', '" + mname + "');\n")

                lines.append("}\n\n")

    return "".join(lines)


# ── 从 uiux.yaml 生成 SvelteKit 页面骨架 ─────────────────
def _build_page_svelte(page_id: str, path: str, components: list, stores: list) -> str:
    """Build a +page.svelte from component list and stores."""
    page_code = GEN_HEADER
    page_code += "<script lang=\"ts\">\n"
    page_code += f"  // 页面: {page_id} ({path})\n"
    page_code += f"  // 组件数量: {len(components)}\n\n"

    for store in stores:
        sname = store["store"]
        camel = kebab_to_camel(sname.replace("Store", ""))
        page_code += f"  import {{ {sname} }} from '$lib/stores/{camel}';\n"

    page_code += "\n  // 状态\n"
    page_code += "  let loading = $state(false);\n"

    for comp in components:
        comp_id = snake_to_camel(comp.get("id", comp["_id"] if "_id" in comp else "comp"))
        binding = comp.get("data_binding", "")
        if binding:
            page_code += f"  let {comp_id} = $derived($state({binding}));\n"

    page_code += "\n  // 事件处理\n"
    page_code += "  function handleSubmit(e: SubmitEvent) {\n"
    page_code += "    e.preventDefault();\n"
    page_code += "  }\n"
    page_code += "</script>\n\n"
    page_code += f"<div class=\"page-{page_id}\">\n"

    for comp in components:
        comp_type = comp.get("type", "div")
        comp_id = snake_to_camel(comp.get("id", "comp"))
        content = comp.get("content", "")
        data_bind = comp.get("data_binding", "")
        visibility = comp.get("visibility", "")

        if visibility:
            page_code += f"  <!-- visibility: {visibility} -->\n"

        if comp_type == "text":
            page_code += f"    <span class=\"{comp_id}\">{content}</span>\n"
        elif comp_type == "input":
            placeholder = comp.get("placeholder", "")
            editable = comp.get("editable", False)
            if editable:
                page_code += f"    <input class=\"{comp_id}\" placeholder=\"{placeholder}\" />\n"
            else:
                page_code += f"    <span class=\"{comp_id}\">{content}</span>\n"
        elif comp_type == "button":
            label = comp.get("label", "")
            action = comp.get("action", "")
            page_code += f'    <button class="{comp_id}" onclick={{() => console.log("{action}")}}>{label}</button>\n'
        elif comp_type == "icon-button":
            icon = comp.get("icon", "")
            action = comp.get("action", "")
            page_code += f'    <button class="icon-btn {comp_id}" onclick={{() => console.log("{action}")}}>[{icon}]</button>\n'
        elif comp_type == "status-badge":
            page_code += f"    <span class=\"{comp_id}\">{{$state.status ?? ''}}</span>\n"
        elif comp_type == "scrollable-list":
            page_code += f"    <div class=\"scrollable-list {comp_id}\">\n"
            page_code += f"      <!-- items from: {data_bind} -->\n"
            page_code += f"      <!-- TODO: replace with actual data binding -->\n"
            page_code += f"    </div>\n"
        elif comp_type == "canvas-viewport":
            page_code += f"    <div class=\"canvas-viewport {comp_id}\">\n"
            page_code += f"      <CanvasEditor />\n"
            page_code += f"    </div>\n"
        elif comp_type == "thread-header":
            page_code += f"    <div class=\"thread-header {comp_id}\">\n"
            page_code += f"    <span>{{$threadStore?.currentThread?.name ?? ''}}</span>\n"
            page_code += f"    </div>\n"
        elif comp_type == "execution-stream":
            page_code += f"    <div class=\"execution-stream {comp_id}\">\n"
            page_code += f"      <ExecutionStream />\n"
            page_code += f"    </div>\n"
        elif comp_type == "bottom-drawer":
            page_code += f"    <ArtifactDrawer />\n"
        elif comp_type == "result-zone":
            page_code += f"    <ResultZone />\n"
        elif comp_type == "dynamic-form":
            page_code += f"    <NodeConfigForm />\n"
        elif comp_type == "timeline":
            page_code += f"    <RunTimeline />\n"
        elif comp_type == "context-banner":
            page_code += f"    <ContextBanner />\n"
        elif comp_type == "canvas-toolbar":
            page_code += f"    <CanvasToolbar />\n"
        elif comp_type == "zoom-control":
            page_code += f"    <ZoomControl />\n"
        elif comp_type == "infinite-canvas":
            page_code += f"    <InfiniteCanvas />\n"
        elif comp_type == "draggable-palette":
            page_code += f"    <NodePalette />\n"
        else:
            page_code += f"    <!-- {comp_type}: {comp_id} -->\n"
            page_code += f"    <div class=\"{comp_type} {comp_id}\"></div>\n"

    page_code += "</div>\n"
    return page_code


def gen_routes(spec_dir: Path) -> dict[str, str]:
    """从 uiux.yaml 生成路由文件"""
    files = {}

    for uiux_file in spec_root(spec_dir).rglob("canvas_uiux.yaml"):
        data = load_yaml(uiux_file)
        pages = data.get("content", {}).get("pages", [])
        stores = data.get("content", {}).get("state_management", {}).get("stores", [])

        # 如果没有显式 pages，从 stores 推断默认路由
        if not pages:
            print(f"  [gen_routes] canvas_uiux.yaml 无显式 pages，从 stores 推断默认路由")

            # 默认首页 → 重定向到 /canvas
            files["routes/+page.svelte"] = (
                GEN_HEADER
                + "<script lang=\"ts\">\n"
                + "  import { goto } from '$app/navigation';\n"
                + "  import { onMount } from 'svelte';\n"
                + "  onMount(() => goto('/canvas'));\n"
                + "</script>\n\n"
                + "<div class=\"home-redirect\">\n"
                + "  <p>Redirecting to VibeX Canvas...</p>\n"
                + "</div>\n"
            )

            # Canvas 列表页
            canvas_list_comps = [
                {"id": "canvas-toolbar", "type": "canvas-toolbar"},
                {"id": "canvas-list", "type": "scrollable-list", "data_binding": "$canvasStore.canvases"},
            ]
            files["routes/canvas/+page.svelte"] = _build_page_svelte(
                "canvas-list", "/canvas", canvas_list_comps, stores
            )
            files["routes/canvas/+page.ts"] = (
                GEN_HEADER
                + "import { ListCanvases } from '$lib/api';\n\n"
                + "export async function load() {\n"
                + "  const data = await ListCanvases();\n"
                + "  return { canvases: data.items, total: data.total };\n"
                + "}\n"
            )

            # 单个 Canvas 视图（画布编辑器主页）
            canvas_editor_comps = [
                {"id": "context-banner", "type": "context-banner"},
                {"id": "canvas-toolbar", "type": "canvas-toolbar"},
                {"id": "canvas-viewport", "type": "canvas-viewport"},
                {"id": "bottom-drawer", "type": "bottom-drawer"},
            ]
            files["routes/canvas/[id]/+page.svelte"] = _build_page_svelte(
                "canvas-editor", "/canvas/[id]", canvas_editor_comps, stores
            )
            files["routes/canvas/[id]/+page.ts"] = (
                GEN_HEADER
                + "import { GetCanvas } from '$lib/api';\n"
                + "import { error } from '@sveltejs/kit';\n\n"
                + "export async function load({{ params }}: {{ params: {{ id: string }} }}) {\n"
                + "  const canvas = await GetCanvas(params.id);\n"
                + "  if (!canvas) throw error(404, 'Canvas not found');\n"
                + "  return { canvas };\n"
                + "}\n"
            )

            # Thread 视图
            thread_comps = [
                {"id": "thread-header", "type": "thread-header"},
                {"id": "scrollable-list", "type": "scrollable-list", "data_binding": "$threadStore.currentThread?.messages"},
                {"id": "execution-stream", "type": "execution-stream"},
            ]
            files["routes/canvas/[id]/thread/[threadId]/+page.svelte"] = _build_page_svelte(
                "thread-view", "/canvas/[id]/thread/[threadId]", thread_comps, stores
            )
            files["routes/canvas/[id]/thread/[threadId]/+page.ts"] = (
                GEN_HEADER
                + "import { error } from '@sveltejs/kit';\n\n"
                + "export async function load({{ params }}: {{ params: {{ id: string; threadId: string }} }}) {\n"
                + "  // TODO: load thread data\n"
                + "  return {{ threadId: params.threadId, canvasId: params.id }};\n"
                + "}\n"
            )

            # Run 视图
            run_comps = [
                {"id": "timeline", "type": "timeline"},
                {"id": "result-zone", "type": "result-zone"},
                {"id": "dynamic-form", "type": "dynamic-form"},
            ]
            files["routes/canvas/[id]/run/[runId]/+page.svelte"] = _build_page_svelte(
                "run-view", "/canvas/[id]/run/[runId]", run_comps, stores
            )
            files["routes/canvas/[id]/run/[runId]/+page.ts"] = (
                GEN_HEADER
                + "import { error } from '@sveltejs/kit';\n\n"
                + "export async function load({{ params }}: {{ params: {{ id: string; runId: string }} }}) {\n"
                + "  // TODO: load run data\n"
                + "  return {{ runId: params.runId, canvasId: params.id }};\n"
                + "}\n"
            )

            # layout.ts — 共享布局
            files["routes/+layout.ts"] = (
                GEN_HEADER
                + "// 共享布局：加载 UI store 初始状态\n"
                + "export const ssr = false;\n"
                + "export const prerender = false;\n"
            )

            return files

        for page in pages:
            page_id = page["id"]
            path = page.get("path", "/")
            regions = page.get("regions", [])

            # 提取组件列表
            components = []
            for region in regions:
                for comp in region.get("components", []):
                    components.append(comp)

            # 生成 +page.svelte
            page_code = GEN_HEADER
            page_code += "<script lang=\"ts\">\n"
            page_code += f"  // 页面: {page_id} ({path})\n"
            page_code += f"  // 组件数量: {len(components)}\n\n"

            # import stores
            for store in stores:
                sname = store["store"]
                camel = kebab_to_camel(sname.replace("Store", ""))
                page_code += f"  import {{ {sname} }} from '$lib/stores/{camel}';\n"

            page_code += "\n  // 状态\n"
            page_code += "  let loading = $state(false);\n"

            # 生成响应式变量
            for comp in components:
                comp_id = snake_to_camel(comp.get("id", comp["_id"] if "_id" in comp else "comp"))
                binding = comp.get("data_binding", "")
                if binding:
                        page_code += "  let " + comp_id + " = $derived($state(" + binding + "));" + chr(10)

            page_code += "\n  // 事件处理\n"
            page_code += "  function handleSubmit(e: SubmitEvent) {\n"
            page_code += "    e.preventDefault();\n"
            page_code += "    // TODO: implement\n"
            page_code += "  }\n"
            page_code += "</script>\n\n"

            page_code += f"<div class=\"page-{page_id}\">\n"

            # 生成 HTML 结构
            for region in regions:
                pos = region.get("position", "main")
                page_code += f"  <!-- {pos} region -->\n"
                page_code += f"  <div class=\"region region-{pos}\">\n"

                for comp in region.get("components", []):
                    comp_type = comp.get("type", "div")
                    comp_id = snake_to_camel(comp.get("id", "comp"))
                    content = comp.get("content", "")
                    data_bind = comp.get("data_binding", "")
                    visibility = comp.get("visibility", "")

                    if visibility:
                        page_code += f"    <!-- visibility: {visibility} -->\n"

                    if comp_type == "text":
                        page_code += f"    <span class=\"{comp_id}\">{content}</span>\n"
                    elif comp_type == "input":
                        placeholder = comp.get("placeholder", "")
                        editable = comp.get("editable", False)
                        if editable:
                            page_code += f"    <input class=\"{comp_id}\" placeholder=\"{placeholder}\" />\n"
                        else:
                            page_code += f"    <span class=\"{comp_id}\">{content}</span>\n"
                    elif comp_type == "button":
                        label = comp.get("label", "")
                        action = comp.get("action", "")
                        page_code += f'    <button class="{comp_id}" onclick={{() => console.log("{action}")}}>{label}</button>\n'
                    elif comp_type == "icon-button":
                        icon = comp.get("icon", "")
                        action = comp.get("action", "")
                        page_code += f'    <button class="icon-btn {comp_id}" onclick={{() => console.log("{action}")}}>[{icon}]</button>\n'
                    elif comp_type == "status-badge":
                        states = comp.get("states", [])
                        page_code += "    <span class=\"" + comp_id + "\">" + "{$state.status ?? ''}" + "</span>\n"
                    elif comp_type == "scrollable-list":
                        item_template = comp.get("item_template", "").strip()
                        page_code += f"    <div class=\"scrollable-list {comp_id}\">\n"
                        page_code += f"      <!-- items from: {data_bind} -->\n"
                        page_code += f"      {{#each items as item}}\n"
                        page_code += f"        <div class=\"list-item\">{{item.name ?? item.label ?? item.id}}</div>\n"
                        page_code += f"      {{/each}}\n"
                        page_code += f"    </div>\n"
                    elif comp_type == "canvas-viewport":
                        page_code += f"    <div class=\"canvas-viewport {comp_id}\">\n"
                        page_code += f"      <!-- Canvas 节点编辑器 -->\n"
                        page_code += f"      <CanvasEditor />\n"
                        page_code += f"    </div>\n"
                    elif comp_type == "thread-header":
                        page_code += f"    <div class=\"thread-header {comp_id}\">\n"
                        page_code += "    <span class=\"" + comp_id + "\">" + "{$threadStore.currentThread?.name ?? ''}" + "</span>\n"
                        page_code += f"    </div>\n"
                    elif comp_type == "execution-stream":
                        page_code += f"    <div class=\"execution-stream {comp_id}\">\n"
                        page_code += f"      <ExecutionStream />\n"
                        page_code += f"    </div>\n"
                    elif comp_type == "bottom-drawer":
                        page_code += f"    <ArtifactDrawer />\n"
                    elif comp_type == "result-zone":
                        page_code += f"    <ResultZone />\n"
                    elif comp_type == "dynamic-form":
                        page_code += f"    <NodeConfigForm />\n"
                    elif comp_type == "timeline":
                        page_code += f"    <RunTimeline />\n"
                    elif comp_type == "context-banner":
                        page_code += f"    <ContextBanner />\n"
                    elif comp_type == "canvas-toolbar":
                        page_code += f"    <CanvasToolbar />\n"
                    elif comp_type == "zoom-control":
                        page_code += f"    <ZoomControl />\n"
                    elif comp_type == "infinite-canvas":
                        page_code += f"    <InfiniteCanvas />\n"
                    elif comp_type == "draggable-palette":
                        page_code += f"    <NodePalette />\n"
                    elif comp_type == "canvas-toolbar":
                        page_code += f"    <CanvasToolbar />\n"
                    else:
                        page_code += f"    <!-- {comp_type}: {comp_id} -->\n"
                        page_code += f"    <div class=\"{comp_type} {comp_id}\"></div>\n"

                page_code += "  </div>\n"

            page_code += "</div>\n"

            # 路由路径
            route_path = path.strip("/")
            if not route_path:
                route_path = "index"
            files[f"routes/{route_path}/+page.svelte"] = page_code

            # +page.ts
            page_load = GEN_HEADER
            page_load += f"// load data for {path}\n"
            page_load += "export async function load() {\n"
            if "canvas" in route_path and "list" not in route_path:
                page_load += "  // 加载画布列表\n"
                page_load += "  return {};\n"
            elif "canvas" in route_path and "list" in route_path:
                page_load += "  // 加载画布列表\n"
                page_load += "  return {};\n"
            else:
                page_load += "  return {};\n"
            page_load += "}\n"
            files[f"routes/{route_path}/+page.ts"] = page_load

    return files


# ── 生成 stores ────────────────────────────────────────────
def gen_stores(spec_dir: Path) -> dict[str, str]:
    """从 uiux.yaml 的 state_management 生成 Zustand stores"""
    files = {}

    for uiux_file in spec_root(spec_dir).rglob("canvas_uiux.yaml"):
        data = load_yaml(uiux_file)
        stores_data = data.get("content", {}).get("state_management", {}).get("stores", [])

        for store_data in stores_data:
            sname = store_data["store"]
            camel = kebab_to_camel(sname.replace("Store", ""))

            code = GEN_HEADER
            code += f"// {sname} — generated from canvas_uiux.yaml\n\n"

            if sname == "canvasStore":
                code += """import { writable, derived } from 'svelte/store';
import type { Canvas, Node, Edge } from '../types';

function createCanvasStore() {
  const { subscribe, set, update } = writable<{
    canvases: Canvas[];
    currentCanvas: Canvas | null;
    nodes: Node[];
    edges: Edge[];
    selectedNodes: string[];
    viewport: { x: number; y: number; zoom: number };
  }>({
    canvases: [],
    currentCanvas: null,
    nodes: [],
    edges: [],
    selectedNodes: [],
    viewport: { x: 0, y: 0, zoom: 1 },
  });

  return {
    subscribe,
    setCanvases: (canvases: Canvas[]) => update(s => ({ ...s, canvases })),
    setCurrent: (canvas: Canvas | null) => update(s => {
      if (!canvas) return { ...s, currentCanvas: null, nodes: [], edges: [] };
      return { ...s, currentCanvas: canvas };
    }),
    addNode: (node: Node) => update(s => ({ ...s, nodes: [...s.nodes, node] })),
    updateNode: (id: string, changes: Partial<Node>) => update(s => ({
      ...s,
      nodes: s.nodes.map(n => n.id === id ? { ...n, ...changes } : n)
    })),
    deleteNode: (id: string) => update(s => ({
      ...s,
      nodes: s.nodes.filter(n => n.id !== id),
      edges: s.edges.filter(e => e.source_node_id !== id && e.target_node_id !== id),
    })),
    addEdge: (edge: Edge) => update(s => ({ ...s, edges: [...s.edges, edge] })),
    deleteEdge: (id: string) => update(s => ({
      ...s,
      edges: s.edges.filter(e => e.id !== id)
    })),
    setSelected: (ids: string[]) => update(s => ({ ...s, selectedNodes: ids })),
    setViewport: (vp: { x: number; y: number; zoom: number }) =>
      update(s => ({ ...s, viewport: vp })),
    reset: () => set({
      canvases: [],
      currentCanvas: null,
      nodes: [],
      edges: [],
      selectedNodes: [],
      viewport: { x: 0, y: 0, zoom: 1 },
    }),
  };
}

export const canvasStore = createCanvasStore();
"""
            elif sname == "threadStore":
                code += """import { writable } from 'svelte/store';

export interface Thread {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  status: string;
}

export interface ContextSummary {
  referenced_files: string[];
  current_context: string;
  temp_attachments: string[];
  persistent_memory: string;
}

function createThreadStore() {
  const { subscribe, set, update } = writable<{
    threads: Thread[];
    currentThread: Thread | null;
    context_summary: ContextSummary;
  }>({
    threads: [],
    currentThread: null,
    context_summary: {
      referenced_files: [],
      current_context: '',
      temp_attachments: [],
      persistent_memory: '',
    },
  });

  return {
    subscribe,
    add: (thread: Thread) => update(s => ({ ...s, threads: [...s.threads, thread] })),
    setCurrent: (thread: Thread | null) => update(s => ({ ...s, currentThread: thread })),
    updateContext: (summary: Partial<ContextSummary>) =>
      update(s => ({ ...s, context_summary: { ...s.context_summary, ...summary } })),
  };
}

export const threadStore = createThreadStore();
"""
            elif sname == "runStore":
                code += """import { writable, derived } from 'svelte/store';

export type RunStatus = 'idle' | 'queued' | 'planning' | 'running' | 'completed' | 'failed';

export interface ToolInvocation {
  id: string;
  tool_name: string;
  tool_icon: string;
  stage: string;
  status: string;
  duration_ms: number;
  retry_count: number;
  raw_log: string;
}

export interface Run {
  id: string;
  status: RunStatus;
  message_stream: string;
  tool_invocations: ToolInvocation[];
  final_output: unknown;
  created_at: string;
}

function createRunStore() {
  const { subscribe, set, update } = writable<{
    currentRun: Run | null;
    runs: Run[];
  }>({
    currentRun: null,
    runs: [],
  });

  return {
    subscribe,
    setCurrent: (run: Run | null) => update(s => ({ ...s, currentRun: run })),
    addRun: (run: Run) => update(s => ({ ...s, runs: [...s.runs, run], currentRun: run })),
    appendMessage: (delta: string) =>
      update(s => {
        if (!s.currentRun) return s;
        return {
          ...s,
          currentRun: { ...s.currentRun, message_stream: s.currentRun.message_stream + delta }
        };
      }),
    addToolInvocation: (ti: ToolInvocation) =>
      update(s => {
        if (!s.currentRun) return s;
        return { ...s, currentRun: { ...s.currentRun, tool_invocations: [...s.currentRun.tool_invocations, ti] } };
      }),
    updateToolInvocation: (id: string, changes: Partial<ToolInvocation>) =>
      update(s => {
        if (!s.currentRun) return s;
        return {
          ...s,
          currentRun: {
            ...s.currentRun,
            tool_invocations: s.currentRun.tool_invocations.map(ti =>
              ti.id === id ? { ...ti, ...changes } : ti
            )
          }
        };
      }),
    markCompleted: () =>
      update(s => {
        if (!s.currentRun) return s;
        return { ...s, currentRun: { ...s.currentRun, status: 'completed' } };
      }),
    markFailed: (error?: string) =>
      update(s => {
        if (!s.currentRun) return s;
        return { ...s, currentRun: { ...s.currentRun, status: 'failed' } };
      }),
  };
}

export const runStore = createRunStore();
"""
            elif sname == "artifactStore":
                code += """import { writable } from 'svelte/store';

export interface Artifact {
  id: string;
  name: string;
  type: string;
  preview: string;
  url?: string;
  content?: unknown;
}

function createArtifactStore() {
  const { subscribe, update } = writable<{
    artifacts: Artifact[];
    pending: Artifact[];
  }>({ artifacts: [], pending: [] });

  return {
    subscribe,
    add: (artifact: Artifact) =>
      update(s => ({ ...s, artifacts: [...s.artifacts, artifact] })),
    update: (id: string, changes: Partial<Artifact>) =>
      update(s => ({
        ...s,
        artifacts: s.artifacts.map(a => a.id === id ? { ...a, ...changes } : a)
      })),
    remove: (id: string) =>
      update(s => ({ ...s, artifacts: s.artifacts.filter(a => a.id !== id) })),
  };
}

export const artifactStore = createArtifactStore();
"""
            elif sname == "uiStore":
                code += """import { writable } from 'svelte/store';

function createUiStore() {
  const { subscribe, update } = writable<{
    sidebarLeft: boolean;
    sidebarRight: boolean;
    artifactDrawerOpen: boolean;
    logLayer: 1 | 2 | 3;
    canvasViewActive: boolean;
  }>({
    sidebarLeft: true,
    sidebarRight: true,
    artifactDrawerOpen: false,
    logLayer: 1,
    canvasViewActive: true,
  });

  return {
    subscribe,
    toggleLeft: () => update(s => ({ ...s, sidebarLeft: !s.sidebarLeft })),
    toggleRight: () => update(s => ({ ...s, sidebarRight: !s.sidebarRight })),
    toggleArtifactDrawer: () => update(s => ({ ...s, artifactDrawerOpen: !s.artifactDrawerOpen })),
    cycleLogLayer: () => update(s => ({ ...s, logLayer: ((s.logLayer % 3) + 1) as 1 | 2 | 3 })),
    toggleCanvasView: () => update(s => ({ ...s, canvasViewActive: !s.canvasViewActive })),
  };
}

export const uiStore = createUiStore();
"""
            else:
                # 通用 store
                state_fields = []
                for st in store_data.get("state", []):
                    fname = st["name"]
                    ftype = st.get("type", "unknown")
                    state_fields.append(f"  {fname}: {ftype};")

                code += f"""import {{ writable }} from 'svelte/store';

function create{store_data['store']}() {{
  const initial = {{
{chr(10).join(state_fields)}
  }};
  return writable(initial);
}}

export const {store_data['store']} = create{store_data['store']}();
"""
            files[f"src/lib/stores/{camel}.ts"] = code

    return files


# ── 生成 PWA（manifest + service worker）─────────────────
def gen_pwa(spec_name: str = "canvas") -> dict[str, str]:
    """生成 PWA 必需文件：manifest.json、service worker、layout.svelte"""
    manifest = {
        "static/manifest.json": """{
  "name": "VibeX Canvas",
  "short_name": "VibeX",
  "description": "AI-powered canvas workspace",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#6366f1",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
"""
    }
    sw = {
        "static/sw.js": GEN_HEADER + """// Service Worker — VibeX PWA offline support
const CACHE_NAME = 'vibex-v1';
const STATIC_ASSETS = [
  '/',
  '/canvas',
  '/canvas/list',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(STATIC_ASSETS);
    })
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  if (event.request.method !== 'GET') return;
  event.respondWith(
    caches.match(event.request).then((cached) => {
      if (cached) return cached;
      return fetch(event.request)
        .then((response) => {
          if (!response || response.status !== 200 || response.type === 'opaque') {
            return response;
          }
          const clone = response.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(event.request, clone));
          return response;
        })
        .catch(() => caches.match('/'));
    })
  );
});
"""
    }
    layout_svelte = {
        "src/routes/+layout.svelte": GEN_HEADER + """<script>
  import { onMount } from 'svelte';
  onMount(() => {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }
  });
</script>

<slot />
"""
    }
    return {**manifest, **sw, **layout_svelte}


# ── 生成 DB（local 模式用 Dexie.js）───────────────────────
def gen_db() -> str:
    return GEN_HEADER + """// Dexie.js 数据库（local 模式）
import Dexie from 'dexie';

export class VibexDB extends Dexie {
  canvas!: Dexie.Table<{
    id: string; name: string; description: string;
    viewport_x: number; viewport_y: number; zoom: number;
    created_at: string; updated_at: string; is_deleted: number;
  }>;
  node!: Dexie.Table<{
    id: string; canvas_id: string; type: string; label: string;
    position_x: number; position_y: number; config: string;
    created_at: string; is_deleted: number;
  }>;
  edge!: Dexie.Table<{
    id: string; canvas_id: string; source_node_id: string;
    source_port: string; target_node_id: string; target_port: string;
    edge_type: string; condition_expression: string; is_deleted: number;
  }>;
  snapshot!: Dexie.Table<{
    id: string; canvas_id: string; data: string;
    created_at: string; is_auto: number;
  }>;

  constructor() {
    super('VibexDB');
    this.version(1).stores({
      canvas: 'id, name, updated_at, is_deleted',
      node: 'id, canvas_id, type, is_deleted',
      edge: 'id, canvas_id, source_node_id, target_node_id, is_deleted',
      snapshot: 'id, canvas_id, created_at',
    });
  }
}

export const db = new VibexDB();
"""


# ── 主函数 ────────────────────────────────────────────────
def main():
    print(f"spec-to-sveltekit v0.1")
    print(f"  spec_dir : {SPEC_DIR}")
    print(f"  output_dir: {OUT_DIR}")
    print(f"  mode     : {MODE}")

    ensure_dir(OUT_DIR / "src" / "lib" / "stores")
    ensure_dir(OUT_DIR / "src" / "generated" / "components")
    ensure_dir(OUT_DIR / "src" / "routes" / "canvas")
    ensure_dir(OUT_DIR / "src" / "routes" / "canvas" / "list")
    ensure_dir(OUT_DIR / "static")
    ensure_dir(OUT_DIR / "static" / "icons")

    # 生成文件
    print("  生成 src/lib/types.ts...")
    (OUT_DIR / "src" / "lib" / "types.ts").write_text(gen_types(SPEC_DIR), encoding="utf-8")

    print("  生成 src/lib/api.ts...")
    (OUT_DIR / "src" / "lib" / "api.ts").write_text(gen_api(SPEC_DIR), encoding="utf-8")

    print("  生成 src/lib/db.ts (Dexie.js)...")
    (OUT_DIR / "src" / "lib" / "db.ts").write_text(gen_db(), encoding="utf-8")

    print("  生成 stores...")
    for path, content in gen_stores(SPEC_DIR).items():
        full_path = OUT_DIR / "src" / "lib" / "stores" / Path(path).name
        full_path.write_text(content, encoding="utf-8")

    print("  生成 routes...")
    for path, content in gen_routes(SPEC_DIR).items():
        full_path = OUT_DIR / "src" / path
        ensure_dir(full_path.parent)
        full_path.write_text(content, encoding="utf-8")

    print("  生成 src/generated/README.txt...")
    (OUT_DIR / "src" / "generated" / "README.txt").write_text(
        "此目录由 spec-to-sveltekit 自动生成，所有文件可安全覆盖。\n"
        "手动编辑请放在 src/lib/ 或 src/components/ 目录。\n",
        encoding="utf-8"
    )

    print("  生成 PWA (manifest + sw.js + layout.svelte)...")
    for path, content in gen_pwa().items():
        full_path = OUT_DIR / path
        ensure_dir(full_path.parent)
        full_path.write_text(content, encoding="utf-8")
        print(f"  生成 {path}...")

    print(f"\n✅ 生成完成！")
    print(f"   运行: cd {OUT_DIR} && npm run dev")


if __name__ == "__main__":
    main()
