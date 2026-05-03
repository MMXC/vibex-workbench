import { get, writable } from 'svelte/store';
import { stripReasoningTags } from '$lib/stores/thread-store';
import type { SpecDisplayMeta, SpecSlotSummary } from '$lib/workbench/spec-display';
import {
	createSlotPlanGraph,
	previewToolRoute,
	toFireworksGraph,
	type FireworksGraph,
	type PlanGraph,
	type RoutePreview,
} from '$lib/services/tool-routing-client';

export type SpecSlotMessage = {
	id: string;
	role: 'user' | 'assistant' | 'system';
	content: string;
	createdAt: string;
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
		return {
			activeKey: parsed.activeKey ?? null,
			drawerOpen: false,
			sessions: parsed.sessions ?? {},
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

	return [
		userInput,
		'',
		'[Spec Slot Session]',
		`spec_path: ${session.spec.path}`,
		`spec_name: ${session.spec.name}`,
		`spec_level: ${session.spec.level}`,
		`slot_id: ${session.slot.id}`,
		`slot_label: ${session.slot.label}`,
		`slot_status: ${session.slot.status}`,
		`slot_summary: ${session.slot.summary}`,
		`compact_summary: ${session.compactSummary || 'none'}`,
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
		try {
			const graph = await createSlotPlanGraph({
				goal: `为 ${session.spec.path} 的 ${session.slot.label} 槽位生成工具路由和可视化验证图`,
				specPath: session.spec.path,
				slotId: session.slot.id,
			});
			const route = await previewToolRoute(graph);
			const fireworks = toFireworksGraph(graph, route);
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
						body: JSON.stringify({ threadId: session.threadId, input: prompt }),
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
