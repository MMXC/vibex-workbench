---
name: yaml-block-scalar-pitfall
description: YAML解析器将双引号字符串内的竖线符误判为块标量指示符，导致后续内容被吞掉。常见于字段值含union type语法的场景。
category: skills
---

# YAML 块标量歧义陷阱

## 问题现象

YAML 解析器报错：
```
expected <block end>, but found '<scalar>'
```

且错误指向的行恰好是一个含 `|` 的字段值。

## 根因

当 YAML 字段值以双引号包裹但内容含 `|` 时，解析器在双引号结束后遇到顶层 `|` 后面的 `-` 映射项，把整个后续 YAML 当作块标量内容，导致块映射不匹配。

```yaml
# ❌ 会报错
- name: "viewMode"
  type: "'mermaid' | 'canvas'"
  description: "dsl-canvas 的视图模式
```

## 修复方案

**方案1（推荐）：描述性类型名**
```yaml
# ✅ 替代 union type，用描述性类型名
- name: "viewMode"
  type: "mermaid_or_canvas"
  description: "dsl-canvas 的视图模式，值: mermaid | canvas"
```

**方案2：统一单行字符串**
```yaml
# ✅ 所有多行内容用单行字符串
clarification_loop: "澄清循环：agent提问 -> user回答 -> agent建议 -> user确认"
behavior_preview: "Canvas展示：输入 -> spec变更 -> 迭代循环 -> 验证门"
```

## 陷阱2：patch 对复杂 YAML 的结构性损坏

### 问题现象

用 `patch` 编辑包含多行块标量（`|`, `>-`）和嵌套缩进的 YAML 文件时，patch 工具不理解 YAML 结构，会把替换文本插入到错误位置，导致：
- 重复的顶级 key（如两个 `io_contract:`）
- 块内容被截断
- 缩进层级错乱

### 典型案例

`routing-panel_feature.yaml` 的 `io_contract` 块被 patch 后插入到文件中间，导致重复顶级 key，验证报错：
```
YAML ERROR: while parsing a block mapping
  in "routing-panel_feature.yaml", line 66, column 7
expected <block end>, but found '<block sequence start>'
  in "routing-panel_feature.yaml", line 72, column 11
```

修复方式：直接用 `write_file` 重写整个文件，不用 patch 修。

### 判断规则

| 条件 | 用 patch | 用 write_file |
|---|---|---|
| 单行字符串值（`name: "xxx"`） | ✅ | 可选 |
| 简单布尔/数字值 | ✅ | 可选 |
| 含 `\|` 或 `>-` 块标量 | ❌ | ✅ |
| 嵌套超过3层缩进 | ❌ | ✅ |
| 多行列表+映射混合结构 | ❌ | ✅ |
| 顶级 key 增删（干净结构） | ✅ | 可选 |

### 验证命令

每次 patch 后立即验证：
```bash
python3 -c "
import yaml
try:
    yaml.safe_load(open('specs/feature/routing-panel/routing-panel_feature.yaml'))
    print('✅ YAML OK')
except Exception as e:
    print(f'❌ YAML ERROR: {e}')
"
```

### 陷阱3：嵌套键下的块标量返回空字符串

#### 问题现象

`yaml.safe_load` 解析含块标量的 YAML，但 `template` 字段内容为空：

```python
import yaml
with open('feature.yaml') as f:
    data = yaml.safe_load(f)
print(data.get('content', {}).get('template', ''))
# 输出: ''  ← 块标量内容被吞掉
```

YAML 文件中 `template: |` 明明写了多行内容，但解析结果是空。

#### 根因

当 `template: |` 嵌套在另一个映射键下（如 `content.template: |`），YAML 解析器在某些缩进/上下文组合下将块标量的**缩进基准线**设在错误位置，导致所有内容行都被判定为缩进不足而丢弃。

```yaml
# ❌ 嵌套在 content 下，块标量可能为空
content:
  description: "xxx"
  template: |
    # Feature Spec ${ID}
    spec:
      name: "${NAME}"

# ❌ 同样有问题：顶层但缩进层级深
content:
  sub_specs:
    template: |
      spec:
        name: "xxx"
```

#### 修复方案

**将 `template` 提升为顶层键**，与 `spec:` / `content:` 同级：

```yaml
# ✅ template 作为顶层键（YAML 解析行为最稳定）
spec:
  id: feature_template
  level: meta_template
  name: feature_template_feature

content:
  description: "L4 Feature spec 生成模板"

template: |
  # Feature Spec ${FEATURE_ID}
  spec:
    id: "${FEATURE_ID}"
    name: "${SAFE_NAME}"
```

解析时直接读顶层：

```python
data = yaml.safe_load(f)
template_content = data.get("template")  # ✅ 顶层键，内容完整
# 而不是：
# template_content = data.get("content", {}).get("template", "")  # ❌ 可能为空
```

## 预防规则

1. **YAML spec 中的 type 字段**：永远不用 `|` union syntax，改用 `_or_` 连词
2. **含块标量的复杂字段**：永远用 `write_file` 重写整个文件
3. **写完 YAML 后立即运行** `make validate` 捕捉语法错误
4. **`template` 等块标量字段**：永远作为 YAML 顶层键，放在 `spec:` / `content:` 同级，不要嵌套在任何映射键下方

---

## 陷阱4：`yaml.dump_all` 写入多文档文件导致 `--- null` 污染

### 问题现象

VibeX spec 中某些文件（如 GPT5.5 生成的 L5 specs）原是单文档 YAML，但文件末尾有 `---`，被 `safe_load_all` 识别为多文档边界。当你用 `yaml.dump_all` 写回时：

```python
# ❌ 错误：dump_all 会产生 "--- null" 分隔符
with open(f) as fh:
    docs = list(yaml.safe_load_all(fh))  # 读取
spec = docs[0]
modify_in_memory(spec)
with open(f, 'w') as fh:
    yaml.dump_all(docs, fh, ...)  # ← 写回后文件末尾出现 "--- null"
```

解析时报错：
```
expected a single document in the stream
but found another document
in "...", line N, column 1:
    --- null
```

### 根因

`dump_all` 对每个文档末尾都写 `---`，当原始文件末尾有孤立 `---` 时，写出后变成：
```yaml
spec:
  name: "xxx"
---   ← dump_all 写出的
null  ← dump_all 写出的（表示 None）
```

### 修复方案

**永远用 `yaml.safe_dump`（单文档）替代 `dump_all`**：

```python
with open(f) as fh:
    docs = list(yaml.safe_load_all(fh))  # 用 safe_load_all 读（兼容多文档）
spec = docs[0]  # 取第一个文档
modify_in_memory(spec)
with open(f, 'w') as fh:
    yaml.safe_dump(spec, fh,
        default_flow_style=False,
        allow_unicode=True,
        sort_keys=False,
        width=200)  # ← safe_dump 单文档，永不产生 --- null
```

### 判断规则

| 情况 | 用 `dump_all` | 用 `safe_dump` |
|------|--------------|----------------|
| 明确单文档文件 | ❌ | ✅ |
| 多文档文件（如多个 `---` 开头） | ✅ | ❌（会丢数据） |
| 不确定的 spec YAML | ❌ | ✅ |
| 批量处理（不确定文档数） | ❌ | ✅（每次只取 docs[0]） |

### 验证命令

```bash
python3 -c "
import yaml, glob
errors = []
for f in glob.glob('specs/L5-slice/*.yaml'):
    try:
        docs = list(yaml.safe_load_all(open(f)))
        count = len([d for d in docs if d is not None])
        if count != 1:
            errors.append(f'{f}: {count} docs')
    except Exception as e:
        errors.append(f'{f}: {e}')
if errors:
    for e in errors: print(f'❌ {e}')
else:
    print('✅ All single-document YAML OK')
"
```

---

## 陷阱5：文件末尾 `---` 导致多文档解析失败

### 问题现象

YAML 解析器报错：
```
YAML error xxx.yaml: expected a single document in the stream
expected a single document in the stream, but found another document
```

错误指向文件第 2 行（`spec:`），同时提示第 N 行有另一个 document。

### 根因

当 YAML 文件以 `---` 结尾（即使 `---` 后面没有任何内容），`safe_load` 会把它当作 **YAML 多文档分隔符**，认为文档在 `---` 处结束，文件剩余部分（空行）是第二个空文档。

常见场景：write_file 写入内容后自动追加了 `\n---\n`，或者手工在文件末尾加了 `---` 作为习惯性结尾。

```yaml
# ❌ 文件末尾有 ---，safe_load 认为是多文档
io_contract:
  input: "..."
...
changelog:
  - version: "0.1"
---
# ↑ 这里开始是第二个"文档"（实际是空的），导致解析失败
```

### 修复方案

**写完文件后去掉末尾的 `---\n`**：

```python
new_content = content.rstrip()
if new_content.endswith('---\n') or new_content.endswith('---'):
    new_content = new_content.rsplit('---', 1)[0].rstrip()
    with open(path, 'w') as fh:
        fh.write(new_content)
```

或者写作时直接不在末尾加 `---`。

### 验证命令

```bash
python3 -c "
import yaml
try:
    with open('xxx.yaml') as f:
        yaml.safe_load(f)
    print('✅ YAML OK')
except Exception as e:
    print(f'❌ YAML ERROR: {e}')
"
```

### 预防规则

1. **write_file 写入后立即 strip 末尾 `---`**：用脚本批量处理，防止手工遗漏
2. **模板文件末尾禁止 `---`**：spec-templates 的模板文件不应以 `---` 结尾
3. **批量修复脚本**（如需修复已有文件）：
   ```python
   import os
   for root, dirs, files in os.walk('specs/'):
       for f in files:
           if f.endswith('.yaml'):
               path = os.path.join(root, f)
               content = open(path).read()
               if content.rstrip().endswith('---'):
                   new = content.rstrip().rsplit('---', 1)[0].rstrip()
                   open(path, 'w').write(new)
                   print(f'Fixed: {path}')
   ```
