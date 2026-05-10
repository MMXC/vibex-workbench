#!/usr/bin/env python3
"""
Digital Wallpaper Spec Code Generator
Generates types, components, and stubs from spec YAML files
"""

import os
import yaml
from pathlib import Path

SPECS_DIR = Path("specs")
OUTPUT_DIR = Path("generated")


def load_specs():
    """Load all spec YAML files"""
    specs = {}
    for spec_file in SPECS_DIR.rglob("*.yaml"):
        rel_path = spec_file.relative_to(SPECS_DIR)
        specs[str(rel_path)] = yaml.safe_load(spec_file.read_text())
    return specs


def generate_types(specs):
    """Generate TypeScript types from specs"""
    types = ["// Auto-generated types from specs\n"]
    types.append("export interface SpecBase {")
    types.append("  id: string;")
    types.append("  name: string;")
    types.append("  description: string;")
    types.append("}\n")
    return "\n".join(types)


def main():
    """Main generator entry point"""
    print("Digital Wallpaper Code Generator")
    print(f"Specs directory: {SPECS_DIR}")
    print(f"Output directory: {OUTPUT_DIR}")
    
    specs = load_specs()
    print(f"Loaded {len(specs)} specs")


if __name__ == "__main__":
    main()
