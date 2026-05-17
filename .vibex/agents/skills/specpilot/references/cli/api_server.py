#!/usr/bin/env python3
"""
SpecPilot API Server — HTTP wrapper around CLI capabilities
Svelte 前端通过 REST API 调用 CLI 能力
"""
import json
import subprocess
import threading
import os
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

CLI_DIR = '/tmp/specpilot'
API_PORT = 7890


def cli(args: list) -> dict:
    """Run specpilot CLI and return parsed JSON output"""
    try:
        result = subprocess.run(
            [sys.executable, '-m', 'cli'] + args,
            cwd=CLI_DIR,
            capture_output=True,
            text=True,
            timeout=10
        )
        output = result.stdout.strip()
        if output:
            try:
                return json.loads(output)
            except json.JSONDecodeError:
                return {'raw': output, 'error': None}
        return {'error': result.stderr.strip() or 'no output'}
    except subprocess.TimeoutExpired:
        return {'error': 'timeout'}
    except Exception as e:
        return {'error': str(e)}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass  # silent

    def send_json(self, data, status=200):
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()

    def do_GET(self):
        u = urlparse(self.path)
        path = u.path.rstrip('/')

        if path == '/api/dc':
            data = cli(['dc', 'list'])
            self.send_json(data)

        elif path == '/api/adapters':
            data = cli(['ad', 'list'])
            self.send_json(data)

        elif path == '/api/specs':
            data = cli(['spec', 'list'])
            self.send_json(data)

        elif path == '/api/mf':
            data = cli(['mf', 'list'])
            self.send_json(data)

        elif path.startswith('/api/spec/'):
            name = path.split('/')[-1]
            data = cli(['spec', 'get', name])
            self.send_json(data)

        elif path.startswith('/api/binding/'):
            name = path.split('/')[-1]
            data = cli(['spec', 'binding', name])
            self.send_json(data)

        elif path == '/api/ec/history':
            params = parse_qs(u.query)
            limit = int(params.get('limit', [20])[0])
            # ec history prints one JSON per line
            result = subprocess.run(
                [sys.executable, '-m', 'cli', 'ec', 'history', '--limit', str(limit)],
                cwd=CLI_DIR, capture_output=True, text=True
            )
            lines = [json.loads(l) for l in result.stdout.strip().split('\n') if l.strip()]
            self.send_json({'events': lines})

        elif path == '/api/health':
            self.send_json({'ok': True, 'layers': ['L1-MF', 'L2-DC', 'L3-EC', 'L4-Spec']})

        else:
            self.send_json({'error': 'not found', 'path': path}, 404)

    def do_POST(self):
        u = urlparse(self.path)
        path = u.path.rstrip('/')

        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length).decode() if length > 0 else '{}'
        try:
            data = json.loads(body)
        except:
            data = {}

        if path == '/api/dc/set':
            key = data.get('key', '')
            value = data.get('value')
            if key:
                result = cli(['dc', 'set', key, json.dumps(value)])
                self.send_json(result)
            else:
                self.send_json({'error': 'key required'}, 400)

        elif path == '/api/dc/query':
            q = data.get('query', '')
            result = cli(['dc', 'query', q])
            self.send_json(result)

        elif path == '/api/ad/switch':
            name = data.get('adapter', '')
            if name:
                result = cli(['ad', 'switch', name])
                self.send_json(result)
            else:
                self.send_json({'error': 'adapter name required'}, 400)

        elif path == '/api/ad/query':
            q = data.get('query', 'SELECT * FROM servers')
            result = cli(['ad', 'query', q])
            self.send_json(result)

        elif path == '/api/ec/emit':
            event = data.get('event', '')
            payload = data.get('payload', {})
            if event:
                result = cli(['ec', 'emit', event, json.dumps(payload)])
                self.send_json(result)
            else:
                self.send_json({'error': 'event required'}, 400)

        elif path == '/api/mf/register':
            name = data.get('name', '')
            path2 = data.get('path', '')
            if name:
                # Pass contract fields as JSON strings
                args = ['mf', 'register', name, path2]
                inputs = data.get('inputs')
                if inputs is not None:
                    args += ['--inputs', json.dumps(inputs)]
                outputs = data.get('outputs')
                if outputs is not None:
                    args += ['--outputs', json.dumps(outputs)]
                events = data.get('events')
                if events is not None:
                    args += ['--events', json.dumps(events)]
                result = cli(args)
                self.send_json(result)
            else:
                self.send_json({'error': 'name required'}, 400)

        elif path == '/api/mf/resolve':
            spec = data.get('spec', '')
            result = cli(['mf', 'resolve', spec])
            self.send_json(result)

        elif path == '/api/run':
            scenario = data.get('scenario', 'basic')
            verbose = data.get('verbose', False)
            args = ['run', scenario]
            if verbose:
                args.append('-v')
            result = cli(args)
            self.send_json(result)

        else:
            self.send_json({'error': 'not found', 'path': path}, 404)


def main():
    server = HTTPServer(('127.0.0.1', API_PORT), Handler)
    print(f'SpecPilot API running on http://127.0.0.1:{API_PORT}')
    print('Endpoints:')
    print('  GET  /api/health          — health check')
    print('  GET  /api/dc              — data center state')
    print('  GET  /api/adapters        — list adapters')
    print('  GET  /api/specs           — list specs')
    print('  GET  /api/specs           — get spec detail')
    print('  GET  /api/mf              — list MF components')
    print('  GET  /api/ec/history      — event history')
    print('  POST /api/dc/set          — set data')
    print('  POST /api/dc/query        — query via adapter')
    print('  POST /api/ad/switch       — switch adapter')
    print('  POST /api/ad/query        — query current adapter')
    print('  POST /api/ec/emit         — emit event')
    print('  POST /api/mf/register     — register component')
    print('  POST /api/mf/resolve      — resolve from spec')
    print('  POST /api/run             — run integration')
    print()
    sys.stdout.flush()
    server.serve_forever()


if __name__ == '__main__':
    main()
