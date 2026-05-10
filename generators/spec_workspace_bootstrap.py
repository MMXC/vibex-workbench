#!/usr/bin/env python3
"""
从 vibex-workbench 仓库内 spec-templates/ 复制并占位符替换，生成与
L0 workspace_bootstrap_contract + 各层 content.workspace_bootstrap 对齐的 L1–L5 占位 specs。

用法:
    python3 spec_workspace_bootstrap.py <workspace_root> [--project-slug NAME] [--owner USER]
        [--overwrite] [--dry-run] [--json]

约定:
    - template 根目录默认 = 本脚本所在目录的上一级 / spec-templates
    - L1 specs/L1-goal/{slug}-goal.yaml （与 scaffold ENTRY.yaml 可并存）
    - 链: {slug}-goal <- {slug}-skeleton <- MOD-{slug}-shell <- FEAT-{slug}-starter <- SLICE-{slug}-starter-slice
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from datetime import datetime
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent.resolve()
DEFAULT_TEMPLATE_ROOT = SCRIPT_DIR.parent / "spec-templates"

BOOTSTRAP_MAP = (
    ("L1-goal/L1-goal-template.yaml", "specs/L1-goal/{slug}-goal.yaml"),
    ("L2-skeleton/L2-skeleton-template.yaml", "specs/L2-skeleton/{slug}-skeleton.yaml"),
    ("L3-module/L3-module-template.yaml", "specs/L3-module/MOD-{slug}-shell.yaml"),
    ("L4-feature/L4-feature-template.yaml", "specs/L4-feature/FEAT-{slug}-starter.yaml"),
    ("L5-slice/L5-slice-template.yaml", "specs/L5-slice/SLICE-{slug}-starter-slice.yaml"),
)


def slugify(raw: str) -> str:
    s = raw.strip().lower().replace(" ", "-")
    s = re.sub(r"[^a-z0-9._-]", "-", s)
    s = re.sub(r"-+", "-", s).strip("-")
    return s or "project"


def build_context(slug: str, owner: str) -> dict:
    now_day = datetime.now().strftime("%Y-%m-%d")
    now_ts = datetime.now().strftime("%Y-%m-%dT%H:%M:%S")
    l1 = f"{slug}-goal"
    l2 = f"{slug}-skeleton"
    l3 = f"MOD-{slug}-shell"
    l4 = f"FEAT-{slug}-starter"
    l5 = f"SLICE-{slug}-starter-slice"
    return {
        "slug": slug,
        "l1_spec_name": l1,
        "l2_spec_name": l2,
        "l3_spec_name": l3,
        "l4_spec_name": l4,
        "l5_spec_name": l5,
        "owner": owner,
        "YYYY-MM-DD": now_day,
        "YYYY-MM-DDTHH:MM:SS": now_ts,
    }


def patch_l3_yaml(content: str, l3_name: str) -> str:
    content = re.sub(
        r"^(\s*)name:\s*\"\{module-name\}\"",
        rf'\1name: "{l3_name}"',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    content = re.sub(
        r"^(\s*)module:\s*\"\{module-name\}\"",
        r'\1module: "frontend"',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    return content


def patch_l4_yaml(content: str, l4_name: str) -> str:
    content = re.sub(
        r"^(\s*)name:\s*\"\{feature-name\}\"",
        rf'\1name: "{l4_name}"',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    content = re.sub(
        r"^(\s*)module:\s*\"\{module-name\}\"",
        r'\1module: "frontend"',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    return content


def patch_l5_yaml(content: str, l5_name: str) -> str:
    content = re.sub(
        r"^(\s*)name:\s*\"\{slice-name\}\"",
        rf'\1name: "{l5_name}"',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    content = re.sub(
        r"^(\s*)module:\s*\"\{module-name\}\"",
        r'\1module: "frontend"',
        content,
        count=1,
        flags=re.MULTILINE,
    )
    return content


def apply_plain_replacements(content: str, ctx: dict, kind: str) -> str:
    pairs = [
        ("{YYYY-MM-DDTHH:MM:SS}", ctx["YYYY-MM-DDTHH:MM:SS"]),
        ("{YYYY-MM-DD}", ctx["YYYY-MM-DD"]),
        ("{l4-spec-name}", ctx["l4_spec_name"]),
        ("{l3-spec-name}", ctx["l3_spec_name"]),
        ("{l2-spec-name}", ctx["l2_spec_name"]),
        ("{l1-spec-name}", ctx["l1_spec_name"]),
        ("{short-name}", ctx["l2_spec_name"]),
        ("{project-name}", ctx["l1_spec_name"]),
        ("{feature-name}", ctx["l4_spec_name"]),
        ("{slice-name}", ctx["l5_spec_name"]),
        ("{main-module-name}", "frontend"),
        ("{owner}", ctx["owner"]),
    ]
    if kind == "l5":
        pass
    for old, new in pairs:
        content = content.replace(old, new)

    # L2 skeleton 里 modules_matrix 等仍可出现 {module-name}，替换为占位词便于检索
    if kind == "l2":
        content = content.replace('"{module-name}"', '"_fill-module-name"')

    # L3/L4/L5：spec.name 已处理；其余示例中的 {module-name} 统一弱化
    if kind in ("l3", "l4", "l5"):
        content = content.replace("{module-name}", "stub-module")

    return content


def run_bootstrap(
    workspace_root: Path,
    template_root: Path,
    slug: str,
    owner: str,
    overwrite: bool,
    dry_run: bool,
) -> dict:
    ctx = build_context(slug, owner)
    written = []
    skipped = []
    errors = []

    if not workspace_root.is_dir():
        return {"ok": False, "error": f"目录不存在: {workspace_root}"}

    plan = []
    for src_rel, dst_pattern in BOOTSTRAP_MAP:
        src = template_root / src_rel
        dst_rel = dst_pattern.format(slug=slug)
        plan.append({"src": str(src), "dest": dst_rel, "exists": src.is_file()})
        if not src.is_file():
            errors.append(f"模板缺失: {src}")

    if errors:
        return {"ok": False, "error": "; ".join(errors), "plan": plan}

    if dry_run:
        return {"ok": True, "dry_run": True, "slug": slug, "plan": plan, "workspaceRoot": str(workspace_root)}

    for src_rel, dst_pattern in BOOTSTRAP_MAP:
        src = template_root / src_rel
        dst_rel = dst_pattern.format(slug=slug)
        dst = workspace_root / dst_rel
        if "L1-goal" in dst_rel:
            kind = "l1"
        elif "L2-skeleton" in dst_rel:
            kind = "l2"
        elif "L3-module" in dst_rel:
            kind = "l3"
        elif "L4-feature" in dst_rel:
            kind = "l4"
        else:
            kind = "l5"

        try:
            dst.parent.mkdir(parents=True, exist_ok=True)
        except OSError as e:
            errors.append(f"{dst_rel}: mkdir {e}")
            continue

        if dst.exists() and not overwrite:
            skipped.append(dst_rel)
            continue

        text = src.read_text(encoding="utf-8")

        if kind == "l3":
            text = patch_l3_yaml(text, ctx["l3_spec_name"])
        elif kind == "l4":
            text = patch_l4_yaml(text, ctx["l4_spec_name"])
        elif kind == "l5":
            text = patch_l5_yaml(text, ctx["l5_spec_name"])

        text = apply_plain_replacements(text, ctx, kind)

        dst.write_text(text, encoding="utf-8")
        written.append(dst_rel)

    result = {
        "ok": True,
        "written_files": written,
        "skipped_files": skipped,
        "errors": errors,
        "slug": slug,
        "chain": {
            "L1": ctx["l1_spec_name"],
            "L2": ctx["l2_spec_name"],
            "L3": ctx["l3_spec_name"],
            "L4": ctx["l4_spec_name"],
            "L5": ctx["l5_spec_name"],
        },
        "workspaceRoot": str(workspace_root.resolve()),
    }
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description="VibeX workspace L1–L5 spec bootstrap from templates")
    parser.add_argument("workspace_root", help="目标工作区根目录")
    parser.add_argument("--template-root", default=str(DEFAULT_TEMPLATE_ROOT), help="spec-templates 根目录")
    parser.add_argument("--project-slug", default="", help="kebab-case 项目短名（默认取目录 basename）")
    parser.add_argument("--owner", default="user")
    parser.add_argument("--overwrite", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    ws = Path(args.workspace_root).resolve()
    slug_src = args.project_slug.strip() if args.project_slug else ws.name
    slug = slugify(slug_src)
    tr = Path(args.template_root).resolve()

    result = run_bootstrap(ws, tr, slug, args.owner.strip() or "user", args.overwrite, args.dry_run)

    if args.json or result.get("dry_run") or not result.get("ok"):
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        print(f"[spec-bootstrap] slug={slug} written={len(result.get('written_files', []))} skipped={len(result.get('skipped_files', []))}")
        for p in result.get("written_files", []):
            print(f"  + {p}")
        for p in result.get("skipped_files", []):
            print(f"  (skip exists) {p}")
        if result.get("errors"):
            for e in result["errors"]:
                print(f"  ! {e}")

    return 0 if result.get("ok") else 1


if __name__ == "__main__":
    sys.exit(main())
