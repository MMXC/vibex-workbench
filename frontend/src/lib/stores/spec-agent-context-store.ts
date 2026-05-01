import { get, writable } from 'svelte/store';
import type { SpecDisplayMeta, SpecLevelToken } from '$lib/workbench/spec-display';

export type SpecContextItem = SpecDisplayMeta & {
	content?: string;
	attachedAt: string;
};

type SpecAgentContextState = {
	items: SpecContextItem[];
	expanded: boolean;
	focusedPath: string | null;
};

export type SpecCommand = {
	name: string;
	levels: SpecLevelToken[];
	sample: string;
	description: string;
};

export const specCommands: SpecCommand[] = [
	{
		name: '/add',
		levels: ['L1', 'L2', 'L3', 'L4'],
		sample: '/add "描述要新增的子 spec"',
		description: '添加唯一合法下一层子 spec',
	},
	{
		name: '/validate',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/validate "校验现有输入输出及 spec 是否合理"',
		description: '校验输入输出、层级、parent chain 与边界',
	},
	{
		name: '/deduce',
		levels: ['L1', 'L2', 'L3', 'L4'],
		sample: '/deduce "根据现有 spec 及子 spec 推导新子 spec"',
		description: '根据当前 spec 与 children 推导缺失子 spec',
	},
	{
		name: '/analyse',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/analyse "分析现有 spec"',
		description: '只分析，不修改文件',
	},
	{
		name: '/confirm',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/confirm "确认 spec"',
		description: '确认草稿并进入写盘前检查',
	},
	{
		name: '/implement',
		levels: ['L3', 'L4', 'L5', 'IMPL'],
		sample: '/implement "实现当前 spec"',
		description: '实现当前 spec 或实现切片',
	},
	{
		name: '/implement-loop',
		levels: ['L4', 'L5', 'IMPL'],
		sample: '/implement-loop "循环直至实现"',
		description: '实现、验证、修复循环直到通过或阻塞',
	},
];

function createSpecAgentContextStore() {
	const { subscribe, update, set } = writable<SpecAgentContextState>({
		items: [],
		expanded: false,
		focusedPath: null,
	});

	return {
		subscribe,
		addSpec(spec: SpecDisplayMeta, content?: string) {
			update(state => {
				const item: SpecContextItem = {
					...spec,
					content,
					attachedAt: new Date().toISOString(),
				};
				const items = [item, ...state.items.filter(existing => existing.path !== spec.path)].slice(0, 8);
				return { ...state, items, focusedPath: spec.path };
			});
		},
		removeSpec(path: string) {
			update(state => {
				const items = state.items.filter(item => item.path !== path);
				return {
					...state,
					items,
					focusedPath: state.focusedPath === path ? items[0]?.path ?? null : state.focusedPath,
				};
			});
		},
		toggleExpanded() {
			update(state => ({ ...state, expanded: !state.expanded }));
		},
		clear() {
			set({ items: [], expanded: false, focusedPath: null });
		},
	};
}

export const specAgentContextStore = createSpecAgentContextStore();

export function getAvailableSpecCommands(level: SpecLevelToken | null | undefined): SpecCommand[] {
	const current = level ?? 'UNKNOWN';
	return specCommands.filter(command => command.levels.includes(current));
}

export function currentFocusedSpecContext(): SpecContextItem | null {
	const state = get(specAgentContextStore);
	return state.items.find(item => item.path === state.focusedPath) ?? state.items[0] ?? null;
}

export function formatSpecContextForPrompt(): string {
	const state = get(specAgentContextStore);
	if (state.items.length === 0) return '';
	const lines = state.items.map(item => {
		const focus = item.path === state.focusedPath ? 'focus' : 'context';
		return [
			`- ${focus}: ${item.display.title}`,
			`  name: ${item.name}`,
			`  level: ${item.level}`,
			`  path: ${item.path}`,
			`  summary: ${item.display.summary}`,
		].join('\n');
	});
	return `\n\n[Spec Context]\n${lines.join('\n')}\n[/Spec Context]\n`;
}
