#!/usr/bin/env python3
"""
VibeX Spec 批量结构对比工具
用法:
  python3 spec-compare.py --dir DIR          # 分析目录下所有yaml
  python3 spec-compare.py --dir-a A --dir-b B  # 两目录对比

输出: 各维度均值对比表 + sections 集合差异
"""
import yaml, glob, argparse, statistics, sys, os

def load_dir(pattern):
    """按glob pattern加载所有yaml，返回 (filename->spec, filename->path)"""
    specs, paths = {}, {}
    for f in sorted(glob.glob(pattern)):
        name = os.path.basename(f)
        try:
            docs = list(yaml.safe_load_all(open(f)))
            if docs and docs[0]:
                specs[name] = docs[0]
                paths[name] = f
        except Exception as e:
            print(f"  ⚠ {name}: {e}", file=sys.stderr)
    return specs, paths

def sections(spec):
    return set(k for k in spec.keys() if k not in ('spec', 'meta'))

def count_behaviors(spec):
    b = spec.get('content', {}).get('behaviors', [])
    return len(b) if isinstance(b, list) else 0

def count_us(spec):
    u = spec.get('content', {}).get('user_stories', [])
    return len(u) if isinstance(u, list) else 0

def count_tc(spec):
    t = spec.get('content', {}).get('test_scenarios', [])
    return len(t) if isinstance(t, list) else 0

def count_con(spec):
    c = spec.get('content', {}).get('constraints', [])
    return len(c) if isinstance(c, list) else 0

def io_lines(spec):
    io = spec.get('io_contract', {})
    if isinstance(io, dict):
        inp = [l for l in str(io.get('input', '')).split('\n') if l.strip()]
        out = [l for l in str(io.get('output', '')).split('\n') if l.strip()]
        return len(inp), len(out)
    return 0, 0

def avg(lst):
    return round(statistics.mean(lst), 1) if lst else 0

def analyze(specs, paths, label):
    all_sects = set()
    for s in specs.values():
        all_sects |= sections(s)
    lines = [len(open(paths[n]).readlines()) for n in specs if n in paths]
    io_in  = [io_lines(v)[0] for v in specs.values()]
    io_out = [io_lines(v)[1] for v in specs.values()]
    return {
        'count':    len(specs),
        'sections': sorted(all_sects),
        'avg_lines': avg(lines),
        'avg_sects': avg([len(sections(v)) for v in specs.values()]),
        'avg_bhv':  avg([count_behaviors(v) for v in specs.values()]),
        'avg_us':   avg([count_us(v) for v in specs.values()]),
        'avg_tc':   avg([count_tc(v) for v in specs.values()]),
        'avg_con':  avg([count_con(v) for v in specs.values()]),
        'avg_io_in': avg(io_in),
        'avg_io_out':avg(io_out),
    }

def compare(a_specs, a_paths, b_specs, b_paths, label_a, label_b):
    a = analyze(a_specs, a_paths, label_a)
    b = analyze(b_specs, b_paths, label_b)
    print(f"\n{'═'*60}")
    print(f"  {label_a} ({a['count']}份) vs {label_b} ({b['count']}份)")
    print(f"{'═'*60}")
    print(f"  {'维度':<16} {label_a:>10} {label_b:>10} {'胜出':>10}")
    print(f"  {'─'*48}")
    for lbl, av, bv in [
        ("总行数",          a['avg_lines'],  b['avg_lines']),
        ("顶层sections数",  a['avg_sects'],  b['avg_sects']),
        ("behaviors数",    a['avg_bhv'],    b['avg_bhv']),
        ("user_stories数", a['avg_us'],     b['avg_us']),
        ("test_scenarios数",a['avg_tc'],    b['avg_tc']),
        ("constraints数",  a['avg_con'],    b['avg_con']),
        ("io input行数",   a['avg_io_in'],  b['avg_io_in']),
        ("io output行数",  a['avg_io_out'], b['avg_io_out']),
    ]:
        winner = "★ A" if av > bv else ("★ B" if bv > av else "≈")
        print(f"  {lbl:<16} {av:>10} {bv:>10} {winner:>10}")
    print(f"  {'─'*48}")
    print(f"  {label_a} sections: {a['sections']}")
    print(f"  {label_b} sections: {b['sections']}")
    only_a = sorted(set(a['sections']) - set(b['sections']))
    only_b = sorted(set(b['sections']) - set(a['sections']))
    if only_a: print(f"  {label_a}独有: {only_a}")
    if only_b: print(f"  {label_b}独有: {only_b}")

if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--dir", help="单个目录（分析所有yaml）")
    ap.add_argument("--dir-a", help="目录A")
    ap.add_argument("--dir-b", help="目录B")
    args = ap.parse_args()

    def load(p):
        p = p.rstrip('/')
        return load_dir(p + '/*.yaml')

    if args.dir:
        specs, paths = load(args.dir)
        if not specs: print("❌ 目录为空"); sys.exit(1)
        a = analyze(specs, paths, args.dir)
        print(f"\n{'═'*60}\n  {args.dir} ({a['count']}份)\n{'═'*60}")
        for k in ['avg_lines','avg_sects','avg_bhv','avg_us','avg_tc','avg_con','avg_io_in','avg_io_out']:
            print(f"  {k:<18} {a[k]:>10}")
        print(f"  sections: {a['sections']}")

    elif args.dir_a and args.dir_b:
        a_s, a_p = load(args.dir_a)
        b_s, b_p = load(args.dir_b)
        if not a_s or not b_s: print("❌ 目录为空"); sys.exit(1)
        compare(a_s, a_p, b_s, b_p, args.dir_a, args.dir_b)
    else:
        print(__doc__)
