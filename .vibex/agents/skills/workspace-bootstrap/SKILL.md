---
name: workspace-bootstrap
description: |
  根据用户指定的原型文件，自动绑定/创建对应层级的 L3/L4/L5 specs。
  读取原型 HTML 中的 `data-vibex-annot` 标注，按 L3 → L4 → L5 层级逐一生成/绑定 YAML spec 文件到工作区 `.vibex/specs/` 目录。
  **生成后自动校验命名规范和 YAML 格式，不合规自动调整。**
  触发条件：用户指定了原型路径，且该原型已通过 prototype-annotator 技能完成 L1/L3/L4/L5 标注。
  触发词：绑定spec、初始化spec、绑定原型、spec绑定。
---

## A. 命名规范（强制）

所有 spec 文件必须遵守以下命名规则，校验脚本会自动检查并调整。

| 层级 | 正确前缀 | 正确示例 | 错误示例 |
|------|---------|---------|---------|
| L3 module | `MOD-` | `MOD-ide-chrome.yaml` | `L3-module-titlebar.yaml` |
| L4 feature | `FEAT-` | `FEAT-ide-titlebar.yaml` | `L4-feature-titlebar.yaml` |
| L5 slice | `SLICE-` | `SLICE-titlebar-menu-btn.yaml` | `L5-slice-titlebar.yaml` |

**Slug 转换规则**：
- `data-vibex-title` 格式为 `L{n} · 中文标题`，取中文部分
- 中文转拼音（空格→`-`），例：`顶部标题栏` → `top-titlebar`
- 最终文件名：`{前缀}{slug}.yaml`，如 `MOD-top-titlebar.yaml`

**校验不通过时**：自动调整文件名，不拒绝执行。

---

## B. YAML 格式规范

所有生成的 YAML 必须符合以下结构：

```yaml
---
spec:
  version: "0.1"
  level: "3_module"        # L3固定填 3_module，L4填 4_feature，L5填 5_slice
  name: "MOD-top-titlebar" # 与文件名去掉 .yaml 后一致
  parent: "vibex-workbench-mvp"  # L3的parent为L1名，L4的parent为所属L3的name
  status: "proposal"
  prototype_ref: "../../prototypes/vibex-workbench-mvp.html"
meta:
  type: "module"           # L3填module，L4填feature，L5填slice
  owner: "user"
  created: "2026-05-14"
  updated: "2026-05-14"
lifecycle:
  current: "proposal"
  updated: "2026-05-14"
display:
  title: L3 · 顶部标题栏
  summary: 模块一句话描述
  description: 模块详细描述（多行）
structure:
  parent: vibex-workbench-mvp
  parent_selector: ".titlebar"
  children:
    - FEAT-top-titlebar-menu
l4_features: []    # L3有，L4/L5为空列表
io:
  input: []
  output: []
  boundary: ""
prototype:
  file: ".vibex/prototypes/vibex-workbench-mvp.html"
  validates: []
  status: annotated
content:
  l3_l4_lineage:
    summary: ""
    which_features: []
```

**YAML 格式校验规则**：
- 必须为有效 YAML
- 顶层必须有 `spec:` 字段
- `spec.level` 必须是 `3_module` / `4_feature` / `5_slice`
- `spec.name` 必须与文件名去掉 `.yaml` 一致
- `meta.type` 必须是 `module` / `feature` / `slice`
- `structure.children` 必须是列表（L4/L5 可为空列表 `[]`）

校验不通过时输出错误信息，**拒绝写盘**。

---

## C. 校验脚本

校验脚本在生成文件后执行，路径为 `scripts/validate_and_rename.sh`（Git Bash / Linux）或内联执行：

```bash
#!/bin/bash
# scripts/validate_and_rename.sh
# 用法：bash scripts/validate_and_rename.sh <specs_dir>
# 功能：校验所有 .yaml 文件的命名和格式，不合规则自动重命名/修正

SPECS_DIR="$1"
ERRORS=0

for f in $(find "$SPECS_DIR/L3-module/" -name "*.yaml" 2>/dev/null); do
  NAME=$(basename "$f" .yaml)
  # 命名校验：L3 必须以 MOD- 开头
  if [[ "$NAME" != MOD-* ]]; then
    NEW_NAME="MOD-${NAME#L3-module-}"
    echo "[RENAME] $NAME -> $NEW_NAME"
    mv "$f" "$(dirname $f)/$NEW_NAME.yaml"
    sed -i "s/^  name: \"$NAME\"/  name: \"$NEW_NAME\"/" "$(dirname $f)/$NEW_NAME.yaml"
    ((ERRORS++))
  fi
done

for f in $(find "$SPECS_DIR/L4-feature/" -name "*.yaml" 2>/dev/null); do
  NAME=$(basename "$f" .yaml)
  if [[ "$NAME" != FEAT-* ]]; then
    NEW_NAME="FEAT-${NAME#L4-feature-}"
    echo "[RENAME] $NAME -> $NEW_NAME"
    mv "$f" "$(dirname $f)/$NEW_NAME.yaml"
    ((ERRORS++))
  fi
done

for f in $(find "$SPECS_DIR/L5-slice/" -name "*.yaml" 2>/dev/null); do
  NAME=$(basename "$f" .yaml)
  if [[ "$NAME" != SLICE-* ]]; then
    NEW_NAME="SLICE-${NAME#L5-slice-}"
    echo "[RENAME] $NAME -> $NEW_NAME"
    mv "$f" "$(dirname $f)/$NEW_NAME.yaml"
    ((ERRORS++))
  fi
done

if [ $ERRORS -gt 0 ]; then
  echo "[ERROR] $ERRORS naming issues auto-fixed"
fi
```

**Python 校验脚本（推荐，跨平台）**：`scripts/validate_specs.py`

```python
#!/usr/bin/env python3
"""校验并自动修正 spec 文件命名和 YAML 格式。"""
import os, sys, re, yaml
from pathlib import Path

PREFIX_MAP = {
    "L3-module/": ("MOD-", "3_module", "module"),
    "L4-feature/": ("FEAT-", "4_feature", "feature"),
    "L5-slice/": ("SLICE-", "5_slice", "slice"),
}
CHINESE_TO_SLUG = {
    "顶部标题栏": "top-titlebar",
    "左侧栏": "left-composite",
    "主内容区": "main-area",
    "右侧AI栏": "right-ai-panel",
    "底部Dock": "bottom-dock",
    "底部状态栏": "bottom-statusbar",
    "菜单栏": "menu-strip",
    "命令中心": "command-center",
    "窗口控制": "window-controls",
    "活动图标栏": "activity-bar",
    "Spec资源管理器": "spec-explorer",
    "中心视图": "center-view",
    "Agent对话": "agent-chat",
    "Dock面板": "dock-panel",
    "DockTab栏": "dock-tab-bar",
    "Dock内容区": "dock-content",
    "状态栏左": "statusbar-left",
    "状态栏右": "statusbar-right",
    "资源管理器": "explorer",
    "搜索": "search",
    "Git": "git",
    "扩展": "extensions",
    "账户": "account",
    "Spec栏标题": "spec-hdr",
    "刷新Spec": "spec-reload",
    "Spec文件节点": "spec-file-node",
    "编辑器标签组": "editor-tab-bar",
    "AI栏标题": "ai-hdr",
    "会话历史": "chat-history",
    "快捷操作": "quick-actions",
    "对话历史": "conversation-history",
    "作曲输入": "composer",
    "DockTab": "dock-tab",
    "菜单按钮": "menu-btn",
    "初始化Specs": "init-specs",
}

def title_to_slug(title: str) -> str:
    # 从 "L3 · 顶部标题栏" 提取中文部分
    m = re.search(r'[· ]+([^L].+)$', title)
    if not m:
        return re.sub(r'[^a-z0-9-]', '-', title.lower())
    zh = m.group(1).strip()
    return CHINESE_TO_SLUG.get(zh, re.sub(r'[^a-z0-9\u4e00-\u9fff]', '-', zh).lower())

def validate_and_fix(spec_path: Path):
    name = spec_path.stem
    parent = spec_path.parent.name  # e.g. L3-module
    expected_prefix, level_key, type_val = PREFIX_MAP[parent]

    issues = []
    if not name.startswith(expected_prefix):
        issues.append(f"[NAMING] {name} should start with {expected_prefix}")

    try:
        with open(spec_path) as f:
            data = yaml.safe_load(f)
    except yaml.YAMLError as e:
        issues.append(f"[YAML] {spec_path}: {e}")
        return issues

    if data:
        spec_level = data.get("spec", {}).get("level", "")
        if spec_level != level_key:
            issues.append(f"[LEVEL] spec.level should be '{level_key}', got '{spec_level}'")
        spec_name = data.get("spec", {}).get("name", "")
        if spec_name != name:
            issues.append(f"[NAME] spec.name '{spec_name}' should match filename '{name}'")

    return issues

def main(specs_dir: str):
    specs_path = Path(specs_dir)
    all_issues = []
    for subdir, _, _ in PREFIX_MAP.items():
        sub = specs_path / subdir
        if not sub.exists():
            continue
        for yaml_file in sub.glob("*.yaml"):
            issues = validate_and_fix(yaml_file)
            all_issues.extend(issues)

    if all_issues:
        print("[VALIDATE] Issues found:")
        for issue in all_issues:
            print(f"  {issue}")
    else:
        print("[VALIDATE] All specs passed!")

if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
```

---

## D. 工作流程（含校验）

```
原型标注检查
    ↓
解析标注树
    ↓
生成/绑定 spec 文件（暂存内存，不写盘）
    ↓
执行校验脚本
    ↓
  ┌─ 有问题 → 自动修正文件名 + YAML 内容 → 重新校验 → 直到通过
  ↓
写盘
    ↓
最终报告
```

# workspace-bootstrap

**规范路径**：与本目录 **`agent.json`** 成对放置于 **`.vibex/agents/skills/workspace-bootstrap/`**。

---

## 0. 前置条件检查

**必须在执行任何操作前完成此检查，未通过则拒绝执行。**

### 0.1 必填输入

| 参数 | 来源 | 说明 |
|------|------|------|
| `workspace_root` | 用户指定或 `ps_get_page_context` | 工作区根目录 |
| `prototype_path` | 用户指定 | 原型 HTML 文件路径（绝对路径或相对于 `workspace_root/.vibex/prototypes/`） |

### 0.2 原型标注检查

1. 使用 `read_file` 读取原型 HTML 内容
2. 检查文件中是否存在 `data-vibex-annot` 属性
3. **如果不存在任何 `data-vibex-annot` 属性**：
   - 向用户输出错误信息：**"该原型尚未完成标注，请先使用 prototype-annotator 技能完成 L1/L3/L4/L5 标注后再执行本技能。"**
   - **拒绝执行任何后续操作**
4. **如果存在标注但 L1/L3/L4 不完整**：
   - 向用户列出缺失的层级
   - 提示使用 prototype-annotator 补充标注

### 0.3 确认提示

向用户输出当前原型标注状态：

```
### 原型标注状态

| 层级 | 数量 | 示例 |
|------|------|------|
| L1 | 1 | wb-root |
| L3 | 6 | titlebar / wb-left-composite / wb-main / wb-right / wb-dock-wrap / statusbar |
| L4 | N | menu-strip / activity-bar / spec-explorer / ai-column / dock ... |
| L5 | M | button / tree-btn / composer-region ... |

确认以上标注完整后，我将按层级逐一绑定/创建 spec 文件。
```

---

## 1. Spec 目录结构

所有 spec 文件写入工作区 `.vibex/specs/` 下对应层级目录：

```
.vibex/specs/
├── L1-goal/         # 由 prototype-spec-extractor 或用户手动管理
├── L2-skeleton/      # 由 prototype-spec-extractor 或用户手动管理
├── L3-module/        # 本技能生成/绑定
│   └── MOD-{slug}.yaml
├── L4-feature/       # 本技能生成/绑定
│   └── FEAT-{slug}.yaml
└── L5-slice/         # 本技能生成/绑定
    └── SLICE-{slug}.yaml
```

**命名规则（强制，见 §A）**：`{前缀}-{slug}.yaml`，`slug` 由 `data-vibex-title` 中文部分查表转拼音（如 `顶部标题栏` → `top-titlebar`）。

---

## 2. 分步工作流程（含校验）

> ⚠️ **生成后必须执行校验，不合规自动修正后再写盘。**

### 步骤一：解析原型标注（自动执行）

1. `read_file` 原型 HTML
2. 提取所有带 `data-vibex-annot` 的元素，建立标注树：

   ```
   L1: wb-root
    └─ L3: titlebar (L3 · 顶部标题栏)
    └─ L3: wb-left-composite (L3 · 左侧栏)
       └─ L4: activity-bar (L4 · 活动图标栏)
          └─ L5: button[资源管理器]
          └─ L5: button[搜索]
          └─ ...
       └─ L4: spec-explorer (L4 · Spec 资源管理器)
          └─ L5: button[刷新]
          └─ L5: button[.vibex]
          └─ ...
    └─ L3: wb-main (L3 · 主内容区)
    └─ L3: wb-right (L3 · 右侧 AI 栏)
    └─ L3: wb-dock-wrap (L3 · 底部 Dock)
    └─ L3: statusbar (L3 · 底部状态栏)
   ```

### 步骤二：生成（暂存，不写盘）

对每个 L3/L4/L5 节点，按 §B 格式生成 YAML 内容。
**文件名使用 §A 规范前缀（如 `MOD-top-titlebar.yaml`）。**
生成后立即执行 Python 校验脚本（见 §C），不合规自动修正。

### 步骤三：校验与修正

运行 `python scripts/validate_specs.py .vibex/specs`：
- **命名不合规**：自动重命名文件 + 更新内部 `spec.name`
- **YAML 格式不合规**：输出错误，**拒绝写盘**
- **校验通过**：进入下一步

1. 输出完整绑定计划清单（含最终规范文件名）：

   ```
   ### 绑定计划

   #### L3 Modules ({N} 个)
   | # | 标题 | 文件（规范） | 状态 |
   |---|------|------|------|
   | 1 | L3 · 顶部标题栏 | MOD-top-titlebar.yaml | 创建 |
   | 2 | L3 · 左侧栏 | MOD-left-composite.yaml | 绑定 |
   ...

   #### L4 Features ({M} 个)
   ...

   #### L5 Slices ({K} 个)
   ...
   ```

2. 执行校验：`python scripts/validate_specs.py .vibex/specs`
3. 向用户确认：`请确认绑定计划，确认后我将写入 spec 文件。`
4. 用户确认后，逐个 `write_file` 生成/更新 YAML 文件

---

## 3. 输出规范

每次任务输出：

1. **标注解析结果**：标注树结构
2. **绑定计划清单**：L3/L4/L5 各层级文件列表 + 状态（创建/绑定）
3. **最终报告**：
   - `created[]`：新建的 spec 文件
   - `bound[]`：已绑定（已存在）的 spec 文件
   - `errors[]`：任何失败信息

---

## 4. 约束

- **幂等**：已存在的 spec 文件不覆盖内容，仅追加/更新 `children` 引用
- **仅写 `.vibex/specs/`**：不改任何业务代码
- **中文 slug 处理**：L3/L4/L5 的文件名 slug 使用 `data-vibex-title` 中中文部分的全拼（空格→下划线）或直接用中文文件名（如果文件系统支持）
- **原型路径**：`prototype_ref` 字段存储相对于 `.vibex/specs/` 的路径（向上三级：`../../prototypes/...`）

---

## 5. 示例

**输入**：`workspace_root=C:/project/vibex-workbench`，`prototype_path=.vibex/prototypes/vibex-workbench-mvp.html`

**前置检查**：通过（存在 `data-vibex-annot`）

**绑定计划**（部分）：

```
### L3 → L4 → L5 绑定计划

L3: titlebar (顶部标题栏)
 └─ L4: menu-strip (菜单栏)
     └─ L5: button[文件] / button[编辑] / button[视图] / button[终端] / button[帮助]
 └─ L4: command-center (命令中心)
 └─ L4: window-controls (窗口控制)
     └─ L5: button[初始化specs]
```

确认后生成文件：

- `.vibex/specs/L3-module/MOD-top-titlebar.yaml`（新建）
- `.vibex/specs/L4-feature/FEAT-menu-strip.yaml`（新建）
- `.vibex/specs/L5-slice/SLICE-menu-btn-file.yaml`（新建）
- ...
