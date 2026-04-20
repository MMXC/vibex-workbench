#!/usr/bin/env python3
"""io_contract 验证器：
- active 状态的核心 spec（feature/behavior/service）→ 检查 content.io_contract 完整性
- draft 状态或 supplement 层（test/uiux）→ 跳过（允许无 io_contract）
- module 层 → 检查 io_contract + changelog
- goal/architecture 层 → 跳过（定义标准，不遵循标准）
"""
import sys
import yaml
import os
from pathlib import Path

FEATURE_IO_CONTRACT = ["input", "output", "boundary"]

def get_spec_meta(spec):
    """Extract spec metadata from either root or content level"""
    if spec.get("spec"):
        return spec.get("spec", {})
    content = spec.get("content", {})
    return content.get("spec", {})

def get_level_info(spec):
    """Get level category and status from spec metadata"""
    spec_meta = get_spec_meta(spec)
    level = spec_meta.get("level", "")
    status = spec_meta.get("status", "active")

    # Handle numeric prefixes: "1_project-goal", "2_architecture", "L1", "L3"
    level_base = level.split("_")[0].upper().replace("L", "")

    if level_base in ("1", "2"):
        return "architecture", status, level
    if level_base == "3":
        return "module", status, level
    if level_base in ("4", "5"):
        return "feature", status, level

    # Handle string-based levels
    if level.startswith("L1") or level.startswith("L2"):
        return "architecture", status, level
    if level.startswith("L3") or "module" in level:
        return "module", status, level
    if level.startswith("L4") or level.startswith("L5"):
        return "feature", status, level
    if "feature" in level or "service" in level or "uiux" in level or "data" in level or "test" in level:
        return "feature", status, level

    return "module", status, level

def validate_io_contract(spec_path, spec):
    """验证单个 spec 文件的 io_contract"""
    violations = []
    if not spec:
        return violations

    content = spec.get("content", {})
    spec_meta = get_spec_meta(spec)
    level_cat, status, level_raw = get_level_info(spec)
    spec_name = spec_meta.get("name", os.path.basename(spec_path))

    # Skip spec files without content (root-level skeletons)
    if not content and level_cat != "architecture":
        return []

    # Skip draft specs entirely
    if status == "draft":
        return []

    # Skip supplement layers (_test.yaml, _uiux.yaml) — they define test cases and UI layout
    if "_test" in spec_name or "_uiux" in spec_name:
        return []

    # Skip goal/architecture level specs — they define the standard
    if level_cat == "architecture":
        return []

    io_contract = content.get("io_contract", {})

    # Module specs (L3) — need io_contract + changelog
    if level_cat == "module":
        if not io_contract:
            violations.append("缺少 content.io_contract")
            return violations
        missing_io = [f for f in FEATURE_IO_CONTRACT if f not in io_contract]
        if missing_io:
            violations.append(f"缺少 io_contract 字段: {', '.join(missing_io)}")
        if "behavior" not in io_contract:
            violations.append("缺少 io_contract.behavior")
        if "changelog" not in io_contract:
            violations.append("缺少 io_contract.changelog")
        return violations

    # Feature/service/behavior/data specs — need io_contract with input/output/boundary
    if not io_contract:
        violations.append("缺少 content.io_contract")
        return violations

    missing_io = [f for f in FEATURE_IO_CONTRACT if f not in io_contract]
    if missing_io:
        violations.append(f"缺少 io_contract 字段: {', '.join(missing_io)}")

    # behavior layer specs need behavior + changelog
    if "behavior" in level_raw or "_behavior" in spec_name:
        if "behavior" not in io_contract:
            violations.append("缺少 io_contract.behavior")
        if "changelog" not in io_contract:
            violations.append("缺少 io_contract.changelog")

    return violations

def main():
    specs_dir = Path(sys.argv[1] if len(sys.argv) > 1 else "specs")
    all_violations = []
    checked = 0
    skipped = 0

    for spec_file in sorted(specs_dir.rglob("*.yaml")):
        try:
            with open(spec_file) as f:
                spec = yaml.safe_load(f)
        except yaml.YAMLError as e:
            all_violations.append((str(spec_file), [f"YAML 语法错误: {e}"]))
            continue

        violations = validate_io_contract(str(spec_file), spec)
        if violations:
            all_violations.append((str(spec_file), violations))
        else:
            spec_meta = get_spec_meta(spec)
            status = spec_meta.get("status", "active")
            name = spec_meta.get("name", os.path.basename(spec_file))
            level_cat, _, _ = get_level_info(spec)
            if status == "draft" or "_test" in name or "_uiux" in name or level_cat == "architecture":
                skipped += 1
            else:
                checked += 1

    print(f"✅ {checked} 个 spec 通过验证 ({skipped} 个已跳过)")
    if all_violations:
        print(f"❌ {len(all_violations)} 个 spec 有问题:")
        for path, viols in all_violations:
            for v in viols:
                print(f"  {os.path.basename(path)}: {v}")
        sys.exit(1)
    else:
        print(f"✅ 所有 active spec 包含完整的 io_contract 字段")
        sys.exit(0)

if __name__ == "__main__":
    main()
