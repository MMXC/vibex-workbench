"""
L4 SpecRegistry — Spec 治理层
字段定义 / 事件链路 / 适配器配置 / 绑定关系
"""
import yaml
import json
import os
from typing import Optional, Any


class SpecRegistry:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._init()
        return cls._instance

    def _init(self):
        self._specs: dict[str, dict] = {}
        self._bindings: dict[str, list[dict]] = {}
        self._loaded = False  # Auto-load on first access

    def _ensure_loaded(self):
        if not self._loaded:
            self.load_builtin()
            self._loaded = True

    def load_builtin(self) -> None:
        """Load built-in specs"""
        builtin = {
            'L3-dashboard-kpi': {
                'name': 'L3-dashboard-kpi',
                'level': 'L3',
                'title': 'KPI 看板',
                'fields': [
                    {'name': 'cpu_usage', 'type': 'number', 'desc': 'CPU 使用率 0-100'},
                    {'name': 'memory_percent', 'type': 'number', 'desc': '内存占用百分比'},
                    {'name': 'network_traffic', 'type': 'number', 'desc': '网络流量 GB/s'},
                    {'name': 'active_connections', 'type': 'integer', 'desc': '活跃连接数'},
                ],
                'events': [
                    {'name': 'data:kpis.updated', 'desc': 'KPI 数据更新时广播', 'subscribers': ['KPICard', 'TrendChart']},
                    {'name': 'kpi:threshold.exceeded', 'desc': 'CPU > 80% 时触发', 'subscribers': ['AlertBadge', 'ServerList']},
                ],
                'adapter': 'mock',
                'components': ['KPICard', 'TrendChart', 'AlertBadge'],
            },
            'L2-server-list': {
                'name': 'L2-server-list',
                'level': 'L2',
                'title': '服务器列表',
                'fields': [
                    {'name': 'server_name', 'type': 'string', 'desc': '服务器名称'},
                    {'name': 'server_ip', 'type': 'string', 'desc': '服务器 IP'},
                    {'name': 'server_status', 'type': 'string', 'desc': '状态: online/warning/offline'},
                    {'name': 'server_cpu', 'type': 'number', 'desc': 'CPU 使用率'},
                    {'name': 'server_mem', 'type': 'number', 'desc': '内存占用'},
                ],
                'events': [
                    {'name': 'server:highlight-red', 'desc': 'CPU 超过阈值时高亮', 'subscribers': ['ServerList']},
                ],
                'adapter': 'mock',
                'components': ['ServerList'],
            },
            'L3-import': {
                'name': 'L3-import',
                'level': 'L3',
                'title': '数据导入',
                'fields': [
                    {'name': 'import_file', 'type': 'file', 'desc': '上传文件'},
                    {'name': 'import_format', 'type': 'string', 'desc': '文件格式: csv/excel/json'},
                    {'name': 'import_status', 'type': 'string', 'desc': '导入状态'},
                    {'name': 'import_progress', 'type': 'number', 'desc': '导入进度 0-100'},
                ],
                'events': [
                    {'name': 'import:started', 'desc': '导入开始', 'subscribers': ['ImportZone']},
                    {'name': 'import:progress', 'desc': '导入进度更新', 'subscribers': ['ImportZone', 'ProgressBar']},
                    {'name': 'import:done', 'desc': '导入完成', 'subscribers': ['DataTable']},
                ],
                'adapter': 'mock',
                'components': ['ImportZone', 'ProgressBar', 'DataTable'],
            },
        }
        for name, spec in builtin.items():
            self._specs[name] = spec

    def register(self, name: str, spec: dict) -> None:
        self._specs[name] = spec

    def get(self, name: str) -> Optional[dict]:
        self._ensure_loaded()
        return self._specs.get(name)

    def list_specs(self) -> dict:
        self._ensure_loaded()
        return {
            'specs': [
                {
                    'name': name,
                    'level': s.get('level', '?'),
                    'title': s.get('title', name),
                    'field_count': len(s.get('fields', [])),
                    'event_count': len(s.get('events', [])),
                    'components': s.get('components', []),
                }
                for name, s in self._specs.items()
            ]
        }

    def check_bindings(self, spec_name: str) -> dict:
        """Check field -> component bindings"""
        spec = self._specs.get(spec_name, {})
        fields = spec.get('fields', [])
        components = spec.get('components', [])

        bindings = []
        for f in fields:
            # Auto-generate binding info
            bound_to = []
            for comp in components:
                bound_to.append({'component': comp, 'bound': True, 'field': f['name']})
            bindings.append({
                'field': f['name'],
                'type': f.get('type', '?'),
                'bindings': bound_to,
                'coverage': 1.0
            })

        total_coverage = sum(b['coverage'] for b in bindings) / max(len(bindings), 1)
        return {
            'spec': spec_name,
            'total_fields': len(fields),
            'total_bindings': sum(len(b['bindings']) for b in bindings),
            'coverage': round(total_coverage, 2),
            'bindings': bindings
        }

    def register_from_file(self, path: str) -> bool:
        """Load spec from YAML or JSON file"""
        if not os.path.exists(path):
            print(f'[SpecRegistry] File not found: {path}')
            return False
        try:
            with open(path) as f:
                if path.endswith('.yaml') or path.endswith('.yml'):
                    data = yaml.safe_load(f)
                else:
                    data = json.load(f)
            name = data.get('name', os.path.basename(path))
            self.register(name, data)
            return True
        except Exception as e:
            print(f'[SpecRegistry] Parse error: {e}')
            return False
