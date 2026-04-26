---
name: spec-deletion-protocol
description: Spec 文件删除安全协议 — 在 spec-driven 项目中删除文件前必须验证内容，防止误删其他 agent/session 生成的资产。适用于 vibex-workbench 多 agent 协作环境。
category: spec-driven
title: Spec Deletion Protocol
triggers:
- 删除 spec
- 清理旧文件
- rm 存根
- 删除旧 spec
- 误删恢复
- spec cleanup
related_skills:
- spec-first-workflow
- validate-specs
---

# Spec 删除安全协议

## 核心教训（2026-04-26）

在 vibex-workbench 中，误删了 GPT5.5 agent 生成的 9 个 SLICE-ide-*.yaml 文件。
原因：文件在磁盘上是多文档格式（`safe_load_all` 能解析），但我扫描时只看到了文件名，就判断为空存根并删除。

**教训：文件名不能判断内容价值。必须读文件内容再决定。**

---

## 删除前检查清单

在删除任何 spec 文件前，必须完成以下步骤：

### Step 1: 确认文件来源

```bash
# 查看 git 历史，谁在什么 commit 写了这个文件
git log --oneline --all -- {path}

# 如果有 parent commit，查看提交者信息
git show {commit} --stat | grep {filename}
```

### Step 2: 验证文件内容

```bash
# 先读前 30 行，确认有实际内容
head -30 {path}

# 检查 frontmatter 是否完整
python3 -c "
import yaml
docs = list(yaml.safe_load_all(open('{path}')))
for i, d in enumerate(docs):
    if d:
        print(f'Doc {i}: level={d[\"spec\"].get(\"level\",\"?\")} name={d[\"spec\"].get(\"name\",\"?\")} parent={d[\"spec\"].get(\"parent\",\"?\")}')
"
```

### Step 3: 评估价值

| 情况 | 决策 |
|------|------|
| 有完整 frontmatter + content | **保留** — 不要删 |
| 有 frontmatter 但 content 为空 | 确认是否「占位桩」，询问用户 |
| 纯格式错误（无法 parse） | 可删，但先 commit 再删 |
| 确认是旧版本已迁移 | 可删，但记录迁移目标 |

### Step 4: 删除方式

```bash
# 永远不用 rm — 用 git rm + commit
git rm {path}
git commit -m "chore: remove stale spec — {reason}"
```

### Step 5: 误删恢复

如果已经删了，立即恢复：

```bash
# 从 parent commit 恢复
git show {parent_commit}:{path} > {path}

# 或者用 git checkout
git checkout {parent_commit} -- {path}
```

---

## 多 Agent 环境特别注意事项

在 vibex-workbench 中：
- **不同的 agent**（Go agent / GPT5.5 / M2.7 / 手动）可能在不同 session 生成 spec
- **文件名相同不代表内容相同** — 同名文件在不同时刻由不同 agent 生成，内容可能完全不同
- **磁盘空 ≠ 无价值** — 文件可能在某个 commit 里，但后来被错误覆盖

**原则：看到不熟悉的文件名时，假设它有价值，先读再删。**

---

## 快速检查脚本

```bash
# 检查目录中所有 yaml 文件的 frontmatter 完整性
python3 -c "
import yaml, glob, sys
for f in glob.glob('specs/**/*.yaml', recursive=True):
    try:
        docs = list(yaml.safe_load_all(open(f)))
        for d in docs:
            if d and 'spec' in d:
                name = d['spec'].get('name','?')
                level = d['spec'].get('level','?')
                parent = d['spec'].get('parent','?')
                print(f'OK  {level} {name} (parent: {parent})')
    except Exception as e:
        print(f'ERR {f}: {e}')
"
```
