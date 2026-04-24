#!/usr/bin/env python3
"""
VibeX State Detector — 探测仓库根目录状态

检测信号：
  empty:   specs/ 不存在 AND generators/gen.py 不存在
  partial: specs/ 存在 AND generators/gen.py 不存在
  ready:   specs/ 存在 AND generators/gen.py 存在 AND Makefile 含 lint-specs

用法：
  python3 state_detector.py /path/to/workspace
  python3 state_detector.py /path/to/workspace --json
"""

import os
import sys
import json


def detect_state(workspace_root: str) -> dict:
    """返回仓库状态对象（empty | half | ready）。"""
    workspace_root = os.path.abspath(workspace_root)
    signals = []

    # Signal 1: specs/ 目录
    specs_path = os.path.join(workspace_root, "specs")
    specs_exists = os.path.isdir(specs_path)
    signals.append({
        "path": "specs/",
        "exists": specs_exists,
        "reason": "specs 目录存在" if specs_exists else "无 specs 目录"
    })

    # Signal 2: generators/gen.py
    gen_path = os.path.join(workspace_root, "generators", "gen.py")
    gen_exists = os.path.isfile(gen_path)
    signals.append({
        "path": "generators/gen.py",
        "exists": gen_exists,
        "reason": "生成器入口存在" if gen_exists else "无生成器入口"
    })

    # Signal 3: Makefile 含 lint-specs
    makefile_path = os.path.join(workspace_root, "Makefile")
    makefile_has_lint = False
    makefile_exists = os.path.isfile(makefile_path)
    if makefile_exists:
        with open(makefile_path, "r", encoding="utf-8") as f:
            content = f.read()
            makefile_has_lint = "lint-specs" in content or "lint_specs" in content
    signals.append({
        "path": "Makefile",
        "exists": makefile_exists,
        "reason": "lint-specs target 存在" if makefile_has_lint else "Makefile 存在但无 lint-specs"
    })

    # 综合判断状态
    if specs_exists and gen_exists and makefile_has_lint:
        state = "ready"
    elif specs_exists or gen_exists:
        state = "half"
    else:
        state = "empty"

    # 生成建议
    suggestions = {
        "empty": ["点击「初始化脚手架」开始搭建项目"],
        "half": ["运行「生成脚手架」补全目录结构", "或手动创建 generators/gen.py 和 Makefile"],
        "ready": ["在 spec 编辑器中打开或新建规格文件", "运行「校验」检查 spec 质量"],
    }[state]

    return {
        "state": state,
        "workspace_root": workspace_root,
        "signals": signals,
        "suggestions": suggestions
    }


def main():
    if len(sys.argv) < 2:
        print("Usage: state_detector.py <workspace_root> [--json]", file=sys.stderr)
        sys.exit(1)

    workspace = sys.argv[1]
    as_json = "--json" in sys.argv

    if not os.path.isdir(workspace):
        result = {
            "state": "error",
            "error": f"目录不存在: {workspace}",
            "workspace_root": workspace
        }
    else:
        result = detect_state(workspace)

    if as_json or not sys.stdout.isatty():
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        # 人类可读输出
        icons = {"empty": "⬜", "half": "🟨", "ready": "🟩", "error": "❌"}
        icon = icons.get(result["state"], "?")
        print(f"{icon} State: {result['state']}")
        print(f"   目录: {result['workspace_root']}")
        print("   检测信号:")
        for s in result.get("signals", []):
            status = "✅" if s["exists"] else "❌"
            print(f"     {status} {s['path']}: {s['reason']}")
        if "suggestions" in result:
            print("   建议:")
            for sug in result["suggestions"]:
                print(f"     → {sug}")
        if "error" in result:
            print(f"   错误: {result['error']}")

    # Exit code: 0=success, 1=error
    sys.exit(0 if result["state"] != "error" else 1)


if __name__ == "__main__":
    main()
