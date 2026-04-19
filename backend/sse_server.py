#!/usr/bin/env python3
"""
Minimal SSE Backend — VibeX Workbench
从 service.yaml 接口契约派生，实现 mock 事件流

端口: 33335
SSE 端点: GET /api/sse/threads/<threadId>
Run API: POST /api/runs
"""

import http.server
import socketserver
import threading
import json
import time
import uuid
from urllib.parse import urlparse, parse_qs
from http.server import BaseHTTPRequestHandler, HTTPServer

PORT = 33335

# ── SSE Client Registry ────────────────────────────────────────
clients: dict[str, list['SSEClient']] = {}

class SSEClient:
    def __init__(self, thread_id: str, handler: 'SSEHandler'):
        self.thread_id = thread_id
        self.handler = handler
        self.closed = False

    def send(self, event: str, data: dict):
        if self.closed:
            return
        payload = json.dumps(data, ensure_ascii=False)
        # SSE format: event:<type>\ndata:<json>\n\n
        self.handler.wfile.write(f"event: {event}\ndata: {payload}\n\n".encode())
        self.handler.wfile.flush()

    def close(self):
        self.closed = True

def broadcast(thread_id: str, event: str, data: dict):
    """向指定 threadId 的所有客户端广播事件"""
    for client in clients.get(thread_id, []):
        try:
            client.send(event, data)
        except Exception:
            pass

# ── Mock Run Executor ──────────────────────────────────────────
def run_mock_run(thread_id: str):
    """模拟一个 Run: started → tool×3 → completed"""
    run_id = str(uuid.uuid4())[:8]
    goal = "mock task"

    time.sleep(0.3)
    broadcast(thread_id, "run.started", {
        "runId": run_id,
        "threadId": thread_id,
        "status": "queued",
        "stage": "planning",
        "goal": goal,
    })

    time.sleep(0.5)
    broadcast(thread_id, "run.stage_changed", {
        "runId": run_id,
        "stage": "executing",
    })

    tools = [
        ("read_file", {"path": "/workspace/main.py"}),
        ("terminal", {"command": "ls -la"}),
        ("search_files", {"pattern": "*.py", "path": "/workspace"}),
    ]
    for i, (tool_name, args) in enumerate(tools):
        inv_id = f"{run_id}-t{i}"
        time.sleep(0.6)
        broadcast(thread_id, "tool.called", {
            "invocationId": inv_id,
            "runId": run_id,
            "toolName": tool_name,
            "args": args,
        })

        time.sleep(0.3)
        result = {"output": f"[mock] {tool_name} done", "status": "success"}
        broadcast(thread_id, "tool.completed", {
            "invocationId": inv_id,
            "result": result,
        })

        time.sleep(0.2)
        broadcast(thread_id, "message.delta", {
            "runId": run_id,
            "role": "assistant",
            "delta": f"[{tool_name}] 执行完成，输出：{result['output']}\n",
            "is_final": False,
        })

    # Artifact
    time.sleep(0.3)
    artifact_id = str(uuid.uuid4())[:8]
    broadcast(thread_id, "artifact.created", {
        "artifact": {
            "id": artifact_id,
            "threadId": thread_id,
            "runId": run_id,
            "type": "code",
            "name": "output.py",
            "content": "print('hello from mock backend!')\n",
            "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
    })

    time.sleep(0.3)
    broadcast(thread_id, "run.completed", {
        "runId": run_id,
        "summary": f"Run {run_id} completed. 3 tools executed.",
    })

# ── HTTP Handler ──────────────────────────────────────────────
class SSEHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        print(f"[SSE] {args[0]}")

    def send_sse_headers(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()

    def do_GET(self):
        parsed = urlparse(self.path)

        # GET /api/sse/threads/<threadId>
        if parsed.path.startswith("/api/sse/threads/"):
            thread_id = parsed.path.split("/")[-1]
            client = SSEClient(thread_id, self)

            if thread_id not in clients:
                clients[thread_id] = []
            clients[thread_id].append(client)

            print(f"[SSE] Client connected: threadId={thread_id} (total={len(clients[thread_id])})")

            # 发送 SSE HTTP 响应头
            self.send_sse_headers()

            # 发送连接确认事件
            client.send("connected", {"threadId": thread_id, "status": "connected"})

            # 自动触发一个 mock run（只触发一次）
            threading.Thread(target=run_mock_run, args=(thread_id,), daemon=True).start()

            # 保持连接，超时检查
            try:
                while not client.closed:
                    time.sleep(1)
            except (BrokenPipeError, ConnectionResetError):
                pass
            finally:
                clients[thread_id].remove(client)
                print(f"[SSE] Client disconnected: threadId={thread_id}")
            return

        # GET /api/health
        if parsed.path == "/api/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"status":"ok","port":%d}' % PORT)
            return

        self.send_error(404, "Not Found")

    def do_POST(self):
        parsed = urlparse(self.path)

        # POST /api/runs → 手动触发一个 mock run
        if parsed.path == "/api/runs":
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length) if length > 0 else b"{}"
            try:
                data = json.loads(body)
            except:
                data = {}
            thread_id = data.get("threadId", "default")
            goal = data.get("goal", "用户目标")

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            response = {"runId": str(uuid.uuid4())[:8], "threadId": thread_id, "status": "queued"}
            self.wfile.write(json.dumps(response).encode())

            # 在后台执行 mock run
            threading.Thread(target=run_mock_run, args=(thread_id,), daemon=True).start()
            return

        self.send_error(404, "Not Found")

# ── Main ─────────────────────────────────────────────────────
class ThreadedHTTPServer(socketserver.ThreadingMixIn, http.server.HTTPServer):
    allow_reuse_address = True
    daemon_threads = True

if __name__ == "__main__":
    server = ThreadedHTTPServer(("", PORT), SSEHandler)
    print(f"[VibeX SSE Backend] Listening on http://0.0.0.0:{PORT}")
    print(f"  SSE:  GET  http://localhost:{PORT}/api/sse/threads/<threadId>")
    print(f"  Runs: POST http://localhost:{PORT}/api/runs")
    server.serve_forever()
