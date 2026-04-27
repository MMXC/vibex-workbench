---
spec:
  version: "0.1"
  level: skill
  name: mvp-impl-by-hand
  status: active
  created: "2026-04-27"
  updated: "2026-04-27"
meta:
  type: workflow
  module: coordination
  owner: hermes
  tags: [vibex-workbench, implementation, mvp, spec-driven]
lifecycle:
  current: active
  history:
    - at: "2026-04-27"
      by: hermes
      note: "mvp-impl cron job failed 24x; direct implementation proved reliable
---

# MVP Spec Implementation — Direct Closed-Loop

## When to Use

When implementing a sequence of specs in order (e.g. P0→P1→P2),
rather than delegating to an unreliable cron/agent job.

## Per-Spec Workflow

### Step 1 — Read spec
read_file the L4/L5 spec file; note `file_path`, `io_contract`, `behavior`

### Step 2 — Find implementation target
search_files frontend/src/ or agent/cmd/web/ for the target file path from the spec

### Step 3 — QA before coding
Verify preconditions first:

Python syntax + logic:
```
python3 -m py_compile generators/state_detector.py
python3 generators/state_detector.py /tmp/qa-empty --json
```

Go build (run from go.mod root, not single-file vet):
```
cd agent && go build ./cmd/web/
```

### Step 4 — Write code
Use patch for precise edits; match existing style

### Step 5 — Commit (two commits)
1. Feature: `git add -A && git commit -m "feat: FEAT-xxx — short desc"`
2. Doc update: `git add -A && git commit -m "docs: FEAT-xxx → done in spec_status"`

### Step 6 — Update state file
Edit /root/.hermes/mvp-impl-state.json:
- current spec: status=done, commit=<hash>, completed_at=today
- current += 1

### Step 7 — Update spec_status
Edit specs/L1-goal/vibex-workbench-mvp.yaml:
- spec_status table: FEAT-xxx status=done, note=brief summary
- changelog prepend: P0-N: FEAT-xxx done

### Step 8 — Push
git push

### Step 9 — Final validation
```
cd /root/vibex-workbench && python3 $(find . -name validate_specs.py)
# Expect: [OK] All specs passed.
```

## Common Pitfalls

### Go build failure
- Duplicate declarations: grep -n 'func Name' target.go to confirm uniqueness
- vet single-file cfg missing: run `go build ./cmd/web/` from the go.mod root dir, not `go build file.go`
- Stale residual packages: grep -r 'import "old/pkg"' agent/ to find references, then delete them
- **handlers_workspace.go**: stale antch backend duplicate, delete if blocking build

### Svelte type errors
Known issue: missing @sveltejs/adapter-static causes svelte-check to error on workspace/+page.svelte
Validating: check if your changed filename appears in the error output
Real new error pattern: `Error: YourFile.svelte:N:M — your-code-here`

### API endpoint not registered
First: grep -n 'HandleFunc.*path' agent/cmd/web/main.go to verify registration
If missing: add `http.HandleFunc("/api/xxx", withCORS(xxxHandler))`

## State File Format

/root/.hermes/mvp-impl-state.json:
```
{
  "current": 0,
  "specs": [
    {
      "name": "FEAT-xxx",
      "file": "specs/L4-feature/FEAT-xxx.yaml",
      "phase": "P0",
      "status": "proposal",
      "commit": null,
      "completed_at": null
    }
  ]
}
```

## Anti-patterns
- Do NOT try to fix flaky cron/agent jobs: direct implementation is more reliable
- Do NOT rewrite entire files: use patch for targeted replacements
- Do NOT skip validate_specs.py: run before every push
