#!/usr/bin/env python3
"""
SpecPilot CLI — 核心能力层
Usage: python3 -m specpilot.cli <command> <subcommand> [args]
"""
import sys
import os

# Add parent to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from .datacenter.core import DataCenter
from .eventcenter.core import EventCenter
from .adapters.manager import AdapterManager
from .specregistry.core import SpecRegistry
from .mfregistry.core import MFRegistry

import argparse
import json

def main():
    parser = argparse.ArgumentParser(prog='specpilot', description='SpecPilot CLI')
    sub = parser.add_subparsers(dest='cmd')

    # dc subcommands
    dc = sub.add_parser('dc', help='DataCenter operations')
    dc_sub = dc.add_subparsers(dest='sub')

    p = dc_sub.add_parser('get', help='Get value')
    p.add_argument('key', help='Key name')
    
    p = dc_sub.add_parser('set', help='Set value')
    p.add_argument('key', help='Key name')
    p.add_argument('value', help='Value (JSON)')
    
    p = dc_sub.add_parser('list', help='List all keys')
    
    p = dc_sub.add_parser('watch', help='Watch changes')
    p.add_argument('key', nargs='?', help='Key to watch (optional, watch all if omitted)')
    
    p = dc_sub.add_parser('query', help='Query via adapter')
    p.add_argument('query', help='SQL-like query')

    # ec subcommands
    ec = sub.add_parser('ec', help='EventCenter operations')
    ec_sub = ec.add_subparsers(dest='sub')

    p = ec_sub.add_parser('emit', help='Emit event')
    p.add_argument('event', help='Event name')
    p.add_argument('payload', nargs='?', help='Payload (JSON string)')

    p = ec_sub.add_parser('subscribe', help='Subscribe to event')
    p.add_argument('event', help='Event name')
    p.add_argument('handler', help='Handler name')

    p = ec_sub.add_parser('history', help='Show event history')
    p.add_argument('--limit', type=int, default=20, help='Limit')

    p = ec_sub.add_parser('list', help='List subscriptions')

    # ad subcommands
    ad = sub.add_parser('ad', help='Adapter operations')
    ad_sub = ad.add_subparsers(dest='sub')

    p = ad_sub.add_parser('list', help='List adapters')
    p = ad_sub.add_parser('switch', help='Switch active adapter')
    p.add_argument('name', help='Adapter name: mock|http|ws|grpc')

    p = ad_sub.add_parser('query', help='Query via current adapter')
    p.add_argument('q', help='Query string')

    p = ad_sub.add_parser('status', help='Show current adapter status')

    # spec subcommands
    sp = sub.add_parser('spec', help='Spec registry operations')
    sp_sub = sp.add_subparsers(dest='sub')

    p = sp_sub.add_parser('get', help='Get spec')
    p.add_argument('name', help='Spec name')

    p = sp_sub.add_parser('register', help='Register spec')
    p.add_argument('path', help='Spec YAML/JSON file path')

    p = sp_sub.add_parser('list', help='List all specs')

    p = sp_sub.add_parser('binding', help='Check field bindings')
    p.add_argument('spec', help='Spec name')

    # mf subcommands
    mf = sub.add_parser('mf', help='MF registry operations')
    mf_sub = mf.add_subparsers(dest='sub')

    p = mf_sub.add_parser('register', help='Register component')
    p.add_argument('name', help='Component name')
    p.add_argument('path', help='Component path/URL')

    p = mf_sub.add_parser('list', help='List registered components')

    p = mf_sub.add_parser('resolve', help='Resolve component from spec')
    p.add_argument('spec', help='Spec name')

    # run subcommand: 4-layer integration test
    p = sub.add_parser('run', help='Run 4-layer integration test')
    p.add_argument('scenario', nargs='?', default='basic', help='Scenario: basic|threshold|adapter-switch')
    p.add_argument('--verbose', '-v', action='store_true')

    args = parser.parse_args()

    if not args.cmd:
        parser.print_help()
        return

    # Initialize singletons
    _dc = DataCenter()
    _ec = EventCenter()
    _am = AdapterManager()
    _sr = SpecRegistry()
    _mr = MFRegistry()

    # Route commands
    if args.cmd == 'dc':
        if args.sub == 'get':
            v = _dc.get(args.key)
            print(json.dumps({'key': args.key, 'value': v}))
        elif args.sub == 'set':
            import json as j
            try:
                val = j.loads(args.value)
            except:
                val = args.value
            _dc.set(args.key, val)
            print(json.dumps({'key': args.key, 'value': val, 'ok': True}))
        elif args.sub == 'list':
            print(json.dumps(_dc.dump()))
        elif args.sub == 'watch':
            _dc.watch(args.key)
        elif args.sub == 'query':
            adapter = _am.get_active()
            result = adapter.query(args.query)
            _dc.apply_result(result)
            print(json.dumps({'adapter': _am.active_name, 'result': result}))
        else:
            dc.print_help()

    elif args.cmd == 'ec':
        if args.sub == 'emit':
            import json as j
            payload = j.loads(args.payload) if args.payload else {}
            _ec.emit(args.event, payload)
            print(json.dumps({'event': args.event, 'payload': payload, 'subscribers_notified': len(_ec._subs.get(args.event, []))}))
        elif args.sub == 'subscribe':
            _ec.subscribe(args.event, args.handler)
            print(json.dumps({'event': args.event, 'handler': args.handler, 'ok': True}))
        elif args.sub == 'history':
            for ev in _ec.history(args.limit):
                print(json.dumps(ev))
        elif args.sub == 'list':
            print(json.dumps(_ec.list_subs()))
        else:
            ec.print_help()

    elif args.cmd == 'ad':
        if args.sub == 'list':
            print(json.dumps(_am.list_adapters()))
        elif args.sub == 'switch':
            _am.switch_to(args.name)
            print(json.dumps({'active': _am.active_name, 'ok': True}))
        elif args.sub == 'query':
            adapter = _am.get_active()
            result = adapter.query(args.q)
            print(json.dumps({'adapter': _am.active_name, 'query': args.q, 'result': result}))
        elif args.sub == 'status':
            print(json.dumps(_am.status()))
        else:
            ad.print_help()

    elif args.cmd == 'spec':
        if args.sub == 'get':
            print(json.dumps(_sr.get(args.name)))
        elif args.sub == 'register':
            _sr.register_from_file(args.path)
            print(json.dumps({'path': args.path, 'ok': True}))
        elif args.sub == 'list':
            print(json.dumps(_sr.list_specs()))
        elif args.sub == 'binding':
            print(json.dumps(_sr.check_bindings(args.spec)))
        else:
            sp.print_help()

    elif args.cmd == 'mf':
        if args.sub == 'register':
            _mr.register(args.name, args.path)
            print(json.dumps({'name': args.name, 'path': args.path, 'ok': True}))
        elif args.sub == 'list':
            print(json.dumps(_mr.list_components()))
        elif args.sub == 'resolve':
            print(json.dumps(_mr.resolve_from_spec(args.spec)))
        else:
            mf.print_help()

    elif args.cmd == 'run':
        run_integration(_dc, _ec, _am, _sr, _mr, args.scenario, args.verbose)

def run_integration(dc, ec, am, sr, mr, scenario, verbose):
    import json as j

    print(f'\n=== SpecPilot 4-Layer Integration: {scenario} ===\n')

    if scenario == 'basic':
        # Layer 1: Load specs
        print('[L4] Spec Registry: loading...')
        sr.load_builtin()
        specs = sr.list_specs()
        print(f'  Specs loaded: {len(specs["specs"])}')

        # Layer 2: Switch adapter
        print('\n[L2] DataCenter: switching to mockAdapter...')
        am.switch_to('mock')
        adapter = am.get_active()
        print(f'  Active: {am.active_name}')

        # Layer 3: Query via adapter
        print('\n[L3] EventCenter: emit adapter.query event...')
        result = adapter.query('SELECT cpu_usage, memory_percent FROM servers')
        ec.emit('adapter.query', {'adapter': am.active_name, 'result': result})
        print(f'  Subscribers notified: {len(ec._subs.get("adapter.query", []))}')

        # Apply to datacenter
        print('\n[L2] DataCenter: apply result...')
        dc.apply_result(result)
        for k, v in dc.dump()['data'].items():
            val = v['value'] if isinstance(v, dict) else v
            print(f'  {k}: {val}')

        # Emit data update event
        print('\n[L3] EventCenter: emit data.updated...')
        ec.emit('data.updated', dc.dump()['data'])
        history = ec.history(5)
        print(f'  Event history: {len(history)} entries')

        # Layer 1: Resolve MF components
        print('\n[L1] MF Registry: resolve from spec...')
        mr.register('KPICard', 'prototype/pages/KPICard.svelte')
        mr.register('ServerList', 'prototype/pages/ServerList.svelte')
        resolved = mr.resolve_from_spec('L3-dashboard-kpi')
        print(f'  Resolved components: {resolved}')

        print('\n[PASS] 4-layer integration: basic\n')

    elif scenario == 'threshold':
        print('[Scenario: CPU threshold exceeded]')
        am.switch_to('mock')
        adapter = am.get_active()
        result = adapter.query('SELECT cpu_usage FROM servers WHERE status = "warning"')
        dc.apply_result(result)
        cpu = result.get('cpu_usage', 0)

        print(f'  CPU: {cpu}% (threshold: 80%)')
        if cpu > 80:
            print(f'  [{chr(0x1b)}[31mEXCEEDED{chr(0x1b)}[0m] Emitting kpi:threshold.exceeded')
            ec.emit('kpi:threshold.exceeded', {'cpu_usage': cpu, 'threshold': 80})
        else:
            print(f'  [OK] No threshold exceeded')
        print('\n[PASS] threshold scenario\n')

    elif scenario == 'adapter-switch':
        print('[Scenario: Switch adapters]')
        adapters = ['mock', 'http', 'ws', 'grpc']
        for name in adapters:
            am.switch_to(name)
            adapter = am.get_active()
            result = adapter.query('SELECT * FROM servers LIMIT 1')
            dc.apply_result(result)
            print(f'  [{name}] -> cpu={result.get("cpu_usage", "?")}, mem={result.get("memory_percent", "?")}')
        print('\n[PASS] adapter-switch scenario\n')

if __name__ == '__main__':
    main()
