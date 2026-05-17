"""DataCenter HTTP API server — in-memory KV store with HTTP API."""
import json
import os
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse

PORT = 7890
WORKSPACE = "."  # overridden at init


class DataCenter:
    """In-memory key-value store."""

    def __init__(self):
        self._store: dict = {}
        self._lock = threading.RLock()
        self._version = 0

    def set(self, key: str, value) -> dict:
        with self._lock:
            self._store[key] = value
            self._version += 1
        return {"ok": True, "key": key, "value": value, "version": self._version}

    def get(self, key: str):
        with self._lock:
            return self._store.get(key)

    def list(self) -> dict:
        with self._lock:
            return dict(self._store)

    def delete(self, key: str):
        with self._lock:
            if key in self._store:
                del self._store[key]
                self._version += 1
                return {"ok": True, "deleted": key}
        return {"ok": False, "error": "key not found"}

    def subscribe(self, key: str, callback):
        """For EC integration."""
        pass


# Global instance
_dc = DataCenter()


def set_dc_port(port: int):
    global PORT
    PORT = port


class DCHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(f"[DC] {fmt % args}")

    def send_json(self, data: dict, code: int = 200):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", len(body))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET,PUT,DELETE,OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/")

        # GET /api/health
        if path == "/api/health":
            with _dc._lock:
                count = len(_dc._store)
            self.send_json({
                "ok": True,
                "service": "specpilot-dc",
                "version": _dc._version,
                "key_count": count,
                "dc_port": PORT,
                "mf_port": int(os.environ.get("SPECPILOT_MF_PORT", 5177)),
            })
            return

        # GET /api/dc/ → list all
        if path == "/api/dc" or path == "/api/dc/":
            self.send_json({"ok": True, "keys": _dc.list()})
            return

        # GET /api/dc/list
        if path == "/api/dc/list":
            self.send_json({"ok": True, "keys": _dc.list()})
            return

        # GET /api/dc/<key>
        if path.startswith("/api/dc/"):
            key = path[8:]
            val = _dc.get(key)
            if val is None:
                self.send_json({"ok": False, "error": "not found"}, 404)
            else:
                self.send_json({"ok": True, "key": key, "value": val})
            return

        self.send_json({"ok": False, "error": "not found"}, 404)

    def do_PUT(self):
        parsed = urlparse(self.path)
        if not parsed.path.startswith("/api/dc/"):
            self.send_json({"ok": False, "error": "invalid path"}, 400)
            return

        key = parsed.path[8:]
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        try:
            data = json.loads(body)
            value = data.get("value")
        except (json.JSONDecodeError, KeyError):
            self.send_json({"ok": False, "error": "invalid JSON"}, 400)
            return

        result = _dc.set(key, value)
        self.send_json(result)

    def do_DELETE(self):
        parsed = urlparse(self.path)
        if not parsed.path.startswith("/api/dc/"):
            self.send_json({"ok": False, "error": "invalid path"}, 400)
            return
        key = parsed.path[8:]
        result = _dc.delete(key)
        self.send_json(result, 200 if result["ok"] else 404)


class DataCenterServer(HTTPServer):
    allow_reuse_address = True

    def __init__(self, port: int = PORT, workspace: str = "."):
        super().__init__(("127.0.0.1", port), DCHandler)
        self._port = port
        self._workspace = workspace

    def serve_forever(self):
        print(f"[DC] DataCenter listening on 127.0.0.1:{self._port}")
        super().serve_forever()


# ── CLI helpers ──────────────────────────────────────────────
def seed_demo_data():
    """Seed some demo data for quick testing."""
    _dc.set("kpi.revenue", "1.2M")
    _dc.set("kpi.users", 8420)
    _dc.set("kpi.conversion", 0.0342)
    _dc.set("statusbar.layout", {
        "items": [
            {"id": "sp", "label": "SpecPilot", "icon": "🚀", "color": "#7c3aed"},
            {"id": "dc", "label": "DC", "icon": "💾", "color": "#059669"},
            {"id": "ec", "label": "EC", "icon": "⚡", "color": "#d97706"},
            {"id": "status", "label": "Ready", "icon": "✅", "color": "#10b981"},
        ]
    })
    _dc.set("metrics.workbench", {"active_specs": 3, "sessions": 1, "mode": "proto"})
    _dc.set("ui.theme", {
        "primary": "#7c3aed",
        "bg": "#0f0f1a",
        "text": "#e0e0f0",
    })
    _dc.set("table.users", {
        "columns": ["ID", "Name", "Role", "Status"],
        "rows": [
            ["U001", "Alice", "Admin", "Active"],
            ["U002", "Bob", "Dev", "Active"],
            ["U003", "Carol", "Reviewer", "Away"],
        ]
    })


seed_demo_data()
