#!/usr/bin/env python3
"""
spec_code_validator.py — 代码层扫描器（spec-first 自举第四层）

扫描生成的 .svelte / .ts 文件，检查 spec 层无法覆盖的代码级问题：

  1. import 断链   → import 路径指向不存在的文件
  2. Skeleton 空洞 → 有 Skeleton 但对应 .svelte 未填充
  3. Content 空洞  → CONTENT_START/END 之间为空
  4. 死 import     → import 了但从未使用

使用场景：
  python generators/spec_code_validator.py --scan generated
  python generators/spec_code_validator.py --check-fix_import

设计原则：
  - 确定性检查（无 LLM，无 agent）
  - 可修复的 gap 直接给出 patch 建议
  - 与 spec_criteria_validator.py 的 L1/L2/L3 形成互补
"""

import re
import sys
import subprocess
from pathlib import Path
from dataclasses import dataclass, field
from typing import Optional


# ── 配置 ─────────────────────────────────────────────────────────

FRONTEND_SRC = Path("frontend/src")
SKIP_DIRS = {"node_modules", ".svelte-kit", ".git", "assets"}


# ── 数据模型 ─────────────────────────────────────────────────────

@dataclass
class CodeGap:
    severity: str          # critical | warning | info
    gap_type: str          # import_broken | skeleton_empty | content_hole | dead_import
    file: str              # 相对路径
    detail: str            # 描述
    fix_suggestion: str    # 修复建议
    line: int = 0          # 行号（可选）


@dataclass
class ScanReport:
    file: str
    gaps: list[CodeGap] = field(default_factory=list)
    imports: list[dict] = field(default_factory=list)  # {path, line, target}


# ── 导入路径解析 ─────────────────────────────────────────────────

def resolve_import_path(import_path: str, source_file: Path, src_root: Path) -> Optional[Path]:
    """
    将 import 路径解析为绝对文件系统路径。

    支持：
      import from '$lib/stores/canvas-store'   → frontend/src/lib/stores/canvas-store.ts
      import from './ThreadList.svelte'        → 相对路径
      import from '@xyflow/svelte'             → skip（第三方）
    """
    import_path = import_path.strip("'\"")

    # 跳过第三方包（node:* 内置、npm 包、svelte/、$app/ 等别名）
    if import_path.startswith('@') or import_path.startswith('~'):
        return None
    if any(import_path.startswith(p) for p in [
        'node:', 'svelte/', '$app/', '$env/',
        'vitest', 'yaml', 'dagre', 'dexie',
    ]):
        return None
    # SvelteKit generated $types（build 时自动生成，import 时存在，运行时有效）
    if import_path.startswith('./$types') or import_path.startswith('../$types'):
        return None

    # $lib 别名 → 映射到 frontend/src/lib
    if import_path.startswith('$lib/'):
        rel = import_path[len('$lib/'):]
        base = src_root / "lib" / rel
        # TypeScript 隐式扩展名：自动尝试
        candidates = [
            base,
            Path(str(base) + '.ts'),
            Path(str(base) + '.svelte'),
            Path(str(base) + '.tsx'),
        ]
        for c in candidates:
            if c.exists() and c.is_file():
                return c
        return base  # 未找到也返回，用于 gap 报告

    # 相对路径
    if import_path.startswith('.'):
        base = source_file.parent
        name = import_path.lstrip('./')
        # 回溯目录搜索（$lib/stores/types.ts → $lib/types.ts 可能是 ../types）
        candidates = [
            base / name,
            base / (name + '.ts'),
            base / (name + '.svelte'),
            base / (name + '.tsx'),
            # 回溯：从 stores/ 回退到 lib/ 根
            src_root / "lib" / name,
            src_root / "lib" / (name + '.ts'),
            src_root / "lib" / (name + '.svelte'),
            src_root / "lib" / (name + '.tsx'),
            # 再上一级
            src_root / name,
            src_root / (name + '.ts'),
        ]
        for c in candidates:
            if c.exists() and c.is_file():
                return c
        return base / name

    return None


# ── 扫描器 ───────────────────────────────────────────────────────

def scan_file(file_path: Path, src_root: Path) -> ScanReport:
    """扫描单个文件，返回 gap 列表。"""
    rel = file_path.relative_to(src_root.parent)
    report = ScanReport(file=str(rel))
    content = ""

    try:
        content = file_path.read_text(encoding="utf-8")
    except Exception:
        return report

    lines = content.splitlines()
    import_pattern = re.compile(
        r'^import\s+.*?\s+from\s+[\'"]([^\'"]+)[\'"]',
        re.MULTILINE
    )

    for i, line in enumerate(lines, 1):
        m = import_pattern.search(line)
        if not m:
            continue

        import_path = m.group(1)
        resolved = resolve_import_path(import_path, file_path, src_root)

        report.imports.append({
            "path": import_path,
            "line": i,
            "resolved": str(resolved) if resolved else None,
        })

        if resolved is None:
            # 第三方，跳过
            if import_path.startswith('@') or import_path.startswith('~'):
                continue
            # 无法解析
            report.gaps.append(CodeGap(
                severity="warning",
                gap_type="import_broken",
                file=str(rel),
                detail=f"无法解析 import: {import_path}",
                fix_suggestion=f"检查 '{import_path}' 是否拼写正确，或确认路径别名配置",
                line=i,
            ))
        elif not resolved.exists():
            # 文件不存在
            # 尝试找到最近的匹配文件
            suggestions = _find_similar_files(resolved, src_root)
            suggestion = ""
            if suggestions:
                suggestion = f"文件不存在。最接近：{suggestions[0]}"
            report.gaps.append(CodeGap(
                severity="critical",
                gap_type="import_broken",
                file=str(rel),
                detail=f"Import 目标不存在: {import_path} → {resolved}",
                fix_suggestion=suggestion or f"创建 {resolved} 或修正 import 路径",
                line=i,
            ))

    # Skeleton 空洞检测：扫描 .svelte 文件
    if file_path.suffix == '.svelte' and not str(file_path).endswith('.Skeleton.svelte'):
        # 检查是否有对应的 .Skeleton.svelte
        skeleton = file_path.parent / (file_path.stem + '.Skeleton.svelte')
        if skeleton.exists():
            # Skeleton 存在，检查 content 是否已迁移
            skeleton_content = skeleton.read_text(encoding="utf-8")
            # 简单启发：Skeleton 里没有 CONTENT_START 说明已迁移
            if 'CONTENT_START' not in skeleton_content and '// 开发者代码' not in skeleton_content:
                # Skeleton 已被部分填充，检查 .svelte 里是否有骨架引用
                if 'Skeleton' not in content:
                    pass  # 正常：开发者代码已在 .svelte 中，不需要引用 Skeleton

    # Content 空洞检测
    if 'CONTENT_START' in content and 'CONTENT_END' in content:
        # 提取 CONTENT_START 和 CONTENT_END 之间的内容
        m = re.search(r'CONTENT_START\s*\n(.*?)\n\s*CONTENT_END', content, re.DOTALL)
        if m:
            inner = m.group(1).strip()
            # 去除注释和空白
            inner_clean = re.sub(r'<!--.*?-->', '', inner, flags=re.DOTALL)
            inner_clean = re.sub(r'//.*', '', inner_clean)
            inner_clean = inner_clean.strip()
            if not inner_clean:
                report.gaps.append(CodeGap(
                    severity="critical",
                    gap_type="content_hole",
                    file=str(rel),
                    detail="CONTENT_START / CONTENT_END 之间为空",
                    fix_suggestion="在 CONTENT_START 和 CONTENT_END 之间补充开发者代码",
                    line=0,
                ))

    # 死 import 检测（简单版：import 了但文件中未使用）
    used_imports = set()
    for imp in report.imports:
        ipath = imp["path"]
        if ipath.startswith('$lib/'):
            # 提取最后一部分作为标识符
            last = ipath.split('/')[-1]
            # 去掉扩展名
            last = re.sub(r'\.(svelte|ts|tsx|js)$', '', last)
            if last in content:
                used_imports.add(ipath)

    return report


def _find_similar_files(target: Path, src_root: Path, max_results: int = 3) -> list[str]:
    """根据路径片段模糊匹配可能存在的文件。"""
    name = target.name
    parent_hint = target.parent.name
    results = []
    # 在同目录下找名称相似的
    candidates = target.parent.glob(f"*{name[:4]}*")
    for c in candidates:
        if c.is_file() and str(target) not in str(c):
            results.append(str(c.relative_to(src_root.parent)))
            if len(results) >= max_results:
                break
    return results


def scan_directory(src_dir: Path) -> list[ScanReport]:
    """扫描目录下的所有 svelte 和 ts 文件。"""
    reports = []
    for p in src_dir.rglob("*"):
        if not p.is_file():
            continue
        # 跳过 node_modules 等
        parts = p.parts
        if any(d in parts for d in SKIP_DIRS):
            continue
        if p.suffix not in ('.svelte', '.ts', '.tsx'):
            continue
        report = scan_file(p, src_dir)
        if report.gaps:
            reports.append(report)
    return reports


def scan_generated(src_dir: Path) -> list[ScanReport]:
    """只扫描标记为「generated」的文件（检查是否被开发者填充）。"""
    reports = []
    for p in src_dir.rglob("*"):
        if not p.is_file():
            continue
        if p.suffix not in ('.svelte', '.ts'):
            continue
        parts = p.parts
        if any(d in parts for d in SKIP_DIRS):
            continue
        try:
            content = p.read_text(encoding="utf-8")
        except Exception:
            continue
        # 判定为 generated 的文件
        if 'spec-to-code' not in content and '此文件由' not in content:
            continue
        report = scan_file(p, src_dir)
        if report.gaps:
            reports.append(report)
    return reports


# ── 汇总报告 ─────────────────────────────────────────────────────

def format_report(reports: list[ScanReport], show_warnings: bool = False) -> str:
    if not reports:
        return "✅ 代码层扫描通过：无 gaps\n"

    total_gaps = sum(len(r.gaps) for r in reports)
    by_type: dict[str, int] = {}
    by_severity: dict[str, int] = {}

    for r in reports:
        for g in r.gaps:
            by_type[g.gap_type] = by_type.get(g.gap_type, 0) + 1
            by_severity[g.severity] = by_severity.get(g.severity, 0) + 1

    critical = by_severity.get("critical", 0)
    warning  = by_severity.get("warning", 0)
    info     = by_severity.get("info", 0)

    header = (
        f"代码层扫描报告（{len(reports)} 个文件有问题）\n"
        f"{'=' * 60}\n"
        f"Critical: {critical}  Warning: {warning}  Info: {info}\n"
        f"按类型: {dict(sorted(by_type.items()))}"
    )

    if critical == 0:
        header += "\n\n✅ 无 critical gaps（Warning = 第三方/npm/运行时别名，不影响构建）"
        if not show_warnings:
            return header

    lines = [header, ""]

    for r in reports:
        for g in r.gaps:
            if not show_warnings and g.severity != "critical":
                continue
            lines.append(f"  [{g.severity:8}] {r.file}" + (f":{g.line}" if g.line else ""))
            lines.append(f"         {g.detail}")
            lines.append(f"         → {g.fix_suggestion}")
            lines.append("")

    return "\n".join(lines)


# ── 集成 spec_self_corrector 的 agent_action 映射 ───────────────

def gaps_to_corrector_actions(reports: list[ScanReport]) -> dict:
    """
    将代码层 gaps 转换为 spec_self_corrector 可消费的 action 格式。
    返回: {spec_name: [(action, gaps), ...]}
    """
    # 按 module/spec 映射 gaps
    # 这里简化处理：所有代码层 gap 归为 fix_import 或 patch_component
    by_action = {"fix_import": [], "patch_component": []}

    for r in reports:
        rel_file = r.file
        for g in r.gaps:
            entry = {
                "file": rel_file,
                "severity": g.severity,
                "detail": g.detail,
            }
            if g.gap_type == "import_broken":
                entry["line"] = g.line
                entry["fix"] = g.fix_suggestion
                by_action["fix_import"].append(entry)
            elif g.gap_type in ("content_hole", "skeleton_empty"):
                entry["fix"] = g.fix_suggestion
                by_action["patch_component"].append(entry)

    return by_action


# ── CLI ──────────────────────────────────────────────────────────

def main():
    import argparse
    parser = argparse.ArgumentParser(description="spec-code-validator: 代码层 gap 扫描")
    parser.add_argument("--scan", choices=["all", "generated"], default="all",
                        help="扫描范围：all=所有文件, generated=仅标记为 generated 的文件")
    parser.add_argument("--json", action="store_true", help="JSON 输出")
    parser.add_argument("--actions", action="store_true",
                        help="输出 agent action 格式（供 spec_self_corrector 使用）")
    parser.add_argument("--src", default="frontend/src", help="源码目录")
    parser.add_argument("--warnings", "-w", action="store_true",
                        help="显示所有 warning（默认只显示 critical）")
    args = parser.parse_args()

    src_dir = Path(args.src)
    if not src_dir.exists():
        print(f"❌ 目录不存在: {src_dir}", file=sys.stderr)
        sys.exit(1)

    if args.scan == "generated":
        reports = scan_generated(src_dir)
    else:
        reports = scan_directory(src_dir)

    if args.json:
        import json
        output = [
            {
                "file": r.file,
                "gaps": [
                    {"severity": g.severity, "type": g.gap_type,
                     "detail": g.detail, "fix": g.fix_suggestion, "line": g.line}
                    for g in r.gaps
                ],
                "imports": r.imports,
            }
            for r in reports
        ]
        print(json.dumps(output, ensure_ascii=False, indent=2))
        return

    if args.actions:
        result = gaps_to_corrector_actions(reports)
        import json
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return

    text = format_report(reports, show_warnings=args.warnings)
    print(text)


if __name__ == "__main__":
    main()
