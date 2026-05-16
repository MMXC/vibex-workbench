#!/usr/bin/env python3
"""
spec-prototype-fragment · wc-gen.py
────────────────────────────────────────────────────────────────────────────
参数化派生原型生成脚本。

使用方式：
    python wc-gen.py --spec-name MOD-build-panel --selector ".build-group" \
                     --output ../prototypes/MOD-build-panel.html

原理：
    1. 复制完整 MVP HTML（不做裁剪）
    2. 在最后一个 </script> 后注入 WC 定义脚本（从 wc-template.js 读取，DOMContentLoaded 包裹）
    3. 替换指定 selector 区域为 <wc-{spec-name}> 元素
    4. 写入目标路径

⚠ 注入点规则：
    - 必须用 rfind('</script>') 追加到最后一个 script 块之后
    - 禁止注入到 </style> 之前（会被当作文本不执行）
    - wc_script() 输出的脚本必须用 DOMContentLoaded 包裹，确保 DOM 就绪
"""

import argparse
import os
import re
import sys

MVP_PATH = ".vibex/prototypes/vibex-workbench-mvp.html"

CSS_VARS = {
    "--wb-bg": "#0d1117",
    "--wb-bg-panel": "#0f1117",
    "--wb-bg-hover": "#161b22",
    "--wb-text": "#c0caf5",
    "--wb-text-sec": "#787c99",
    "--wb-border": "rgba(255,255,255,0.07)",
    "--wb-accent": "#72d6d0",
    "--font-mono": "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    "--accent-green": "#87cf8a",
    "--accent-red": "#e16d75",
    "--accent-yellow": "#efc66b",
    "--accent-orange": "#f09a6a",
    "--accent-blue": "#72d6d0",
}


def wc_script(spec_name: str, trigger_label: str = "") -> str:
    """
    读取 scripts/wc-template.js，替换占位符，
    用 DOMContentLoaded 包裹后返回完整 <script> 块（确保 DOM 就绪）。
    """
    if not trigger_label:
        trigger_label = spec_name.replace("-", " ").replace("_", " ").title()

    script_dir = os.path.dirname(os.path.abspath(__file__))
    template_path = os.path.join(script_dir, "wc-template.js")
    with open(template_path, encoding="utf-8") as f:
        tmpl = f.read()

    css_vars = css_vars_block()
    safe_name = safe_class_name(spec_name)

    tmpl = tmpl.replace("{{SPEC_NAME}}", spec_name)
    tmpl = tmpl.replace("{{SAFE_CLASS_NAME}}", safe_name)
    tmpl = tmpl.replace("{{TRIGGER_LABEL}}", trigger_label)
    tmpl = tmpl.replace("{{CSS_VARS}}", css_vars)

    wrapped = f"""document.addEventListener('DOMContentLoaded', function () {{
  (function () {{
{tmpl}  }})();
}});
"""
    return f"<script>\n{wrapped}\n</script>\n"


def css_vars_block() -> str:
    """生成 shadow DOM 内联 CSS 变量声明"""
    indent = "        "
    lines = [indent + ":root {"]
    for name, val in CSS_VARS.items():
        lines.append(f"{indent}  {name}: {val};")
    lines.append(indent + "}")
    return "\n".join(lines)


def safe_class_name(spec_name: str) -> str:
    """将 spec 名称转换为合法的 JS 类名"""
    parts = spec_name.replace("-", "_").replace(".", "_").split("_")
    return "".join(p.title() for p in parts)


def generate(spec_name: str, selector: str, output_path: str,
             trigger_label: str = "") -> None:
    """
    核心生成函数。
    1. 读取 MVP HTML
    2. 在 </style> 后注入 WC 脚本
    3. 替换 selector 区域为 <wc-*> 元素
    4. 写入输出文件
    """
    script_dir = os.path.dirname(os.path.abspath(__file__))
    # scripts/spec-prototype-fragment/scripts/ → 向上 5 层到项目根
    #   up1: spec-prototype-fragment/, up2: skills/, up3: agents/, up4: .vibex/, up5: project-root/
    mvp_file = os.path.normpath(
        os.path.join(script_dir, "..", "..", "..", "..", "..", MVP_PATH)
    )
    if not os.path.exists(mvp_file):
        print(f"Error: MVP file not found: {mvp_file}", file=sys.stderr)
        sys.exit(1)

    with open(mvp_file, "r", encoding="utf-8") as f:
        html = f.read()

    # 注入 WC 脚本（追加到最后一个 </script> 后，DOMContentLoaded 包裹）
    script_end = html.rfind("</script>")
    if script_end == -1:
        print("Error: no </script> found in MVP HTML", file=sys.stderr)
        sys.exit(1)
    script_end += len("</script>")

    script = wc_script(spec_name, trigger_label)
    html = html[:script_end] + "\n" + script + html[script_end:]

    # 替换 selector 区域
    sel_name = selector.lstrip(".")
    pattern = re.compile(
        r'(<\w+(?:\s[^>]*)?\sclass="[^"]*' + re.escape(sel_name) + r'[^"]*"\s[^>]*>)',
        re.IGNORECASE
    )
    match = pattern.search(html)

    if match:
        wc_tag = (
            f'<wc-{spec_name} '
            f'data-vibex-annot="L3" '
            f'data-vibex-title="L3 · {spec_name}（派生）"'
            f'></wc-{spec_name}>'
        )
        html = _replace_element(html, match.start(), selector, wc_tag)
        print(f"Replaced selector '{selector}' with <wc-{spec_name}>")
    else:
        body_end = html.rfind("</body>")
        if body_end == -1:
            print(f"Warning: selector '{selector}' not found and </body> not found", file=sys.stderr)
        else:
            wc_tag = (
                f'\n<wc-{spec_name} '
                f'data-vibex-annot="L3" '
                f'data-vibex-title="L3 · {spec_name}（派生）"'
                f'></wc-{spec_name}>\n'
            )
            html = html[:body_end] + wc_tag + html[body_end:]
            print(f"Selector '{selector}' not found, appended to body")

    # 更新标题
    html = html.replace(
        "<title>VibeX Workbench",
        "<title>原型 · L3 · " + spec_name + " — VibeX Workbench"
    )

    # 确保目录存在
    out_dir = os.path.dirname(output_path)
    if out_dir and not os.path.exists(out_dir):
        os.makedirs(out_dir)

    with open(output_path, "w", encoding="utf-8") as f:
        f.write(html)

    size = os.path.getsize(output_path)
    print(f"Generated: {output_path} ({size} bytes)")


def _replace_element(html: str, match_start: int, selector: str, wc_tag: str) -> str:
    """将匹配元素替换为 WC 标签"""
    sel_name = selector.lstrip(".")

    gt = html.find(">", match_start)
    if gt == -1:
        return html

    start_tag_end = gt + 1

    tag_match = re.match(r'<(\w+)', html[match_start:])
    if not tag_match:
        return html
    tag_name = tag_match.group(1)

    depth = 1
    pos = start_tag_end
    while pos < len(html) and depth > 0:
        if html[pos:pos+2] == "</":
            end_pos = html.find(">", pos)
            if end_pos == -1:
                break
            end_tag = html[pos+2:end_pos].strip().split()[0]
            if end_tag == tag_name:
                depth -= 1
                if depth == 0:
                    break
            pos = end_pos + 1
        elif html[pos:pos+1] == "<":
            space = html.find(" ", pos)
            gt2 = html.find(">", pos)
            if gt2 == -1:
                break
            if space != -1 and space < gt2:
                next_tag = html[pos+1:space]
            else:
                next_tag = html[pos+1:gt2]
            if next_tag and not next_tag.startswith("!"):
                depth += 1
            pos = gt2 + 1
        else:
            pos += 1

    if depth == 0:
        end_len = len(f"</{tag_name}>")
        return html[:match_start] + wc_tag + html[pos + end_len:]

    return html[:match_start] + wc_tag + html[start_tag_end:]


def main():
    parser = argparse.ArgumentParser(
        description="spec-prototype-fragment · 参数化派生原型生成"
    )
    parser.add_argument("--spec-name", required=True,
                        help="spec 名称（将用作 wc-* 标签名）")
    parser.add_argument("--selector", required=True,
                        help="MVP 中要替换的区域选择器（如 .build-group）")
    parser.add_argument("--output", required=True,
                        help="输出路径（如 ../prototypes/MOD-xxx.html）")
    parser.add_argument("--trigger-label", default="",
                        help="触发按钮文字（默认从 spec-name 生成）")
    args = parser.parse_args()

    generate(args.spec_name, args.selector, args.output, args.trigger_label)


if __name__ == "__main__":
    main()
