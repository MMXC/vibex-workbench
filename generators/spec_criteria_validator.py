#!/usr/bin/env python3
"""
spec_criteria_validator.py -- VibeX Spec Criteria Validator
spec-first 自举循环的核心验证层。

设计原则（对应用户的 spec-first 理念）：
  - Spec 先行：criteria 在 spec YAML 中定义，验证器不猜
  - 允许不完美：L1/L2 不通过 → 标记 gap，不阻塞（除非 critical）
  - Agent 聚焦关键区域：gap 分类 → critical 推给 agent，structural 自愈
  - 三层分级：
    L1 (structural)    : YAML 语法 + parent chain   → validate_specs.py 已有
    L2 (semantic)      : 文件存在 + import 可达       → 本文件
    L3 (behavioral)    : acceptance_criteria          → 需 Agent/测试运行时

输出结构（ goskill 风格）：
  {
    "success": bool,
    "spec_name": str,
    "criteria_report": {
        "L1": { "passed": bool, "checks": [...] },
        "L2": { "passed": bool, "checks": [...] },
        "L3": { "passed": bool, "checks": [...] }
    },
    "gaps": [
        {
            "severity": "critical|warning|info",
            "layer": "L1|L2|L3",
            "criterion_id": "...",
            "message": "...",
            "agent_action": "fix_import|patch_component|write_test|skip"
        }
    ],
    "agent_targets": [gap for gap in gaps if gap.severity == "critical"]
  }

使用方式：
  python generators/spec_criteria_validator.py [--spec <name>|--all] [--level L1|L2|L3|L1L2]
"""
import sys
import re
import subprocess
from pathlib import Path
from dataclasses import dataclass, field, asdict
from typing import Literal

# ── 路径配置 ────────────────────────────────────────────────
SPEC_DIR = Path("specs")
GEN_DIR = Path("frontend/src")
TEMPLATE_DIR = Path("generators/templates")
MANIFEST_PATH = Path("generators/.manifest.json")


# ── 数据模型 ─────────────────────────────────────────────────
@dataclass
class Gap:
    severity: Literal["critical", "warning", "info"]
    layer: Literal["L1", "L2", "L3"]
    criterion_id: str
    message: str
    agent_action: Literal["fix_import", "patch_component", "write_test",
                          "self_heal", "skip", "notify"] = "skip"
    spec_path: str = ""


@dataclass
class CriteriaReport:
    spec_name: str
    L1: dict = field(default_factory=dict)
    L2: dict = field(default_factory=dict)
    L3: dict = field(default_factory=dict)
    gaps: list = field(default_factory=list)
    success: bool = False

    def to_dict(self):
        d = asdict(self)
        d["success"] = self.success
        return d


# ── Manifest reader ───────────────────────────────────────────
# 读取 gen.py 输出的 .manifest.json，提供生成物 ground truth
def load_manifest() -> dict:
    """读取 manifest，返回 dict；不存在时返回空结构。"""
    if not MANIFEST_PATH.exists():
        return {}
    try:
        return json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    except Exception:
        return {}


# ── L2 验证：语义层 ───────────────────────────────────────────
def check_imports_reachable(spec_name: str, content: dict,
                             check_generated_files: bool = False) -> list[Gap]:
    """检查 spec 声明的组件/模块在生成目录中是否可导入。

    L2 语义层验证（对应 spec-first 理念）：
    - 读取 manifest.json 了解实际生成了哪些文件（ground truth）
    - spec 可声明 generated_outputs（可选），检查这些是否与 manifest 对齐
    - 检查 manifest 中的文件是否存在 import 断链
    - 不依赖 spec 的 generated_outputs 也能工作（manifest 是 ground truth）

    check_generated_files: 若为 True，严格检查 manifest vs spec 声明
    """
    gaps = []
    spec_path = content.get("spec", {}).get("path", "")
    level = content.get("spec", {}).get("level", "")
    manifest = load_manifest()
    generated = set(manifest.get("all_files", []))

    if level not in ("4_feature", "5a_uiux", "5b_service", "5c_data", "5d_test"):
        return gaps

    # 从 spec 的 generated_outputs 提取声明的路径（可选）
    spec_declared = set(content.get("generated_outputs", []))

    if check_generated_files and spec_declared:
        # 严格模式：spec 声明的必须出现在 manifest
        for rel_path in spec_declared:
            # 尝试规范化路径（manifest 可能用 frontend/src/ 前缀）
            candidates = {rel_path, f"frontend/{rel_path}", f"src/{rel_path}"}
            matched = any(p in generated for p in candidates)
            if not matched:
                gaps.append(Gap(
                    severity="warning",
                    layer="L2",
                    criterion_id="manifest_declared_mismatch",
                    message=f"Spec 声明但 manifest 未包含: {rel_path}",
                    agent_action="fix_import",
                    spec_path=spec_path
                ))

    # 检查 manifest 中与本 spec 相关的文件是否有 import 断链
    # 从文件名推断关联：文件名包含 spec_name（允许 _/- 差异）
    import re
    safe = spec_name.replace("-", "[-_]").replace("_", "[-_]")
    spec_files = [f for f in generated if re.search(safe, f)]
    if not spec_files:
        return gaps  # 没有生成物，无可检查的 imports

    for rel_path in spec_files:
        file_path = GEN_DIR / rel_path.replace("frontend/", "")
        # 规范化路径
        if not str(file_path).startswith(str(GEN_DIR)):
            file_path = Path("frontend") / rel_path
        if not file_path.exists():
            # manifest 中有但文件不在，说明未生成
            gaps.append(Gap(
                severity="critical" if check_generated_files else "warning",
                layer="L2",
                criterion_id="manifest_file_not_found",
                message=f"Manifest 中记录但文件不存在: {rel_path}",
                agent_action="fix_import",
                spec_path=spec_path
            ))
        elif not _has_valid_imports(file_path):
            gaps.append(Gap(
                severity="warning",
                layer="L2",
                criterion_id="import_reachability",
                message=f"文件存在但 import 语句可能有断链: {rel_path}",
                agent_action="patch_component",
                spec_path=spec_path
            ))

    return gaps


def _has_valid_imports(file_path: Path) -> bool:
    """检查文件内 import 语句引用的模块是否存在（轻量 heuristic）。"""
    import re
    if not file_path.exists():
        return True  # 文件不存在已经在 manifest_file_not_found 捕获
    try:
        text = file_path.read_text(encoding="utf-8")
    except Exception:
        return True

    # Svelte: import from '$lib/...' 模式
    svelte_imports = re.findall(r"from ['\"](?:@/|./|\.\./|\$lib/)([^'\"]+)['\"]", text)
    for imp in svelte_imports:
        if imp.startswith("$lib/"):
            resolved = GEN_DIR / "lib" / imp[5:]
        elif imp.startswith("./") or imp.startswith("../"):
            resolved = file_path.parent / imp
        else:
            resolved = GEN_DIR / imp
        # 轻量检查：文件是否存在，或在 frontend/src 下有同名 ts/svelte/js/py
        if not resolved.exists() and not list(GEN_DIR.rglob(f"{resolved.name}.*")):
            return False
    return True


def check_parent_chain_alignment(spec_name: str, content: dict) -> list[Gap]:
    """检查 spec 的 parent 是否在文件系统中真实存在（L1 已有路径验证，这里做语义补充）。"""
    gaps = []
    spec_path = content.get("spec", {}).get("path", "")
    parent = content.get("spec", {}).get("parent")
    level = content.get("spec", {}).get("level", "")

    if not parent or level in ("1_project_goal", "1_project-goal"):
        return gaps

    # meta_template 级别不做 parent chain 检查
    if level == "meta_template":
        return gaps

    # 推断 parent 路径（复用 validate_specs 的逻辑）
    # _is_level_in 处理 5a_uiux / 5b_service 等变体
    if _is_level_in(level, ["3_module", "4_feature", "5", "2_skeleton", "2_architecture", "meta_template"]):
        parent_candidates = _resolve_parent_candidates(level, parent)
        if not any(p.exists() for p in parent_candidates):
            gaps.append(Gap(
                severity="critical",
                layer="L1",
                criterion_id="parent_file_exists",
                message=f"Parent '{parent}' (level={level}) 的文件不存在于任何候选路径",
                agent_action="self_heal",
                spec_path=spec_path
            ))

    return gaps


def _find_feature_spec(parent: str, subtypes: list[str]) -> Path | None:
    """
    在 specs/feature/ 下查找 parent feature 的 spec 文件。
    parent: feature name (e.g. 'workbench-shell', 'dsl-canvas', 'canvas_renderer')
    subtypes: 候选文件名后缀 (e.g. ['_feature.yaml', '-feature.yaml'])
    先用原名，再用 underscore 变体。
    """
    direct_candidates = [
        SPEC_DIR / "L4-feature" / f"{parent}.yaml",
        SPEC_DIR / "L4-feature" / f"{parent.replace('_', '-')}.yaml",
    ]
    for path in direct_candidates:
        if path.exists():
            return path

    # 尝试原名（支持 workbench-shell）
    for st in subtypes:
        path = SPEC_DIR / "feature" / parent / f"{parent}{st}"
        if path.exists():
            return path
    # 尝试 underscore 变体（支持 canvas_renderer → canvas-renderer）
    safe = parent.replace("-", "_").replace(" ", "_")
    if safe != parent:
        for st in subtypes:
            path = SPEC_DIR / "feature" / safe / f"{safe}{st}"
            if path.exists():
                return path
            # 或者 hyphen 版本目录
            hyphen = safe.replace("_", "-")
            path2 = SPEC_DIR / "feature" / hyphen / f"{hyphen}{st}"
            if path2.exists():
                return path2
    return None


def _resolve_parent_candidates(level: str, parent: str) -> list[Path]:
    """
    根据 spec level 解析 parent 的可能路径。
    实际目录结构：
      specs/project-goal/<name>.yaml                  (L1)
      specs/architecture/<name>.yaml                   (L2)
      specs/module/<name>_module.yaml                  (L3, MOD-* 前缀)
      specs/feature/<feat>/<feat>_feature.yaml         (L4 feature)
      specs/feature/<feat>/<name>_<subtype>.yaml      (L5)
    优先用原名（支持 hyphens），再尝试 underscore 变体。
    """
    candidates = []
    safe = parent.replace("-", "_")
    hyphen = safe.replace("_", "-")

    if level == "3_module":
        # L3 parent: architecture spec 或 module spec
        candidates = [
            SPEC_DIR / "L2-skeleton" / f"{parent}.yaml",
            SPEC_DIR / "L2-skeleton" / f"{hyphen}.yaml",
            SPEC_DIR / "architecture" / f"{parent}.yaml",
            SPEC_DIR / "architecture" / f"{safe}.yaml",
            SPEC_DIR / "L3-module" / f"{parent}.yaml",
            SPEC_DIR / "L3-module" / f"{hyphen}.yaml",
            SPEC_DIR / "module" / f"{parent}_module.yaml",
            SPEC_DIR / "module" / f"{safe}_module.yaml",
            SPEC_DIR / "L1-goal" / f"{parent}.yaml",
            SPEC_DIR / "L1-goal" / f"{hyphen}.yaml",
            SPEC_DIR / "project-goal" / f"{parent}.yaml",
            SPEC_DIR / "project-goal" / f"{safe}.yaml",
        ]

    elif level in ("2_skeleton", "2_architecture"):
        # L2 architecture → project-goal 或 architecture 目录
        candidates = [
            SPEC_DIR / "L1-goal" / f"{parent}.yaml",
            SPEC_DIR / "L1-goal" / f"{hyphen}.yaml",
            SPEC_DIR / "project-goal" / f"{parent}.yaml",
            SPEC_DIR / "project-goal" / f"{safe}.yaml",
            SPEC_DIR / "L2-skeleton" / f"{parent}.yaml",
            SPEC_DIR / "L2-skeleton" / f"{hyphen}.yaml",
            SPEC_DIR / "architecture" / f"{parent}.yaml",
            SPEC_DIR / "architecture" / f"{safe}.yaml",
        ]

    elif level == "4_feature":
        # L4 feature: parent 是 MOD-* (module) 或另一 L4 feature
        candidates = [
            SPEC_DIR / "L3-module" / f"{parent}.yaml",
            SPEC_DIR / "L3-module" / f"{hyphen}.yaml",
            SPEC_DIR / "module" / f"{parent}_module.yaml",
            SPEC_DIR / "module" / f"{safe}_module.yaml",
            SPEC_DIR / "L2-skeleton" / f"{parent}.yaml",
            SPEC_DIR / "L2-skeleton" / f"{hyphen}.yaml",
        ]
        # 找 feature spec（支持 workbench-shell、canvas_renderer 等）
        feat_spec = _find_feature_spec(parent, ["_feature.yaml", "-feature.yaml"])
        if feat_spec:
            candidates.append(feat_spec)
        feat_spec_safe = _find_feature_spec(safe, ["_feature.yaml", "-feature.yaml"])
        if feat_spec_safe and feat_spec_safe not in candidates:
            candidates.append(feat_spec_safe)

    elif _is_level_in(level, ["5"]):
        # L5 subtype: parent 指向同名 L4 feature（workbench-shell → workbench-shell_feature.yaml）
        candidates.extend([
            SPEC_DIR / "L4-feature" / f"{parent}.yaml",
            SPEC_DIR / "L4-feature" / f"{hyphen}.yaml",
        ])
        feat_spec = _find_feature_spec(parent, ["_feature.yaml", "-feature.yaml"])
        if feat_spec:
            candidates.append(feat_spec)
        feat_spec_safe = _find_feature_spec(safe, ["_feature.yaml", "-feature.yaml"])
        if feat_spec_safe and feat_spec_safe not in candidates:
            candidates.append(feat_spec_safe)
        # 也接受 MOD-* parent
        if parent.startswith("MOD-"):
            candidates.append(SPEC_DIR / "L3-module" / f"{parent}.yaml")
            candidates.append(SPEC_DIR / "module" / f"{parent}_module.yaml")

    return [p for p in candidates if p]


def _is_level_in(level: str, prefixes: list[str]) -> bool:
    """检查 level 是否属于给定前缀集合（处理 5a/5b 等变体）。"""
    for p in prefixes:
        if level == p or level.startswith(p):
            return True
    return False


# ── L3 验证：行为层 ──────────────────────────────────────────
def check_acceptance_criteria(spec_name: str, content: dict) -> list[Gap]:
    """检查 acceptance_criteria 是否完整（只检查声明存在，不做运行时验证）。"""
    gaps = []
    spec_path = content.get("spec", {}).get("path", "")
    level = content.get("spec", {}).get("level", "")

    if level != "4_feature":
        return gaps

    ac_list = content.get("acceptance_criteria", [])
    if not ac_list:
        gaps.append(Gap(
            severity="warning",
            layer="L3",
            criterion_id="acceptance_criteria_present",
            message="L4 feature spec 缺少 acceptance_criteria（建议补充 Gherkin 格式）",
            agent_action="write_test",
            spec_path=spec_path
        ))
        return gaps

    for i, ac in enumerate(ac_list):
        criterion_id = ac.get("id", f"AC-{i+1}")
        if not ac.get("given") or not ac.get("when") or not ac.get("then"):
            gaps.append(Gap(
                severity="info",
                layer="L3",
                criterion_id=criterion_id,
                message=f"AC [{criterion_id}] 格式不完整（given/when/then 缺字段）",
                agent_action="skip",
                spec_path=spec_path
            ))

        # 检查是否有对应的测试文件（启发式）
        test_name = f"test_{spec_name.replace('_', '-')}_{criterion_id.lower()}"
        test_patterns = [
            GEN_DIR / "tests" / f"{test_name}.ts",
            GEN_DIR / "tests" / f"{test_name}.test.ts",
        ]
        if not any(p.exists() for p in test_patterns):
            gaps.append(Gap(
                severity="warning",
                layer="L3",
                criterion_id=criterion_id,
                message=f"AC [{criterion_id}] 缺少对应测试文件",
                agent_action="write_test",
                spec_path=spec_path
            ))

    return gaps


# ── Meta-template 校验 ───────────────────────────────────────
def check_meta_template_consistency(spec_name: str, content: dict) -> list[Gap]:
    """检查 meta_template spec 的 template 字段是否与实际模板文件一致。"""
    gaps = []
    spec_path = content.get("spec", {}).get("path", "")
    level = content.get("spec", {}).get("level", "")

    if level != "meta_template":
        return gaps

    template_content = content.get("template") or content.get("content", {}).get("template", "")
    if not template_content:
        gaps.append(Gap(
            severity="critical",
            layer="L1",
            criterion_id="meta_template_has_content",
            message="meta_template spec 缺少 template 字段",
            agent_action="self_heal",
            spec_path=spec_path
        ))
        return gaps

    # 检查模板文件是否已同步（gen.py E3 会写入 generators/templates/*.yaml.tpl）
    tpl_file = TEMPLATE_DIR / f"{spec_name}.yaml.tpl"
    if tpl_file.exists():
        existing = tpl_file.read_text(encoding="utf-8")
        if existing != template_content:
            gaps.append(Gap(
                severity="info",
                layer="L2",
                criterion_id="template_sync",
                message=f"template 与 generators/templates/{spec_name}.yaml.tpl 不同步（运行 make generate）",
                agent_action="self_heal",
                spec_path=spec_path
            ))
    else:
        gaps.append(Gap(
            severity="warning",
            layer="L2",
            criterion_id="template_file_missing",
            message=f"meta_template spec 存在但 generators/templates/{spec_name}.yaml.tpl 不存在（运行 make generate 同步）",
            agent_action="self_heal",
            spec_path=spec_path
        ))

    return gaps


# ── 主验证循环 ───────────────────────────────────────────────
def validate_single_spec(spec_path: Path, levels: str = "L1L2L3",
                          check_generated_files: bool = False) -> CriteriaReport | None:
    """对单个 spec 执行三层验证，返回结构化报告。"""
    import yaml
    try:
        data = yaml.safe_load(spec_path.read_text(encoding="utf-8"))
    except yaml.YAMLError:
        return None
    if not data:
        return None

    spec_meta = data.get("spec", {})
    spec_name = spec_meta.get("name")
    if not spec_name:
        return None

    report = CriteriaReport(spec_name=spec_name)

    # L1: YAML 结构 + parent chain（delegate to validate_specs 的结果，这里做补充）
    if "L1" in levels:
        report.L1 = {
            "passed": True,
            "checks": [],
            "note": "L1 parent chain 验证由 validate_specs.py 负责"
        }
        gaps_l1 = check_parent_chain_alignment(spec_name, data)
        report.gaps.extend(gaps_l1)

    # L2: 语义层
    if "L2" in levels:
        report.L2 = {
            "passed": True,
            "checks": []
        }
        report.gaps.extend(check_imports_reachable(spec_name, data, check_generated_files))
        report.gaps.extend(check_meta_template_consistency(spec_name, data))

    # L3: 行为层
    if "L3" in levels:
        report.L3 = {
            "passed": True,
            "checks": []
        }
        report.gaps.extend(check_acceptance_criteria(spec_name, data))

    # 汇总
    critical_gaps = [g for g in report.gaps if g.severity == "critical"]
    report.success = len(critical_gaps) == 0

    return report


def validate_all(levels: str = "L1L2L3",
                 check_generated_files: bool = False) -> list[CriteriaReport]:
    """扫描所有 spec 并验证。"""
    reports = []
    for path in SPEC_DIR.rglob("*.yaml"):
        if "node_modules" in str(path):
            continue
        r = validate_single_spec(path, levels, check_generated_files)
        if r:
            reports.append(r)
    return reports


def print_report(report: CriteriaReport):
    """格式化输出报告（供 human + agent 解析）。"""
    status = "✅ PASS" if report.success else "❌ FAIL"
    print(f"\n{'='*60}")
    print(f"[{status}] {report.spec_name}")

    if report.gaps:
        print(f"  Gaps ({len(report.gaps)}):")
        for g in report.gaps:
            icon = {"critical": "🔴", "warning": "🟡", "info": "🔵"}[g.severity]
            print(f"    {icon} [{g.layer}] {g.criterion_id}: {g.message}")
            if g.agent_action != "skip":
                print(f"         → Agent action: {g.agent_action}")
    else:
        print(f"  No gaps found.")


# ── CLI ─────────────────────────────────────────────────────
def main():
    import argparse
    parser = argparse.ArgumentParser(description="VibeX Spec Criteria Validator")
    parser.add_argument("--spec", help="Spec name to validate (default: all)")
    parser.add_argument("--level", default="L1L2L3",
                        help="Layers to check: L1, L2, L3, or combination like L1L2 (default: L1L2L3)")
    parser.add_argument("--check-generated", action="store_true",
                        help="检查 generated_outputs 声明的文件是否存在（需要先运行 make generate）")
    parser.add_argument("--json", action="store_true", help="Output machine-readable JSON")
    parser.add_argument("--critical-only", action="store_true", help="Only show critical gaps")
    parser.add_argument("--agent-targets", action="store_true",
                        help="Output only agent target gaps (severity=critical)")
    args = parser.parse_args()

    if args.spec:
        # 单 spec 验证
        spec_path = SPEC_DIR / f"{args.spec}.yaml"
        if not spec_path.exists():
            # 搜索
            hits = list(SPEC_DIR.rglob(f"*/{args.spec}.yaml"))
            if hits:
                spec_path = hits[0]
        report = validate_single_spec(spec_path, args.level, args.check_generated)
        if report:
            if args.json:
                import json
                if args.agent_targets:
                    d = report.to_dict()
                    d["agent_targets"] = [asdict(g) for g in report.gaps if g.severity == "critical"]
                    print(json.dumps(d, indent=2, ensure_ascii=False))
                else:
                    print(json.dumps(report.to_dict(), indent=2, ensure_ascii=False))
            else:
                print_report(report)
                sys.exit(0 if report.success else 1)
        else:
            print(f"Spec not found: {args.spec}")
            sys.exit(1)
    else:
        # 全量验证
        reports = validate_all(args.level, args.check_generated)
        if args.json:
            import json
            output = [r.to_dict() for r in reports]
            if args.agent_targets:
                output = [r.to_dict() for r in reports
                          for g in r.gaps if g.severity == "critical"]
            print(json.dumps(output, indent=2, ensure_ascii=False))
        else:
            for r in reports:
                print_report(r)

            total = sum(len(r.gaps) for r in reports)
            critical = sum(1 for r in reports for g in r.gaps if g.severity == "critical")
            print(f"\n{'='*60}")
            print(f"Total: {len(reports)} specs, {total} gaps ({critical} critical)")
            sys.exit(0 if critical == 0 else 1)


if __name__ == "__main__":
    main()
