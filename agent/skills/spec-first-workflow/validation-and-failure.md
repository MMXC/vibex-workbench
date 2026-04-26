# Validation And Failure Reference

Use this reference to interpret command results without relying on free-form terminal logs.

## Primary Commands

| Command | Purpose | Success Signal |
|---|---|---|
| `make validate` | YAML syntax + parent chain | exit 0, `[validate] OK` |
| `make generate` | spec-to-code generation | exit 0, `[generate] OK` |
| `make gen-graph` | graph emission | exit 0, output file written |
| `npm --prefix frontend run build` | frontend compile | exit 0, `✓ built` |
| `go build .` | Wails Go compile | exit 0 |
| `make wails-dev` | desktop runtime | manual steady state |

Prefer L5 `verification.commands` over this table.

## Failure Categories

### yaml_parse_error

Symptoms:

- `YAML error`
- `could not find expected ':'`
- `expected a single document`

Action:

1. Fix YAML formatting only.
2. Do not change implementation files.
3. Re-run `make validate`.

### parent_not_found

Symptoms:

- `parent '<name>' not found`

Action:

1. Search spec index for the intended parent.
2. Fix `spec.parent` if typo.
3. Create missing parent spec only if the parent concept truly exists.
4. Re-run `make validate`.

### level_mismatch

Symptoms:

- spec appears under wrong level group
- level field conflicts with filename/path convention

Action:

1. Fix `spec.level`.
2. Move file only if explicitly required.
3. Re-run `make validate`.

### missing_l5_boundary

Symptoms:

- User asks for code implementation but only L4 exists.
- L5 lacks `implementation_boundary`.
- L5 lacks `verification.commands`.

Action:

1. Create/update L5 slice.
2. Run `make validate`.
3. Implement after validation passes.

### build_error

Symptoms:

- `npm run build` or `go build .` exits non-zero.
- Compiler points to implementation file.

Action:

1. Check whether failing file is inside L5 boundary.
2. If yes, fix implementation.
3. If no, update/add L5 before touching file.

### typecheck_error

Symptoms:

- `svelte-check` / TypeScript diagnostics
- Existing generated `types.ts` errors

Action:

1. Determine whether diagnostics are pre-existing.
2. Fix only diagnostics introduced by current task.
3. If pre-existing, report separately and do not broaden scope.

### runtime_manual_check

Symptoms:

- Behavior requires Wails window interaction.
- Drag/no-drag, native menu visibility, window buttons.

Action:

1. Run compile commands.
2. Ask for or perform manual Wails check if possible.
3. Report as manual verification if not run.

### generated_drift

Symptoms:

- `make generate` modifies `.Skeleton.svelte`, services, or `types.ts`.
- Git status shows generated files after generator run.

Action:

1. Identify whether generated changes are expected by spec.
2. Do not commit unrelated generated drift with implementation unless requested.
3. If generated output conflicts with desired code, update generator/spec rather than hand-editing generated files.

## Response Rules

Report failures in this format:

```markdown
Validation failed: <category>

Evidence:
- <short output excerpt>

Next action:
- <fix spec / fix implementation / ask user / manual check>
```

Do not paste long terminal logs. Summarize the few lines needed for the category.

## Long-Running Tasks

For commands like `make wails-dev`:

1. Treat startup as successful only when a healthy steady-state message appears.
2. If blocked by backend spawn, classify as `runtime_manual_check` or `build_error` depending on source.
3. Do not wait indefinitely for interactive prompts.
4. Prefer explicit task IDs or logs over streaming noise when available.
