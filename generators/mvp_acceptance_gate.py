#!/usr/bin/env python3
"""
MVP Acceptance Gate Runner
输入：make validate + spec_criteria_validator 输出
输出：MVP usable / blocked 摘要（JSON + 可读摘要）
用法：python generators/mvp_acceptance_gate.py [--workspace-root DIR]
"""
import subprocess
import sys
import json
import os
import re
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class GateResult:
    usable: bool = True  # default usable unless blocked
    critical_failures: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    manual_checks: list[dict] = field(default_factory=list)
    details: dict = field(default_factory=dict)

    def to_json(self) -> str:
        return json.dumps({
            "usable": self.usable,
            "critical_failures": self.critical_failures,
            "warnings": self.warnings,
            "manual_checks": self.manual_checks,
            "details": self.details,
        }, indent=2, ensure_ascii=False)


def run_command(cmd: list[str], cwd: Optional[str] = None, timeout: int = 120) -> tuple[int, str, str]:
    """Run a shell command, return (exit_code, stdout, stderr)."""
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=cwd,
            timeout=timeout,
            env={**os.environ, "PYTHONIOENCODING": "utf-8", "PYTHONUTF8": "1"},
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", "Command timed out"
    except FileNotFoundError:
        return -1, "", f"Command not found: {cmd[0]}"
    except Exception as e:
        return -1, "", str(e)


def check_validate(ws_root: str) -> tuple[int, str]:
    """Run make validate, return (exit_code, output)."""
    return run_command(["make", "validate"], cwd=ws_root, timeout=180)


def check_criteria(ws_root: str) -> tuple[int, str]:
    """Run spec_criteria_validator with critical-only, return (exit_code, output)."""
    script = os.path.join(ws_root, "generators", "spec_criteria_validator.py")
    return run_command(
        ["python3", script, "--level", "L1L2L3", "--critical-only"],
        cwd=ws_root, timeout=120,
    )


def check_scaffold_smoke(ws_root: str) -> tuple[int, str]:
    """Check scaffold script exists and is valid Python."""
    script = os.path.join(ws_root, "generators", "scaffolder.py")
    if not os.path.exists(script):
        return 1, f"scaffolder.py not found at {script}"
    # syntax check
    code, out, err = run_command(["python3", "-m", "py_compile", script])
    if code != 0:
        return code, f"scaffolder.py syntax error: {err}"
    return 0, "scaffolder.py syntax OK"


def check_frontend_build(ws_root: str) -> tuple[int, str]:
    """Check frontend package.json and basic structure."""
    frontend_dir = os.path.join(ws_root, "frontend")
    pkg = os.path.join(frontend_dir, "package.json")
    if not os.path.exists(pkg):
        return 1, "frontend/package.json not found"
    return 0, "frontend/package.json exists"


def run_gate(ws_root: str) -> GateResult:
    """Run all MVP gate checks."""
    result = GateResult()
    ws_root = ws_root.rstrip("/")

    # 1. make validate
    code, out, err = check_validate(ws_root)
    result.details["make_validate"] = {"exit": code, "output_len": len(out)}
    if code != 0:
        result.critical_failures.append(f"make validate failed (exit {code})")
        result.usable = False
    else:
        result.warnings.append("make validate passed — but review spec quality manually")

    # 2. spec_criteria_validator --critical-only
    code2, out2, err2 = check_criteria(ws_root)
    result.details["criteria_validator"] = {"exit": code2, "output_len": len(out2)}
    # parse critical gaps from output
    m = re.search(r"(\d+)\s+critical", out2)
    if m and int(m.group(1)) > 0:
        result.critical_failures.append(f"spec_criteria_validator: {m.group(1)} critical gaps found")
        result.usable = False
    elif code2 != 0:
        result.warnings.append("spec_criteria_validator returned non-zero (may have parse errors)")

    # 3. scaffolder smoke
    code3, out3 = check_scaffold_smoke(ws_root)
    result.details["scaffolder_smoke"] = {"exit": code3}
    if code3 != 0:
        result.warnings.append(f"scaffolder.py issue: {out3}")

    # 4. frontend structure
    code4, out4 = check_frontend_build(ws_root)
    result.details["frontend_structure"] = {"exit": code4}
    if code4 != 0:
        result.warnings.append(f"frontend/package.json: {out4}")

    # 5. manual checks — Wails runtime, UI screenshots
    result.manual_checks = [
        {
            "id": "wails-runtime",
            "description": "Wails 运行时检查：运行 make wails-dev 后 WebView2 窗口正常打开",
            "status": "pending",
        },
        {
            "id": "workspace-empty-flow",
            "description": "空仓库流程：选择一个空目录 → 检测状态为 empty → 初始化脚手架 → 进入 workbench",
            "status": "pending",
        },
        {
            "id": "spec-edit-flow",
            "description": "Spec 编辑流程：在 workbench 打开 spec → 点击编辑 → 修改 → 保存 → 刷新页面内容一致",
            "status": "pending",
        },
        {
            "id": "new-l1-wizard-flow",
            "description": "新建 L1 流程：点击新建 L1 → 填写表单 → 生成文件 → 文件出现在 specs/L1-goal/ 下",
            "status": "pending",
        },
    ]

    return result


def print_summary(result: GateResult):
    """Print human-readable summary."""
    print("\n" + "=" * 60)
    print("  MVP Acceptance Gate Summary")
    print("=" * 60)

    status = "✅ USABLE" if result.usable else "❌ BLOCKED"
    print(f"\n  Overall: {status}")

    if result.critical_failures:
        print(f"\n  Critical failures ({len(result.critical_failures)}):")
        for f in result.critical_failures:
            print(f"    - {f}")
    else:
        print("\n  Critical failures: none")

    if result.warnings:
        print(f"\n  Warnings ({len(result.warnings)}):")
        for w in result.warnings:
            print(f"    - {w}")
    else:
        print("\n  Warnings: none")

    print(f"\n  Manual checks ({len(result.manual_checks)}):")
    for mc in result.manual_checks:
        print(f"    [{mc['status']:8s}] {mc['id']}")
        print(f"                 {mc['description']}")

    print("\n" + "=" * 60)
    print(f"  JSON output: {result.to_json()}")
    print("=" * 60 + "\n")


def main():
    import argparse
    parser = argparse.ArgumentParser(description="MVP Acceptance Gate Runner")
    parser.add_argument(
        "--workspace-root", "-w",
        default=os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
        help="Workspace root directory (default: parent of generators/)",
    )
    args = parser.parse_args()

    if not os.path.isdir(args.workspace_root):
        print(f"ERROR: workspace root not found: {args.workspace_root}", file=sys.stderr)
        sys.exit(1)

    result = run_gate(args.workspace_root)
    print_summary(result)

    # Exit code: 0 = usable, 1 = blocked
    sys.exit(0 if result.usable else 1)


if __name__ == "__main__":
    main()
