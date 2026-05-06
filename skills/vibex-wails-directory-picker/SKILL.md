---
name: vibex-wails-directory-picker
title: Wails Directory Picker Full Path Fix
description: Fix Wails v2 OpenDirectoryDialog on Windows returning folder name instead of full path — two-layer solution with sqweek/dialog and frontend fallback logic
tags:
  - vibex-workbench
  - wails
  - windows
  - directory-picker
updated: 2026-04-30
---

# vibex-wails-directory-picker

## Problem

Wails v2 `runtime.OpenDirectoryDialog` on Windows (WebView2) only returns the **folder name**, not the full absolute path.

**Symptom:** User selects `C:\Users\xxx\vibex-workbench` but the app receives `"vibex-workbench"` as the workspace root.

**Root cause:** Wails delegates to WebView2's native folder picker, which doesn't expose the full path in Wails v2 on Windows.

## Solution: Two-layer fallback

### Layer 1 — Frontend (wails-runtime.ts)

Call `openDirectoryDialog()` which tries in order:

1. **File System Access API** `showDirectoryPicker()` — Returns `FileSystemDirectoryHandle` with optional `.path` property (Chromium extension). If `path` exists, use it. If not, **do not return the folder name** — throw and continue.
2. **Wails runtime** `rt.OpenDirectoryDialog()` — After switch to `sqweek/dialog` on Go side, this returns full path. Check: if result has path separators (`/` or `` ), accept it. If only folder name, do NOT accept — continue to fallback.
3. **`<input webkitdirectory>`** — Last resort, path may be incomplete.

**Critical catch logic:**
```typescript
catch (e: any) {
  if (e?.name === 'AbortError' || e?.message?.includes('cancelled')) {
    return ''; // user cancelled — stop here
  }
  // other errors (no path property) — continue to next priority
}
```

Do NOT `return ''` on "no path property" error — that short-circuits the Wails fallback.

### Layer 2 — Go Backend (main.go)

Replace Wails `runtime.OpenDirectoryDialog` with `github.com/sqweek/dialog`.

**Install:**
```bash
GOPROXY=https://rsproxy.cn,direct GONOSUMDB='*' go get github.com/sqweek/dialog
```

**Usage:**
```go
import "github.com/sqweek/dialog"

func (a *App) OpenDirectoryDialog(ctx context.Context) (string, error) {
    dir, err := dialog.Directory().
        SetStartDir(a.workspaceRoot).
        Title("选择工作区目录").
        Browse()
    if err != nil {
        return "", nil // user cancelled
    }
    if dir == "" {
        return "", nil
    }
    // Ensure absolute path
    if !filepath.IsAbs(dir) {
        abs, err := filepath.Abs(dir)
        if err != nil {
            return "", fmt.Errorf("failed to resolve absolute path: %w", err)
        }
        dir = abs
    }
    a.workspaceRoot = dir
    runtime.EventsEmit(ctx, "workspace:selected", dir)
    return dir, nil
}
```

`sqweek/dialog` calls native Win32 `SHBrowseForFolder` APIs directly — not through WebView2 — so it returns the real filesystem path.

## Console diagnostics to look for

| Log | Meaning |
|-----|---------|
| `[openDirectoryDialog] showDirectoryPicker no path property, got name: xxx` | File System Access API returned folder name only — should continue to Wails fallback |
| `[openDirectoryDialog] Wails returned folder name only: xxx` | Wails runtime returned folder name — sqweek/dialog not active yet |
| `[openDirectoryDialog] showDirectoryPicker cancelled` | User cancelled — expected, no action needed |

## Why not just use `document.title` or URL tricks?

Those don't work. WebView2 sandboxed environment doesn't expose the full path through document properties. The only reliable path comes from native Win32 APIs (sqweek/dialog) or Chromium's `showDirectoryPicker().path` (unreliable in WebView2).

## Related files

- `frontend/src/lib/wails-runtime.ts` — `openDirectoryDialog()` implementation
- `main.go` — `OpenDirectoryDialog` Wails binding
- `specs/L5-slice/SLICE-ide-titlebar-component.yaml` — Runtime fallback spec rule
