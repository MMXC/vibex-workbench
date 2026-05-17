"""EventCenter — pub/sub with HTTP SSE."""
import json
import threading
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse
import queue

PORT = 7891  # separate from DC


class EventCenter:
    """In-process pub/sub bus."""

    def __init__(self):
        self._subscribers: dict[str, list[queue.Queue]] = {}
        self._lock = threading.RLock()
        self._seq = 0

    def emit(self, channel: str, payload) -> dict:
        with self._lock:
            self._seq += 1
            event = {
                "seq": self._seq,
                "channel": channel,
                "payload": payload,
                "ts": time.time(),
            }
            for q in self._subscribers.get(channel, []):
                q.put_nowait(event)
            # broadcast to wildcard subscribers
            for q in self._subscribers.get("*", []):
                q.put_nowait(event)
        return {"ok": True, "seq": self._seq, "channel": channel}

    def subscribe(self, channel: str) -> queue.Queue:
        q = queue.Queue(maxsize=100)
        with self._lock:
            self._subscribers.setdefault(channel, []).append(q)
        return q


_ec = EventCenter()


class ECHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(f"[EC] {fmt % args}")

    def send_json(self, data: dict, code: int = 200):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        parsed = urlparse(self.path)
        if parsed.path != "/api/ec/emit":
            self.send_json({"ok": False, "error": "not found"}, 404)
            return
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        try:
            data = json.loads(body)
            channel = data.get("channel", "default")
            payload = data.get("payload")
            result = _ec.emit(channel, payload)
            self.send_json(result)
        except Exception as e:
            self.send_json({"ok": False, "error": str(e)}, 400)

    def do_GET(self):
        parsed = urlparse(self.path)
        if not parsed.path.startswith("/api/ec/subscribe/"):
            self.send_json({"ok": False, "error": "not found"}, 404)
            return
        channel = parsed.path[18:]
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        q = _ec.subscribe(channel)
        try:
            while True:
                try:
                    event = q.get(timeout=30)
                    data = f"data: {json.dumps(event)}\n\n"
                    self.wfile.write(data.encode())
                    self.wfile.flush()
                except queue.Empty:
                    heartbeat = f"data: {json.dumps({'type':'heartbeat','ts':time.time()})}\n\n"
                    self.wfile.write(heartbeat.encode())
                    self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError):
            pass


class EventCenterServer(HTTPServer):
    allow_reuse_address = True

    def __init__(self, port: int = PORT):
        super().__init__(("127.0.0.1", port), ECHandler)
        self._port = port

    def serve_forever(self):
        print(f"[EC] EventCenter listening on 127.0.0.1:{self._port}")
        super().serve_forever()
