"""
L1 MF Registry — Module Federation 组件注册
按 Spec 自动解析需要加载的 MF 组件
"""
from typing import Optional


class MFRegistry:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._init()
        return cls._instance

    def _init(self):
        self._components: dict[str, dict] = {}

    def register(self, name: str, path: str, spec_hint: str = '') -> None:
        """Register a component with its remote path"""
        self._components[name] = {
            'name': name,
            'path': path,
            'spec_hint': spec_hint,
            'registered': True,
        }

    def get(self, name: str) -> Optional[dict]:
        return self._components.get(name)

    def list_components(self) -> dict:
        return {
            'components': [
                {
                    'name': name,
                    'path': info['path'],
                    'spec_hint': info.get('spec_hint', ''),
                }
                for name, info in self._components.items()
            ],
            'total': len(self._components)
        }

    def resolve_from_spec(self, spec_name: str) -> dict:
        """Resolve which components to load based on a spec"""
        # In real impl, this reads from SpecRegistry
        # Here we use a hardcoded map
        spec_component_map = {
            'L3-dashboard-kpi': ['KPICard', 'TrendChart', 'AlertBadge'],
            'L2-server-list': ['ServerList'],
            'L3-import': ['ImportZone', 'ProgressBar', 'DataTable'],
        }
        components = spec_component_map.get(spec_name, [])

        resolved = []
        for comp_name in components:
            info = self._components.get(comp_name, {'path': f'pending:{comp_name}'})
            resolved.append({
                'component': comp_name,
                'path': info.get('path', f'<remote>/{comp_name}'),
                'status': 'pending' if not info.get('registered') else 'ready',
            })

        return {
            'spec': spec_name,
            'components': resolved,
            'total': len(resolved),
        }

    def unregister(self, name: str) -> None:
        if name in self._components:
            del self._components[name]
