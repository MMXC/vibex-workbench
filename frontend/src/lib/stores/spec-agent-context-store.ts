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
	draftCommand: { id: string; text: string } | null;
};

export type SpecCommand = {
	name: string;
	levels: SpecLevelToken[];
	sample: string;
	description: string;
};

export type WorkbenchAgentContextSnapshot = {
	workspaceRoot?: string;
	workspaceState?: string;
	backendStatus?: string;
	leftActivity?: string;
	centerView?: string;
	selectedSpecPath?: string | null;
	dashboardLevel?: string;
	specCount?: number;
	activeRun?: {
		id?: string;
		status?: string;
		toolCount?: number;
	};
	recentOutput?: string;
	mode?: string;
};

export const specCommands: SpecCommand[] = [
	{
		name: '/workspace',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/workspace "说明当前工作区状态和下一步"',
		description: '查看/解释当前 workspace 状态',
	},
	{
		name: '/specs',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/specs "列出当前 specs 并按槽位完整度归纳"',
		description: '读取当前 spec 列表和槽位摘要',
	},
	{
		name: '/open-spec',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/open-spec "输入 spec 名称、路径或标题关键词"',
		description: '搜索并打开匹配的 spec 详情',
	},
	{
		name: '/search-spec',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/search-spec "输入 spec 名称、路径或标题关键词"',
		description: '搜索 spec 并显示候选项',
	},
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
		name: '/generate',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/generate "运行生成并解释输出"',
		description: '运行或规划 make generate',
	},
	{
		name: '/route',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/route "为当前目标展示 plan graph 和工具路由"',
		description: '生成计划图与工具路由预览',
	},
	{
		name: '/governance',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/governance "检查 parent chain、槽位完整度和漂移风险"',
		description: '执行治理/一致性分析',
	},
	{
		name: '/open-slot',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/open-slot "打开当前 spec 的输入槽位澄清"',
		description: '进入 Spec 槽位抽屉澄清',
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
		name: '/clarify',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample: '/clarify "交互式澄清当前 spec 的缺失槽位"',
		description: '进入交互式可视化澄清，补齐结构、输入、输出、原型或实现边界',
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
	/** 与控制台「新旧项目分流」按钮一致；预填后可在 Composer 中改 root 说明 */
	{
		name: '/spec-init-chain',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample:
			'/workspace "按「新项目 / 无既有 spec」推进：先与用户澄清项目 slug 与目标边界，再调用 workspace_specs_bootstrap（或 POST /api/workspace/spec-bootstrap，overwrite=false）生成 L1–L5 占位链；遵循 .agents/skills/workspace-bootstrap。写盘前必须用户确认。"',
		description: '新仓库：澄清澈后初始化 L1–L5 spec 链',
	},
	{
		name: '/spec-legacy-align',
		levels: ['L1', 'L2', 'L3', 'L4', 'L5', 'IMPL', 'UNKNOWN'],
		sample:
			'/workspace "按「已有代码 / 旧项目」推进：先 workspace_detect_state 与 /governance 看 parent 链与槽位缺口，与用户澄清是「文档对齐」还是「增量补链」；禁止默认 overwrite 已有 YAML；必要时 verify_spec_suite。仅在用户明确要求时覆盖文件。"',
		description: '旧项目：检测、治理与增量补链',
	},
];

function createSpecAgentContextStore() {
	const { subscribe, update, set } = writable<SpecAgentContextState>({
		items: [],
		expanded: false,
		focusedPath: null,
		draftCommand: null,
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
				return { ...state, items, expanded: true, focusedPath: spec.path };
			});
		},
		prefillCommand(text: string) {
			update(state => ({
				...state,
				expanded: true,
				draftCommand: { id: crypto.randomUUID(), text },
			}));
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
			set({ items: [], expanded: false, focusedPath: null, draftCommand: null });
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

export function formatSpecContextForPrompt(options?: {
	workspaceRoot?: string;
	specs?: { path: string; level: number; name: string; status: string; display?: SpecDisplayMeta['display'] }[];
	workbench?: WorkbenchAgentContextSnapshot;
}): string {
	const state = get(specAgentContextStore);
	const workspaceRoot = options?.workspaceRoot?.trim() ?? '';
	const specList = options?.specs ?? [];
	const workbench = options?.workbench;
	if (state.items.length === 0 && !workspaceRoot && specList.length === 0 && !workbench) return '';
	const sections: string[] = [];
	if (workspaceRoot) {
		sections.push(`[Workspace]\nroot: ${workspaceRoot}\n[/Workspace]`);
	}
	if (workbench) {
		const lines = [
			`workspace_state: ${workbench.workspaceState ?? 'unknown'}`,
			`backend_status: ${workbench.backendStatus ?? 'unknown'}`,
			`left_activity: ${workbench.leftActivity ?? 'unknown'}`,
			`center_view: ${workbench.centerView ?? 'unknown'}`,
			`selected_spec_path: ${workbench.selectedSpecPath ?? 'none'}`,
			`dashboard_level: ${workbench.dashboardLevel ?? 'unknown'}`,
			`spec_count: ${workbench.specCount ?? specList.length}`,
			`composer_mode: ${workbench.mode ?? 'text'}`,
			`active_run: ${workbench.activeRun?.status ?? 'none'} (${workbench.activeRun?.toolCount ?? 0} tools)`,
			workbench.recentOutput ? `recent_output: ${workbench.recentOutput}` : '',
		].filter(Boolean);
		sections.push(`[Workbench Context]\n${lines.join('\n')}\n[/Workbench Context]`);
	}
	if (specList.length > 0) {
		const rows = specList
			.map(spec => {
				const title = spec.display?.title ?? spec.name;
				const summary = spec.display?.summary ? ` — ${spec.display.summary}` : '';
				return `- L${spec.level} ${spec.status} ${spec.path} :: ${title}${summary}`;
			})
			.join('\n');
		sections.push(`[Spec List]\n${rows}\n[/Spec List]`);
	}
	if (state.items.length === 0) return `\n\n${sections.join('\n\n')}\n`;
	const lines = state.items.map(item => {
		const focus = item.path === state.focusedPath ? 'focus' : 'context';
		const slots = item.slots.all
			.map(slot => `${slot.label}:${slot.status}${slot.count ? `(${slot.count})` : ''}`)
			.join(', ');
		return [
			`- ${focus}: ${item.display.title}`,
			`  name: ${item.name}`,
			`  level: ${item.level}`,
			`  path: ${item.path}`,
			`  summary: ${item.display.summary}`,
			`  slots: ${slots}`,
		].join('\n');
	});
	sections.push(`[Spec Context]\n${lines.join('\n')}\n[/Spec Context]`);
	return `\n\n${sections.join('\n\n')}\n`;
}
