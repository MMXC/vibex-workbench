---
name: spec-chain-audit
description: Audit and fix vibex-workbench spec parent chains — detect breaks, fix missing parents, clean trailing empty docs, reconcile L2 naming conventions.
triggers:
  - "validate_specs.py 报错 parent chain"
  - "L4 parent 指向 L2 而非 L3"
  - "L2 目录叫 architecture 还是 L2-skeleton"
  - "多文档 YAML safe_load 失败"
---

## Audit Workflow

### 1. Run validation
```bash
python3 generators/validate_specs.py specs/
```

### 2. Check directory structure
Canonical layout uses `L1-goal / L2-skeleton / L3-module / L4-feature / L5-slice`.
Legacy dirs to remove: `architecture/`, `module/`, `feature/`, `project-goal/`, `meta/`, `p-metaspec/`, `L2-skeleton/` (if it duplicates the canonical one).

### 3. Analyze each parent chain error

| Error pattern | Cause | Fix |
|---|---|---|
| `parent 'XXX' not found (specs/L5-slice/L4-feature/...)` | `SPEC_DIR` wrongly set to scan subdir | Fix validate_specs.py head (see below) |
| `L4 → L2 skeleton name` | L4 parent should be L3 MOD, not L2 | Change parent to `MOD-xxx` |
| `YAML ComposerError: expected single document` | Trailing `---` empty doc marker | Strip it (see clean_trailing_docs.py below) |
| `L2 in architecture/ not L2-skeleton/` | Wrong directory name | Move to `specs/L2-skeleton/`, delete `architecture/` |

### 4. L2 naming rule
**Always use `L2-skeleton/`, NOT `architecture/`** — spec-templates uses `L2-skeleton/` naming.

### 5. L4 parent rule
L4 must point to L3 MOD (e.g. `MOD-workspace-root`), not L2 skeleton.

## validate_specs.py SPEC_DIR Bug

When called with a subdirectory arg (e.g. `specs/L5-slice/`), `SPEC_DIR` was set to that subdir, causing parent lookups to resolve to wrong paths.

**Fix** — patch the head of `validate_specs.py`:
```python
_scan_root_arg = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("specs")
SPEC_DIR = _scan_root_arg
# If scanning a subdirectory (e.g. specs/L5-slice/), resolve SPEC_DIR to specs/ root
if SPEC_DIR.name in ("L1-goal", "L2-skeleton", "L3-module", "L4-feature", "L5-slice",
                     "architecture", "module", "feature", "project-goal"):
    SPEC_DIR = SPEC_DIR.parent
# Also update the scan line in main():
# FROM: yaml_files = list(SPEC_DIR.rglob("*.yaml"))
# TO:   yaml_files = list(_scan_root_arg.rglob("*.yaml"))
```

## Clean trailing empty doc markers

```python
import os
for root, dirs, files in os.walk("/path/to/specs"):
    for f in files:
        if not f.endswith('.yaml'): continue
        path = os.path.join(root, f)
        with open(path) as fh:
            content = fh.read()
        if content.rstrip().endswith('\n---'):
            new_content = content.rstrip()[:-4].rstrip() + '\n'
            with open(path, 'w') as fh:
                fh.write(new_content)
            print(f"Fixed: {path}")
```

## Verification

```bash
python3 generators/validate_specs.py specs/      # must pass
python3 generators/validate_specs.py specs/L5-slice/  # must also pass
ls specs/  # must show: L1-goal L2-skeleton L3-module L4-feature L5-slice
```
