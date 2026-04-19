#!/usr/bin/env python3
"""从属链验证器 - 检查每个 spec 的 parent 是否存在且层级正确"""

import sys
import yaml
from pathlib import Path

def load_specs(spec_dir):
    specs = {}
    for path in Path(spec_dir).rglob("*.yaml"):
        try:
            with open(path) as f:
                data = yaml.safe_load(f)
            if data and "spec" in data:
                name = data["spec"].get("name")
                level = data["spec"].get("level", "")
                parent = data["spec"].get("parent")
                specs[name] = {
                    "level": level,
                    "parent": parent,
                    "path": str(path),
                }
        except Exception as e:
            print(f"  PARSE ERROR: {path}: {e}", file=sys.stderr)
    return specs

def level_num(level_str):
    """Extract numeric level for comparison"""
    s = str(level_str)
    if s.startswith("1"):
        return 1
    elif s.startswith("2"):
        return 2
    elif s.startswith("3"):
        return 3
    elif s.startswith("4"):
        return 4
    elif s.startswith("5"):
        return 5
    return 0

def validate(specs):
    violations = []
    
    # L1 specs should have no parent
    for name, info in specs.items():
        if info["level"] in ("1_project-goal", "1"):
            if info["parent"] is not None:
                violations.append(f"L1 has parent: {name} -> {info['parent']}")
    
    for name, info in specs.items():
        parent = info["parent"]
        if parent is None:
            continue
        
        # Parent must exist
        if parent not in specs:
            violations.append(f"MISSING PARENT: {name} (L{info['level']}) -> parent '{parent}' not found")
            continue
        
        # Level must be parent.level + 1 (approximately)
        child_lv = level_num(info["level"])
        parent_lv = level_num(specs[parent]["level"])
        
        # Allow same-level (e.g., multiple L5 specs under same L4)
        # But L4 must have L3 parent, L3 must have L2 parent, etc.
        if child_lv > 0 and parent_lv > 0:
            if child_lv <= parent_lv:
                # Same-level is OK for L5 specs
                if not (child_lv == 5 and parent_lv == 4):
                    violations.append(f"LEVEL SKIP: {name} (L{info['level']}) -> parent {parent} (L{specs[parent]['level']})")
    
    return violations

def main():
    spec_dir = sys.argv[1] if len(sys.argv) > 1 else "specs"
    
    print(f"Loading specs from: {spec_dir}")
    specs = load_specs(spec_dir)
    print(f"Found {len(specs)} specs")
    
    violations = validate(specs)
    
    if violations:
        print(f"\n❌ {len(violations)} violation(s):")
        for v in violations:
            print(f"  {v}")
        sys.exit(1)
    else:
        print("\n✅ All parent chains valid!")
        print("\nSpec tree:")
        levels = {}
        for name, info in specs.items():
            lv = info["level"]
            if lv not in levels:
                levels[lv] = []
            levels[lv].append(name)
        
        for lv in sorted(levels.keys()):
            print(f"  L{lv}: {', '.join(sorted(levels[lv]))}")
        
        sys.exit(0)

if __name__ == "__main__":
    main()
