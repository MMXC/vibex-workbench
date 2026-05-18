#!/usr/bin/env python3
"""
SpecPilot API Server — DC State / EC Event / MF Registry HTTP API
前端通过 /api/* 端点直接操作 workspace-local JSON 状态文件。
"""
import json
import os
import sys
from datetime import datetime, timezone
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse

# Default ports (can be overridden by env)
API_PORT = int(os.getenv('SPECPILOT_DC_PORT', '7890'))
WORKSPACE_ROOT = os.getenv('SPECPILOT_WORKSPACE_ROOT', '')


def state_file(name: str) -> str:
    root = WORKSPACE_ROOT or os.getcwd()
    return os.path.join(root, '.specpilot', f'{name}.json')


def read_json(path: str, default: dict) -> dict:
    try:
        with open(path) as f:
            return json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        return default


def write_json(path: str, data: dict):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass  # silent

    def send_json(self, data, status=200):
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def send_text(self, data: str, status=200):
        self.send_response(status)
        self.send_header('Content-Type', 'text/plain')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(data.encode())

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS, PUT, DELETE')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()

    def do_GET(self):
        u = urlparse(self.path)
        path = u.path.rstrip('/')

        # GET /api/health
        if path == '/api/health':
            self.send_json({'status': 'ok', 'port': API_PORT})
            return

        # GET /api/dc — list all DC key-values
        if path == '/api/dc':
            data = read_json(state_file('dc_state'), {'data': {}, 'total': 0})
            self.send_json(data)
            return

        # GET /api/dc/:key
        if path.startswith('/api/dc/'):
            key = path.split('/')[-1]
            all_data = read_json(state_file('dc_state'), {'data': {}})
            value = all_data.get('data', {}).get(key)
            self.send_json({'key': key, 'value': value})
            return

        # GET /api/ec/history?limit=N
        if path == '/api/ec/history':
            limit = 20
            qs = u.query
            if 'limit=' in qs:
                try:
                    limit = int(qs.split('limit=')[1].split('&')[0])
                except ValueError:
                    pass
            history = read_json(state_file('ec_state'), [])
            if isinstance(history, list):
                records = history[-limit:]
            else:
                records = []
            self.send_json(records)
            return

        # GET /api/mf/components
        if path in ('/api/mf/components', '/api/mf'):
            data = read_json(state_file('mf_registry'), {'components': {}})
            comps = data.get('components', {})
            items = []
            for name, info in comps.items():
                if isinstance(info, dict):
                    items.append({**info, 'name': name})
            self.send_json({'components': items, 'total': len(items)})
            return

        self.send_text('Not Found', 404)

    def do_POST(self):
        u = urlparse(self.path)
        path = u.path.rstrip('/')
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length) if content_length > 0 else b''
        try:
            payload = json.loads(body.decode()) if body else {}
        except json.JSONDecodeError:
            payload = {}

        # POST /api/dc/set { key, value }
        if path == '/api/dc/set':
            key = payload.get('key', '')
            if not key:
                self.send_json({'ok': False, 'error': 'missing key'}, 400)
                return
            value = payload.get('value')
            dc = read_json(state_file('dc_state'), {'data': {}})
            dc.setdefault('data', {})[key] = value
            dc['total'] = len(dc['data'])
            write_json(state_file('dc_state'), dc)
            self.send_json({'ok': True, 'key': key})
            return

        # POST /api/dc/seed { component? }
        if path == '/api/dc/seed':
            component = payload.get('component', 'Dashboard')
            seed_demo_data(component)
            self.send_json({'ok': True, 'message': f'seeded {component}'})
            return

        # POST /api/ec/publish { event, payload }
        if path == '/api/ec/publish':
            event_name = payload.get('event', '')
            event_payload = payload.get('payload')
            if not event_name:
                self.send_json({'ok': False, 'error': 'missing event'}, 400)
                return
            record = {
                'event': event_name,
                'payload': event_payload,
                'emitted_at': datetime.now(timezone.utc).isoformat(),
                'subscriber_count': 0,
            }
            history = read_json(state_file('ec_state'), [])
            if not isinstance(history, list):
                history = []
            history.append(record)
            if len(history) > 200:
                history = history[-200:]
            write_json(state_file('ec_state'), history)
            self.send_json({'ok': True, 'record': record})
            return

        # POST /api/mf/components/register { name }
        if path == '/api/mf/components/register':
            name = payload.get('name', '')
            if not name:
                self.send_json({'ok': False, 'error': 'missing name'}, 400)
                return
            mf = read_json(state_file('mf_registry'), {'components': {}})
            mf.setdefault('components', {})[name] = {
                'name': name,
                'version': '1.0.0',
                'inputs': [],
                'outputs': [],
                'events': [],
                'registered_at': datetime.now(timezone.utc).isoformat(),
            }
            write_json(state_file('mf_registry'), mf)
            self.send_json({'ok': True, 'name': name})
            return

        self.send_text('Not Found', 404)

    def do_PUT(self):
        """PUT /api/dc/:key { value }"""
        u = urlparse(self.path)
        path = u.path.rstrip('/')
        if not path.startswith('/api/dc/'):
            self.send_text('Not Found', 404)
            return
        key = path.split('/')[-1]
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length) if content_length > 0 else b''
        try:
            payload = json.loads(body.decode())
        except json.JSONDecodeError:
            payload = {}
        value = payload.get('value')
        dc = read_json(state_file('dc_state'), {'data': {}})
        dc.setdefault('data', {})[key] = value
        dc['total'] = len(dc['data'])
        write_json(state_file('dc_state'), dc)
        self.send_json({'ok': True, 'key': key})

    def do_DELETE(self):
        """DELETE /api/dc/:key"""
        u = urlparse(self.path)
        path = u.path.rstrip('/')
        if not path.startswith('/api/dc/'):
            self.send_text('Not Found', 404)
            return
        key = path.split('/')[-1]
        dc = read_json(state_file('dc_state'), {'data': {}})
        dc.get('data', {}).pop(key, None)
        dc['total'] = len(dc.get('data', {}))
        write_json(state_file('dc_state'), dc)
        self.send_json({'ok': True})


def seed_demo_data(component='Dashboard'):
    demos = {
        'Dashboard': {
            'kpi.revenue': 843250,
            'kpi.revenue.trend': 12.4,
            'kpi.users': 12847,
            'kpi.users.trend': -2.1,
            'kpi.conversion': 3.8,
            'kpi.conversion.trend': 0.3,
            'alert.status': 'healthy',
            'alert.count': 0,
            'table.transactions': [
                {'id': 'TXN001', 'user': 'alice@example.com', 'amount': 249.99, 'status': 'completed'},
                {'id': 'TXN002', 'user': 'bob@example.com', 'amount': 89.50, 'status': 'pending'},
                {'id': 'TXN003', 'user': 'carol@example.com', 'amount': 1240.00, 'status': 'completed'},
            ],
            'chart.daily_revenue': [
                {'date': '2026-05-11', 'value': 28400},
                {'date': '2026-05-12', 'value': 31200},
                {'date': '2026-05-13', 'value': 29800},
                {'date': '2026-05-14', 'value': 35600},
                {'date': '2026-05-15', 'value': 33150},
            ],
        },
        'Default': {
            'app.status': 'running',
            'app.version': '1.0.0',
            'env': 'development',
        },
    }
    data = demos.get(component, demos['Default'])
    dc = {'data': data, 'total': len(data)}
    write_json(state_file('dc_state'), dc)
    write_json(state_file('ec_state'), [{
        'event': 'system.boot',
        'payload': {'component': component, 'version': '1.0.0'},
        'emitted_at': datetime.now(timezone.utc).isoformat(),
        'subscriber_count': 0,
    }])


def main():
    os.makedirs(state_file(''), exist_ok=True)
    dc = read_json(state_file('dc_state'), None)
    if dc is None:
        seed_demo_data()
    server = HTTPServer(('127.0.0.1', API_PORT), Handler)
    print(f'SpecPilot API listening on 127.0.0.1:{API_PORT}', file=sys.stderr)
    print(f'Workspace: {WORKSPACE_ROOT or os.getcwd()}', file=sys.stderr)
    sys.stderr.flush()
    server.serve_forever()


if __name__ == '__main__':
    main()
