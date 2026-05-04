import { get, writable } from 'svelte/store';
import { stripReasoningTags } from '$lib/stores/thread-store';
import type { SpecDisplayMeta, SpecSlotSummary } from '$lib/workbench/spec-display';
import {
	createSlotPlanGraph,
	previewToolRoute,
	toA2UIModel,
	toFireworksGraph,
	type A2UIModel,
	type FireworksGraph,
	type PlanGraph,
	type RoutePreview,
} from '$lib/services/tool-routing-client';
import { specExplorerStore } from '$lib/stores/spec-explorer-store';

export type SpecSlotA2UIStatus = 'idle' | 'loading' | 'ready';

export type SpecSlotChatPrefill = {
	id: string;
	text: string;
};

export type SpecSlotSession = {
	key: string;
	threadId: string;
	spec: SpecDisplayMeta;
	slot: SpecSlotSummary;
	content: string;
	messages: SpecSlotMessage[];
	compactSummary: string;
	status: 'idle' | 'running' | 'error';
	error: string | null;
	planGraph?: PlanGraph;
	routePreview?: RoutePreview;
	fireworksGraph?: FireworksGraph;
	toolDrafts?: unknown[];
	/** A2UI 核心区模型（由 slot + plan + route 推导） */
	a2uiRevision: number;
	a2uiModel?: A2UIModel;
	a2uiStatus: SpecSlotA2UIStatus;
	/** 由右侧「微调」写入，左侧输入框消费后清除 */
	chatPrefill: SpecSlotChatPrefill | null;
	updatedAt: string;
};

export type SpecSlotSessionState = {
	activeKey: string | null;
	drawerOpen: boolean;
	sessions: Record<string, SpecSlotSession>;
};

type OpenSlotSessionInput = {
	spec: SpecDisplayMeta;
	slot: SpecSlotSummary;
	content: string;
};

const STORAGE_KEY = 'vibex-spec-slot-sessions';
const SSE_URL = import.meta.env.VITE_SSE_URL || 'http://localhost:33338';
const useMockBackend =
	import.meta.env.VITE_MOCK_SSE === '1' || import.meta.env.VITE_MOCK_SSE === 'true';

let activeSource: EventSource | null = null;

function nowIso(): string {
	return new Date().toISOString();
}

function id(): string {
	return crypto.randomUUID();
}

function sessionKey(specPath: string, slotId: string): string {
	return `${specPath.replace(/\\/g, '/')}::${slotId}`;
}

function loadState(): SpecSlotSessionState {
	if (typeof localStorage === 'undefined') {
		return { activeKey: null, drawerOpen: false, sessions: {} };
	}
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return { activeKey: null, drawerOpen: false, sessions: {} };
		const parsed = JSON.parse(raw) as SpecSlotSessionState;
		const sessions: Record<string, SpecSlotSession> = {};
		for (const [k, v] of Object.entries(parsed.sessions ?? {})) {
			const s = v as SpecSlotSession;
			sessions[k] = {
				...s,
				a2uiRevision: s.a2uiRevision ?? 0,
				a2uiStatus: s.a2uiStatus ?? 'idle',
				chatPrefill: s.chatPrefill ?? null,
			};
		}
		return {
			activeKey: parsed.activeKey ?? null,
			drawerOpen: false,
			sessions,
		};
	} catch {
		return { activeKey: null, drawerOpen: false, sessions: {} };
	}
}

function persist(state: SpecSlotSessionState) {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(
		STORAGE_KEY,
		JSON.stringify({
			...state,
			drawerOpen: false,
		})
	);
}

function buildPrompt(session: SpecSlotSession, userInput: string): string {
	const recent = session.messages
		.slice(-8)
		.map(message => `${message.role}: ${message.content}`)
		.join('\n');
	const explorerState = get(specExplorerStore);
	const specList = explorerState.specs
		.map(spec => {
			const title = spec.display?.title ?? spec.name;
			const summary = spec.display?.summary ? ` — ${spec.display.summary}` : '';
			return `- L${spec.level} ${spec.status} ${spec.path} :: ${title}${summary}`;
		})
		.join('\n');

	return [
		userInput,
		'',
		'[Spec Slot Session]',
		`workspace_root: ${explorerState.workspaceRoot || 'unknown'}`,
		`spec_path: ${session.spec.path}`,
		`spec_name: ${session.spec.name}`,
		`spec_level: ${session.spec.level}`,
		`slot_id: ${session.slot.id}`,
		`slot_label: ${session.slot.label}`,
		`slot_status: ${session.slot.status}`,
		`slot_summary: ${session.slot.summary}`,
		`compact_summary: ${session.compactSummary || 'none'}`,
		'',
		...(session.slot.id === 'prototype'
			? [
					'[Design Kit / 原型物料]',
					'工作区约定：.vibex/design/DESIGN.md（栈、令牌、组件边界）；可交付 HTML 放在 .vibex/prototypes/；当前 spec 的 prototype.file 引用相对工作区根路径。',
					'生成或修改原型前须对齐 DESIGN.md；写入当前 spec 的 prototype 字段须经用户明确确认后再 specs/write；写入物料文件使用原型槽工具条 extract/scaffold（已 confirm）。',
					'[/Design Kit]',
					'',
				]
			: []),
		'[A2UI]',
		session.a2uiModel
			? JSON.stringify(
					{
						headline: session.a2uiModel.headline,
						revision: session.a2uiModel.revision,
						routeSummary: session.a2uiModel.routeSummaryLines,
						hints: session.a2uiModel.componentHints,
					},
					null,
					2
				)
			: 'none',
		'[/A2UI]',
		'',
		'要求：这是针对当前 spec 槽位的可延续交互式澄清。先提出必要问题或给出可确认方案；除非用户明确确认，不要直接写文件。',
		'如果需要工具路由，请先展示 plan graph / route preview 的建议步骤。',
		'工具策略：优先造工具、注册路由、使用路由、调试工具、完善工具；不要默认直接生成业务代码。',
		'如果现有工具不足，先输出 metadata-only 工具草案，且必须包含 ai_fill_area(prompt_template/domain_rules/examples/personalization_notes)。',
		'',
		'[Current Tool Route Preview]',
		session.routePreview ? JSON.stringify(session.routePreview, null, 2) : 'none',
		'[/Current Tool Route Preview]',
		'',
		'[Spec List]',
		specList || 'none',
		'[/Spec List]',
		'',
		'[Recent Session Messages]',
		recent || 'none',
		'[/Recent Session Messages]',
		'',
		'[Current Spec Content]',
		session.content,
		'[/Current Spec Content]',
		'[/Spec Slot Session]',
	].join('\n');
}

function createSpecSlotSessionStore() {
	const initial = loadState();
	const { subscribe, update } = writable<SpecSlotSessionState>(initial);

	function commit(mutator: (state: SpecSlotSessionState) => SpecSlotSessionState) {
		update(state => {
			const next = mutator(state);
			persist(next);
			return next;
		});
	}

	function closeStream() {
		activeSource?.close();
		activeSource = null;
	}

	function appendAssistantDelta(key: string, delta: string, isFinal: boolean) {
		if (!delta && !isFinal) return;
		commit(state => {
			const session = state.sessions[key];
			if (!session) return state;
			const messages = [...session.messages];
			const last = messages[messages.length - 1];
			if (last?.role === 'assistant' && session.status === 'running') {
				messages[messages.length - 1] = {
					...last,
					content: isFinal ? stripReasoningTags(delta || last.content) : last.content + delta,
					createdAt: nowIso(),
				};
			} else if (delta) {
				messages.push({ id: id(), role: 'assistant', content: delta, createdAt: nowIso() });
			}
			return {
				...state,
				sessions: {
					...state.sessions,
					[key]: {
						...session,
						messages,
						status: isFinal ? 'idle' : session.status,
						updatedAt: nowIso(),
					},
				},
			};
		});
	}

	async function refreshToolRouting(key: string) {
		const state = get({ subscribe });
		const session = state.sessions[key];
		if (!session) return;
		commit(current => {
			const active = current.sessions[key];
			if (!active) return current;
			return {
				...current,
				sessions: {
					...current.sessions,
					[key]: {
						...active,
						a2uiStatus: 'loading',
						error: null,
						updatedAt: nowIso(),
					},
				},
			};
		});
		try {
			const workspaceRoot = get(specExplorerStore).workspaceRoot;
			const graph = await createSlotPlanGraph({
				goal: `为 ${session.spec.path} 的 ${session.slot.label} 槽位生成工具路由和可视化验证图`,
				specPath: session.spec.path,
				slotId: session.slot.id,
				workspaceRoot,
			});
			const route = await previewToolRoute(graph, workspaceRoot);
			const fireworks = toFireworksGraph(graph, route);
			const fresh = get({ subscribe }).sessions[key] ?? session;
			const nextRev = (fresh.a2uiRevision ?? 0) + 1;
			const a2uiModel = toA2UIModel({
				revision: nextRev,
				slotId: fresh.slot.id,
				slotLabel: fresh.slot.label,
				specPath: fresh.spec.path,
				specTitle: fresh.spec.display.title,
				goal: graph.goal,
				graph,
				route,
			});
			commit(current => {
				const active = current.sessions[key];
				if (!active) return current;
				return {
					...current,
					sessions: {
						...current.sessions,
						[key]: {
							...active,
							planGraph: graph,
							routePreview: route,
							fireworksGraph: fireworks,
							a2uiModel,
							a2uiRevision: nextRev,
							a2uiStatus: 'ready',
							updatedAt: nowIso(),
						},
					},
				};
			});
		} catch (e) {
			commit(current => {
				const active = current.sessions[key];
				if (!active) return current;
				return {
					...current,
					sessions: {
						...current.sessions,
						[key]: {
							...active,
							a2uiStatus: 'idle',
							error: `工具路由预览失败: ${e instanceof Error ? e.message : String(e)}`,
							updatedAt: nowIso(),
						},
					},
				};
			});
		}
	}

	return {
		subscribe,
		open(input: OpenSlotSessionInput) {
			const key = sessionKey(input.spec.path, input.slot.id);
			commit(state => {
				const existing = state.sessions[key];
				const session: SpecSlotSession = existing
					? {
							...existing,
							spec: input.spec,
							slot: input.slot,
							content: input.content,
							a2uiRevision: existing.a2uiRevision ?? 0,
							a2uiStatus: existing.a2uiStatus ?? 'idle',
							chatPrefill: existing.chatPrefill ?? null,
							updatedAt: nowIso(),
						}
					: {
							key,
							threadId: crypto.randomUUID(),
							spec: input.spec,
							slot: input.slot,
							content: input.content,
							messages: [
								{
									id: id(),
									role: 'system',
									content: `已打开「${input.slot.label}」槽位澄清会话。状态：${input.slot.status}。`,
									createdAt: nowIso(),
								},
							],
							compactSummary: '',
							status: 'idle',
							error: null,
							a2uiRevision: 0,
							a2uiStatus: 'idle',
							chatPrefill: null,
							updatedAt: nowIso(),
						};
				return {
					...state,
					activeKey: key,
					drawerOpen: true,
					sessions: { ...state.sessions, [key]: session },
				};
			});
			void refreshToolRouting(key);
		},
		clearChatPrefillActive() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session) return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: { ...session, chatPrefill: null, updatedAt: nowIso() },
					},
				};
			});
		},
		/** 原型物料库等工具条：向左侧对话预填可发送文本 */
		prefillActiveChat(text: string) {
			const t = text.trim();
			if (!t) return;
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session) return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							chatPrefill: { id: id(), text: t },
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		confirmActiveA2UI() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session?.a2uiModel) return state;
				const m = session.a2uiModel;
				const lines = [
					'[A2UI 确认] 用户确认当前槽位可视化摘要。',
					`slot: ${session.slot.id} (${session.slot.label})`,
					...m.cards.slice(0, 8).map(c => `- ${c.title}: ${c.body.slice(0, 200)}`),
				];
				if (m.routeSummaryLines.length) {
					lines.push('--- tool route ---', ...m.routeSummaryLines.slice(0, 12));
				}
				const msg: SpecSlotMessage = {
					id: id(),
					role: 'system',
					content: lines.join('\n'),
					createdAt: nowIso(),
				};
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							messages: [...session.messages, msg],
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		tuneActiveA2UI() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session) return state;
				const bullets =
					session.a2uiModel?.cards.map(c => `- ${c.title}: ${c.body.slice(0, 120)}`).join('\n') ??
					session.routePreview?.decisions.map(d => `- ${d.tool || d.node_id}`).join('\n') ??
					'（尚无路由卡片）';
				const text = `[A2UI 微调] 当前槽位「${session.slot.label}」。请根据右侧 A2UI 确认区调整澄清问题或输出更精确的草案。\n${bullets}`;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							chatPrefill: { id: id(), text },
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		draftActiveA2UI() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session?.a2uiModel) return state;
				const body = session.a2uiModel.cards
					.map(c => `## ${c.title}\n${c.body}`)
					.join('\n\n');
				const msg: SpecSlotMessage = {
					id: id(),
					role: 'system',
					content: `[A2UI 草稿]（未写盘）\n\n${body}`,
					createdAt: nowIso(),
				};
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							messages: [...session.messages, msg],
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		cancelActiveA2UI() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session) return state;
				const msg: SpecSlotMessage = {
					id: id(),
					role: 'system',
					content: '[A2UI 取消] 已放弃当前可视化候选摘要；可点击「重新生成」恢复路由与 A2UI。',
					createdAt: nowIso(),
				};
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							a2uiModel: undefined,
							a2uiStatus: 'idle',
							messages: [...session.messages, msg],
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		regenerateActiveA2UI() {
			const key = get({ subscribe }).activeKey;
			if (key) void refreshToolRouting(key);
		},
		close() {
			closeStream();
			commit(state => ({ ...state, drawerOpen: false }));
		},
		resetActive() {
			closeStream();
			commit(state => {
				if (!state.activeKey) return state;
				const sessions = { ...state.sessions };
				delete sessions[state.activeKey];
				return { ...state, sessions, activeKey: null, drawerOpen: false };
			});
		},
		compactActive() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session) return state;
				const kept = session.messages.slice(-6);
				const compacted = session.messages
					.slice(0, -6)
					.map(message => `${message.role}: ${message.content}`)
					.join('\n');
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							messages: kept.length ? kept : session.messages,
							compactSummary: [session.compactSummary, compacted].filter(Boolean).join('\n\n').slice(-6000),
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		async submitActive(input: string) {
			const text = input.trim();
			if (!text) return;
			const state = get({ subscribe });
			const key = state.activeKey;
			const session = key ? state.sessions[key] : null;
			if (!key || !session || session.status === 'running') return;

			commit(current => {
				const active = current.sessions[key];
				if (!active) return current;
				return {
					...current,
					sessions: {
						...current.sessions,
						[key]: {
							...active,
							messages: [
								...active.messages,
								{ id: id(), role: 'user', content: text, createdAt: nowIso() },
								{ id: id(), role: 'assistant', content: '', createdAt: nowIso() },
							],
							status: 'running',
							error: null,
							updatedAt: nowIso(),
						},
					},
				};
			});

			closeStream();
			const streamPath = useMockBackend
				? `${SSE_URL}/api/sse/threads/${session.threadId}`
				: `${SSE_URL}/api/sse/${session.threadId}`;
			activeSource = new EventSource(streamPath);
			activeSource.addEventListener('message.delta', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const role = String(data.role ?? 'assistant');
					if (role !== 'assistant') return;
					appendAssistantDelta(
						key,
						typeof data.delta === 'string' ? data.delta : '',
						data.is_final === true
					);
				} catch {
					// Ignore malformed stream chunks.
				}
			});
			activeSource.addEventListener('run.completed', () => {
				closeStream();
				commit(current => {
					const active = current.sessions[key];
					if (!active) return current;
					return {
						...current,
						sessions: {
							...current.sessions,
							[key]: { ...active, status: 'idle', updatedAt: nowIso() },
						},
					};
				});
			});
			activeSource.addEventListener('run.failed', () => {
				closeStream();
				commit(current => {
					const active = current.sessions[key];
					if (!active) return current;
					return {
						...current,
						sessions: {
							...current.sessions,
							[key]: { ...active, status: 'error', error: '运行失败', updatedAt: nowIso() },
						},
					};
				});
			});

			try {
				const fresh = get({ subscribe }).sessions[key] ?? session;
				const prompt = buildPrompt(fresh, text);
				if (useMockBackend) {
					await fetch(`${SSE_URL}/api/runs`, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ threadId: session.threadId, goal: prompt }),
					});
				} else {
					await fetch(`${SSE_URL}/api/chat`, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							threadId: session.threadId,
							input: prompt,
							workspaceRoot: get(specExplorerStore).workspaceRoot,
						}),
					});
				}
			} catch (e) {
				closeStream();
				commit(current => {
					const active = current.sessions[key];
					if (!active) return current;
					return {
						...current,
						sessions: {
							...current.sessions,
							[key]: {
								...active,
								status: 'error',
								error: e instanceof Error ? e.message : String(e),
								updatedAt: nowIso(),
							},
						},
					};
				});
			}
		},
	};
}

export const specSlotSessionStore = createSpecSlotSessionStore();

export function activeSpecSlotSession(state: SpecSlotSessionState): SpecSlotSession | null {
	return state.activeKey ? state.sessions[state.activeKey] ?? null : null;
}
