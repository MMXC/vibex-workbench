"""Preview HTTP server — serves prototype files with hot reload."""
import json, os, threading, time, sys
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse

_PORT_ARG = None
_WS_ARG = None
for i, a in enumerate(sys.argv):
    if a == "--port" and i + 1 < len(sys.argv):
        _PORT_ARG = int(sys.argv[i + 1])
        break

PORT = _PORT_ARG or int(os.environ.get("SPECPILOT_MF_PORT", "5177"))
WORKSPACE = os.environ.get("SPECPILOT_WORKSPACE", ".")

# Track last modification time for hot reload
_file_mtimes: dict[str, float] = {}


class PreviewHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    close_connection = True  # always close — prevents HTTP/1.1 keep-alive hang

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
            self.wfile.flush()
            return
        with open(filepath, "rb") as f:
            data = f.read()
        self.send_response(code)
        self.send_header("Content-Type", self.guess_mime(filepath))
        self.send_header("Content-Length", len(data))
        self.send_header("Cache-Control", "no-cache")
        self.end_headers()
        self.wfile.write(data)
        self.wfile.flush()

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

    def inject_dc_base(self, html: bytes) -> bytes:
        """Inject DC/EC base URLs into MF component HTML as window vars."""
        dc_port = os.environ.get("SPECPILOT_DC_PORT", "7890")
        mf_port = os.environ.get("SPECPILOT_MF_PORT", "5177")
        script = (
            f'<script>\n'
            f'window.__specpilotDcBase = "http://localhost:{dc_port}/api";\n'
            f'window.__specpilotMfBase = "http://localhost:{mf_port}";\n'
            f'// Listen for init message from parent\n'
            f'window.addEventListener("message", function(e) {{\n'
            f'  if (e.data && e.data.type === "__specpilot_init") {{\n'
            f'    window.__specpilotDcBase = e.data.dcBase;\n'
            f'    window.__specpilotMfBase = e.data.mfBase;\n'
            f'  }}\n'
            f'}});\n'
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
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            self.wfile.flush()
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
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            self.wfile.flush()
            return

        # API: spec file mtime for hot reload

        # Serve /preview → previews/index.html
        if path == "/preview" or path == "/":
            filepath = os.path.join(ws, ".specpilot", "previews", "index.html")
            if not os.path.exists(filepath):
                filepath = os.path.join(ws, ".specpilot", "previews", "index.html")
            html = open(filepath, "rb").read() if os.path.exists(filepath) else b"<h1>No preview yet</h1>"
            html = self.inject_hot_reload(html)
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Content-Length", len(html))
            self.end_headers()
            self.wfile.write(html)
            return

        # API: spec file mtime for hot reload
        if path.startswith("/api/spec/mtime"):
            import urllib.parse
            qs = urllib.parse.parse_qs(parsed.query)
            spec_path = qs.get("path", [None])[0]
            if spec_path and os.path.isabs(spec_path) and os.path.exists(spec_path):
                mtime = os.path.getmtime(spec_path)
                body = json.dumps({"mtime": mtime}).encode()
            else:
                body = json.dumps({"mtime": None}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)
            return

        # Serve /components/<name> — MF component shell
        if path.startswith("/components/"):
            name = path[12:]
            comp_dir = os.path.join(ws, ".specpilot", "components", name)
            for candidate in ["index.html", f"{name}.html"]:
                fp = os.path.join(comp_dir, candidate)
                if os.path.exists(fp):
                    html = open(fp, "rb").read()
                    html = self.inject_hot_reload(html)
                    self.send_response(200)
                    self.send_header("Content-Type", "text/html")
                    self.send_header("Content-Length", len(html))
                    self.end_headers()
                    self.wfile.write(html)
                    return
            # Fallback: serve MF shell from global install
            # __file__ = /tmp/specpilot/cli/specpilot/preview/server.py
            # go up 3 levels to /tmp/specpilot/, then mf/index.html
            global_mf = os.path.join(os.path.dirname(__file__), "..", "..", "..", "mf", "index.html")
            global_mf = os.path.normpath(global_mf)
            if os.path.exists(global_mf):
                html = open(global_mf, "rb").read()
                html = self.inject_dc_base(html)
                html = self.inject_hot_reload(html)
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.send_header("Content-Length", str(len(html)))
                self.end_headers()
                self.wfile.write(html)
                self.wfile.flush()
                return
            self.send_response(404)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(f"Component not found: {name}".encode())
            self.wfile.flush()
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


if __name__ == "__main__":
    print(f"[Preview] PreviewServer listening on 127.0.0.1:{PORT}")
    print(f"[Preview] Preview URL: http://127.0.0.1:{PORT}/preview")
    PreviewServer(port=PORT, workspace=WORKSPACE).serve_forever()
