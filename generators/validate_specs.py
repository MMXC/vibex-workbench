#!/usr/bin/env python3
"""
validate_specs.py -- VibeX Workbench Spec Validator
Validates spec YAML syntax and basic parent-chain resolution.
"""
import sys
import yaml
from pathlib import Path

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")

LEVEL_ORDER = {
    "1_project_goal": 1,
    "1_project-goal": 1,
    "2_architecture": 2,
    "2_skeleton": 2,
    "3_module": 3,
    "4_feature": 4,
    "5_implementation": 5,
    "5_slice": 5,
    "5a_uiux": 5,
    "5b_service": 5,
    "5c_data": 5,
    "5d_test": 5,
}
LEVEL_PARENT = {
    "2_architecture": "1_project_goal",
    "2_skeleton": "1_project_goal",
    "3_module": "2_skeleton",
    "4_feature": "3_module",
    "5_implementation": "4_feature",
    "5_slice": "4_feature",
    "5a_uiux": "4_feature",
    "5b_service": "4_feature",
    "5c_data": "4_feature",
    "5d_test": "4_feature",
}

errors = []
warnings = []
specs = {}

def first_existing(candidates: list[Path]) -> Path:
    """Return the first existing path, or the first candidate as a useful error target."""
    for path in candidates:
        if path.exists():
            return path
    return candidates[0]

def get_spec_path(level: str, name: str) -> Path:
    """Resolve spec file path from level and name.
    Checks multiple candidate directories to support both legacy
    (specs/module/) and new (specs/L3-module/) layouts.
    """
    name_base = name.replace("-", "_").replace(" ", "_")
    hyphen_name = name_base.replace("_", "-")

    if level in ("1_project_goal", "1_project-goal"):
        return first_existing([
            SPEC_DIR / "L1-goal" / f"{name}.yaml",
            SPEC_DIR / "L1-goal" / f"{hyphen_name}.yaml",
            SPEC_DIR / "project-goal" / f"{name}.yaml",
            SPEC_DIR / "project-goal" / f"{hyphen_name}.yaml",
        ])
    elif level in ("2_architecture", "2_skeleton"):
        return first_existing([
            SPEC_DIR / "L2-skeleton" / f"{name}.yaml",
            SPEC_DIR / "L2-skeleton" / f"{hyphen_name}.yaml",
            SPEC_DIR / "architecture" / f"{name}.yaml",
            SPEC_DIR / "architecture" / f"{hyphen_name}.yaml",
            SPEC_DIR / "architecture" / "architecture.yaml",
        ])
    elif level == "3_module":
        return first_existing([
            SPEC_DIR / "L3-module" / f"{name}.yaml",
            SPEC_DIR / "L3-module" / f"{hyphen_name}.yaml",
            SPEC_DIR / "module" / f"{name}_module.yaml",
            SPEC_DIR / "module" / f"{name_base}_module.yaml",
        ])
    elif level.startswith("4_") or level == "4_feature":
        if name.startswith("MOD-"):
            return get_spec_path("3_module", name)
        return first_existing([
            SPEC_DIR / "L4-feature" / f"{name}.yaml",
            SPEC_DIR / "L4-feature" / f"{hyphen_name}.yaml",
            SPEC_DIR / "feature" / name / f"{name}_feature.yaml",
            SPEC_DIR / "feature" / hyphen_name / f"{hyphen_name}_feature.yaml",
        ])
    elif level.startswith("5_"):
        feat = name.split("_")[0]
        fname = name
        return first_existing([
            SPEC_DIR / "L5-slice" / f"{name}.yaml",
            SPEC_DIR / "L5-slice" / f"{hyphen_name}.yaml",
            SPEC_DIR / "feature" / feat / fname,
            SPEC_DIR / "feature" / feat / f"{fname}.yaml",
        ])
    return Path(name + ".yaml")

def validate_file(path: Path) -> dict | None:
    """Parse and validate a single spec file.
    Handles two YAML formats:
    - Markdown with YAML frontmatter (--- separator): extract frontmatter only
    - Pure YAML with top-level fields (new VibeX format, no spec: wrapper)
    """
    if not path.exists():
        return None
    try:
        with open(path, encoding="utf-8") as f:
            raw = f.read()
        # Strip Markdown YAML frontmatter: keep only content before first ---
        yaml_content = raw.split("\n---\n")[0].split("\n---\r\n")[0]
        data = yaml.safe_load(yaml_content)
        if data is None:
            return None
        # Support both nested spec: wrapper and flat root-level fields
        if isinstance(data, dict) and "spec" in data:
            spec_meta = data.get("spec", {})
        else:
            spec_meta = data  # new flat format: name/level/parent at root
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
            get_spec_path("2_skeleton", parent_name),
            get_spec_path("2_architecture", parent_name),
            get_spec_path("1_project-goal", parent_name),
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
    path = spec["path"]

    # p-metaspec/ files are meta-specs about specs themselves; skip parent chain
    if "p-metaspec" in str(path):
        return

    if level not in LEVEL_PARENT and level not in ("1_project_goal", "1_project-goal"):
        return

    if level == "meta_template":
        return

    if level in ("1_project_goal", "1_project-goal"):
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
