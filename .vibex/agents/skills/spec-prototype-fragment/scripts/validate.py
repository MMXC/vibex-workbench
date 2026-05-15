#!/usr/bin/env python3
"""
spec-prototype-fragment · validate.py
────────────────────────────────────────────────────────────────────────────
约束校验脚本 — 检查生成的派生原型 HTML 是否满足约束。

使用方式：
    python validate.py prototypes/MOD-{name}.html

检查项（硬编码，非 markdown 描述）：
    1. MVP 全量复制（wb-root 区块完整，无意外裁剪）
    2. WC 定义存在（customElements.define）
    3. shadow DOM 存在（attachShadow）
    4. 四态 panels 存在（ph-idle / ph-open / ph-done / ph-error）
    5. 初始 hidden（所有 phase panel 含 class="hidden" 或 class="phase ... hidden"）
    6. 无绝对颜色（样式用 var(--xxx)，避免自己发明颜色）
    7. CSS 变量声明（shadow DOM 内有 CSS 自定义属性声明）

退出码：
    0  全部通过
    1  检查失败
────────────────────────────────────────────────────────────────────────────
"""

import argparse
import re
import sys


MVP_WB_ROOT_BLOCKS = [
    ".wb-root",
    ".wb-titlebar",
    ".wb-center",
    ".wb-left",
    ".wb-footer",
    ".statusbar",
    ".vibex-annot-ctrl",
]

MVP_CSS_VAR_NAMES = [
    "--wb-bg-panel", "--wb-bg-panel-2", "--wb-bg-base",
    "--wb-text", "--wb-text-sec",
    "--wb-accent", "--accent-green", "--accent-red",
    "--accent-orange", "--accent-yellow",
    "--font-ui", "--font-mono", "--wb-border",
]


def read_file(path: str) -> str:
    try:
        with open(path, "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        print(f"Error: file not found: {path}", file=sys.stderr)
        sys.exit(1)


def check_wb_root_blocks(html: str) -> list:
    """检查 wb-root 核心区块是否完整"""
    issues = []
    for block in MVP_WB_ROOT_BLOCKS:
        sel = block.lstrip(".")
        # 检查 class 或 id 包含该名称
        pattern = re.compile(rf'class="[^"]*{re.escape(sel)}[^"]*"', re.IGNORECASE)
        id_pattern = re.compile(rf'id="{re.escape(sel)}"', re.IGNORECASE)
        if not pattern.search(html) and not id_pattern.search(html):
            issues.append(f"MVP 核心区块缺失: {block}")
    return issues


def check_wc_define(html: str) -> list:
    """检查 customElements.define 是否存在"""
    issues = []
    if not re.search(r'customElements\.define\s*\(', html):
        issues.append("customElements.define 未找到")
    return issues


def check_shadow_dom(html: str) -> list:
    """检查 attachShadow 是否存在"""
    issues = []
    if not re.search(r'attachShadow\s*\(\s*\{', html):
        issues.append("attachShadow 未找到（web component 需要 shadow DOM）")
    return issues


def check_phase_panels(html: str) -> list:
    """检查四态 panel 是否存在"""
    required_panels = ["ph-open", "ph-loading", "ph-done", "ph-error"]
    issues = []
    for panel in required_panels:
        pattern = re.compile(rf'id="ph-[^"]*"\s*class="[^"]*phase[^"]*"', re.IGNORECASE)
        if not pattern.search(html):
            issues.append(f"四态 panel 缺失: {panel}")
    return issues


def check_initial_hidden(html: str) -> list:
    """检查所有 phase panel 初始是否含 hidden"""
    issues = []
    # 找到所有 phase class 的元素
    phase_pattern = re.compile(r'<[^>]+class="[^"]*phase[^"]*"[^>]*>', re.IGNORECASE)
    matches = phase_pattern.findall(html)
    if not matches:
        issues.append("未找到任何 class 包含 'phase' 的元素")
        return issues

    for el in matches:
        if 'hidden' not in el:
            # 提取 id
            id_m = re.search(r'id="([^"]+)"', el)
            el_id = id_m.group(1) if id_m else "<unknown>"
            issues.append(f"phase 元素初始无 hidden: #{el_id}")
    return issues


def check_css_vars_usage(html: str) -> list:
    """检查是否使用了 var(--xxx) 引用 MVP CSS 变量"""
    issues = []
    var_pattern = re.compile(r'var\s*\(\s*--[\w-]+')
    has_var = bool(var_pattern.search(html))
    if not has_var:
        issues.append("未找到 var(--xxx) 引用（应使用 MVP CSS 变量）")

    # 检查是否有可疑的硬编码颜色（排除 rgba 和已有 var）
    color_issues = []
    # 查找 background:\s*#[0-9a-f]{3,6} 或 color:\s*#[0-9a-f]{3,6}
    hardcoded = re.findall(
        r'(?:background|color|border-color|fill|stroke)\s*:\s*#([0-9a-fA-F]{3,6})\s*;',
        html
    )
    # 允许的颜色（透明、继承、无）
    allowed = {"000", "000000", "fff", "ffffff", "none", "inherit", "transparent", "currentcolor"}
    for color in hardcoded:
        if color.lower() not in allowed:
            color_issues.append(f"硬编码颜色: #{color}")
    issues.extend(color_issues)
    return issues


def check_no_hidden_unrelated(html: str) -> list:
    """检查非关联区域是否有意外的 hidden"""
    issues = []
    # 找到所有带 hidden 的元素
    hidden_pattern = re.compile(r'<[^>]+class="[^"]*hidden[^"]*"[^>]*>', re.IGNORECASE)
    hidden_els = hidden_pattern.findall(html)

    for el in hidden_els:
        # 检查是否是 phase panel（允许）
        if re.search(r'\bphase\b', el, re.IGNORECASE):
            continue
        # 检查是否是 overlay（允许）
        if re.search(r'\boverlay\b', el, re.IGNORECASE):
            continue
        # 检查是否是 trigger btn（允许）
        if re.search(r'\btrigger\b', el, re.IGNORECASE):
            continue
        # 检查是否是 wc-* 元素（允许）
        if re.search(r'<wc-', el, re.IGNORECASE):
            continue
        # 提取 id/class
        id_m = re.search(r'id="([^"]+)"', el)
        cls_m = re.search(r'class="([^"]+)"', el)
        el_desc = f"#{id_m.group(1)}" if id_m else (f".{cls_m.group(1)}" if cls_m else el[:60])
        issues.append(f"非关联区域含 hidden（应为 phase/overlay/trigger）: {el_desc}")

    return issues


def check_wc_element(html: str) -> list:
    """检查是否有 <wc-*> 元素"""
    issues = []
    if not re.search(r'<wc-[\w-]+', html):
        issues.append("未找到 <wc-*> 元素（spec 关联区域应被替换为 component）")
    return issues


def run(path: str) -> bool:
    """运行所有检查，返回 True 表示全部通过"""
    html = read_file(path)

    all_issues = {}
    checks = [
        ("MVP 全量复制", check_wb_root_blocks),
        ("WC 定义", check_wc_define),
        ("shadow DOM", check_shadow_dom),
        ("WC 元素", check_wc_element),
        ("四态 panels", check_phase_panels),
        ("初始 hidden", check_initial_hidden),
        ("CSS 变量使用", check_css_vars_usage),
        ("无意外 hidden", check_no_hidden_unrelated),
    ]

    passed = 0
    failed = 0

    for name, fn in checks:
        issues = fn(html)
        all_issues[name] = issues
        if issues:
            failed += 1
            print(f"  ✗ {name}:")
            for issue in issues:
                print(f"    - {issue}")
        else:
            passed += 1
            print(f"  ✓ {name}")

    print(f"\n{passed}/{passed+failed} 检查通过")
    return failed == 0


def main():
    parser = argparse.ArgumentParser(
        description="spec-prototype-fragment · 派生原型约束校验"
    )
    parser.add_argument("file", help="派生原型 HTML 文件路径")
    args = parser.parse_args()

    ok = run(args.file)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
