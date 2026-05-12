---
name: vibex-spec-yaml-duplicate-key-fix
description: Fix duplicate top-level YAML keys in VibeX specs
triggers:
  - yaml unmarshal mapping key already defined
  - DuplicateKeyError in ruamel.yaml
  - SKIP in verify_specs output
---

# VibeX Spec YAML Duplicate Key Fix

## Trigger
`verify_specs` reports `SKIP: yaml unmarshal: mapping key "content" already defined at line N`

## Root Cause
VibeX L5 spec files (e.g. `SLICE-go-agent-cli.yaml`) have `content` as a top-level key 2-3 times. YAML standard forbids duplicate mapping keys at the same level.

## Fix Workflow

### Step 1: Locate duplicate content: lines
```bash
grep -n "^content:" specs/L5-slice/SLICE-*.yaml
```

### Step 2: Use ruamel.yaml (NOT safe_load)

Python's `yaml.safe_load` and `yaml.CSafeLoader` both reject duplicate keys. Must use `ruamel.yaml` + rename strategy.

```python
from ruamel.yaml import YAML
from io import StringIO

yaml_ruamel = YAML()
yaml_ruamel.preserve_quotes = True
yaml_ruamel.indent(mapping=2, sequence=4, offset=2)

with open('spec.yaml', 'r') as f:
    raw = f.read()

# Rename 2nd and 3rd "content:" to different keys so ruamel can parse
lines = raw.split('\n')
content_count = 0
new_lines = []
for line in lines:
    stripped = line.lstrip()
    if stripped == 'content:' and (len(line) - len(stripped)) == 0:
        content_count += 1
        if content_count == 1:
            new_lines.append(line)
        elif content_count == 2:
            new_lines.append(line.replace('content:', 'file_entry:'))
        elif content_count == 3:
            new_lines.append(line.replace('content:', 'file_entry_2:'))
    else:
        new_lines.append(line)

modified = '\n'.join(new_lines)
stream = StringIO(modified)
data = yaml_ruamel.load(stream)

# Extract blocks and merge into list
content_first = data.get('content', {})
file_entry1 = data.get('file_entry', {})
file_entry2 = data.get('file_entry_2', {})

new_content_list = [content_first]
if file_entry1:
    new_content_list.append(file_entry1)
if file_entry2:
    new_content_list.append(file_entry2)

data['content'] = new_content_list
for extra in ['file_entry', 'file_entry_2']:
    if extra in data:
        del data[extra]

stream_out = StringIO()
yaml_ruamel.dump(data, stream_out)
output = stream_out.getvalue()

import yaml
test = yaml.safe_load(output)  # verify
print(f"content type: {type(test['content'])}")
print(f"entries: {[c.get('file_path') for c in test['content']]}")

with open('spec.yaml', 'w') as f:
    f.write(output)
```

### Step 3: Verify with Go tool
```bash
./verify_specs --workspace . --format summary | grep -E "^(PASS|FAIL|SKIP)" | head -20
```

## Other common spec YAML issues

### generation_order multi-line string
Go expects `[]string` but YAML has a multi-line scalar.  
Fix: convert to YAML list:
```yaml
generation_order:
  - '1. first step'
  - '2. second step'
```

## Install dependency
```bash
pip install ruamel.yaml -t /root/.hermes/hermes-agent/venv/lib/python3.11/site-packages/
```

## Key pitfalls
- `ruamel.yaml` raises `DuplicateKeyError` by default on duplicate keys; rename first before parsing
- After fix, `content` type changes from `map[string]any` to `[]map[string]any`; Go code `Content` field must use `any` type
- `execute_code` uses sandbox Python, `terminal` uses venv Python; install ruamel to the correct environment
