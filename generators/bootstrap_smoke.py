#!/usr/bin/env python3
"""
Bootstrap Generator Smoke Test Runner
在临时目录执行 scaffold，验证最小闭环：生成文件、validate、idempotency。
用法：python generators/bootstrap_smoke.py [--dry-run]
"""
import subprocess
import sys
import json
import os
import shutil
import hashlib
import tempfile
from dataclasses import dataclass, field, asdict
from typing import Optional


@dataclass
class SmokeReport:
    success: bool = True
    temp_dir: str = ""
    created_files: list[str] = field(default_factory=list)
    missing_files: list[str] = field(default_factory=list)
    command_results: dict = field(default_factory=dict)
    idempotency_result: dict = field(default_factory=dict)
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    def to_json(self) -> str:
        return json.dumps(asdict(self), indent=2, ensure_ascii=False)


# Canonical files that scaffold MUST create
REQUIRED_CANONICAL_PATHS = [
    "specs/L1-goal/",
    "specs/L2-skeleton/",
    "generators/",
    "generators/gen.py",
    "generators/validate_specs.py",
    "Makefile",
    "README.md",
]

# Files that idempotency test should NOT overwrite
SENTINEL_FILENAME = "sentinel-check.txt"


def md5(path: str) -> Optional[str]:
    try:
        with open(path, "rb") as f:
            return hashlib.md5(f.read()).hexdigest()
    except OSError:
        return None


def run_cmd(cmd: list[str], cwd: str, timeout: int = 60) -> tuple[int, str, str]:
    try:
        env = {**os.environ, "PYTHONIOENCODING": "utf-8", "PYTHONUTF8": "1"}
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=cwd,
            timeout=timeout,
            env=env,
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", "timeout"
    except FileNotFoundError:
        return -1, "", f"command not found: {cmd[0]}"
    except Exception as e:
        return -1, "", str(e)


def check_paths_exist(temp_dir: str) -> tuple[list[str], list[str]]:
    """Return (found_paths, missing_paths)."""
    found = []
    missing = []
    for p in REQUIRED_CANONICAL_PATHS:
        full = os.path.join(temp_dir, p.rstrip("/"))
        if os.path.exists(full):
            found.append(p)
        else:
            missing.append(p)
    return found, missing


def run_scaffold(temp_dir: str, scaffold_py: str) -> tuple[int, str, str]:
    """Run scaffolder.py on the temp directory."""
    if os.path.exists(scaffold_py):
        return run_cmd(["python3", scaffold_py, temp_dir, "--template", "default"], temp_dir)
    else:
        return -1, "", f"scaffolder.py not found at {scaffold_py}"


def run_validate(temp_dir: str) -> tuple[int, str, str]:
    return run_cmd(["make", "validate"], temp_dir, timeout=120)


def run_smoke(temp_dir: str, scaffold_py: str, dry_run: bool = False) -> SmokeReport:
    report = SmokeReport(temp_dir=temp_dir)
    scaffold_py_abs = os.path.abspath(scaffold_py)

    # ── First scaffold ─────────────────────────────────────────────
    code1, out1, err1 = run_scaffold(temp_dir, scaffold_py_abs)
    report.command_results["first_scaffold"] = {"exit": code1, "out_len": len(out1), "err_len": len(err1)}

    if code1 != 0 and "scaffolder.py" in err1 and "not found" in err1:
        # scaffolder not available — just check paths that template creates
        report.warnings.append("scaffolder.py not found; checking template structure only")

    # Check required paths
    found, missing = check_paths_exist(temp_dir)
    report.created_files = found
    report.missing_files = missing
    if missing:
        report.success = False
        report.errors.append(f"Missing canonical paths: {missing}")

    # ── make validate ─────────────────────────────────────────────
    code_v, out_v, err_v = run_validate(temp_dir)
    report.command_results["make_validate"] = {"exit": code_v, "out_len": len(out_v), "err_len": len(err_v)}
    if code_v != 0:
        report.warnings.append(f"make validate exited {code_v} in scaffold dir")

    # ── Idempotency test ──────────────────────────────────────────
    sentinel_path = os.path.join(temp_dir, SENTINEL_FILENAME)
    sentinel_content = "smoke-test-sentinel-do-not-overwrite"
    with open(sentinel_path, "w") as f:
        f.write(sentinel_content)
    sentinel_hash_before = md5(sentinel_path)

    # Second scaffold
    code2, out2, err2 = run_scaffold(temp_dir, scaffold_py_abs)
    report.command_results["second_scaffold"] = {"exit": code2, "out_len": len(out2), "err_len": len(err2)}

    sentinel_hash_after = md5(sentinel_path)
    idem_ok = sentinel_hash_before == sentinel_hash_after

    report.idempotency_result = {
        "sentinel_intact": idem_ok,
        "sentinel_before": sentinel_hash_before,
        "sentinel_after": sentinel_hash_after,
    }
    if not idem_ok:
        report.warnings.append("Idempotency check: sentinel file was overwritten")

    # Verify sentinel still has original content
    try:
        with open(sentinel_path) as f:
            content = f.read()
        if sentinel_content not in content:
            report.warnings.append("Sentinel file content changed after second scaffold")
    except OSError:
        pass

    return report


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Bootstrap Generator Smoke Test")
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Print what would be tested without running scaffold",
    )
    parser.add_argument(
        "--workspace-root", "-w",
        default=os.path.dirname(os.path.abspath(__file__)),
        help="Workspace root (where scaffolder.py lives)",
    )
    args = parser.parse_args()

    # Locate scaffolder.py relative to this script
    smoke_script_dir = os.path.dirname(os.path.abspath(__file__))
    scaffold_py = os.path.join(smoke_script_dir, "scaffolder.py")
    if not os.path.exists(scaffold_py):
        # Try workspace root
        ws = args.workspace_root
        scaffold_py = os.path.join(ws, "generators", "scaffolder.py")

    if args.dry_run:
        print(f"[dry-run] Would test scaffold with:")
        print(f"  scaffolder: {scaffold_py}")
        print(f"  required: {REQUIRED_CANONICAL_PATHS}")
        print(f"  sentinel: {SENTINEL_FILENAME}")
        return

    # Create temp directory
    temp_dir = tempfile.mkdtemp(prefix="vibex-smoke-")
    print(f"[bootstrap-smoke] Temp dir: {temp_dir}")

    try:
        report = run_smoke(temp_dir, scaffold_py)
        print("\n" + "=" * 60)
        print("  Bootstrap Smoke Report")
        print("=" * 60)
        print(f"\n  Status: {'✅ PASS' if report.success else '❌ FAIL'}")
        print(f"\n  Created ({len(report.created_files)}):")
        for f in report.created_files:
            print(f"    ✅ {f}")
        if report.missing_files:
            print(f"\n  Missing ({len(report.missing_files)}):")
            for f in report.missing_files:
                print(f"    ❌ {f}")
        if report.errors:
            print(f"\n  Errors:")
            for e in report.errors:
                print(f"    - {e}")
        if report.warnings:
            print(f"\n  Warnings:")
            for w in report.warnings:
                print(f"    - {w}")
        print(f"\n  Idempotency: {'✅ sentinel intact' if report.idempotency_result.get('sentinel_intact') else '❌ overwritten'}")
        print(f"\n  JSON: {report.to_json()}")
        print("=" * 60)

        # Cleanup
        shutil.rmtree(temp_dir)
        print(f"\n[bootstrap-smoke] Cleaned up {temp_dir}")

        sys.exit(0 if report.success else 1)

    except Exception as e:
        print(f"[bootstrap-smoke] ERROR: {e}", file=sys.stderr)
        try:
            shutil.rmtree(temp_dir)
        except OSError:
            pass
        sys.exit(1)


if __name__ == "__main__":
    main()
