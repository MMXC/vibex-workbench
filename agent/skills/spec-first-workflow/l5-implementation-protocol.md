# L5 Implementation Protocol

Use this reference before modifying implementation files from specs.

## Required L5 Fields

An L5 slice is actionable only if it contains:

```yaml
content:
  file_path: "path/to/file"
  file_type: "svelte|typescript|go|python|html|..."
  implementation_boundary:
    allowed:
      - "..."
    forbidden:
      - "..."
  generation_rules:
    - rule: "..."
      detail: "..."
  verification:
    commands:
      - "..."
    manual_checks:
      - "..."
    failure_conditions:
      - "..."
  dependencies:
    - file: "..."
      reason: "..."
```

If any required field is missing, update L5 before code.

## Implementation Loop

1. Read the L5 slice.
2. Read `content.file_path`.
3. Check `implementation_boundary.allowed`.
4. Check `implementation_boundary.forbidden`.
5. Edit only allowed areas.
6. Run `verification.commands`.
7. If commands fail, classify using `validation-and-failure.md`.
8. Report manual checks separately.

## Boundary Rules

### Allowed Means Allowed

Only edit listed areas. If the desired edit falls outside the allowed list:

1. Stop.
2. Update L5 with the new boundary.
3. Run `make validate`.
4. Then implement.

### Forbidden Means Forbidden

Do not bypass forbidden items, even if it seems easier.

Examples:

- If L5 says "do not edit backend spawn", do not change `SpawnGoBackend`.
- If L5 says "do not fetch in this component", do not add API calls there.
- If L5 says "no global titlebar in app.html", do not add window buttons there.

## Dependency Handling

If implementation requires changing a dependency file:

1. Check whether the dependency has its own L5.
2. If yes, follow that L5.
3. If no, create a small L5 for that file.

Do not silently expand scope.

## Verification Rules

Run commands in the L5 in order unless the command is clearly impossible in the current environment.

Common commands:

```bash
make validate
npm --prefix frontend run build
go build .
```

For manual checks, report them as not automated:

```text
Manual check required:
- Run make wails-dev and verify only WebView titlebar is visible.
```

## Reporting Template

After implementation, report:

```markdown
Implemented from L5:
- <slice name>

Files changed:
- <file>

Verification:
- <command>: pass/fail
- Manual: <needed/not run>

Residual risk:
- <risk or none>
```

## When to Update Spec Again

Update spec before continuing if:

- The target file differs substantially from L5 assumptions.
- A forbidden edit becomes necessary.
- A required verification command is missing.
- The implementation requires another file not listed in dependencies.
- The user changes the product boundary mid-implementation.
