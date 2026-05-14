#!/usr/bin/env python3
"""校验并自动修正 spec 文件命名和 YAML 格式。"""
import os, sys, re, yaml, shutil
from pathlib import Path

PREFIX_MAP = {
    "L3-module/": ("MOD-", "3_module", "module"),
    "L4-feature/": ("FEAT-", "4_feature", "feature"),
    "L5-slice/": ("SLICE-", "5_slice", "slice"),
}

# 中文 title → slug 映射表
CHINESE_TO_SLUG = {
    "顶部标题栏": "top-titlebar",
    "左侧栏": "left-composite",
    "主内容区": "main-area",
    "右侧AI栏": "right-ai-panel",
    "右侧 AI 栏": "right-ai-panel",
    "底部Dock": "bottom-dock",
    "底部 Dock": "bottom-dock",
    "底部状态栏": "bottom-statusbar",
    "菜单栏": "menu-strip",
    "命令中心": "command-center",
    "窗口控制": "window-controls",
    "活动图标栏": "activity-bar",
    "Spec资源管理器": "spec-explorer",
    "Spec 资源管理器": "spec-explorer",
    "中心视图": "center-view",
    "Agent对话": "agent-chat",
    "Agent 对话": "agent-chat",
    "Dock面板": "dock-panel",
    "Dock 面板": "dock-panel",
    "DockTab栏": "dock-tab-bar",
    "Dock Tab 栏": "dock-tab-bar",
    "Dock内容区": "dock-content",
    "Dock 内容区": "dock-content",
    "状态栏左": "statusbar-left",
    "状态栏右": "statusbar-right",
    "资源管理器": "explorer",
    "搜索": "search",
    "Git": "git",
    "扩展": "extensions",
    "账户": "account",
    "Spec栏标题": "spec-hdr",
    "Spec 栏标题": "spec-hdr",
    "刷新Spec": "spec-reload",
    "刷新 Spec": "spec-reload",
    "Spec文件节点": "spec-file-node",
    "Spec 文件节点": "spec-file-node",
    "编辑器标签组": "editor-tab-bar",
    "AI栏标题": "ai-hdr",
    "AI 栏标题": "ai-hdr",
    "会话历史": "chat-history",
    "快捷操作": "quick-actions",
    "对话历史": "conversation-history",
    "作曲输入": "composer",
    "DockTab": "dock-tab",
    "Dock Tab": "dock-tab",
    "菜单按钮": "menu-btn",
    "初始化Specs": "init-specs",
    "初始化 Specs": "init-specs",
}


def title_to_slug(title: str) -> str:
    """从 'L3 · 顶部标题栏' 提取中文并转 slug。"""
    title = title.strip()
    # 移除前缀 L1/L2/L3/L4/L5 和分隔符
    m = re.search(r'[· ]+(.+)$', title)
    if m:
        zh = m.group(1).strip()
    else:
        zh = title
    slug = CHINESE_TO_SLUG.get(zh)
    if slug:
        return slug
    # 兜底：移除非字母数字中文，空白转-
    slug = re.sub(r'[^a-zA-Z0-9\u4e00-\u9fff]', '-', zh)
    slug = re.sub(r'-+', '-', slug).strip('-').lower()
    return slug


def validate_and_fix(spec_path: Path, dry_run: bool = False) -> list:
    """校验单个 spec 文件，返回问题列表。返回空列表=通过。"""
    name = spec_path.stem
    parent_dir = spec_path.parent.name  # e.g. L3-module
    if parent_dir not in PREFIX_MAP:
        return []

    expected_prefix, level_key, type_val = PREFIX_MAP[parent_dir]
    issues = []

    # 1. 命名检查
    if not name.startswith(expected_prefix):
        correct_name = expected_prefix + name.split('-', 1)[-1]
        issues.append(f"[NAMING] '{name}' should be '{correct_name}' [auto-fixable]")

    # 2. YAML 格式检查
    try:
        with open(spec_path, encoding="utf-8") as f:
            data = yaml.safe_load(f)
    except yaml.YAMLError as e:
        issues.append(f"[YAML] parse error: {e}")
        return issues

    if data is None:
        issues.append(f"[YAML] empty file")
        return issues

    spec_block = data.get("spec", {})
    if not spec_block:
        issues.append(f"[YAML] missing 'spec:' block")
        return issues

    spec_level = spec_block.get("level", "")
    if spec_level != level_key:
        issues.append(f"[LEVEL] spec.level should be '{level_key}', got '{spec_level}'")

    spec_name = spec_block.get("name", "")
    if spec_name != name:
        issues.append(f"[NAME] spec.name '{spec_name}' should match filename '{name}'")

    meta_type = data.get("meta", {}).get("type", "")
    if meta_type != type_val:
        issues.append(f"[TYPE] meta.type should be '{type_val}', got '{meta_type}'")

    return issues


def auto_fix(spec_path: Path) -> bool:
    """自动修正文件名和文件内容。返回是否做了修改。"""
    name = spec_path.stem
    parent_dir = spec_path.parent.name
    if parent_dir not in PREFIX_MAP:
        return False

    expected_prefix, level_key, type_val = PREFIX_MAP[parent_dir]
    modified = False

    # 修正文件名
    if not name.startswith(expected_prefix):
        suffix = name.split('-', 1)[-1] if '-' in name else name
        new_name = expected_prefix + suffix
        new_path = spec_path.parent / (new_name + ".yaml")
        print(f"  [RENAME] {name}.yaml -> {new_name}.yaml")
        shutil.move(str(spec_path), str(new_path))
        spec_path = new_path
        modified = True

        # 更新文件内的 spec.name
        try:
            with open(spec_path, encoding="utf-8") as f:
                data = yaml.safe_load(f)
            if data and "spec" in data and data["spec"].get("name") == name:
                data["spec"]["name"] = new_name
            with open(spec_path, "w", encoding="utf-8") as f:
                yaml.safe_dump(data, f, allow_unicode=True, sort_keys=False)
            print(f"  [UPDATE] spec.name updated to '{new_name}'")
        except Exception as e:
            print(f"  [WARN] could not update spec.name: {e}")

    # 修正 spec.level
    try:
        with open(spec_path, encoding="utf-8") as f:
            data = yaml.safe_load(f)
        if data:
            if data.get("spec", {}).get("level") != level_key:
                data.setdefault("spec", {})["level"] = level_key
                modified = True
            if data.get("meta", {}).get("type") != type_val:
                data.setdefault("meta", {})["type"] = type_val
                modified = True
            with open(spec_path, "w", encoding="utf-8") as f:
                yaml.safe_dump(data, f, allow_unicode=True, sort_keys=False)
    except Exception as e:
        print(f"  [WARN] YAML update failed: {e}")

    return modified


def main(specs_dir: str):
    specs_path = Path(specs_dir)
    all_issues = []
    auto_fixed = []

    for subdir in PREFIX_MAP:
        sub = specs_path / subdir
        if not sub.exists():
            continue
        for yaml_file in sorted(sub.glob("*.yaml")):
            issues = validate_and_fix(yaml_file)
            if issues:
                all_issues.append((yaml_file, issues))
                # 自动修正命名类问题
                if any("[NAMING]" in i or "[LEVEL]" in i or "[NAME]" in i or "[TYPE]" in i for i in issues):
                    if auto_fix(yaml_file):
                        auto_fixed.append(str(yaml_file))

    # 二次校验
    if auto_fixed:
        still_issues = []
        for fp_str in auto_fixed:
            fp = Path(fp_str)
            issues = validate_and_fix(fp)
            if issues:
                still_issues.append((fp, issues))
        all_issues = [x for x in all_issues if Path(x[0]) not in [Path(f) for f in auto_fixed]]
        all_issues.extend(still_issues)

    if all_issues:
        print("[VALIDATE] Issues found:")
        for fp, issues in all_issues:
            print(f"  {fp.name}:")
            for issue in issues:
                print(f"    {issue}")
    else:
        print("[VALIDATE] All specs passed!")

    if auto_fixed:
        print(f"[VALIDATE] Auto-fixed: {len(auto_fixed)} files")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
