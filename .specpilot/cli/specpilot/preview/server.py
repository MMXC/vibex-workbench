"""Preview HTTP server — serves prototype files with hot reload."""
import json
import os
import threading
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse

PORT = 5177
WORKSPACE = "."

# Track last modification time for hot reload
_file_mtimes: dict[str, float] = {}


class PreviewHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(f"[Preview] {fmt % args}")

    def guess_mime(self, path: str) -> str:
        ext = os.path.splitext(path)[1].lower()
        return {
            ".html": "text/html",
            ".css": "text/css",
            ".js": "application/javascript",
            ".json": "application/json",
            ".png": "image/png",
            ".jpg": "image/jpeg",
            ".svg": "image/svg+xml",
            ".ico": "image/x-icon",
            ".woff2": "font/woff2",
        }.get(ext, "application/octet-stream")

    def serve_file(self, filepath: str, code: int = 200):
        if not os.path.isfile(filepath):
            self.send_response(404)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"404 Not Found")
            return
        with open(filepath, "rb") as f:
            data = f.read()
        self.send_response(code)
        self.send_header("Content-Type", self.guess_mime(filepath))
        self.send_header("Content-Length", len(data))
        self.send_header("Cache-Control", "no-cache")
        self.end_headers()
        self.wfile.write(data)

    def inject_hot_reload(self, html: bytes) -> bytes:
        """Inject hot reload script into HTML."""
        script = b'<script>\n'
        script += b'// SpecPilot Hot Reload\n'
        script += b'(() => {\n'
        script += b'  const ws = new WebSocket(`ws://${location.host}/__live`);\n'
        script += b'  ws.onmessage = () => location.reload();\n'
        script += b'  ws.onclose = () => setTimeout(() => location.reload(), 1000);\n'
        script += b'})();\n'
        script += b'</script>\n'
        body = html.replace(b"</body>", script + b"</body>")
        if b"</body>" not in html:
            body = html + script
        return body

    def inject_dc_api(self, html: bytes) -> bytes:
        """Inject DataCenter API helpers into HTML."""
        dc_port = os.environ.get("SPECPILOT_DC_PORT", "7890")
        script = (
            f'<script>\n'
            f'window.__DC_PORT__ = {dc_port};\n'
            f'window.__dc = {{\n'
            f'  async get(key) {{ const r = await fetch(`http://127.0.0.1:${{window.__DC_PORT__}}/api/dc/${{key}}`); return (await r.json()).value; }},\n'
            f'  async set(key, val) {{ await fetch(`http://127.0.0.1:${{window.__DC_PORT__}}/api/dc/${{key}}`, {{method:"PUT",headers:{{"Content-Type":"application/json"}},body:JSON.stringify({{value:val}})}}); }},\n'
            f'  async list() {{ const r = await fetch(`http://127.0.0.1:${{window.__DC_PORT__}}/api/dc/`); return (await r.json()).keys; }},\n'
            f'}};\n'
            f'</script>\n'
        ).encode()
        body = html.replace(b"</head>", script + b"</head>")
        if b"</head>" not in html:
            body = script + html
        return body

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/")
        ws = self.server._workspace

        # Health
        if path == "/api/health":
            body = json.dumps({"ok": True, "service": "specpilot-preview"}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)
            return

        # API: list registered components
        if path == "/api/components":
            comp_path = os.path.join(ws, ".specpilot", "components", "index.json")
            if os.path.exists(comp_path):
                with open(comp_path) as f:
                    body = f.read().encode()
            else:
                body = json.dumps({"components": []}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            self.wfile.write(body)
            return

        # Serve /preview → previews/index.html
        if path == "/preview" or path == "/":
            filepath = os.path.join(ws, ".specpilot", "previews", "index.html")
            if not os.path.exists(filepath):
                filepath = os.path.join(ws, ".specpilot", "previews", "index.html")
            html = open(filepath, "rb").read() if os.path.exists(filepath) else b"<h1>No preview yet</h1>"
            html = self.inject_dc_api(html)
            html = self.inject_hot_reload(html)
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Content-Length", len(html))
            self.end_headers()
            self.wfile.write(html)
            return

        # Serve /components/<name>
        if path.startswith("/components/"):
            name = path[12:]
            comp_dir = os.path.join(ws, ".specpilot", "components", name)
            # Try index.html first
            for candidate in ["index.html", f"{name}.html"]:
                fp = os.path.join(comp_dir, candidate)
                if os.path.exists(fp):
                    html = open(fp, "rb").read()
                    html = self.inject_dc_api(html)
                    html = self.inject_hot_reload(html)
                    self.send_response(200)
                    self.send_header("Content-Type", "text/html")
                    self.send_header("Content-Length", len(html))
                    self.end_headers()
                    self.wfile.write(html)
                    return
            self.send_response(404)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Component not found: " + name.encode())
            return

        # Serve static files from .specpilot/
        if path.startswith("/static/"):
            real_path = path[8:]
            filepath = os.path.join(ws, ".specpilot", "static", real_path)
            self.serve_file(filepath)
            return

        # Fallback: serve from previews/
        filepath = os.path.join(ws, ".specpilot", "previews", os.path.basename(path))
        if os.path.isdir(filepath):
            filepath = os.path.join(filepath, "index.html")
        self.serve_file(filepath)


class LiveReloadWebSocket:
    """Minimal live reload broadcaster."""

    def __init__(self):
        self._clients: list = []
        self._lock = threading.Lock()

    def broadcast(self):
        with self._lock:
            for client in self._clients[:]:
                try:
                    client.send(b"reload\n")
                except Exception:
                    self._clients.remove(client)


_live = LiveReloadWebSocket()


class PreviewServer(HTTPServer):
    allow_reuse_address = True

    def __init__(self, port: int = PORT, workspace: str = "."):
        super().__init__(("127.0.0.1", port), PreviewHandler)
        self._port = port
        self._workspace = workspace

    def serve_forever(self):
        print(f"[Preview] PreviewServer listening on 127.0.0.1:{self._port}")
        print(f"[Preview] Preview URL: http://127.0.0.1:{self._port}/preview")
        super().serve_forever()


def reload_all():
    """Call after a file change to notify all connected clients."""
    _live.broadcast()
