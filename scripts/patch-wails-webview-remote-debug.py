#!/usr/bin/env python3
"""
Append Chromium --remote-debugging-port into Wails Windows setupChromium when
VIBEX_WEBVIEW2_REMOTE_DEBUG_PORT is used by Makefile wails-dev-cdp.

Why not WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS alone?
github.com/wailsapp/go-webview2/webviewloader/native_module.go calls
preventEnvAndRegistryOverrides(), which sets:
    os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", additionalBrowserArgs)
so any user-supplied WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS is overwritten with
the args computed inside Wails/go-webview2 (typically excluding CDP).

This patch is idempotent (marker: VIBEX_REMOTE_DEBUG_PATCH).
Patches the module copy under $(go env GOMODCACHE); affects all apps sharing
that exact module version on this machine.
"""
from __future__ import annotations

import os
import subprocess
import sys


MARKER = "VIBEX_REMOTE_DEBUG_PATCH"

PATCH_SNIPPET = (
	"\t// "
	+ MARKER
	+ " (see scripts/patch-wails-webview-remote-debug.py + Makefile wails-dev-cdp).\n"
	'\tif p := strings.TrimSpace(os.Getenv("VIBEX_WEBVIEW2_REMOTE_DEBUG_PORT")); p != "" {\n'
	'\t\tchromium.AdditionalBrowserArgs = append(chromium.AdditionalBrowserArgs, "--remote-debugging-port="+p)\n'
	"\t}\n"
)

ANCHOR_BEFORE = "\tif f.frontendOptions.DragAndDrop != nil && f.frontendOptions.DragAndDrop.DisableWebViewDrop {"


def module_root() -> str:
	out = subprocess.check_output(
		["go", "list", "-m", "-f", "{{.Dir}}", "github.com/wailsapp/wails/v2"],
		text=True,
		cwd=os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
	).strip()
	if not out:
		raise RuntimeError("go list -m github.com/wailsapp/wails/v2 returned empty")
	return out


def main() -> int:
	if sys.platform != "win32":
		print("[patch-wails-webview-remote-debug] skip: Windows WebView2 only")
		return 0

	root = module_root()
	path = os.path.join(root, "internal", "frontend", "desktop", "windows", "frontend.go")
	if not os.path.isfile(path):
		print(f"[patch-wails-webview-remote-debug] missing file: {path}", file=sys.stderr)
		return 1

	try:
		os.chmod(path, 0o644)
	except OSError:
		pass

	with open(path, encoding="utf-8") as f:
		text = f.read()

	if MARKER in text:
		print("[patch-wails-webview-remote-debug] already applied")
		return 0

	if ANCHOR_BEFORE not in text:
		print(
			"[patch-wails-webview-remote-debug] anchor not found; Wails version/layout may have changed",
			file=sys.stderr,
		)
		return 1

	text = text.replace(ANCHOR_BEFORE, PATCH_SNIPPET + "\n" + ANCHOR_BEFORE, 1)

	with open(path, "w", encoding="utf-8", newline="\n") as f:
		f.write(text)

	print(f"[patch-wails-webview-remote-debug] patched {path}")
	return 0


if __name__ == "__main__":
	sys.exit(main())
