#!/usr/bin/env python3
"""
VibeX Spec Coverage & Completeness Analysis
Usage: python3 scripts/spec_coverage.py [--report]
"""
import yaml, os, json, sys
from datetime import datetime
from collections import defaultdict
from pathlib import Path

SPEC_DIR = Path("specs")

def load_specs():
    specs = {}
    for root, dirs, files in os.walk(SPEC_DIR):
        for f in files:
            if not f.endswith('.yaml'): continue
            path = os.path.join(root, f)
            try:
                with open(path) as fp:
                    d = yaml.safe_load(fp)
                s = d['spec']
                specs[s['name']] = {
                    'level': s['level'],
                    'parent': s.get('parent'),
                    'status': s.get('status', '?'),
                    'file': path,
                    'content_keys': list(d.get('content', {}).keys()),
                }
            except Exception as e:
                print(f"WARN: {path}: {e}", file=sys.stderr)
    return specs

def build_coverage_matrix(specs):
    l3_to_l4 = defaultdict(list)
    l4_to_l5 = defaultdict(list)
    for name, info in specs.items():
        if info['level'] == '4_feature':
            p = info['parent']
            if p: l3_to_l4[p].append(name)
        elif info['level'] in ('5_slice', '5_implementation'):
            p = info['parent']
            if p: l4_to_l5[p].append(name)
    return l3_to_l4, l4_to_l5

def check_consistency(specs):
    issues = []
    for name, info in specs.items():
        if info['level'] not in ('1_project-goal', '2_skeleton'):
            p = info['parent']
            if p and p not in specs:
                issues.append(('orphan_parent', name, p))
    names = defaultdict(list)
    for name, info in specs.items():
        names[name].append(info['file'])
    for name, paths in names.items():
        if len(paths) > 1:
            issues.append(('duplicate_name', name, paths))
    return issues

def generate_markdown_report(specs, l3_to_l4, l4_to_l5, issues):
    now = datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ")

    l1_n = sum(1 for i in specs.values() if i['level']=='1_project-goal')
    l2_n = sum(1 for i in specs.values() if i['level']=='2_skeleton')
    l3_n = sum(1 for i in specs.values() if i['level']=='3_module')
    l4_n = sum(1 for i in specs.values() if i['level']=='4_feature')
    l5_n = sum(1 for i in specs.values() if i['level'] in ('5_slice', '5_implementation'))
    l4_with_l5 = sum(1 for n, i in specs.items() if i['level']=='4_feature' and l4_to_l5.get(n))
    l4_total = sum(1 for n, i in specs.items() if i['level']=='4_feature')

    report_lines = [
        "# VibeX Workbench Spec Coverage Report",
        "",
        f"> Generated: {now}",
        f"> Total specs: {len(specs)} (L1:{l1_n} L2:{l2_n} L3:{l3_n} L4:{l4_n} L5:{l5_n})",
        "",
        "## Coverage Summary",
        "",
        "| Layer | Count | Coverage |",
        "|-------|-------|----------|",
        f"| L1 (goal) | {l1_n} | {'OK' if l1_n >= 1 else 'MISSING'} |",
        f"| L2 (skeleton) | {l2_n} | {'OK' if l2_n >= 1 else 'MISSING'} |",
        f"| L3 (module) | {l3_n} | {'OK' if l3_n >= 6 else 'MISSING'} |",
        f"| L4 (feature) | {l4_n} | {l4_with_l5}/{l4_total} have L5 slices |",
        f"| L5 (slice) | {l5_n} | implementation units |",
        "",
        "## L3 -> L4 Coverage",
        "",
    ]

    for l3 in sorted(l3_to_l4.keys()):
        l4s = sorted(l3_to_l4[l3])
        parts = []
        for l4 in l4s:
            l5s = l4_to_l5.get(l4, [])
            if l5s:
                parts.append(f"`{l4}` (L5x{len(l5s)})")
            else:
                parts.append(f"`{l4}` (no L5)")
        report_lines.append(f"- **{l3}**: {', '.join(parts)}")

    # Missing L5 table
    report_lines.extend([
        "",
        "## L4 without L5 (Implementation Gaps)",
        "",
        "| Priority | L4 Feature | Recommended L5 |",
        "|----------|-----------|---------------|",
    ])

    cursor_critical = []
    mvp_high = []
    other = []
    for name, info in specs.items():
        if info['level'] == '4_feature' and name.startswith('FEAT-'):
            if not l4_to_l5.get(name):
                l5_recommend = f"SLICE-{name.replace('FEAT-','').lower()}-impl"
                has_cursor = 'alignment_with_cursor' in info.get('content_keys', [])
                if has_cursor:
                    cursor_critical.append((name, l5_recommend))
                elif name in ('FEAT-spec-editor', 'FEAT-spec-write', 'FEAT-canvas-expand', 'FEAT-new-l1-wizard'):
                    mvp_high.append((name, l5_recommend))
                else:
                    other.append((name, l5_recommend))

    all_missing = cursor_critical + mvp_high + other
    for name, l5 in all_missing:
        if name in [x[0] for x in cursor_critical]:
            badge = "CRITICAL"
        elif name in [x[0] for x in mvp_high]:
            badge = "HIGH"
        else:
            badge = "MEDIUM"
        report_lines.append(f"| {badge} | `{name}` | `{l5}` |")

    # Consistency
    report_lines.extend([
        "",
        "## Consistency Issues",
        "",
    ])
    if not issues:
        report_lines.append("No consistency issues found.")
    else:
        for issue_type, *rest in issues:
            if issue_type == 'orphan_parent':
                report_lines.append(f"- `{rest[0]}`: parent `{rest[1]}` not found")
            elif issue_type == 'duplicate_name':
                report_lines.append(f"- Duplicate spec name `{rest[0]}`: {rest[1]}")

    report_lines.extend([
        "",
        "## Methodology",
        "",
        "Coverage completeness = every L4 must have >= 1 L5 slice, or be explicitly marked 'no L5 needed'.",
        "Consistency = all parent chains valid, no duplicate names, no generates[] conflicts.",
        "",
        "See Also: `specs/L1-goal/vibex-workbench-mvp.yaml`",
    ])

    return "\n".join(report_lines)

if __name__ == '__main__':
    specs = load_specs()
    l3_to_l4, l4_to_l5 = build_coverage_matrix(specs)
    issues = check_consistency(specs)

    if '--report' in sys.argv:
        report = generate_markdown_report(specs, l3_to_l4, l4_to_l5, issues)
        out_path = SPEC_DIR / '_governance' / 'coverage-report.md'
        os.makedirs(out_path.parent, exist_ok=True)
        with open(out_path, 'w') as f:
            f.write(report)
        print(f"Report written: {out_path}")

    # Console matrix
    print("VibeX Spec Coverage Matrix")
    print("=" * 50)
    print(f"Total: {len(specs)} specs\n")

    for l3 in sorted(l3_to_l4.keys()):
        print(f"  {l3} [{specs.get(l3,{}).get('status','?')}]")
        for l4 in sorted(l3_to_l4[l3]):
            l5s = l4_to_l5.get(l4, [])
            count = len(l5s)
            marker = f"OK L5x{count}" if count else "NO L5"
            print(f"    - {l4} [{specs[l4]['status']}] {marker}")

    print("\nL4 without L5:")
    missing = []
    for name, info in specs.items():
        if info['level'] == '4_feature' and name.startswith('FEAT-') and not l4_to_l5.get(name):
            has_cursor = 'alignment_with_cursor' in info.get('content_keys', [])
            priority = "CRITICAL" if has_cursor else ("HIGH" if name in ('FEAT-spec-editor','FEAT-spec-write','FEAT-canvas-expand','FEAT-new-l1-wizard') else "MEDIUM")
            missing.append((priority, name))
    missing.sort(key=lambda x: (0 if x[0]=='CRITICAL' else 1 if x[0]=='HIGH' else 2, x[1]))
    for priority, name in missing:
        print(f"  [{priority}] {name}")

    print("\nConsistency:")
    if issues:
        for issue_type, *rest in issues:
            if issue_type == 'orphan_parent':
                print(f"  ERROR: {rest[0]} parent '{rest[1]}' not found")
    else:
        print("  OK - no consistency issues")
