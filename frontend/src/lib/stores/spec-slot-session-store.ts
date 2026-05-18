import { get, writable } from 'svelte/store';
import { stripReasoningTags } from '$lib/stores/thread-store';
import type { SpecDisplayMeta, SpecSlotSummary } from '$lib/workbench/spec-display';
import {
	extractSpecDisplay,
	getSpecPptFileFromYaml,
	upsertSpecBlockPptFile,
} from '$lib/workbench/spec-display';
import {
	createSlotPlanGraph,
	extractHtmlFromMarkdown,
	previewToolRoute,
	lastAssistantHtmlFence,
	toA2UIModel,
	toFireworksGraph,
	type A2UIModel,
	type FireworksGraph,
	type PlanGraph,
	type RoutePreview,
} from '$lib/services/tool-routing-client';
import { specExplorerStore } from '$lib/stores/spec-explorer-store';
import { agentApiUrl, getAgentApiBase, postAgentChat } from '$lib/runtime/agent-transport';
import {
	extractToolWorkspacePath,
	formatToolCalledBody,
	formatToolCompletedFooter,
} from '$lib/workbench/tool-call-chat';
import type {
	OrchestrationTracePayload,
	RepairDecisionPayload,
	StrongValidationPlan,
	ValidationItemRunResult,
	VerificationSubmission,
} from '$lib/workbench/rules-engine-contract';

export type SpecSlotA2UIStatus = 'idle' | 'loading' | 'ready';

export type SpecSlotChatPrefill = {
	id: string;
	text: string;
};

export type SpecSlotIterationNode = {
	id: string;
	label: string;
	type: 'run' | 'plan' | 'tool' | 'result';
	status: 'running' | 'done' | 'error';
	detail?: string;
};

export type SpecSlotIterationEdge = {
	from: string;
	to: string;
	label?: string;
};

export type SpecSlotMessage = {
	id: string;
	role: 'system' | 'user' | 'assistant' | 'tool';
	content: string;
	createdAt: string;
	/** 工具调用关联的工作区相对路径，用于在中央打开只读预览 */
	toolOpenPath?: string | null;
};

/** Stream 阶段的消息（ts: ISO 时间戳），最终合并为 SpecSlotMessage */
export type A2UIStreamMsg = {
	id: string;
	role: 'system' | 'user' | 'assistant' | 'tool';
	content: string;
	ts: string;
};

/** 由嵌入原型的 vibex-prototype-agent-bridge postMessage 上报，用于抽屉内高亮叠层 */
export type PrototypeAgentHighlightRect = {
	sel: string;
	top: number;
	left: number;
	width: number;
	height: number;
};

export type PrototypeAgentUiState = {
	rects: PrototypeAgentHighlightRect[];
	viewport: { width: number; height: number } | null;
	onboard: { title: string; body: string; step?: number; total?: number } | null;
	updatedAt: string;
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
	/** 右侧详情抽屉：多步迭代过程动态图（来自 SSE run/tool 事件） */
	iterationNodes: SpecSlotIterationNode[];
	iterationEdges: SpecSlotIterationEdge[];
	iterationCursorId: string | null;
	/** 由右侧「微调」写入，左侧输入框消费后清除 */
	chatPrefill: SpecSlotChatPrefill | null;
	/** 规则引擎：强校验计划（默认种子或后端下发） */
	strongValidationPlan?: StrongValidationPlan | null;
	/** 用户 RED/GREEN 提交 */
	verificationSubmission?: VerificationSubmission | null;
	/** 强校验逐条运行结果 */
	validationRuns?: Record<string, ValidationItemRunResult>;
	/** Inspector trace（验证节点） */
	traceEvents?: OrchestrationTracePayload[];
	/** Spec 关联演示 HTML（用于图谱泳道区域展示） */
	pptDemoPath?: string | null;
	pptDemoHtml?: string | null;
	pptDemoLoading?: boolean;
	pptDemoError?: string | null;
	/** 最近一次纠错决策（SSE repair.decision） */
	lastRepairDecision?: RepairDecisionPayload | null;
	/** 当前 agent 轮次：用户输入框原文（不含 buildPrompt 包装） */
	runningUserPrompt?: string | null;
	/** 中止或回滚本轮后，可用「继续」预填同一句再发 */
	pendingResumePrompt?: string | null;
	/** 原型页 bridge → 抽屉高亮 / onboard 叠层 */
	prototypeAgentUi?: PrototypeAgentUiState | null;
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
const useMockBackend =
	import.meta.env.VITE_MOCK_SSE === '1' || import.meta.env.VITE_MOCK_SSE === 'true';

let activeSource: EventSource | null = null;
let activeChatAbort: AbortController | null = null;

function nowIso(): string {
	return new Date().toISOString();
}

function id(): string {
	return crypto.randomUUID();
}

function findLastUserMessageIndex(messages: SpecSlotMessage[]): number {
	for (let i = messages.length - 1; i >= 0; i--) {
		if (messages[i].role === 'user') return i;
	}
	return -1;
}

function abortActiveChatRequest(): void {
	activeChatAbort?.abort();
	activeChatAbort = null;
}

function isTrustedPrototypeBridgeOrigin(origin: string): boolean {
	if (typeof window === 'undefined' || !origin) return false;
	try {
		const agentOrigin = new URL(getAgentApiBase()).origin;
		if (origin === agentOrigin) return true;
		if (origin === window.location.origin) return true;
	} catch {
		return false;
	}
	return false;
}

function normalizePrototypeBridgeRects(raw: unknown): PrototypeAgentHighlightRect[] {
	if (!Array.isArray(raw)) return [];
	const out: PrototypeAgentHighlightRect[] = [];
	for (const x of raw) {
		if (!x || typeof x !== 'object') continue;
		const o = x as Record<string, unknown>;
		const sel = String(o.sel ?? '');
		const top = Number(o.top);
		const left = Number(o.left);
		const width = Number(o.width);
		const height = Number(o.height);
		if (!Number.isFinite(top) || !Number.isFinite(left) || !Number.isFinite(width) || !Number.isFinite(height))
			continue;
		if (width <= 0 || height <= 0 || width > 10000 || height > 10000) continue;
		out.push({ sel, top, left, width, height });
	}
	return out;
}

function parsePrototypeBridgeViewport(raw: unknown): { width: number; height: number } | null {
	if (!raw || typeof raw !== 'object') return null;
	const o = raw as Record<string, unknown>;
	const width = Number(o.width);
	const height = Number(o.height);
	if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return null;
	return { width, height };
}

function parsePrototypeBridgeOnboard(raw: unknown): PrototypeAgentUiState['onboard'] {
	if (!raw || typeof raw !== 'object') return null;
	const o = raw as Record<string, unknown>;
	const title = String(o.title ?? '').trim();
	if (!title) return null;
	return {
		title,
		body: String(o.body ?? ''),
		step: typeof o.step === 'number' ? o.step : undefined,
		total: typeof o.total === 'number' ? o.total : undefined,
	};
}

function sessionKey(specPath: string, slotId: string): string {
	return `${specPath.replace(/\\/g, '/')}::${slotId}`;
}

function basenameNoExt(path: string): string {
	const p = path.replace(/\\/g, '/').split('/').pop() ?? 'spec';
	return p.replace(/\.[^.]+$/, '');
}

function derivePptPath(specPath: string, content: string): string {
	const explicit = getSpecPptFileFromYaml(content);
	if (explicit) return explicit;
	return `.vibex/ppt/${basenameNoExt(specPath)}.html`;
}

function derivePptPathFromSession(session: SpecSlotSession): string {
	return derivePptPath(session.spec.path, session.content ?? '');
}

function delay(ms: number): Promise<void> {
	return new Promise(resolve => setTimeout(resolve, ms));
}

function seedStrongValidationPlan(specPath: string, slotId: string): StrongValidationPlan | null {
	if (slotId !== 'prototype' && slotId !== 'implementation') return null;
	const plan_id = `local:${slotId}:${specPath.replace(/\\/g, '/')}`;
	const items =
		slotId === 'prototype'
			? [
					{
						id: 'make-validate',
						label: '全库 spec YAML 校验',
						command: 'make validate',
						timeout_sec: 300,
						expect_signal: 'green',
					},
					{
						id: 'design-md',
						label: '存在 DESIGN.md（原型栈约定）',
						command: 'test -f .vibex/design/DESIGN.md',
						timeout_sec: 30,
						expect_signal: 'green',
					},
				]
			: [
					{
						id: 'make-validate',
						label: '全库 spec YAML 校验',
						command: 'make validate',
						timeout_sec: 300,
						expect_signal: 'green',
					},
					{
						id: 'spec-validate-slot',
						label: '校验当前 spec 文件',
						command: `make validate`,
						timeout_sec: 300,
						expect_signal: 'green',
					},
				];
	return {
		plan_id,
		slot_binding: slotId,
		spec_path: specPath.replace(/\\/g, '/'),
		items,
	};
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
				iterationNodes: s.iterationNodes ?? [],
				iterationEdges: s.iterationEdges ?? [],
				iterationCursorId: s.iterationCursorId ?? null,
				chatPrefill: s.chatPrefill ?? null,
				strongValidationPlan: s.strongValidationPlan ?? null,
				verificationSubmission: s.verificationSubmission ?? null,
				validationRuns: s.validationRuns ?? {},
				traceEvents: s.traceEvents ?? [],
				pptDemoPath: s.pptDemoPath ?? null,
				pptDemoHtml: s.pptDemoHtml ?? null,
				pptDemoLoading: s.pptDemoLoading ?? false,
				pptDemoError: s.pptDemoError ?? null,
				lastRepairDecision: s.lastRepairDecision ?? null,
				runningUserPrompt: s.runningUserPrompt ?? null,
				pendingResumePrompt: s.pendingResumePrompt ?? null,
				prototypeAgentUi: s.prototypeAgentUi ?? null,
				status: s.status === 'running' ? 'idle' : s.status,
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

/** 行首 `/slug` 或 `/slug 描述` → 路由 agent_profile；首行提示词为任务正文（不含命令前缀）。 */
export function parseLeadingAgentSlashCommand(text: string): {
	agentProfile: string | null;
	userTaskLine: string;
} {
	const t = text.trim();
	const m = t.match(/^\/([a-zA-Z0-9][a-zA-Z0-9_-]*)(?:\s+(.*))?$/s);
	if (!m) {
		return { agentProfile: null, userTaskLine: t };
	}
	const slug = m[1].toLowerCase();
	const rest = (m[2] ?? '').trim();
	const userTaskLine = rest || '请结合当前 spec 槽位与上下文完成专业任务。';
	return { agentProfile: slug, userTaskLine };
}

function buildPrompt(session: SpecSlotSession, userInput: string): string {
	const recent = session.messages
		.slice(-8)
		.map(message => `${message.role}: ${message.content}`)
		.join('\n');
	const explorerState = get(specExplorerStore);
	const MAX_SPEC_LIST = 12;
	const activeSpecLine = explorerState.specs.find(spec => spec.path === session.spec.path);
	const others = explorerState.specs.filter(spec => spec.path !== session.spec.path).slice(0, MAX_SPEC_LIST - 1);
	const compactSpecs = [activeSpecLine, ...others].flatMap(s => (s ? [s] : []));
	const specList = compactSpecs
		.map(spec => {
			const title = spec.display?.title ?? spec.name;
			const summary = spec.display?.summary ? ` — ${spec.display.summary}` : '';
			return `- L${spec.level} ${spec.status} ${spec.path} :: ${title}${summary}`;
		})
		.join('\n');
	const omittedCount = Math.max(0, explorerState.specs.length - compactSpecs.length);
	const specListFooter =
		omittedCount > 0
			? `... omitted ${omittedCount} specs to control context size`
			: '';
	const boundedSpecContent =
		session.content.length > 6000
			? `${session.content.slice(0, 6000)}\n... truncated ${session.content.length - 6000} chars`
			: session.content;

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
					'工作区约定：.vibex/design/DESIGN.md（栈、令牌、组件边界）；可交付 HTML 放在 .vibex/prototypes/；演示用 HTML PPT 路径写在顶层 spec.ppt_file（相对工作区根），不要写入 prototype。',
					'写入物料文件优先 write_file；在原型 HTML 中可引入 `.vibex/assets/vibex-prototype-agent-bridge.js` 与抽屉高亮/onboard 联动。',
					'[/Design Kit]',
					'',
				]
			: []),
		...(session.slot.id === 'implementation'
			? [
					'[Implementation / spec 治理联动]',
					'本槽位右侧 iframe 优先预览 YAML 绑定的静态 HTML：在 spec 顶层维护 `implementation:`，例如：',
					'- `implementation.stage`：治理阶段（如 planning / routing / validating / shipped；可按仓库约定扩展）。',
					'- `implementation.preview_html`：主预览页，相对工作区根的 `.html` 路径。',
					'- `implementation.artifacts`（可选）：命名子产物，如 route_graph、plan_review，值为相对 `.html` 路径。',
					'专业 agent 产出页面时使用 write_file 写入目标路径，再通过 specs/write 更新上述字段并推进 `implementation.stage` / `spec.status`；不要在对话里用超长 HTML 代替落盘。',
					'[/Implementation]',
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
						...(session.slot.id === 'prototype' && session.a2uiModel.uiWorkflowGate
							? {
									uiWorkflowGate: session.a2uiModel.uiWorkflowGate,
								}
							: {}),
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
		[specList, specListFooter].filter(Boolean).join('\n') || 'none',
		'[/Spec List]',
		'',
		'[Recent Session Messages]',
		recent || 'none',
		'[/Recent Session Messages]',
		'',
		'[Current Spec Content]',
		boundedSpecContent,
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

	/** 追加消息到指定 session 的 messages 数组（内部使用） */
	function appendMessage(key: string, msg: SpecSlotMessage) {
		commit(state => {
			const session = state.sessions[key];
			if (!session) return state;
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
	}

	/** 实现槽：一轮对话跑完后拉盘刷新 YAML，供右侧 implementation.preview_html iframe 与槽位摘要同步 */
	async function reloadSessionSpecAfterTurn(key: string) {
		const sess = get({ subscribe }).sessions[key];
		if (!sess || sess.slot.id !== 'implementation') return;
		const wsRoot = get(specExplorerStore).workspaceRoot?.trim();
		if (!wsRoot) return;
		try {
			const q = new URLSearchParams({
				path: sess.spec.path,
				workspaceRoot: wsRoot,
			});
			const r = await fetch(`${agentApiUrl('/api/workspace/specs/read')}?${q.toString()}`);
			if (!r.ok) return;
			const j = (await r.json()) as { content?: string };
			const raw = String(j.content ?? '');
			const meta = extractSpecDisplay(raw, sess.spec.path);
			const slot = meta.slots.all.find(s => s.id === sess.slot.id) ?? sess.slot;
			commit(state => {
				const cur = state.sessions[key];
				if (!cur) return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...cur,
							content: raw,
							spec: meta,
							slot,
							updatedAt: nowIso(),
						},
					},
				};
			});
		} catch {
			// best-effort：静默失败，控制台仍可手动刷新
		}
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

			let a2uiModel = session.a2uiModel;
			if (session.slot.id === 'prototype' && a2uiModel) {
				const tail = messages[messages.length - 1];
				if (tail?.role === 'assistant') {
					const extracted = extractHtmlFromMarkdown(tail.content);
					if (extracted) {
						const prev = a2uiModel.prototype;
						if (prev?.mode === 'workspace_file' && prev.fileRel?.trim()) {
							a2uiModel = {
								...a2uiModel,
								prototype: {
									...prev,
									assistantDraftHtml: extracted,
									caption:
										'HTML 原型预览：主区为 prototype.file（工作区静态 HTML）；下方为助手 fenced HTML 草稿（iframe sandbox：allow-scripts）。',
								},
							};
						} else {
							a2uiModel = {
								...a2uiModel,
								prototype: {
									mode: 'html_snippet',
									html: extracted,
									caption:
										'HTML 原型预览（来自助手 fenced html；iframe sandbox：allow-scripts）。',
								},
							};
						}
					}
				}
			}

			return {
				...state,
				sessions: {
					...state.sessions,
					[key]: {
						...session,
						messages,
						a2uiModel,
						status: isFinal ? 'idle' : session.status,
						runningUserPrompt: isFinal ? null : session.runningUserPrompt,
						updatedAt: nowIso(),
					},
				},
			};
		});
	}

	function appendToolCallSessionMessage(key: string, msg: SpecSlotMessage) {
		commit(state => {
			const session = state.sessions[key];
			if (!session) return state;
			if (session.messages.some(m => m.id === msg.id)) return state;
			const messages = [...session.messages];
			const last = messages[messages.length - 1];
			if (last?.role === 'assistant' && session.status === 'running') {
				messages.splice(messages.length - 1, 0, msg);
			} else {
				messages.push(msg);
			}
			return {
				...state,
				sessions: {
					...state.sessions,
					[key]: { ...session, messages, updatedAt: nowIso() },
				},
			};
		});
	}

	/** Inject a tool result message into the active session (used by StatusBar bootstrap button) */
	function injectToolResult(content: string, openPath?: string | null) {
		const { activeKey, sessions } = (() => {
			let current: SpecSlotSessionState = {} as SpecSlotSessionState;
			specSlotSessionStore.subscribe(s => { current = s; })();
			return current;
		})();
		if (!activeKey || !sessions[activeKey]) return;
		const msg: SpecSlotMessage = {
			id: `bootstrap-${Date.now()}`,
			role: 'tool',
			content,
			createdAt: nowIso(),
			toolOpenPath: openPath,
		};
		appendToolCallSessionMessage(activeKey, msg);
	}

	function patchSessionMessage(key: string, messageId: string, patch: Partial<SpecSlotMessage>) {
		commit(state => {
			const session = state.sessions[key];
			if (!session) return state;
			const messages = [...session.messages];
			const i = messages.findIndex(m => m.id === messageId);
			if (i < 0) return state;
			messages[i] = { ...messages[i], ...patch };
			return {
				...state,
				sessions: {
					...state.sessions,
					[key]: { ...session, messages, updatedAt: nowIso() },
				},
			};
		});
	}

	function upsertIterationNode(
		key: string,
		node: SpecSlotIterationNode,
		opts?: { moveCursor?: boolean; edgeLabelFromPrev?: string }
	) {
		commit(state => {
			const session = state.sessions[key];
			if (!session) return state;
			const nodes = [...session.iterationNodes];
			const idx = nodes.findIndex(n => n.id === node.id);
			if (idx >= 0) nodes[idx] = { ...nodes[idx], ...node };
			else nodes.push(node);

			let edges = session.iterationEdges;
			let cursor = session.iterationCursorId;
			if (opts?.moveCursor) {
				if (cursor && cursor !== node.id && !edges.some(e => e.from === cursor && e.to === node.id)) {
					edges = [...edges, { from: cursor, to: node.id, label: opts.edgeLabelFromPrev }];
				}
				cursor = node.id;
			}

			return {
				...state,
				sessions: {
					...state.sessions,
					[key]: {
						...session,
						iterationNodes: nodes,
						iterationEdges: edges,
						iterationCursorId: cursor,
						updatedAt: nowIso(),
					},
				},
			};
		});
	}

	async function emitInspectorTrace(trace: OrchestrationTracePayload) {
		if (useMockBackend) return;
		try {
			await fetch(agentApiUrl('/api/workbench/inspector/trace'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(trace),
			});
		} catch {
			// best-effort only
		}
	}

	function appendTraceEvent(key: string, trace: OrchestrationTracePayload) {
		commit(state => {
			const session = state.sessions[key];
			if (!session) return state;
			const next = [...(session.traceEvents ?? []), trace].slice(-80);
			return {
				...state,
				sessions: {
					...state.sessions,
					[key]: {
						...session,
						traceEvents: next,
						updatedAt: nowIso(),
					},
				},
			};
		});
	}

type ParsedValidationTemplate = {
	tool: string;
	args: Record<string, unknown>;
};

/**
 * Parse StrongValidationItem.tool_call_template.
 * Supported:
 * 1) plain name: "cdp_validate"
 * 2) JSON: {"tool":"cdp_validate","arguments":{...}}
 * 3) JSON: {"name":"cdp_validate","args":{...}}
 */
function parseValidationTemplate(raw: string | undefined): ParsedValidationTemplate | null {
	const src = String(raw ?? '').trim();
	if (!src) return null;
	if (!src.startsWith('{')) return { tool: src, args: {} };
	try {
		const v = JSON.parse(src) as Record<string, unknown>;
		const tool = String(v.tool ?? v.name ?? '').trim();
		const argsRaw = (v.arguments ?? v.args ?? {}) as Record<string, unknown>;
		if (!tool) return null;
		return { tool, args: argsRaw && typeof argsRaw === 'object' ? argsRaw : {} };
	} catch {
		return { tool: src, args: {} };
	}
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
				specYamlContent: fresh.content,
				assistantPrototypeHtml: lastAssistantHtmlFence(fresh.messages),
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
		injectToolResult,
		openDrawer() {
			commit(state => ({
				...state,
				drawerOpen: true,
			}));
		},
		open(input: OpenSlotSessionInput) {
			const key = sessionKey(input.spec.path, input.slot.id);
			commit(state => {
				const existing = state.sessions[key];
				const seededPlan =
					existing?.strongValidationPlan ?? seedStrongValidationPlan(input.spec.path, input.slot.id);
				const session: SpecSlotSession = existing
					? {
							...existing,
							spec: input.spec,
							slot: input.slot,
							content: input.content,
							a2uiRevision: existing.a2uiRevision ?? 0,
							a2uiStatus: existing.a2uiStatus ?? 'idle',
							iterationNodes: existing.iterationNodes ?? [],
							iterationEdges: existing.iterationEdges ?? [],
							iterationCursorId: existing.iterationCursorId ?? null,
							chatPrefill: existing.chatPrefill ?? null,
							strongValidationPlan: seededPlan,
							verificationSubmission: existing.verificationSubmission ?? null,
							validationRuns: existing.validationRuns ?? {},
							traceEvents: existing.traceEvents ?? [],
							pptDemoPath: existing.pptDemoPath ?? derivePptPathFromSession(existing),
							pptDemoHtml: existing.pptDemoHtml ?? null,
							pptDemoLoading: existing.pptDemoLoading ?? false,
							pptDemoError: existing.pptDemoError ?? null,
							lastRepairDecision: existing.lastRepairDecision ?? null,
							runningUserPrompt: existing.runningUserPrompt ?? null,
							pendingResumePrompt: existing.pendingResumePrompt ?? null,
							prototypeAgentUi: existing.prototypeAgentUi ?? null,
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
							iterationNodes: [],
							iterationEdges: [],
							iterationCursorId: null,
							chatPrefill: null,
							strongValidationPlan: seededPlan,
							verificationSubmission: null,
							validationRuns: {},
							traceEvents: [],
							pptDemoPath: derivePptPath(input.spec.path, input.content),
							pptDemoHtml: null,
							pptDemoLoading: false,
							pptDemoError: null,
							lastRepairDecision: null,
							runningUserPrompt: null,
							pendingResumePrompt: null,
							prototypeAgentUi: null,
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
		submitVerificationOutcome(outcome: VerificationSubmission['outcome'], notes?: string) {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session?.strongValidationPlan) return state;
				const plan = session.strongValidationPlan;
				const sub: VerificationSubmission = {
					submission_id: id(),
					plan_id: plan.plan_id,
					slot_id: session.slot.id,
					spec_path: session.spec.path,
					outcome,
					artifact_refs: Object.values(session.validationRuns ?? {})
						.filter(r => !!r.output)
						.map(r => `${r.item_id}:${(r.output || '').slice(0, 120)}`),
					notes: notes?.trim() || undefined,
				};
				const msg: SpecSlotMessage = {
					id: id(),
					role: 'system',
					content: `[强校验提交] outcome=${outcome} plan=${plan.plan_id} submission=${sub.submission_id}${notes ? `\n${notes}` : ''}`,
					createdAt: nowIso(),
				};
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...session,
							verificationSubmission: sub,
							messages: [...session.messages, msg],
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		/** 将默认校验命令预填到左侧对话（由用户在宿主环境执行） */
		prefillValidationRun(itemId?: string) {
			const state = get({ subscribe });
			const key = state.activeKey;
			const session = key ? state.sessions[key] : null;
			if (!session?.strongValidationPlan?.items.length) return;
			const items = session.strongValidationPlan.items;
			const item = itemId ? items.find(i => i.id === itemId) : items[0];
			if (!item?.command) return;
			const cwd = get(specExplorerStore).workspaceRoot || '.';
			const text = [
				`[强校验 · 运行] 请在仓库根（cwd=${cwd}）执行：`,
				item.command,
				item.timeout_sec ? `(建议超时 ${item.timeout_sec}s)` : '',
				'执行后将输出摘要回复；若通过请点击右侧「标记 GREEN」。',
			]
				.filter(Boolean)
				.join('\n');
			const t = text.trim();
			if (!t) return;
			commit(state => {
				const k = state.activeKey;
				if (!k) return state;
				const sess = state.sessions[k];
				if (!sess) return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[k]: {
							...sess,
							chatPrefill: { id: id(), text: t },
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		recordTraceEvent(trace: OrchestrationTracePayload) {
			const key = get({ subscribe }).activeKey;
			if (!key) return;
			appendTraceEvent(key, trace);
		},
		async refreshPptDemoActive() {
			const state = get({ subscribe });
			const key = state.activeKey;
			const session = key ? state.sessions[key] : null;
			if (!key || !session) return;
			const path = session.pptDemoPath || derivePptPathFromSession(session);
			commit(s => {
				const active = s.sessions[key];
				if (!active) return s;
				return {
					...s,
					sessions: {
						...s.sessions,
						[key]: { ...active, pptDemoPath: path, pptDemoError: null, updatedAt: nowIso() },
					},
				};
			});
			try {
				const q = new URLSearchParams({ path });
				const wsRoot = get(specExplorerStore).workspaceRoot;
				if (wsRoot) q.set('workspaceRoot', wsRoot);
				const r = await fetch(`${agentApiUrl('/api/workspace/specs/read')}?${q.toString()}`);
				if (!r.ok) throw new Error(await r.text());
				const j = (await r.json()) as { content?: string };
				commit(s => {
					const active = s.sessions[key];
					if (!active) return s;
					return {
						...s,
						sessions: {
							...s.sessions,
							[key]: {
								...active,
								pptDemoPath: path,
								pptDemoHtml: String(j.content ?? ''),
								pptDemoLoading: false,
								pptDemoError: null,
								updatedAt: nowIso(),
							},
						},
					};
				});
			} catch (e) {
				commit(s => {
					const active = s.sessions[key];
					if (!active) return s;
					return {
						...s,
						sessions: {
							...s.sessions,
							[key]: {
								...active,
								pptDemoPath: path,
								pptDemoHtml: null,
								pptDemoLoading: false,
								pptDemoError: e instanceof Error ? e.message : String(e),
								updatedAt: nowIso(),
							},
						},
					};
				});
			}
		},
		async generatePptDemoActive() {
			const state = get({ subscribe });
			const key = state.activeKey;
			const session = key ? state.sessions[key] : null;
			if (!key || !session) return;
			const wsRoot = get(specExplorerStore).workspaceRoot;
			const path = session.pptDemoPath || derivePptPathFromSession(session);
			commit(s => {
				const active = s.sessions[key];
				if (!active) return s;
				return {
					...s,
					sessions: {
						...s.sessions,
						[key]: {
							...active,
							pptDemoPath: path,
							pptDemoLoading: true,
							pptDemoError: null,
							updatedAt: nowIso(),
						},
					},
				};
			});
			const prompt = [
				'请加载并使用 html-ppt 技能，为当前 spec 生成一份可演示的单文件 HTML 说明稿。',
				`spec_path=${session.spec.path}`,
				`output_path=${path}`,
				'要求：内容覆盖目标、模块、关键流程、验证要点；深色主题；支持键盘翻页；必须使用 write_file 实际写入 output_path。',
				`spec_yaml:\n${session.content.slice(0, 12000)}`,
			].join('\n\n');
			try {
				await postAgentChat({
					threadId: session.threadId,
					input: prompt,
					workspaceRoot: wsRoot,
					agent_profile: 'ppt-generator',
				});
				for (let i = 0; i < 30; i++) {
					await delay(2000);
					const q = new URLSearchParams({ path });
					if (wsRoot) q.set('workspaceRoot', wsRoot);
					const r = await fetch(`${agentApiUrl('/api/workspace/specs/read')}?${q.toString()}`);
					if (!r.ok) continue;
					const j = (await r.json()) as { content?: string };
					const html = String(j.content ?? '');
					if (!html.trim()) continue;
					let nextContent = session.content;
					try {
						const patched = upsertSpecBlockPptFile(session.content, path);
						const wr = await fetch(agentApiUrl('/api/workspace/specs/write'), {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({
								workspace_root: wsRoot,
								path: session.spec.path,
								content: patched,
							}),
						});
						if (wr.ok) nextContent = patched;
					} catch {
						// don't block preview rendering on spec write failure
					}
					commit(s => {
						const active = s.sessions[key];
						if (!active) return s;
						return {
							...s,
							sessions: {
								...s.sessions,
								[key]: {
									...active,
									content: nextContent,
									pptDemoHtml: html,
									pptDemoLoading: false,
									pptDemoError: null,
									updatedAt: nowIso(),
								},
							},
						};
					});
					return;
				}
				throw new Error('生成超时：未检测到 HTML 文件写入');
			} catch (e) {
				commit(s => {
					const active = s.sessions[key];
					if (!active) return s;
					return {
						...s,
						sessions: {
							...s.sessions,
							[key]: {
								...active,
								pptDemoLoading: false,
								pptDemoError: e instanceof Error ? e.message : String(e),
								updatedAt: nowIso(),
							},
						},
					};
				});
			}
		},
		async runValidationItem(itemId: string) {
			const state = get({ subscribe });
			const key = state.activeKey;
			const session = key ? state.sessions[key] : null;
			const item = session?.strongValidationPlan?.items.find(i => i.id === itemId);
			if (!key || !session || !item) return;
			const started = nowIso();
			const wsRoot = get(specExplorerStore).workspaceRoot;
			const parsedTemplate = parseValidationTemplate(item.tool_call_template);
			const templateTool = String(parsedTemplate?.tool ?? '').trim();
			const templateArgs = parsedTemplate?.args ?? {};
			const tmpl = templateTool.toLowerCase();
			const command = String(item.command ?? '').trim().toLowerCase();
			const channel: ValidationItemRunResult['channel'] = tmpl.includes('cdp_validate')
				? 'cdp_validate'
				: tmpl.includes('qa') || command.includes('playwright')
					? 'qa_playwright'
					: tmpl.includes('verify_specs') || command.includes('verify_specs')
						? 'verify_specs'
						: tmpl.includes('tdd_run')
							? 'tdd_run'
							: command.includes('make validate') || tmpl.includes('make_validate')
								? 'make_validate'
								: 'unknown';
			const beginTrace: OrchestrationTracePayload = {
				phase: 'strong_validation_plan',
				node: { node_id: `validation:${item.id}`, kind: channel, label: item.label || item.id },
				run_id: session.threadId,
				timestamp_unix: Math.floor(Date.now() / 1000),
				outcome_summary: 'run_clicked',
			};
			appendTraceEvent(key, beginTrace);
			void emitInspectorTrace(beginTrace);
			let result: ValidationItemRunResult = {
				item_id: item.id,
				ok: false,
				channel,
				started_at: started,
				finished_at: nowIso(),
				error: 'unknown validation channel',
			};
			try {
				if (channel === 'qa_playwright') {
					// 优先联动“用户工作区自定义 agent 流程”验证；
					// 若未配置则回退到 legacy qa/run。
					const flowPath = String(templateArgs.flow_path ?? '.vibex/agents/flows/qa-agent-flow.json');
					const flowResp = await fetch(agentApiUrl('/api/workspace/agent-flow-qa/run'), {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ workspace_root: wsRoot, flow_path: flowPath }),
					});
					const flowData = (await flowResp.json()) as Record<string, unknown>;
					const customFound = Boolean(flowData.custom_agent_found);
					if (customFound) {
						const startupOk = Boolean(flowData.startup_ok);
						const testOk = Boolean(flowData.test_ok);
						const returnOk = Boolean(flowData.return_ok);
						const flowOk = Boolean(flowData.ok);
						result = {
							item_id: item.id,
							ok: flowOk && startupOk && testOk && returnOk,
							channel,
							source: 'custom_agent_flow',
							started_at: started,
							finished_at: nowIso(),
							output: [
								`custom_agent_found=${customFound}`,
								`startup_ok=${startupOk}`,
								`test_ok=${testOk}`,
								`return_ok=${returnOk}`,
								typeof flowData.error === 'string' && flowData.error ? `error=${flowData.error}` : '',
							]
								.filter(Boolean)
								.join('\n'),
							error:
								flowOk && startupOk && testOk && returnOk
									? undefined
									: String(flowData.error ?? 'custom agent flow qa failed'),
						};
					} else {
						const scenario = String(templateArgs.scenario ?? 'spec-verify');
						const tags = String(templateArgs.tags ?? '');
						const r = await fetch(agentApiUrl('/api/workspace/qa/run'), {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({ workspace_root: wsRoot, scenario, tags }),
						});
						const data = (await r.json()) as Record<string, unknown>;
						result = {
							item_id: item.id,
							ok: Boolean(data.ok ?? data.passed),
							channel,
							source: 'legacy_qa',
							started_at: started,
							finished_at: nowIso(),
							output: String(data.output ?? ''),
							exit_code: Number(data.exit_code ?? 0),
							error: data.ok ? undefined : String(data.error ?? ''),
						};
					}
				} else if (channel === 'verify_specs') {
					const format = String(templateArgs.format ?? 'summary');
					const checks = String(templateArgs.checks ?? '');
					const levels = String(templateArgs.levels ?? '');
					const r = await fetch(agentApiUrl('/api/workspace/verify-specs'), {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ workspace_root: wsRoot, format, checks, levels }),
					});
					const txt = await r.text();
					result = {
						item_id: item.id,
						ok: r.ok,
						channel,
						started_at: started,
						finished_at: nowIso(),
						output: txt.slice(0, 2000),
						error: r.ok ? undefined : txt.slice(0, 300),
					};
				} else if (channel === 'make_validate') {
					const target = String(templateArgs.target ?? 'validate');
					const r = await fetch(agentApiUrl('/api/workspace/run-make'), {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ workspace_root: wsRoot, target }),
					});
					const data = (await r.json()) as Record<string, unknown>;
					result = {
						item_id: item.id,
						ok: Boolean(data.ok),
						channel,
						started_at: started,
						finished_at: nowIso(),
						output: String(data.output ?? '').slice(0, 2000),
						exit_code: Number(data.exitCode ?? 0),
					};
				} else if (channel === 'tdd_run') {
					await postAgentChat({
						threadId: session.threadId,
						input: `请使用 tdd_run 运行当前槽位对应 spec 的测试并返回 RED/GREEN 摘要。spec_path=${session.spec.path}`,
						workspaceRoot: wsRoot,
					});
					result = {
						item_id: item.id,
						ok: true,
						channel,
						started_at: started,
						finished_at: nowIso(),
						output: '已触发 tdd_run（结果见会话流）',
					};
				} else if (channel === 'cdp_validate') {
					const defaults = {
						plan_id: `${session.slot.id}:${item.id}:${Date.now()}`,
						target_env: { deployment: 'user_managed', host: '127.0.0.1', port: 9222, timeout_sec: 30 },
						entry_url: 'http://localhost:5173/workbench',
						steps: [{ id: item.id, assertions: [{ id: 'a1', type: 'text_contains', selector: 'body', value: 'Spec' }] }],
						screenshot_on_fail: true,
					};
					const cdpPlan = {
						...defaults,
						...(templateArgs as Record<string, unknown>),
						target_env: {
							...defaults.target_env,
							...((templateArgs.target_env as Record<string, unknown> | undefined) ?? {}),
						},
					};
					const r = await fetch(agentApiUrl('/api/workspace/cdp/validate'), {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							workspace_root: wsRoot,
							...cdpPlan,
						}),
					});
					const data = (await r.json()) as Record<string, unknown>;
					result = {
						item_id: item.id,
						ok: Boolean(data.ok),
						channel,
						started_at: started,
						finished_at: nowIso(),
						output: `${String(data.error ?? '')}\n${String((data.logs as unknown[] | undefined)?.join('\n') ?? '')}`.trim(),
						error: data.ok ? undefined : String(data.error ?? 'cdp_validate failed'),
					};
				}
			} catch (e) {
				result = {
					item_id: item.id,
					ok: false,
					channel,
					started_at: started,
					finished_at: nowIso(),
					error: e instanceof Error ? e.message : String(e),
				};
			}
			commit(s => {
				const active = s.sessions[key];
				if (!active) return s;
				return {
					...s,
					sessions: {
						...s.sessions,
						[key]: {
							...active,
							validationRuns: { ...(active.validationRuns ?? {}), [item.id]: result },
							updatedAt: nowIso(),
						},
					},
				};
			});
			const doneTrace: OrchestrationTracePayload = {
				phase: 'user_verification',
				node: { node_id: `validation:${item.id}`, kind: channel, label: item.label || item.id },
				run_id: session.threadId,
				timestamp_unix: Math.floor(Date.now() / 1000),
				outcome_summary: result.ok ? 'ok' : result.error || 'failed',
			};
			appendTraceEvent(key, doneTrace);
			void emitInspectorTrace(doneTrace);
		},
		clearPrototypeAgentUiActive() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const session = state.sessions[key];
				if (!session) return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: { ...session, prototypeAgentUi: null, updatedAt: nowIso() },
					},
				};
			});
		},
		applyPrototypeBridgeMessage(origin: string, data: unknown) {
			if (typeof window !== 'undefined' && !isTrustedPrototypeBridgeOrigin(origin)) return;
			const rec = data as Record<string, unknown>;
			if (rec?.source !== 'vibex-prototype-bridge') return;
			const kind = String(rec.kind ?? '');
			const st = get({ subscribe });
			const key = st.activeKey;
			const session = key ? st.sessions[key] : null;
			if (!key || !session || session.slot.id !== 'prototype') return;

			commit(state => {
				const cur = state.sessions[key];
				if (!cur || cur.slot.id !== 'prototype') return state;
				if (kind === 'ready') return state;

				if (kind === 'clear' || kind === 'onboard-end') {
					return {
						...state,
						sessions: {
							...state.sessions,
							[key]: { ...cur, prototypeAgentUi: null, updatedAt: nowIso() },
						},
					};
				}

				if (kind === 'highlight-rects' || kind === 'onboard-step') {
					const rects = normalizePrototypeBridgeRects(rec.rects);
					const viewport = parsePrototypeBridgeViewport(rec.viewport);
					const onboard = parsePrototypeBridgeOnboard(rec.onboard);
					return {
						...state,
						sessions: {
							...state.sessions,
							[key]: {
								...cur,
								prototypeAgentUi: {
									rects,
									viewport,
									onboard,
									updatedAt: nowIso(),
								},
								updatedAt: nowIso(),
							},
						},
					};
				}

				return state;
			});
		},
		/** 设置当前会话绑定的原型 HTML 路径（用于 SpecPilot 原型预览） */
		setPrototypePath(path: string) {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const sess = state.sessions[key];
				if (!sess) return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: { ...sess, pptDemoPath: path, updatedAt: nowIso() },
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
			abortActiveChatRequest();
			commit(state => ({ ...state, drawerOpen: false }));
		},
		resetActive() {
			closeStream();
			abortActiveChatRequest();
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
		abortActiveAgent() {
			const state = get({ subscribe });
			const key = state.activeKey;
			if (!key) return;
			const sess = state.sessions[key];
			if (!sess || sess.status !== 'running') return;
			const snap = sess.runningUserPrompt?.trim() ?? '';
			abortActiveChatRequest();
			closeStream();
			commit(s => {
				const cur = s.sessions[key];
				if (!cur || cur.status !== 'running') return s;
				let messages = [...cur.messages];
				const last = messages[messages.length - 1];
				if (last?.role === 'assistant') {
					if (!last.content.trim()) messages.pop();
					else {
						messages[messages.length - 1] = {
							...last,
							content: `${last.content}\n\n---\n*(输出已中断)*`,
							createdAt: nowIso(),
						};
					}
				}
				const sys: SpecSlotMessage = {
					id: id(),
					role: 'system',
					content:
						'[Agent 已中止] 前端已断开 SSE；若请求已进入队列，Agent 仍可能在后台执行直至结束。',
					createdAt: nowIso(),
				};
				return {
					...s,
					sessions: {
						...s.sessions,
						[key]: {
							...cur,
							messages: [...messages, sys],
							status: 'idle',
							error: null,
							pendingResumePrompt: snap || cur.pendingResumePrompt,
							runningUserPrompt: null,
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		revertActiveAgentTurn() {
			const state = get({ subscribe });
			const key = state.activeKey;
			if (!key) return;
			const sess = state.sessions[key];
			if (!sess) return;
			if (sess.status === 'running') {
				abortActiveChatRequest();
				closeStream();
			}
			const idx = findLastUserMessageIndex(sess.messages);
			if (idx < 0) return;
			const userText = sess.messages[idx]?.content?.trim() ?? '';
			commit(s => {
				const cur = s.sessions[key];
				if (!cur) return s;
				const messages = cur.messages.slice(0, idx);
				return {
					...s,
					sessions: {
						...s.sessions,
						[key]: {
							...cur,
							messages,
							status: 'idle',
							error: null,
							iterationNodes: [],
							iterationEdges: [],
							iterationCursorId: null,
							pendingResumePrompt: userText || cur.pendingResumePrompt,
							runningUserPrompt: null,
							updatedAt: nowIso(),
						},
					},
				};
			});
		},
		resumeInterruptedAgentTurn() {
			commit(state => {
				const key = state.activeKey;
				if (!key) return state;
				const cur = state.sessions[key];
				if (!cur) return state;
				const t = cur.pendingResumePrompt?.trim();
				if (!t || cur.status === 'running') return state;
				return {
					...state,
					sessions: {
						...state.sessions,
						[key]: {
							...cur,
							chatPrefill: { id: id(), text: t },
							pendingResumePrompt: null,
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
							runningUserPrompt: text,
							pendingResumePrompt: null,
							iterationNodes: [],
							iterationEdges: [],
							iterationCursorId: null,
							updatedAt: nowIso(),
						},
					},
				};
			});

			closeStream();
			abortActiveChatRequest();
			activeChatAbort = new AbortController();
			const chatAbortSignal = activeChatAbort.signal;
			const streamPath = useMockBackend
				? agentApiUrl(`/api/sse/threads/${session.threadId}`)
				: agentApiUrl(`/api/sse/${session.threadId}`);
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
				upsertIterationNode(
					key,
					{
						id: `result:completed:${Date.now()}`,
						label: 'run.completed',
						type: 'result',
						status: 'done',
					},
					{ moveCursor: true, edgeLabelFromPrev: 'done' }
				);
				closeStream();
				commit(current => {
					const active = current.sessions[key];
					if (!active) return current;
					return {
						...current,
						sessions: {
							...current.sessions,
							[key]: { ...active, status: 'idle', runningUserPrompt: null, pendingResumePrompt: null, updatedAt: nowIso() },
						},
					};
				});
				void reloadSessionSpecAfterTurn(key);
			});
			activeSource.addEventListener('run.failed', () => {
				upsertIterationNode(
					key,
					{
						id: `result:failed:${Date.now()}`,
						label: 'run.failed',
						type: 'result',
						status: 'error',
					},
					{ moveCursor: true, edgeLabelFromPrev: 'failed' }
				);
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
								error: '运行失败',
								runningUserPrompt: null,
								pendingResumePrompt: active.runningUserPrompt?.trim() || active.pendingResumePrompt,
								updatedAt: nowIso(),
							},
						},
					};
				});
			});
			activeSource.addEventListener('run.started', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const runId = String(data.runId ?? data.run_id ?? `run:${Date.now()}`);
					upsertIterationNode(
						key,
						{ id: `run:${runId}`, label: 'run.started', type: 'run', status: 'running' },
						{ moveCursor: true, edgeLabelFromPrev: 'start' }
					);
				} catch {
					// ignore
				}
			});
			activeSource.addEventListener('run.planning', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const runId = String(data.runId ?? data.run_id ?? 'current');
					upsertIterationNode(
						key,
						{
							id: `plan:${runId}:${Date.now()}`,
							label: 'planning',
							type: 'plan',
							status: 'running',
							detail: String(data.status ?? ''),
						},
						{ moveCursor: true, edgeLabelFromPrev: 'plan' }
					);
				} catch {
					// ignore
				}
			});
			activeSource.addEventListener('tool.called', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const callId = String(data.call_id ?? data.invocationId ?? `tool:${Date.now()}`);
					const tool = String(data.tool ?? data.toolName ?? 'tool');
					const args = data.args;
					upsertIterationNode(
						key,
						{
							id: `tool:${callId}`,
							label: tool,
							type: 'tool',
							status: 'running',
						},
						{ moveCursor: true, edgeLabelFromPrev: 'call' }
					);
					const openPath = extractToolWorkspacePath(tool, args);
					appendToolCallSessionMessage(key, {
						id: `tool-inline:${callId}`,
						role: 'tool',
						content: formatToolCalledBody(tool, args),
						createdAt: nowIso(),
						toolOpenPath: openPath,
					});
				} catch {
					// ignore
				}
			});
			activeSource.addEventListener('tool.completed', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const callId = String(data.call_id ?? data.invocationId ?? '');
					const tool = String(data.tool ?? data.toolName ?? 'tool');
					const result = String(data.result ?? '');
					const failed = /^error:|^blocked:|^filter_rejected:/i.test(result.trim());
					upsertIterationNode(
						key,
						{
							id: `tool:${callId || Date.now().toString()}`,
							label: tool,
							type: 'tool',
							status: failed ? 'error' : 'done',
							detail: result.slice(0, 140),
						},
						{ moveCursor: true, edgeLabelFromPrev: failed ? 'error' : 'ok' }
					);
					if (callId) {
						const msgId = `tool-inline:${callId}`;
						const prev =
							get({ subscribe }).sessions[key]?.messages.find(m => m.id === msgId)?.content ?? '';
						patchSessionMessage(key, msgId, {
							content: prev + formatToolCompletedFooter(result, failed),
						});
					}
				} catch {
					// ignore
				}
			});
			activeSource.addEventListener('tool.failed', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const callId = String(data.call_id ?? data.invocationId ?? '');
					const tool = String(data.tool ?? data.toolName ?? 'tool');
					const err = String(data.error ?? '');
					if (callId) {
						upsertIterationNode(
							key,
							{
								id: `tool:${callId}`,
								label: tool,
								type: 'tool',
								status: 'error',
								detail: err.slice(0, 140),
							},
							{ moveCursor: true, edgeLabelFromPrev: 'fail' }
						);
						const msgId = `tool-inline:${callId}`;
						const prev =
							get({ subscribe }).sessions[key]?.messages.find(m => m.id === msgId)?.content ?? '';
						patchSessionMessage(key, msgId, {
							content: prev + formatToolCompletedFooter(err, true),
						});
					}
				} catch {
					// ignore
				}
			});
			activeSource.addEventListener('repair.decision', event => {
				try {
					const data = JSON.parse(String((event as MessageEvent).data)) as Record<string, unknown>;
					const decision = data.decision as RepairDecisionPayload | undefined;
					const envelope = data.envelope as { failure_type?: string; error_message?: string } | undefined;
					if (decision?.reason_code) {
						commit(current => {
							const active = current.sessions[key];
							if (!active) return current;
							return {
								...current,
								sessions: {
									...current.sessions,
									[key]: {
										...active,
										lastRepairDecision: decision,
										updatedAt: nowIso(),
									},
								},
							};
						});
						const detail = [
							decision.reason_code,
							decision.target_phase,
							envelope?.failure_type,
							envelope?.error_message,
						]
							.filter(Boolean)
							.join(' · ');
						upsertIterationNode(
							key,
							{
								id: `repair:${decision.reason_code}:${Date.now()}`,
								label: 'repair.decision',
								type: 'plan',
								status: decision.allowed ? 'done' : 'error',
								detail: detail.slice(0, 200),
							},
							{ moveCursor: true, edgeLabelFromPrev: 'repair' }
						);
					}
				} catch {
					// ignore
				}
			});

			try {
				const fresh = get({ subscribe }).sessions[key] ?? session;
				const { agentProfile, userTaskLine } = parseLeadingAgentSlashCommand(text);
				const prompt = buildPrompt(fresh, userTaskLine);
				const wsRoot = get(specExplorerStore).workspaceRoot;
				const chatBody: Record<string, string> = {
					threadId: session.threadId,
					input: prompt,
					...(wsRoot ? { workspaceRoot: wsRoot } : {}),
				};
				if (agentProfile) {
					chatBody.agent_profile = agentProfile;
				}
				if (useMockBackend) {
					await fetch(agentApiUrl('/api/runs'), {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ threadId: session.threadId, goal: prompt }),
						signal: chatAbortSignal,
					});
				} else {
					await postAgentChat(chatBody, { signal: chatAbortSignal });
				}
			} catch (e) {
				closeStream();
				abortActiveChatRequest();
				const aborted =
					(e instanceof DOMException && e.name === 'AbortError') ||
					(e instanceof Error && e.name === 'AbortError');
				const errMsg = e instanceof Error ? e.message : String(e);
				commit(current => {
					const active = current.sessions[key];
					if (!active) return current;
					if (aborted) {
						const snap = active.runningUserPrompt?.trim() ?? '';
						let messages = [...active.messages];
						const last = messages[messages.length - 1];
						if (last?.role === 'assistant') {
							if (!last.content.trim()) messages.pop();
							else {
								messages[messages.length - 1] = {
									...last,
									content: `${last.content}\n\n---\n*(请求已取消)*`,
									createdAt: nowIso(),
								};
							}
						}
						messages.push({
							id: id(),
							role: 'system',
							content: '[请求已取消] 发起对话的 HTTP 请求被中止。',
							createdAt: nowIso(),
						});
						return {
							...current,
							sessions: {
								...current.sessions,
								[key]: {
									...active,
									messages,
									status: 'idle',
									error: null,
									pendingResumePrompt: snap || active.pendingResumePrompt,
									runningUserPrompt: null,
									updatedAt: nowIso(),
								},
							},
						};
					}
					return {
						...current,
						sessions: {
							...current.sessions,
							[key]: {
								...active,
								status: 'error',
								error: errMsg,
								runningUserPrompt: null,
								pendingResumePrompt:
									active.runningUserPrompt?.trim() || active.pendingResumePrompt,
								updatedAt: nowIso(),
							},
						},
					};
				});
			} finally {
				activeChatAbort = null;
			}
		},
	};
}

export const specSlotSessionStore = createSpecSlotSessionStore();

export function activeSpecSlotSession(state: SpecSlotSessionState): SpecSlotSession | null {
	return state.activeKey ? state.sessions[state.activeKey] ?? null : null;
}
