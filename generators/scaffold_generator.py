#!/usr/bin/env python3
"""
VibeX Scaffold Generator — 从 vibex-workbench 复制最小脚手架到目标目录

用法:
    python3 scaffold_generator.py /target/workspace [--project-name NAME] [--owner USER] [--with-agent]

流程:
    1. 检测目标目录状态（必须是 empty 或 partial）
    2. 打印将要创建的文件清单
    3. 用户确认后写入磁盘
    4. 验证 state_detector.py 返回 ready
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from datetime import datetime

# ── 从当前 vibex-workbench 复制的模板文件（相对路径）────────────
# 这些文件是 vibex-workbench 的核心脚手架，不可删除

SCAFFOLD_MANIFEST = [
    # (src_rel, dest_rel, is_dir, desc)
    ("Makefile", "Makefile", False, "make lint-specs / validate / generate"),
    ("generators/gen.py", "generators/gen.py", False, "spec → code 入口"),
    ("generators/validate_specs.py", "generators/validate_specs.py", False, "YAML 语法 + 必填字段检查"),
    ("spec-engine/validate_chain.py", "spec-engine/validate_chain.py", False, "parent chain 链路检查"),
    ("frontend/package.json", "frontend/package.json", False, "前端依赖"),
    ("frontend/svelte.config.js", "frontend/svelte.config.js", False, "SvelteKit 配置"),
    ("frontend/vite.config.ts", "frontend/vite.config.ts", False, "Vite 配置"),
    ("frontend/tsconfig.json", "frontend/tsconfig.json", False, "TS 配置"),
    ("spec-templates", "spec-templates", True, "L0–L5 spec 模板"),
    ("spec-engine", "spec-engine", True, "spec-engine（validate_chain 等）"),
]

# L1 goal 模板替换占位符
PLACEHOLDERS = {
    "{project-name}": "project-name",
    "{owner}": "owner",
    "{YYYY-MM-DD}": datetime.now().strftime("%Y-%m-%d"),
    "{YYYY-MM-DDTHH:MM:SS}": datetime.now().strftime("%Y-%m-%dT%H:%M:%S"),
}


def substitute(content: str, project_name: str, owner: str) -> str:
    """替换模板占位符"""
    replacements = {
        "{project-name}": project_name,
        "{owner}": owner,
        "{YYYY-MM-DD}": datetime.now().strftime("%Y-%m-%d"),
        "{YYYY-MM-DDTHH:MM:SS}": datetime.now().strftime("%Y-%m-%dT%H:%M:%S"),
    }
    result = content
    for placeholder, value in replacements.items():
        result = result.replace(placeholder, value)
    return result


def detect_state(workspace: str) -> dict:
    """调用 state_detector.py 检测目录状态"""
    detector = os.path.join(os.path.dirname(__file__), "state_detector.py")
    try:
        result = subprocess.run(
            ["python3", detector, workspace, "--json"],
            capture_output=True, text=True, timeout=30
        )
        return json.loads(result.stdout.strip())
    except Exception as e:
        return {"state": "error", "error": str(e)}


def preview_scaffold(workspace: str, with_agent: bool) -> list:
    """列出将要创建的文件（dry-run）"""
    vx_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))  # vibex-workbench root
    files = []
    for src_rel, dest_rel, is_dir, desc in SCAFFOLD_MANIFEST:
        if is_dir:
            files.append({
                "type": "dir",
                "path": dest_rel,
                "desc": desc,
                "note": f"整个目录从 {src_rel} 复制"
            })
        else:
            src_path = os.path.join(vx_root, src_rel)
            if os.path.exists(src_path):
                size = os.path.getsize(src_path)
                files.append({
                    "type": "file",
                    "path": dest_rel,
                    "desc": desc,
                    "size": size,
                })
            else:
                files.append({
                    "type": "file",
                    "path": dest_rel,
                    "desc": desc,
                    "size": 0,
                    "warning": f"源文件不存在: {src_path}"
                })
    return files


def run_scaffold(workspace: str, project_name: str, owner: str, dry_run: bool = False) -> dict:
    """
    执行脚手架生成。
    dry_run=True 时只返回预览，不写磁盘。
    """
    vx_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

    # ── 前置检查：目录必须存在 ──────────────────────────
    if not os.path.isdir(workspace):
        return {"ok": False, "error": f"目录不存在: {workspace}"}

    state = detect_state(workspace)
    if state.get("state") == "ready":
        return {
            "ok": False,
            "error": f"目录已是 ready 状态（{workspace}），无需初始化。\n"
                     f"如需重新生成，请先清理目录。"
        }

    # ── 预览文件清单 ────────────────────────────────────
    files = preview_scaffold(workspace, with_agent=False)

    if dry_run:
        return {"ok": True, "dry_run": True, "files": files, "workspace": workspace}

    # ── 执行写入 ────────────────────────────────────────
    created = []
    errors = []

    for src_rel, dest_rel, is_dir, desc in SCAFFOLD_MANIFEST:
        src_path = os.path.join(vx_root, src_rel)
        dest_path = os.path.join(workspace, dest_rel)

        try:
            if is_dir:
                if os.path.exists(dest_path):
                    continue  # 已存在则跳过
                if os.path.isdir(src_path):
                    shutil.copytree(src_path, dest_path)
                    created.append(dest_rel)
            else:
                if os.path.exists(dest_path):
                    continue  # 已存在则跳过
                os.makedirs(os.path.dirname(dest_path), exist_ok=True)
                with open(src_path, "r", encoding="utf-8") as f:
                    content = f.read()
                # 对 spec-templates 做占位符替换
                if dest_rel.startswith("spec-templates/"):
                    content = substitute(content, project_name, owner)
                with open(dest_path, "w", encoding="utf-8") as f:
                    f.write(content)
                created.append(dest_rel)
        except Exception as e:
            errors.append(f"{dest_rel}: {e}")

    # ── 写 L1 goal 入口文件 ───────────────────────────
    # specs/project-goal/<name>-goal.yaml
    goal_dir = os.path.join(workspace, "specs", "project-goal")
    os.makedirs(goal_dir, exist_ok=True)
    goal_path = os.path.join(goal_dir, f"{project_name}-goal.yaml")

    goal_content = f'''---
spec:
  version: "0.1"
  level: "1_project-goal"
  name: "{project_name}-goal"
  parent: null
  status: "proposal"

meta:
  type: "project-goal"
  owner: "{owner}"
  created: "{datetime.now().strftime("%Y-%m-%d")}"
  updated: "{datetime.now().strftime("%Y-%m-%d")}"

lifecycle:
  current: "proposal"
  updated: "{datetime.now().strftime("%Y-%m-%dT%H:%M:%S")}"
  history:
    - status: "proposal"
      at: "{datetime.now().strftime("%Y-%m-%dT%H:%M:%S")}"
      by: "{owner}"
      trigger: "user:manual"
      note: "项目目标初始化"

io_contract:
  input: "用户描述的产品意图（自然语言）"
  output: "L1 goal spec（使命、约束、价值分层）"
  boundary: "只回答做什么/不做什么/成功什么样，不写技术实现"
  behavior: |
    1. 接收用户意图
    2. 写成 L1 goal spec
    3. 确认 spec 完整
    4. 进入 L2 skeleton 阶段

content:
  mission: "待填写：产品的核心使命是什么"
  target_users: "待填写：目标用户是谁"
  constraints: []
  product_value_layers:
    kernel: "内核价值"
    proof: "证明路径"
    experience: "用户体验"
'''

    with open(goal_path, "w", encoding="utf-8") as f:
        f.write(goal_content)
    created.append(f"specs/project-goal/{project_name}-goal.yaml")

    # ── 写确认摘要 ──────────────────────────────────────
    summary = f"""
脚手架生成完成 ✅
目标目录: {workspace}
项目名: {project_name}
创建文件/目录: {len(created)}
"""
    if errors:
        summary += f"\n错误: {errors}\n"

    # ── 验证 state ──────────────────────────────────────
    new_state = detect_state(workspace)
    summary += f"\n状态验证: {new_state.get('state', 'unknown')}\n"

    return {
        "ok": True,
        "created": created,
        "errors": errors,
        "state": new_state,
        "summary": summary.strip()
    }


def main():
    parser = argparse.ArgumentParser(description="VibeX Scaffold Generator")
    parser.add_argument("workspace", help="目标工作区目录路径")
    parser.add_argument("--project-name", default="my-project",
                        help="项目名称（kebab-case，用于生成 spec 文件名）")
    parser.add_argument("--owner", default="user",
                        help="负责人用户名")
    parser.add_argument("--dry-run", action="store_true",
                        help="仅预览将要创建的文件，不写入磁盘")
    parser.add_argument("--confirm", action="store_true",
                        help="跳过确认，直接写入（用于自动化场景）")
    parser.add_argument("--json", action="store_true",
                        help="JSON 输出格式")
    args = parser.parse_args()

    workspace = os.path.abspath(args.workspace)

    # dry-run preview
    result = run_scaffold(workspace, args.project_name, args.owner, dry_run=True)

    if args.json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
        sys.exit(0)

    # ── 打印预览 ────────────────────────────────────────
    print(f"\n📦 VibeX 脚手架预览")
    print(f"   目标目录: {workspace}")
    print(f"   项目名: {args.project_name}")
    print(f"   负责人: {args.owner}")
    print(f"\n   将要创建:")
    for f in result["files"]:
        if f["type"] == "dir":
            print(f"     📁 {f['path']}/  ← {f['desc']}")
        else:
            size = f.get("size", 0)
            size_str = f"({size} bytes)" if size > 0 else ""
            warn = f" ⚠️ {f.get('warning','')}" if f.get("warning") else ""
            print(f"     📄 {f['path']} {size_str} — {f['desc']}{warn}")
    print(f"\n   共 {len(result['files'])} 项（已存在文件会跳过）")

    if args.dry_run:
        print("\n✅ dry-run 完成，未写入磁盘（加 --confirm 实际写入）")
        sys.exit(0)

    # ── 确认写入 ────────────────────────────────────────
    if not args.confirm:
        print(f"\n确认写入？输入 y / yes 确认:")
        try:
            resp = input("> ").strip().lower()
            if resp not in ("y", "yes"):
                print("已取消。")
                sys.exit(0)
        except (EOFError, IOError):
            print("非交互模式，跳过确认。")
            sys.exit(0)

    # 执行写入
    result = run_scaffold(workspace, args.project_name, args.owner, dry_run=False)
    print(result["summary"])

    if not result["ok"]:
        sys.exit(1)
    sys.exit(0)


if __name__ == "__main__":
    main()
