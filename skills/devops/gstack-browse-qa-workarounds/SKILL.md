---
name: gstack-browse-qa-workarounds
description: gstack-browse-qa-workarounds — skill for devops category
category: devops
triggers:
- deploy
- CI/CD
- Cloudflare
- webhook
- monitoring
- Next.js QA
- gstack browse qa workarounds
related_skills:
- systematic-debugging
- test-driven-development
- gstack-browse
---
repo_tracked: true


## Browse Binary Path

The gstack QA skill's setup check (`NEEDS_SETUP`) is unreliable — it looks for the binary in the wrong place.

**Correct binary location:**
```
/root/.openclaw/skills/gstack-browse/bin/browse
```
This is a standalone ELF binary, NOT in `browse/dist/` or `browse/src/`.

**Safe setup check:**
```bash
B=/root/.openclaw/skills/gstack-browse/bin/browse
if [ -x "$B" ]; then
  echo "READY: $B"
else
  echo "NEEDS_SETUP"
fi
```

**Preamble env vars (always set before browse commands):**
```bash
export CI=true
export BROWSE_SERVER_SCRIPT=/root/.openclaw/skills/gstack-browse/src/server.ts
export PLAYWRIGHT_BROWSERS_PATH=~/.cache/ms-playwright
```

**To start the server (if not already running):**
```bash
B=/root/.openclaw/skills/gstack-browse/bin/browse
$B status 2>&1 | head -3
# If not running: the binary auto-starts on first command
```

---

## Screenshot Path Restriction

Screenshots MUST be saved under `/tmp/` — the browse tool rejects other paths.

```bash
# WRONG — will fail:
$B screenshot ~/.gstack/qa-reports/screenshots/canvas.png

# CORRECT — save to /tmp, then copy:
$B screenshot /tmp/canvas.png
cp /tmp/canvas.png ~/.gstack/qa-reports/screenshots/
```

Workaround: Always save to `/tmp/` first, then `cp` to final destination.

---

## React Error #300 (Hydration) in Next.js Apps

When a Next.js app shows React Error #300 after navigation:

1. Clear the console buffer first to distinguish old from new errors:
   ```bash
   $B console --clear
   $B goto <url>
   sleep 2
   $B console --errors
   ```
2. "Try Again" buttons in Next.js error boundaries usually recover (SPA takes over)
3. For root cause: run `npm run dev` locally and navigate to the same route to get full error stack

---

## Direct URL vs. Redirect URL Behavior

Some Next.js apps behave differently when:
- Navigated via redirect from `/` → `/canvas` (works)
- Directly accessed at `/canvas` (may fail with hydration error)

Always test both paths in QA.

---

## Console Error Deduplication

Console errors from previous navigations can linger in the buffer. Always `console --clear` before a fresh navigation test to get accurate error counts.

---

## Mobile Viewport Testing

```bash
$B viewport 375x812
$B screenshot /tmp/mobile.png
$B viewport 1280x720  # restore desktop
```

---

## `snapshot -C` Deep Scan (Always Do This)

Standard `snapshot -i` only shows ARIA tree. For hidden DOM elements (drawers, alert dialogs, overlays), always append `-C`:

```bash
$B snapshot -C -o /tmp/deep.png   # finds cursor-interactive and hidden elements
```

**When to use deep scan:**
- Drawer/modal content not appearing in standard snapshot
- `@eXX` refs not found after clicking a button that should open something
- Alert dialogs or non-ARIA overlays

---

## "Selector matched multiple elements" — Don't Blame Browse

This error means **the page has duplicate DOM elements** (e.g., a toolbar + shortcuts bar sharing the same buttons). Browse is correctly finding multiple matches.

**Pattern:** If you see 3+ consecutive failures with this error, the page is the problem:
```bash
Selector matched multiple elements. Be more specific or use @refs from 'snapshot'.
```

**Workarounds:**
1. Use `snapshot -C` to find the full DOM tree with cursor-interactive elements
2. Use `$B click` with a `@ref` from the most recent snapshot (refs change between calls — always re-snapshot first)
3. Use `$B press Escape` to close modals first before interacting
4. If a button is genuinely duplicated in the page, use the most recent `@ref` from the last `snapshot -i`

**Rule:** Always re-snapshot before clicking when the previous command changed the page state.

---

## Ref IDs Change Between Snapshot Calls

`@e1`, `@e22`, etc. **do not persist across different snapshot calls**. After any navigation or action, re-snapshot to get fresh refs.

```bash
# WRONG — ref may have changed
$B click @e14
$B screenshot /tmp/step2.png  # uses stale ref from previous snapshot

# CORRECT — re-snapshot after every action
$B click @e14
$B snapshot -i  # fresh refs
$B click @e22  # now using current ref
```

---

## Escape Key for Modal/Drawer Close

Pressing `Escape` is often more reliable than clicking the ✕ button when closing modals, especially when:
- The ✕ button has a conflicting ref (appears in both toolbar and shortcuts bar)
- The button click times out with "Selector matched multiple elements"
- Multiple modals are stacked

```bash
$B press Escape
$B wait --networkidle
$B snapshot -i  # verify closed
```

**Note:** On this app specifically, `Escape` successfully closes the version history modal and shortcuts help dialog. Works on most SPA modals that respond to `keydown` events.

---

## Drawer/Modal State Persists After Page Refresh

In SPAs, drawers and modals can re-open after F5 because their open/closed state is stored in localStorage or component state that survives hydration. This causes:
- "关闭抽屉" button showing instead of "打开抽屉" on fresh load
- Modal appearing immediately after navigation
- Confusing ARIA tree state (`[pressed]` instead of normal button)

**Detection:** After `$B goto`, always check the first snapshot for `[pressed]` buttons — they indicate a persisted open state.

**Impact on QA:** When you see a button labeled "关闭X" after a fresh `goto`, that drawer was already open from a previous session. Use it to your advantage (you can test the drawer without clicking) or close it first with `Escape` for a clean slate.

---

## Template/Drawer Click Timeout (5s)

Some buttons (template, specific drawers) can timeout with:
```
Operation timed out: click: Timeout 5000ms exceeded.
```

This is usually NOT a Playwright issue — it means the button's click handler is either:
1. Hung on a Promise that never resolves
2. Caught in an infinite loop
3. Waiting for network resources that fail silently

**Workaround:** Use `$B press Escape` to recover and continue testing. Document as a Medium-severity bug.
