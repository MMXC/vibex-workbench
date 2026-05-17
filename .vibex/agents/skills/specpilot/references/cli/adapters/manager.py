"""
L4 适配器管理器 — 支持 mock / http / ws / grpc
统一 query() 接口，切换后数据中心自动刷新
"""
import json
import os
from typing import Protocol, Optional
from dataclasses import dataclass
import random
from workspace import adapter_state_file


class Adapter(Protocol):
    """统一适配器接口"""
    def query(self, q: str) -> dict: ...
    def status(self) -> dict: ...


@dataclass
class MockAdapter:
    name: str = 'mock'
    
    def query(self, q: str) -> dict:
        # Simulate KPI data
        return {
            'cpu_usage': random.randint(40, 95),
            'memory_percent': random.randint(50, 90),
            'network_traffic': round(random.uniform(0.5, 4.0), 2),
            'active_connections': random.randint(800, 3500),
            '_adapter': 'mock',
            '_query': q
        }

    def status(self) -> dict:
        return {'name': 'mock', 'ok': True, 'type': 'mock', 'latency_ms': 1}


@dataclass
class HttpAdapter:
    name: str = 'http'
    endpoint: str = 'http://localhost:8080/api'
    
    def query(self, q: str) -> dict:
        # Simulate HTTP response with different data profile
        return {
            'cpu_usage': random.randint(55, 75),
            'memory_percent': random.randint(60, 80),
            'network_traffic': round(random.uniform(1.0, 2.5), 2),
            'active_connections': random.randint(1500, 2800),
            '_adapter': 'http',
            '_query': q
        }

    def status(self) -> dict:
        return {'name': 'http', 'ok': True, 'type': 'rest', 'endpoint': self.endpoint}


@dataclass
class WsAdapter:
    name: str = 'ws'
    url: str = 'ws://localhost:9000/live'
    
    def query(self, q: str) -> dict:
        # WS: higher values, real-time feel
        return {
            'cpu_usage': random.randint(70, 95),
            'memory_percent': random.randint(65, 92),
            'network_traffic': round(random.uniform(2.0, 5.0), 2),
            'active_connections': random.randint(2000, 5000),
            '_adapter': 'ws',
            '_query': q
        }

    def status(self) -> dict:
        return {'name': 'ws', 'ok': True, 'type': 'websocket', 'url': self.url}


@dataclass
class GrpcAdapter:
    name: str = 'grpc'
    addr: str = 'localhost:9090'
    
    def query(self, q: str) -> dict:
        return {
            'cpu_usage': random.randint(30, 65),
            'memory_percent': random.randint(40, 70),
            'network_traffic': round(random.uniform(0.3, 1.8), 2),
            'active_connections': random.randint(400, 1500),
            '_adapter': 'grpc',
            '_query': q
        }

    def status(self) -> dict:
        return {'name': 'grpc', 'ok': True, 'type': 'grpc', 'addr': self.addr}


class AdapterManager:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._init()
        return cls._instance

    def _init(self):
        self._state_file = adapter_state_file()
        self._adapters: dict[str, Adapter] = {
            'mock': MockAdapter(),
            'http': HttpAdapter(),
            'ws': WsAdapter(),
            'grpc': GrpcAdapter(),
        }
        self.active_name = self._load_active()

    def _load_active(self) -> str:
        try:
            if os.path.exists(self._state_file):
                with open(self._state_file) as f:
                    return json.load(f).get('active', 'mock')
        except Exception:
            pass
        return 'mock'

    def _save_active(self) -> None:
        try:
            os.makedirs(os.path.dirname(self._state_file), exist_ok=True)
            with open(self._state_file, 'w') as f:
                json.dump({'active': self.active_name}, f)
        except Exception:
            pass

    def switch_to(self, name: str) -> bool:
        if name not in self._adapters:
            print(f'[AdapterManager] Unknown adapter: {name}')
            return False
        self.active_name = name
        self._save_active()
        return True

    def get_active(self) -> Adapter:
        return self._adapters[self.active_name]

    def list_adapters(self) -> dict:
        return {
            'adapters': [
                {'name': name, 'type': a.status().get('type', 'unknown'), 'active': name == self.active_name}
                for name, a in self._adapters.items()
            ],
            'active': self.active_name
        }

    def status(self) -> dict:
        return {
            'active': self.active_name,
            'adapter_info': self._adapters[self.active_name].status()
        }
