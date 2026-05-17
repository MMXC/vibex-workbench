"""SpecPilot CLI — Agent-accessible prototype workspace manager."""
import argparse
import os
import json
import signal
import socket
import sys
import time
import threading

SP_DIR = ".specpilot"
DC_PORT_ENV = "SPECPILOT_DC_PORT"
MF_PORT_ENV = "SPECPILOT_MF_PORT"
DEFAULT_DC_PORT = 7890
DEFAULT_MF_PORT = 5177

_WS = os.getcwd()  # workspace root


def _sp(*path: str) -> str:
    """Path inside .specpilot/."""
    return os.path.join(_WS, SP_DIR, *path)


# ── Meta / PID ────────────────────────────────────────────────
def get_meta() -> dict:
    path = _sp(".meta.json")
    if os.path.exists(path):
        with open(path) as f:
            return json.load(f)
    return {}


def save_meta(meta: dict):
    os.makedirs(_sp(), exist_ok=True)
    with open(_sp(".meta.json"), "w") as f:
        json.dump(meta, f, indent=2)


def read_pids() -> dict:
    path = _sp(".pids.json")
    if os.path.exists(path):
        with open(path) as f:
            return json.load(f)
    return {}


def save_pids(pids: dict):
    with open(_sp(".pids.json"), "w") as f:
        json.dump(pids, f, indent=2)


# ── Network ───────────────────────────────────────────────────
def is_port_open(port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        return s.connect_ex(("127.0.0.1", port)) == 0


def wait_port(port: int, timeout: int = 10) -> bool:
    start = time.time()
    while time.time() - start < timeout:
        if is_port_open(port):
            return True
        time.sleep(0.2)
    return False


# ── Init ─────────────────────────────────────────────────────
def cmd_init(args):
    os.makedirs(_sp(), exist_ok=True)
    os.makedirs(_sp("components"), exist_ok=True)
    os.makedirs(_sp("previews"), exist_ok=True)
    os.makedirs(_sp("static"), exist_ok=True)

    # Default component registry
    with open(_sp("components", "index.json"), "w") as f:
        json.dump({"components": [], "version": "1.0"}, f, indent=2)

    # Default preview
    with open(_sp("previews", "index.html"), "w") as f:
        f.write(_DEFAULT_PREVIEW_HTML)

    save_meta({
        "initialized": True,
        "workspace": _WS,
        "version": "0.1.0",
    })
    print(f"[specpilot] Initialized at {_sp()}")
    print(f"[specpilot] Run 'specpilot start' to launch services.")


# ── Start / Stop ─────────────────────────────────────────────
def start_dc(dc_port: int):
    from specpilot.dc.server import DataCenterServer
    server = DataCenterServer(port=dc_port)
    t = threading.Thread(target=server.serve_forever)
    t.start()
    wait_port(dc_port)
    print(f"[specpilot] DataCenter: http://127.0.0.1:{dc_port}")


def start_preview(mf_port: int):
    from specpilot.preview.server import PreviewServer
    server = PreviewServer(port=mf_port, workspace=_WS)
    t = threading.Thread(target=server.serve_forever)
    t.start()
    wait_port(mf_port)
    print(f"[specpilot] Preview:    http://127.0.0.1:{mf_port}/preview")


def cmd_start(args):
    dc_port = int(os.environ.get(DC_PORT_ENV, DEFAULT_DC_PORT))
    mf_port = int(os.environ.get(MF_PORT_ENV, DEFAULT_MF_PORT))

    if not is_port_open(dc_port):
        start_dc(dc_port)
    else:
        print(f"[specpilot] DataCenter already running on :{dc_port}")

    if not is_port_open(mf_port):
        start_preview(mf_port)
    else:
        print(f"[specpilot] Preview already running on :{mf_port}")

    save_meta({"dc_port": dc_port, "mf_port": mf_port, "workspace": _WS})
    print(f"[specpilot] Ready. Press Ctrl+C to stop.")
    print(f"[specpilot] DC:      http://127.0.0.1:{dc_port}")
    print(f"[specpilot] Preview: http://127.0.0.1:{mf_port}/preview")

    # Block foreground — run until SIGINT
    stop_event = threading.Event()

    def handler(signum, frame):
        stop_event.set()

    signal.signal(signal.SIGINT, handler)
    signal.signal(signal.SIGTERM, handler)
    stop_event.wait()


def cmd_stop(args):
    pids = read_pids()
    for name, pid in pids.items():
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    if os.path.exists(_sp(".pids.json")):
        os.remove(_sp(".pids.json"))
    print("[specpilot] All services stopped.")


def cmd_status(args):
    meta = get_meta()
    dc_port = meta.get("dc_port", DEFAULT_DC_PORT)
    mf_port = meta.get("mf_port", DEFAULT_MF_PORT)
    dc_ok = is_port_open(dc_port)
    mf_ok = is_port_open(mf_port)

    print(f"Workspace : {_WS}")
    print(f"SpecPilot : {'initialized' if meta.get('initialized') else 'not initialized'}")
    print(f"DataCenter: {'🟢 running' if dc_ok else '⚪ stopped'}  (port {dc_port})")
    print(f"Preview   : {'🟢 running' if mf_ok else '⚪ stopped'}  (port {mf_port})")

    if dc_ok:
        try:
            import urllib.request
            r = urllib.request.urlopen(f"http://127.0.0.1:{dc_port}/api/health", timeout=2)
            health = json.loads(r.read())
            print(f"DC keys   : {health.get('key_count', '?')}")
        except Exception:
            pass


# ── Generate ──────────────────────────────────────────────────
def cmd_generate(args):
    from specpilot.generator import Generator
    gen = Generator(workspace=_WS)
    out = args.output or _sp("previews", "index.html")
    result = gen.generate(args.spec, out)
    mf_port = int(os.environ.get(MF_PORT_ENV, DEFAULT_MF_PORT))
    print(f"[specpilot] Generated: {result}")
    print(f"[specpilot] Preview:   http://127.0.0.1:{mf_port}/preview")


# ── DC ────────────────────────────────────────────────────────
def cmd_dc(args):
    import urllib.request, urllib.error
    meta = get_meta()
    dc_port = meta.get("dc_port", DEFAULT_DC_PORT)
    base = f"http://127.0.0.1:{dc_port}/api"

    sub = args.dc_subcommand
    if sub == "get":
        try:
            r = urllib.request.urlopen(f"{base}/dc/{args.key}", timeout=2)
            print(r.read().decode())
        except urllib.error.HTTPError:
            print(f"[error] key not found: {args.key}", file=sys.stderr)
    elif sub == "set":
        req = urllib.request.Request(
            f"{base}/dc/{args.key}",
            data=json.dumps({"value": args.value}).encode(),
            headers={"Content-Type": "application/json"},
            method="PUT"
        )
        urllib.request.urlopen(req, timeout=2)
        print(f"[specpilot] dc.set {args.key} = {args.value}")
    elif sub == "list":
        r = urllib.request.urlopen(f"{base}/dc/", timeout=2)
        data = json.loads(r.read())
        for k, v in data.get("keys", {}).items():
            print(f"  {k} = {v}")


# ── Register ─────────────────────────────────────────────────
def cmd_register(args):
    from specpilot.registry import ComponentRegistry
    # Registry lives inside .specpilot/, pass the .specpilot/ path as workspace
    sp_workspace = _sp()
    reg = ComponentRegistry(workspace=sp_workspace)
    result = reg.register(args.component, args.mf_url, args.dc_key)
    print(f"[specpilot] Registered: {result['name']} → {result['mf_url']}")


# ── Preview ──────────────────────────────────────────────────
def cmd_preview(args):
    meta = get_meta()
    mf_port = meta.get("mf_port", DEFAULT_MF_PORT)
    url = f"http://127.0.0.1:{mf_port}/preview"
    print(f"[specpilot] Preview: {url}")
    if args.open:
        import webbrowser
        webbrowser.open(url)


# ── Templates ────────────────────────────────────────────────
_DEFAULT_PREVIEW_HTML = """<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>SpecPilot</title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:system-ui,sans-serif;background:#0f0f1a;color:#e0e0f0;min-height:100vh;display:flex;align-items:center;justify-content:center}
    .wrap{text-align:center;padding:40px}
    .title{font-size:22px;font-weight:700;color:#a78bfa;margin-bottom:10px}
    .sub{font-size:13px;color:#555;margin-bottom:24px}
    .status{padding:10px 20px;background:#1a1a2e;border-radius:8px;display:inline-block;font-size:12px;color:#888}
    .hint{margin-top:20px;font-size:12px;color:#444}
  </style>
</head>
<body>
  <div class="wrap">
    <div class="title">🚀 SpecPilot Ready</div>
    <div class="sub">Module Federation · DataCenter · EventCenter</div>
    <div class="status" id="s">DC: connecting...</div>
    <div class="hint">等待 Agent 生成原型...</div>
  </div>
  <script>
    fetch('/api/health').then(r=>r.json()).then(d=>{
      document.getElementById('s').textContent='DC: '+d.key_count+' keys | Running';
    }).catch(()=>{document.getElementById('s').textContent='DC: offline';});
  </script>
</body>
</html>"""


# ── Main ─────────────────────────────────────────────────────
def main():
    parser = argparse.ArgumentParser(prog="specpilot",
        description="SpecPilot CLI — Agent-accessible prototype workspace manager")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("init", help="Initialize SpecPilot in current workspace")
    sub.add_parser("start", help="Start DC + Preview services")
    sub.add_parser("stop", help="Stop all services")
    sub.add_parser("status", help="Show service status")

    g = sub.add_parser("generate", help="Generate prototype from spec YAML")
    g.add_argument("spec", help="Path to spec YAML file")
    g.add_argument("-o", "--output", help="Output HTML path")

    dc = sub.add_parser("dc", help="DataCenter operations")
    dc_sub = dc.add_subparsers(dest="dc_subcommand", required=True)
    dc_get = dc_sub.add_parser("get", help="Get DC key")
    dc_get.add_argument("key", help="Key name")
    dc_set = dc_sub.add_parser("set", help="Set DC key")
    dc_set.add_argument("key", help="Key name")
    dc_set.add_argument("value", help="Value")
    dc_sub.add_parser("list", help="List all DC keys")

    r = sub.add_parser("register", help="Register a MF component")
    r.add_argument("component", help="Component name")
    r.add_argument("--mf-url", required=True, help="MF shell route")
    r.add_argument("--dc-key", help="DC key this component reads from")

    p = sub.add_parser("preview", help="Open preview URL")
    p.add_argument("--open", action="store_true", help="Open in browser")

    args = parser.parse_args()
    globals()["cmd_" + args.command.replace("-", "_")](args)


if __name__ == "__main__":
    main()
