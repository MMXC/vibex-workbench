#!/usr/bin/env python3
"""从 specs 生成 Mermaid 从属关系图"""

import sys
import yaml
import argparse
from pathlib import Path

LEVEL_COLORS = {
    "1": "red",
    "1_project-goal": "red",
    "2": "orange",
    "2_skeleton": "orange",
    "3": "yellow",
    "3_module": "yellow",
    "4": "green",
    "4_feature": "green",
    "5": "blue",
    "5a": "blue",
    "5b": "blue",
    "5c": "blue",
    "5d": "blue",
}

def get_color(level):
    for key, color in LEVEL_COLORS.items():
        if key in str(level):
            return color
    return "gray"

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
                }
        except Exception:
            pass
    return specs

def generate_mermaid(specs, output_file=None):
    lines = ["flowchart TB"]
    lines.append('    subgraph spec_hierarchy["VibeX Workbench Spec Hierarchy"]')
    
    # Add nodes
    for name, info in specs.items():
        color = get_color(info["level"])
        label = name.replace("-", " ").replace("_", " ")
        node_id = name.replace("-", "_").replace(" ", "_")
        lines.append(f'        {node_id}["{label}"]:::lv{info["level"][0]}')
    
    lines.append("    end")
    
    # Add edges
    for name, info in specs.items():
        if info["parent"]:
            parent_id = info["parent"].replace("-", "_").replace(" ", "_")
            child_id = name.replace("-", "_").replace(" ", "_")
            lines.append(f"    {parent_id} --> {child_id}")
    
    # Styles
    for lvl, color in [("1", "red"), ("2", "orange"), ("3", "yellow"), ("4", "green"), ("5", "blue")]:
        lines.append(f'    classDef lv{lvl} fill:#f87171,stroke:#{color},stroke-width:3px,color:#000')
    
    mermaid = "\n".join(lines)
    
    if output_file:
        with open(output_file, "w") as f:
            f.write(mermaid)
        print(f"Mermaid graph written to: {output_file}")
    
    return mermaid

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("spec_dir")
    parser.add_argument("--output", "-o")
    args = parser.parse_args()
    
    specs = load_specs(args.spec_dir)
    print(f"Loaded {len(specs)} specs")
    
    mermaid = generate_mermaid(specs, args.output)
    if not args.output:
        print(mermaid)

if __name__ == "__main__":
    main()
