#!/usr/bin/env python3
"""
workspace-bootstrap skill execute entry.
Thin wrapper around generators/spec_workspace_bootstrap.py.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


def resolve_repo_root() -> Path:
    # .../.agents/skills/workspace-bootstrap/scripts/execute.py
    # -> repo root is parents[4]
    return Path(__file__).resolve().parents[4]


def main() -> int:
    parser = argparse.ArgumentParser(description="workspace-bootstrap skill executor")
    parser.add_argument("--workspace-root", required=True)
    parser.add_argument("--project-slug", default="")
    parser.add_argument("--owner", default="user")
    parser.add_argument("--overwrite", action="store_true")
    parser.add_argument("--confirm", action="store_true")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    if not args.confirm:
        payload = {
            "ok": False,
            "error": "confirm must be true",
            "next_action": "set --confirm after agent clarification",
        }
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return 1

    repo_root = resolve_repo_root()
    bootstrap_script = repo_root / "generators" / "spec_workspace_bootstrap.py"
    if not bootstrap_script.exists():
        payload = {
            "ok": False,
            "error": f"bootstrap script missing: {bootstrap_script}",
        }
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return 1

    cmd = [
        "python3",
        str(bootstrap_script),
        args.workspace_root,
        "--json",
        "--owner",
        args.owner,
    ]
    if args.project_slug.strip():
        cmd += ["--project-slug", args.project_slug.strip()]
    if args.overwrite:
        cmd.append("--overwrite")

    proc = subprocess.run(cmd, cwd=str(repo_root), capture_output=True, text=True)
    out = proc.stdout.strip() or proc.stderr.strip()
    if not out:
        out = json.dumps({"ok": False, "error": "empty output from bootstrap script"}, ensure_ascii=False)

    # passthrough JSON preferred
    try:
        parsed = json.loads(out)
    except json.JSONDecodeError:
        parsed = {"ok": proc.returncode == 0, "raw": out}
    if proc.returncode != 0 and "ok" not in parsed:
        parsed["ok"] = False

    if args.json:
        print(json.dumps(parsed, ensure_ascii=False, indent=2))
    else:
        print(json.dumps(parsed, ensure_ascii=False, indent=2))
    return 0 if bool(parsed.get("ok")) else 1


if __name__ == "__main__":
    sys.exit(main())
