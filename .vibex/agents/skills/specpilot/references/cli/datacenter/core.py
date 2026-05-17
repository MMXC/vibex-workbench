"""
L2 DataCenter — 状态单一来源
支持文件持久化、watch、适配器结果自动写入
"""
import json
import threading
import os
from typing import Any, Optional, Dict, Callable
from dataclasses import dataclass
from datetime import datetime
from workspace import dc_state_file


@dataclass
class DataEntry:
    key: str
    value: Any
    source: str = 'unknown'
    updated_at: str = ''
    version: int = 0

    def __post_init__(self):
        if not self.updated_at:
            self.updated_at = datetime.now().isoformat()

    def to_dict(self) -> dict:
        return {'key': self.key, 'value': self.value, 'source': self.source,
                'updated_at': self.updated_at, 'version': self.version}


class DataCenter:
    _instance = None
    _lock = threading.Lock()

    def __new__(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super().__new__(cls)
                    cls._instance._state_file = dc_state_file()
                    cls._instance._data: Dict[str, DataEntry] = {}
                    cls._instance._watchers: Dict[str, list[Callable]] = {}
                    cls._instance._history: list[dict] = []
                    cls._instance._version = 0
                    cls._instance._load()
        return cls._instance

    def _load(self) -> None:
        try:
            if os.path.exists(self._state_file):
                with open(self._state_file) as f:
                    data = json.load(f)
                for k, v in data.get('data', {}).items():
                    self._data[k] = DataEntry(key=k, value=v.get('value'),
                                              source=v.get('source', 'loaded'),
                                              version=v.get('version', 0))
        except Exception:
            pass

    def _save(self) -> None:
        try:
            os.makedirs(os.path.dirname(self._state_file), exist_ok=True)
            with open(self._state_file, 'w') as f:
                json.dump(self.dump(), f)
        except Exception:
            pass

    def set(self, key: str, value: Any, source: str = 'manual') -> None:
        with self._lock:
            self._version += 1
            entry = DataEntry(key=key, value=value, source=source, version=self._version)
            self._data[key] = entry
            self._history.append({'op': 'set', 'key': key, 'value': value,
                                   'source': source, 'ts': datetime.now().isoformat()})
        self._notify(key, value)
        self._save()

    def get(self, key: str, default=None) -> Any:
        entry = self._data.get(key)
        return entry.value if entry else default

    def dump(self) -> dict:
        return {
            'data': {k: v.to_dict() for k, v in self._data.items()},
            'total': len(self._data)
        }

    def apply_result(self, result: dict, source: str = 'adapter') -> None:
        if isinstance(result, dict):
            for key, value in result.items():
                if not key.startswith('_'):
                    self.set(key, value, source=source)
        elif isinstance(result, list):
            for item in result:
                if isinstance(item, dict):
                    for key, value in item.items():
                        if not key.startswith('_'):
                            self.set(key, value, source=source)

    def watch(self, key: Optional[str] = None) -> None:
        import time
        seen = {}
        while True:
            current = self.dump()
            for k, v in current['data'].items():
                if key is None or k == key:
                    if k not in seen or seen[k] != v['value']:
                        seen[k] = v['value']
                        print(f'[dc.watch] {k} = {v["value"]} (from {v["source"]})')
            time.sleep(0.5)

    def subscribe(self, key: str, handler: Callable) -> None:
        if key not in self._watchers:
            self._watchers[key] = []
        self._watchers[key].append(handler)

    def _notify(self, key: str, value: Any) -> None:
        watchers = self._watchers.get(key, []) + self._watchers.get('*', [])
        for w in watchers:
            try:
                w(key, value)
            except Exception as e:
                print(f'[dc.watch error] {e}')

    def history(self, limit=20) -> list:
        return self._history[-limit:]
