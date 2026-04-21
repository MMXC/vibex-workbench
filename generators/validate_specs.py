#!/usr/bin/env python3
"""
validate_specs.py — VibeX Workbench Spec Validator
验证 spec YAML 语法 + 从属链完整性
"""
import sys
import yaml
from pathlib import Path
from collections import defaultdict

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")

LEVEL_ORDER = {"1_project_goal": 1, "2_architecture": 2, "3_module": 3, "4_feature": 4, "5a_uiux": 5, "5b_service": 5, "5c_data": 5, "5d_test": 5}
LEVEL_PARENT = {
    "2_architecture": "1_project_goal",
    "3_module": "2_architecture",
    "4_feature": "3_module",     # 支持 MOD-* parent（L3 module）
    "5a_uiux": "4_feature",
    "5b_service": "4_feature",
    "5c_data": "4_feature",
    "5d_test": "4_feature",
}

errors = []
warnings = []
specs = {}

def get_spec_path(level: str, name: str) -> Path:
    """根据 level 和 name 找到 spec 文件路径"""
    name_base = name.replace("-", "_").replace(" ", "_")
    
    if level == "1_project_goal":
        return SPEC_DIR / "project-goal" / f"{name}.yaml"
    elif level == "2_architecture":
        # 架构文件名为 architecture.yaml，但 name 可能不同
        arch_path = SPEC_DIR / "architecture" / f"{name}.yaml"
        if arch_path.exists():
            return arch_path
        return SPEC_DIR / "architecture" / "architecture.yaml"
    elif level == "3_module":
        # MOD-* module specs live in specs/module/
        if name.startswith("MOD-"):
            return SPEC_DIR / "module" / f"{name}_module.yaml"
        return SPEC_DIR / "module" / f"{name}_module.yaml"
    elif level.startswith("4_") or level == "4_feature":
        # MOD-* parent 映射到 L3 module 目录
        if name.startswith("MOD-"):
            return SPEC_DIR / "module" / f"{name}_module.yaml"
        return SPEC_DIR / "feature" / name / f"{name}_feature.yaml"
    elif level.startswith("5_"):
        feat = name.split("_")[0]
        fname = name
        return SPEC_DIR / "feature" / feat / fname
    return Path(name + ".yaml")

def validate_file(path: Path) -> dict | None:
    """解析并验证单个 spec 文件"""
    if not path.exists():
        return None
    try:
        with open(path) as f:
            raw = f.read()
        data = yaml.safe_load(raw)
        if data is None:
            return None
        spec_meta = data.get("spec", {})
        return {
            "path": path,
            "name": spec_meta.get("name"),
            "level": spec_meta.get("level"),
            "parent": spec_meta.get("parent"),
            "version": spec_meta.get("version"),
            "status": spec_meta.get("status"),
        }
    except yaml.YAMLError as e:
        errors.append(f"YAML 语法错误 {path}: {e}")
        return None
    except Exception as e:
        errors.append(f"读取错误 {path}: {e}")
        return None

def check_parent_chain(spec: dict):
    """验证从属链：parent 必须存在且 level 递减"""
    level = spec["level"]
    parent = spec["parent"]

    if level not in LEVEL_PARENT and level != "1_project_goal":
        return

    # meta_template 是模板元规格，不做从属链验证（parent 含 ${PLACEHOLDER}）
    if level == "meta_template":
        return
    
    if level == "1_project_goal":
        if parent is not None:
            errors.append(f"{spec['path']}: L1 不应有 parent，当前: {parent}")
        return
    
    expected_parent_type = LEVEL_PARENT.get(level)
    if not parent:
        errors.append(f"{spec['path']}: L{level} 缺少 parent 字段")
        return
    
    parent_path = get_spec_path(expected_parent_type, parent)
    if not parent_path.exists():
        errors.append(f"{spec['path']}: parent '{parent}' 未找到 ({parent_path})")
        return

def main():
    yaml_files = list(SPEC_DIR.rglob("*.yaml"))
    yaml_files = [f for f in yaml_files if "node_modules" not in str(f)]
    
    for path in yaml_files:
        spec = validate_file(path)
        if spec and spec["name"]:
            specs[spec["name"]] = spec
            check_parent_chain(spec)
    
    print(f"\n{'='*60}")
    print(f"VibeX Workbench Spec Validator")
    print(f"{'='*60}")
    print(f"扫描目录: {SPEC_DIR}")
    print(f"发现文件: {len(yaml_files)}")
    print(f"有效 Spec: {len(specs)}")
    
    if errors:
        print(f"\n{'❌ 错误 (' + str(len(errors)) + ')'}:")
        for e in errors:
            print(f"  {e}")
    
    if warnings:
        print(f"\n{'⚠️  警告 (' + str(len(warnings)) + ')'}:")
        for w in warnings:
            print(f"  {w}")
    
    if not errors and not warnings:
        print(f"\n✅ 所有 Spec 验证通过！")
    elif not errors:
        print(f"\n⚠️ 验证完成（{len(warnings)} 个警告）")
    else:
        print(f"\n❌ 验证失败（{len(errors)} 个错误）")
        sys.exit(1)
    
    print(f"\nSpec 清单:")
    for name, spec in sorted(specs.items(), key=lambda x: LEVEL_ORDER.get(x[1]["level"] or "0", 0)):
        lvl = spec["level"] or "?"
        parent = spec["parent"] or "—"
        status = spec["status"] or "?"
        print(f"  [{lvl}] {name} (parent: {parent}, status: {status})")

if __name__ == "__main__":
    main()
