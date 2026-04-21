#!/usr/bin/env python3
"""
validate_specs.py -- VibeX Workbench Spec Validator
Validates spec YAML syntax and basic parent-chain resolution.
"""
import sys
import yaml
from pathlib import Path

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")

LEVEL_ORDER = {"1_project_goal": 1, "2_architecture": 2, "3_module": 3, "4_feature": 4, "5a_uiux": 5, "5b_service": 5, "5c_data": 5, "5d_test": 5}
LEVEL_PARENT = {
    "2_architecture": "1_project_goal",
    "3_module": "2_architecture",
    "4_feature": "3_module",
    "5a_uiux": "4_feature",
    "5b_service": "4_feature",
    "5c_data": "4_feature",
    "5d_test": "4_feature",
}

errors = []
warnings = []
specs = {}

def get_spec_path(level: str, name: str) -> Path:
    """Resolve spec file path from level and name."""
    name_base = name.replace("-", "_").replace(" ", "_")

    if level == "1_project_goal":
        return SPEC_DIR / "project-goal" / f"{name}.yaml"
    elif level == "2_architecture":
        arch_path = SPEC_DIR / "architecture" / f"{name}.yaml"
        if arch_path.exists():
            return arch_path
        return SPEC_DIR / "architecture" / "architecture.yaml"
    elif level == "3_module":
        if name.startswith("MOD-"):
            return SPEC_DIR / "module" / f"{name}_module.yaml"
        return SPEC_DIR / "module" / f"{name}_module.yaml"
    elif level.startswith("4_") or level == "4_feature":
        if name.startswith("MOD-"):
            return SPEC_DIR / "module" / f"{name}_module.yaml"
        return SPEC_DIR / "feature" / name / f"{name}_feature.yaml"
    elif level.startswith("5_"):
        feat = name.split("_")[0]
        fname = name
        return SPEC_DIR / "feature" / feat / fname
    return Path(name + ".yaml")

def validate_file(path: Path) -> dict | None:
    """Parse and validate a single spec file."""
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
        errors.append(f"YAML error {path}: {e}")
        return None
    except Exception as e:
        errors.append(f"Read error {path}: {e}")
        return None

def resolve_parent_path(child_level: str, parent_name: str) -> Path | None:
    """
    For L4 features, parent may be MOD-* (module) or another L4 aggregate (e.g. workbench-shell).
    Other levels use LEVEL_PARENT -> get_spec_path.
    """
    if child_level == "4_feature":
        candidates = (
            get_spec_path("3_module", parent_name),
            get_spec_path("4_feature", parent_name),
            get_spec_path("2_architecture", parent_name),
            SPEC_DIR / "project-goal" / f"{parent_name}.yaml",
        )
        for p in candidates:
            if p.exists():
                return p
        return candidates[0]
    expected = LEVEL_PARENT.get(child_level)
    if not expected:
        return None
    return get_spec_path(expected, parent_name)


def check_parent_chain(spec: dict):
    """Ensure parent exists on disk (resolved path)."""
    level = spec["level"]
    parent = spec["parent"]

    if level not in LEVEL_PARENT and level != "1_project_goal":
        return

    if level == "meta_template":
        return

    if level == "1_project_goal":
        if parent is not None:
            errors.append(f"{spec['path']}: L1 must not have parent; got: {parent}")
        return

    if not parent:
        errors.append(f"{spec['path']}: L{level} missing parent field")
        return

    parent_path = resolve_parent_path(level, parent)
    if parent_path is None:
        errors.append(f"{spec['path']}: cannot resolve parent '{parent}' (level={level})")
        return
    if not parent_path.exists():
        errors.append(f"{spec['path']}: parent '{parent}' not found ({parent_path})")
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
    print(f"Scan root: {SPEC_DIR}")
    print(f"YAML files: {len(yaml_files)}")
    print(f"Specs with name: {len(specs)}")

    if errors:
        print(f"\n[ERROR] ({len(errors)}):")
        for e in errors:
            print(f"  {e}")

    if warnings:
        print(f"\n[WARN] ({len(warnings)}):")
        for w in warnings:
            print(f"  {w}")

    if not errors and not warnings:
        print(f"\n[OK] All specs passed.")
    elif not errors:
        print(f"\n[WARN] Done with {len(warnings)} warning(s).")
    else:
        print(f"\n[FAIL] {len(errors)} error(s)")
        sys.exit(1)

    print(f"\nSpec index:")
    for name, spec in sorted(specs.items(), key=lambda x: LEVEL_ORDER.get(x[1]["level"] or "0", 0)):
        lvl = spec["level"] or "?"
        parent = spec["parent"] if spec["parent"] is not None else "-"
        status = spec["status"] or "?"
        print(f"  [{lvl}] {name} (parent: {parent}, status: {status})")

if __name__ == "__main__":
    main()
