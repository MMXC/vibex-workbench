#!/usr/bin/env python3
"""Patch index.html with cache-busting version timestamp."""
import sys
import re
import pathlib

if len(sys.argv) != 2:
    print("Usage: cache_bust.py <index.html>")
    sys.exit(1)

html_path = pathlib.Path(sys.argv[1])
if not html_path.exists():
    print(f"[cache-bust] ERROR: {html_path} not found")
    sys.exit(1)

html = html_path.read_text(encoding="utf-8")
ts = str(int(__import__("time").time()))

def buster(m):
    url = m.group(1)
    sep = "&" if "?" in url else "?"
    return m.group(0).replace(m.group(1), url + sep + "v=" + ts)

html = re.sub(r'(src|href)="(/[^"]+)"', buster, html)
html_path.write_text(html, encoding="utf-8")
print(f"[cache-bust] v={ts}")
