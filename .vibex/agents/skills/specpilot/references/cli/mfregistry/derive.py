#!/usr/bin/env python3
"""
mf-derive-spec.py
────────────────────────────────────────────────────────────────────────────
从 MFRegistry manifest 派生 spec 的 mf_binding 块。

用法：
    python mf-derive-spec.py <spec_name> [--spec-file PATH]
    python mf-derive-spec.py L3-dashboard-kpi --spec-file .vibex/specs/L3-module/MOD-dashboard.yaml

原理：
    1. 调用 MFRegistry.resolve_from_spec(spec_name) 获取 manifest
    2. 从 manifest 提取 components[].contract（inputs/outputs/events）
    3. 生成 YAML mf_binding 块，追加或替换 spec 文件中的对应段落
    4. 计算 manifest_hash 用于漂移检测
    5. 输出 diff 供人工确认
"""
import argparse
import hashlib
import json
import os
import re
import sys
import yaml


def compute_hash(data: dict) -> str:
    s = json.dumps(data, sort_keys=True, ensure_ascii=True)
    return hashlib.sha256(s.encode()).hexdigest()[:16]


def generate_mf_binding_block(manifest: dict) -> str:
    """从 manifest 生成 YAML mf_binding 块"""
    lines = ['mf_binding:']
    lines.append(f'  source: mf_registry')
    lines.append(f'  manifest_hash: "{compute_hash(manifest)}"')
    lines.append('  components:')

    for comp in manifest.get('components', []):
        name = comp['component']
        source = comp.get('path', '')
        status = comp.get('status', 'unknown')
        contract = comp.get('contract', {})

        lines.append(f'    - name: {name}')
        lines.append(f'      source: "{source}"')

        if status == 'pending':
            lines.append(f'      _warn: component not registered in MFRegistry')

        if contract:
            inputs = contract.get('inputs', {})
            outputs = contract.get('outputs', [])
            events = contract.get('events', [])

            lines.append('      contract:')
            if inputs:
                lines.append('        inputs:')
                for key, schema in inputs.items():
                    if isinstance(schema, dict):
                        desc = schema.get('description', '').strip()
                        typ = schema.get('type', 'any')
                        default = schema.get('default')
                        lines.append(f'          {key}:')
                        lines.append(f'            type: {typ}')
                        if desc:
                            lines.append(f'            description: {desc}')
                        if default is not None:
                            lines.append(f'            default: {json.dumps(default)}')
                    else:
                        lines.append(f'          {key}: {json.dumps(schema)}')

            if outputs:
                lines.append('        outputs:')
                for out in outputs:
                    prefix = out.get('prefix', '')
                    desc = out.get('description', '').strip()
                    schema = out.get('schema')
                    lines.append(f'          - prefix: {prefix}')
                    if desc:
                        lines.append(f'            description: {desc}')
                    if schema:
                        lines.append(f'            schema: {json.dumps(schema)}')

            if events:
                lines.append('        events:')
                for ev in events:
                    ev_name = ev.get('event', '')
                    desc = ev.get('description', '').strip()
                    payload = ev.get('payload')
                    lines.append(f'          - event: {ev_name}')
                    if desc:
                        lines.append(f'            description: {desc}')
                    if payload:
                        lines.append(f'            payload: {json.dumps(payload)}')

    return '\n'.join(lines)


def read_spec(path: str) -> str:
    with open(path, encoding='utf-8') as f:
        return f.read()


def update_spec(spec_text: str, binding_block: str) -> tuple[str, str]:
    """
    将 binding_block 注入 spec_text 中。
    - 若已有 mf_binding 块，替换
    - 若没有，追加到文件末尾
    返回 (new_text, action)
    """
    marker = 'mf_binding:'
    marker_re = re.compile(r'^mf_binding:.*$', re.MULTILINE)

    if marker_re.search(spec_text):
        # 找到块首，删除整个旧块
        new_text = marker_re.sub(binding_block, spec_text, count=1)
        return new_text, 'replaced'
    else:
        # 追加
        return spec_text.rstrip() + '\n\n' + binding_block + '\n', 'appended'


def derive_from_registry(spec_name: str, spec_file: str | None = None) -> dict:
    """从 MFRegistry 加载 manifest 并派生"""
    # 延迟导入，避免本脚本独立运行时加载不必要的模块
    sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
    from mfregistry.core import MFRegistry
    registry = MFRegistry()
    manifest = registry.resolve_from_spec(spec_name)
    binding_block = generate_mf_binding_block(manifest)

    result = {
        'spec': spec_name,
        'manifest_hash': manifest.get('manifest_hash', ''),
        'components': manifest.get('components', []),
        'binding_block': binding_block,
    }

    if spec_file and os.path.exists(spec_file):
        spec_text = read_spec(spec_file)
        new_text, action = update_spec(spec_text, binding_block)
        result['spec_file'] = spec_file
        result['action'] = action
        result['new_text'] = new_text

        with open(spec_file, 'w', encoding='utf-8') as f:
            f.write(new_text)
        result['written'] = True

    return result


def main():
    parser = argparse.ArgumentParser(description='Derive spec mf_binding block from MFRegistry manifest')
    parser.add_argument('spec_name', help='Spec name to resolve (e.g. L3-dashboard-kpi)')
    parser.add_argument('--spec-file', '--spec', dest='spec_file',
                        help='Path to spec YAML file to update in-place')
    parser.add_argument('--dry-run', action='store_true',
                        help='Print diff without writing')
    args = parser.parse_args()

    result = derive_from_registry(args.spec_name, None if args.dry_run else args.spec_file)

    print(f"Spec: {result['spec']}")
    print(f"Manifest hash: {result['manifest_hash']}")
    print(f"Components ({len(result['components'])}):")
    for comp in result['components']:
        status = comp.get('status', '')
        contract = comp.get('contract', {})
        print(f"  - {comp['component']} [{status}]")
        if contract:
            in_count = len(contract.get('inputs', {}))
            out_count = len(contract.get('outputs', []))
            ev_count = len(contract.get('events', []))
            print(f"      contract: {in_count} inputs, {out_count} outputs, {ev_count} events")

    print()
    print("Generated mf_binding block:")
    print("─" * 60)
    print(result['binding_block'])
    print("─" * 60)

    if result.get('written'):
        print(f"\n[OK] Written to {result['spec_file']} ({result['action']})")
    elif args.dry_run:
        print("\n[DRY-RUN] No file written (--dry-run)")
    else:
        print("\n[SKIP] No spec file written (use --spec-file to update)")


if __name__ == '__main__':
    main()
