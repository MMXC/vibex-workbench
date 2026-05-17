"""
workspace.py — SpecPilot 工作区根目录统一接口
所有模块通过此文件获取工作区路径，避免硬编码。
"""
import os

# 由 Go agent 在启动时通过环境变量注入
WORKSPACE_ROOT = os.environ.get('SPECPILOT_WORKSPACE_ROOT', os.path.expanduser('~/.specpilot'))


def state_dir() -> str:
    """工作区本地 .specpilot 目录（持久化状态文件）"""
    root = os.path.join(WORKSPACE_ROOT, '.specpilot')
    os.makedirs(root, exist_ok=True)
    return root


def dc_state_file() -> str:
    return os.path.join(state_dir(), 'dc_state.json')


def ec_state_file() -> str:
    return os.path.join(state_dir(), 'ec_state.json')


def mf_registry_file() -> str:
    return os.path.join(state_dir(), 'mf_registry.json')


def adapter_state_file() -> str:
    return os.path.join(state_dir(), 'adapter_state.json')


def services_file() -> str:
    """当前服务端口记录（由 bootstrap 写入，供前端读取）"""
    return os.path.join(state_dir(), 'services.json')


def read_services() -> dict:
    """读取当前运行服务信息 {dcPort, mfPort}"""
    import json
    try:
        with open(services_file(), encoding='utf-8') as f:
            return json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def write_services(dc_port: int, mf_port: int) -> None:
    import json
    with open(services_file(), 'w', encoding='utf-8') as f:
        json.dump({'dcPort': dc_port, 'mfPort': mf_port, 'workspace': WORKSPACE_ROOT}, f)
