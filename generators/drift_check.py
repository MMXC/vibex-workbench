#!/usr/bin/env python3
"""
drift_check.py — VibeX Workbench Schema Drift Detector
比较 spec 中的 entity/type 定义与生成的代码
"""
import sys
import yaml
from pathlib import Path
import re

SPEC_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")
SRC_DIR = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("frontend/src")

def extract_entity_names(spec_dir: Path) -> dict:
    """从所有 L4/L5 spec 中提取 entity/type 定义"""
    entities = {}
    for yaml_file in spec_dir.rglob("*.yaml"):
        if "node_modules" in str(yaml_file):
            continue
        try:
            with open(yaml_file) as f:
                data = yaml.safe_load(f)
            if not data:
                continue
            spec_meta = data.get("spec", {})
            level = spec_meta.get("level", "")
            
            # 从 L4 feature spec 中提取 entities
            if "4_feature" in level or level.startswith("4_"):
                content = data.get("content", {})
                for_key = content.get("entities", content.get("core_entities", {}))
                for name, entity in for_key.items():
                    entities[name] = {
                        "spec": str(yaml_file.relative_to(SPEC_DIR)),
                        "type": "entity",
                        "fields": list(entity.keys()) if isinstance(entity, dict) else [],
                    }
            
            # 从 L5 data spec 中提取 data types
            if "data" in level.lower() or "5c" in level:
                content = data.get("content", {})
                types = content.get("types", content.get("schemas", {}))
                for name, schema in types.items():
                    entities[name] = {
                        "spec": str(yaml_file.relative_to(SPEC_DIR)),
                        "type": "type",
                        "fields": list(schema.keys()) if isinstance(schema, dict) else [],
                    }
        except Exception as e:
            print(f"Warning: 无法读取 {yaml_file}: {e}", file=sys.stderr)
    return entities

def check_drift(entities: dict) -> list:
    """检查 spec → 代码漂移"""
    drift_reports = []
    
    # 扫描生成的代码文件
    if not SRC_DIR.exists():
        return drift_reports
    
    for yaml_name, entity_info in entities.items():
        # 查找对应的 .ts 代码文件
        snake = yaml_name.replace("-", "_")
        camel = re.sub(r"_([a-z])", lambda m: m.group(1).upper(), snake)
        
        found_in_code = False
        for ts_file in SRC_DIR.rglob("*.ts"):
            if snake in ts_file.name or camel in ts_file.name:
                found_in_code = True
                break
        
        if found_in_code:
            drift_reports.append({
                "entity": yaml_name,
                "spec": entity_info["spec"],
                "status": "aligned",
                "type": entity_info["type"],
            })
        else:
            drift_reports.append({
                "entity": yaml_name,
                "spec": entity_info["spec"],
                "status": "spec_only",
                "type": entity_info["type"],
            })
    
    return drift_reports

def main():
    print("=" * 60)
    print("VibeX Workbench — Schema Drift Detector")
    print("=" * 60)
    
    entities = extract_entity_names(SPEC_DIR)
    print(f"\n从 Spec 中提取 {len(entities)} 个 entity/type:")
    for name, info in sorted(entities.items()):
        print(f"  [{info['type']}] {name} ({info['spec']})")
    
    if not SRC_DIR.exists():
        print(f"\n⚠️  代码目录不存在: {SRC_DIR}")
        print("  (运行 'make generate' 生成代码后再检测漂移)")
        return
    
    reports = check_drift(entities)
    
    aligned = [r for r in reports if r["status"] == "aligned"]
    spec_only = [r for r in reports if r["status"] == "spec_only"]
    
    print(f"\n漂移分析:")
    print(f"  ✅ Spec ↔ Code 对齐: {len(aligned)}")
    print(f"  ⚠️  仅 Spec (未生成代码): {len(spec_only)}")
    
    if spec_only:
        print(f"\n以下 Spec 尚未生成对应代码:")
        for r in spec_only:
            print(f"  • {r['entity']} ({r['type']}) — {r['spec']}")
    
    if not aligned and not spec_only:
        print("\n⚠️  未找到任何 Spec-Code 对应关系")
        print("  (生成器可能需要适配当前项目结构)")
    
    print(f"\n{'='*60}")

if __name__ == "__main__":
    main()
