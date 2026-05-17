"""
L1 MF Registry — Module Federation 组件注册
按 Spec 自动解析需要加载的 MF 组件
注册时登记 Data / State / Event 契约，作为 spec 派生的唯一事实来源
"""
import json
import hashlib
import os
from typing import Any, Optional
from workspace import mf_registry_file


def _hash(data: Any) -> str:
    s = json.dumps(data, sort_keys=True, ensure_ascii=True)
    return hashlib.sha256(s.encode()).hexdigest()[:16]


class ComponentContract:
    """MF 组件契约：inputs（props）+ outputs（DC key 前缀）+ events（EC 事件通道）"""

    def __init__(self,
                 inputs: dict | None = None,
                 outputs: list[dict] | None = None,
                 events: list[dict] | None = None):
        self.inputs: dict = inputs or {}
        self.outputs: list[dict] = outputs or []
        self.events: list[dict] = events or []

    def to_dict(self) -> dict:
        return {
            'inputs': self.inputs,
            'outputs': self.outputs,
            'events': self.events,
        }

    def hash(self) -> str:
        return _hash(self.to_dict())

    @classmethod
    def from_dict(cls, d: dict) -> 'ComponentContract':
        return cls(
            inputs=d.get('inputs', {}),
            outputs=d.get('outputs', []),
            events=d.get('events', []),
        )


class MFRegistry:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._init()
        return cls._instance

    def _init(self):
        self._components: dict[str, dict] = {}
        self._contracts: dict[str, ComponentContract] = {}
        self._state_file = mf_registry_file()
        self._load()

    def _load(self) -> None:
        try:
            if os.path.exists(self._state_file):
                with open(self._state_file, encoding='utf-8') as f:
                    data = json.load(f)
                for name, info in data.get('components', {}).items():
                    self._components[name] = {
                        'name': name,
                        'path': info.get('path', ''),
                        'spec_hint': info.get('spec_hint', ''),
                        'registered': True,
                    }
                    if 'contract' in info:
                        self._contracts[name] = ComponentContract.from_dict(info['contract'])
        except Exception:
            pass

    def _save(self) -> None:
        try:
            os.makedirs(os.path.dirname(self._state_file), exist_ok=True)
            with open(self._state_file, 'w', encoding='utf-8') as f:
                json.dump(self.dump(), f, indent=2)
        except Exception:
            pass

    def register(self,
                 name: str,
                 path: str,
                 spec_hint: str = '',
                 inputs: dict | None = None,
                 outputs: list[dict] | None = None,
                 events: list[dict] | None = None) -> None:
        """Register a component with its remote path and Data/State/Event contract"""
        self._components[name] = {
            'name': name,
            'path': path,
            'spec_hint': spec_hint,
            'registered': True,
        }
        self._contracts[name] = ComponentContract(
            inputs=inputs or {},
            outputs=outputs or [],
            events=events or [],
        )
        self._save()

    def get(self, name: str) -> Optional[dict]:
        return self._components.get(name)

    def get_contract(self, name: str) -> ComponentContract | None:
        return self._contracts.get(name)

    def list_components(self) -> dict:
        return {
            'components': [
                {
                    'name': name,
                    'path': info['path'],
                    'spec_hint': info.get('spec_hint', ''),
                    'contract': self._contracts.get(name, ComponentContract()).to_dict(),
                }
                for name, info in self._components.items()
            ],
            'total': len(self._components),
        }

    def resolve_from_spec(self, spec_name: str) -> dict:
        """Resolve which components to load based on a spec, with full contract"""
        spec_component_map = {
            'L3-dashboard-kpi': ['KPICard', 'TrendChart', 'AlertBadge'],
            'L2-server-list': ['ServerList'],
            'L3-import': ['ImportZone', 'ProgressBar', 'DataTable'],
        }
        components = spec_component_map.get(spec_name, [])

        resolved = []
        for comp_name in components:
            info = self._components.get(comp_name, {'path': f'pending:{comp_name}'})
            contract = self._contracts.get(comp_name, ComponentContract())
            resolved.append({
                'component': comp_name,
                'path': info.get('path', f'<remote>/{comp_name}'),
                'status': 'pending' if not info.get('registered') else 'ready',
                'contract': contract.to_dict(),
            })

        manifest = {
            'spec': spec_name,
            'components': resolved,
            'total': len(resolved),
        }
        manifest['manifest_hash'] = _hash(manifest)
        return manifest

    def unregister(self, name: str) -> None:
        if name in self._components:
            del self._components[name]
        if name in self._contracts:
            del self._contracts[name]
        self._save()

    def dump(self) -> dict:
        return {
            'components': {
                name: {
                    **info,
                    'contract': self._contracts.get(name, ComponentContract()).to_dict(),
                }
                for name, info in self._components.items()
            }
        }
