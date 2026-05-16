export type PlanNode = {
	id: string;
	kind: string;
	title: string;
	description: string;
	tool?: string;
	inputs?: Record<string, unknown>;
	outputs?: string[];
	requires_confirmation: boolean;
};

export type PlanEdge = {
	from: string;
	to: string;
	reason?: string;
};

export type PlanGraph = {
	version: string;
	goal: string;
	spec_path?: string;
	slot_id?: string;
	mode: string;
	nodes: PlanNode[];
	edges: PlanEdge[];
};

export type RouteDecision = {
	node_id: string;
	node_kind: string;
	tool: string;
	tool_source: string;
	reason: string;
	can_execute: boolean;
	needs_confirm: boolean;
};

export type RoutePreview = {
	graph_version: string;
	decisions: RouteDecision[];
	warnings?: string[];
};

export type ToolDescriptor = {
	name: string;
	kind: string;
	description: string;
	source: string;
	schema?: Record<string, unknown>;
	permissions?: string[];
	ai_fill_area?: {
		prompt_template?: string;
		domain_rules?: string[];
		examples?: unknown[];
		personalization_notes?: string;
	};
};

export type FireworksGraph = {
	nodes: {
		id: string;
		label: string;
		type: 'plan' | 'tool' | 'gate';
		status: 'ready' | 'confirm' | 'missing';
		detail: string;
	}[];
	edges: { from: string; to: string; label: string }[];
};

export async function listTools(): Promise<ToolDescriptor[]> {
	const res = await fetch('/api/agent/tools');
	if (!res.ok) throw new Error(await res.text());
	const data = await res.json();
	return (data.tools ?? []) as ToolDescriptor[];
}

export async function createSlotPlanGraph(input: {
	goal: string;
	specPath: string;
	slotId: string;
	workspaceRoot?: string;
}): Promise<PlanGraph> {
	const res = await fetch('/api/agent/plan-graph', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			goal: input.goal,
			spec_path: input.specPath,
			slot_id: input.slotId,
			mode: 'spec-slot-routing',
			workspaceRoot: input.workspaceRoot,
		}),
	});
	if (!res.ok) throw new Error(await res.text());
	const data = await res.json();
	return data.graph as PlanGraph;
}

export async function previewToolRoute(graph: PlanGraph, workspaceRoot?: string): Promise<RoutePreview> {
	const res = await fetch('/api/agent/tool-route', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ graph, workspaceRoot }),
	});
	if (!res.ok) throw new Error(await res.text());
	const data = await res.json();
	return data.route as RoutePreview;
}

export function toFireworksGraph(graph: PlanGraph, route: RoutePreview): FireworksGraph {
	const decisionsByNode = new Map(route.decisions.map(decision => [decision.node_id, decision]));
	const nodes: FireworksGraph['nodes'] = [];
	const edges: FireworksGraph['edges'] = [];

	for (const node of graph.nodes) {
		const decision = decisionsByNode.get(node.id);
		nodes.push({
			id: node.id,
			label: node.title,
			type: node.kind === 'gate.confirm' ? 'gate' : 'plan',
			status: node.requires_confirmation || decision?.needs_confirm ? 'confirm' : decision?.tool ? 'ready' : 'missing',
			detail: node.description,
		});
		if (decision?.tool) {
			const toolNodeId = `tool:${decision.tool}`;
			if (!nodes.some(existing => existing.id === toolNodeId)) {
				nodes.push({
					id: toolNodeId,
					label: decision.tool,
					type: 'tool',
					status: decision.can_execute ? 'ready' : 'confirm',
					detail: decision.reason,
				});
			}
			edges.push({ from: node.id, to: toolNodeId, label: decision.tool_source || 'route' });
		}
	}

	for (const edge of graph.edges) {
		edges.push({ from: edge.from, to: edge.to, label: edge.reason ?? 'next' });
	}

	return { nodes, edges };
}

import { getPrototypeFileRel, isSafeWorkspaceRel } from '$lib/workbench/prototype-shell-manifest';
import {
	evaluatePrototypeGate,
	type UiWorkflowGateModel,
} from '$lib/workbench/ui-workflow-gate';

/** 与设计 spec 对齐：等价后端 a2ui_component_hint，当前由前端根据 slot + PlanGraph + Route 推导。 */
export type A2UICardEmphasis = 'confirm' | 'info' | 'warning';

export type A2UICard = {
	id: string;
	title: string;
	body: string;
	bullets?: string[];
	emphasis?: A2UICardEmphasis;
};

export type A2UIPrototypePanel = {
	mode: 'html_snippet' | 'placeholder' | 'workspace_file' | 'mf_remote';
	html?: string;
	/** 当 mode 为 workspace_file：相对工作区根的 HTML 路径（来自 spec prototype.file） */
	fileRel?: string;
	/** workspace_file 下助手 fenced HTML，与磁盘原型并存 */
	assistantDraftHtml?: string;
	caption?: string;
	/** mf_remote 模式：MF remote app 的完整 URL（含 query 参数） */
	mfRemoteUrl?: string;
	/** mf_remote 模式：加载的 MF 组件名 */
	mfComponent?: string;
};

export type A2UIStageLayout = 'cards' | 'fireworks' | 'split';

export type A2UIModel = {
	revision: number;
	slotId: string;
	headline: string;
	primaryStage: A2UIStageLayout;
	cards: A2UICard[];
	prototype?: A2UIPrototypePanel;
	/** prototype 槽位：Intent→UI Spec→Gate 可视化 */
	uiWorkflowGate?: UiWorkflowGateModel;
	showFireworks: boolean;
	routeSummaryLines: string[];
	componentHints: string[];
};

function escapeHtml(s: string): string {
	return s
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

/** Strip vendor reasoning blocks so ```html``` fences match reliably. */
function stripEmbeddedReasoningWrappers(text: string): string {
	return text.replace(/<think>[\s\S]*?<\/redacted_thinking>/gi, '').trim();
}

/**
 * Extract HTML document/snippet from the largest fenced ```html … ``` block in assistant markdown.
 */
export function extractHtmlFromMarkdown(markdown: string): string | null {
	const stripped = stripEmbeddedReasoningWrappers(markdown);
	const re = /```(?:html|htm)?\s*\n([\s\S]*?)```/gi;
	let best = '';
	let m: RegExpExecArray | null;
	while ((m = re.exec(stripped)) !== null) {
		const inner = (m[1] ?? '').trim();
		if (inner.length > best.length) best = inner;
	}
	return best.length > 0 ? best : null;
}

/** Walk assistant messages newest-first; return first fenced HTML found. */
export function lastAssistantHtmlFence(messages: { role: string; content: string }[]): string | null {
	for (let i = messages.length - 1; i >= 0; i--) {
		if (messages[i].role !== 'assistant') continue;
		const h = extractHtmlFromMarkdown(messages[i].content);
		if (h) return h;
	}
	return null;
}

function buildPrototypeShellHtml(specTitle: string, specPath: string, goal: string): string {
	const t = escapeHtml(specTitle);
	const p = escapeHtml(specPath);
	const g = escapeHtml(goal.slice(0, 280));
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><style>
body{font-family:system-ui,Segoe UI,sans-serif;background:#0f1419;color:#e2e8f0;margin:0;padding:16px;line-height:1.45}
h1{font-size:16px;margin:0 0 8px;color:#7aa2ff}
.meta{font-size:11px;color:#94a3b8;word-break:break-all}
.hint{margin-top:14px;padding:10px 12px;border:1px solid rgba(122,162,255,0.35);border-radius:10px;background:rgba(122,162,255,0.08);font-size:12px}
</style></head><body>
<h1>Prototype shell</h1>
<p class="meta">${t}</p>
<p class="meta">${p}</p>
<div class="hint">验证目标（占位）：${g || '在左侧对话中补齐原型要验证的用户路径与通过条件。'}</div>
<p class="meta" style="margin-top:12px">预览 iframe 已启用脚本（allow-scripts，本地工作台）。完整原型由对话/工具生成后可替换。</p>
</body></html>`;
}

export function toA2UIModel(params: {
	revision: number;
	slotId: string;
	slotLabel: string;
	specPath: string;
	specTitle: string;
	goal?: string;
	graph?: PlanGraph;
	route?: RoutePreview;
	/** 当前 spec YAML 全文（prototype 槽用于 Gate） */
	specYamlContent?: string;
	/** 最近一次助手消息中的 ```html```（优先于占位 shell） */
	assistantPrototypeHtml?: string | null;
}): A2UIModel {
	const { slotId, graph, route, revision } = params;
	const goal = (params.goal ?? graph?.goal ?? '').trim();
	const decisions = route?.decisions ?? [];
	const routeSummaryLines = decisions.map(
		d =>
			`${d.node_id} → ${d.tool || '(未命中)'} · ${d.node_kind}${d.needs_confirm ? ' · 需确认' : ''}`
	);
	const componentHints: string[] = [];
	for (const node of graph?.nodes ?? []) {
		if (node.outputs?.length) {
			componentHints.push(`${node.id}: outputs [${node.outputs.join(', ')}]`);
		}
	}

	const cards: A2UICard[] = [];
	switch (slotId) {
		case 'structure':
			cards.push({
				id: 'slot-focus',
				title: '结构与依赖',
				body: '确认 parent、children、dependencies、impacted_files 是否与当前层级一致。',
				bullets: ['层级与 parent chain 合法', '依赖可追踪', '影响文件列表可验证'],
				emphasis: 'info',
			});
			break;
		case 'constraints':
			cards.push({
				id: 'slot-focus',
				title: '约束槽位',
				body: '确认规则、验证方式与例外是否可执行、可测试。',
				bullets: ['规则可判定', '验证路径明确', '与 I/O 不冲突'],
				emphasis: 'info',
			});
			break;
		case 'input':
			cards.push({
				id: 'slot-focus',
				title: '输入契约',
				body: '明确触发条件、输入格式、上游边界与错误语义。',
				bullets: ['可构造最小样例', '边界条件列出', '与输出可配对验收'],
				emphasis: 'info',
			});
			break;
		case 'output':
			cards.push({
				id: 'slot-focus',
				title: '输出契约',
				body: '明确用户可见结果、产物路径、成功/失败信号。',
				bullets: ['可观测验收点', '与输入对齐', '可回归测试'],
				emphasis: 'info',
			});
			break;
		case 'prototype':
			cards.push({
				id: 'slot-focus',
				title: '原型验证',
				body: '用最小 HTML/交互验证关键路径；确认通过后再扩展实现。',
				bullets: [
					'验证目标一句话',
					'主路径与异常路径',
					'与 spec 槽位一致',
					'遵守工作区 .vibex/design/DESIGN.md 与 .vibex/prototypes/ 物料约定',
				],
				emphasis: 'confirm',
			});
			break;
		case 'implementation':
			cards.push({
				id: 'slot-focus',
				title: '实现路由',
				body: '把实现拆成可路由的工具/文件/验证步骤，避免直接堆代码。',
				bullets: ['工具命中可读', '确认门禁在前', '验证步骤可重复'],
				emphasis: 'warning',
			});
			break;
		default:
			cards.push({
				id: 'slot-focus',
				title: '槽位澄清',
				body: '根据当前槽位补齐可确认信息。',
				emphasis: 'info',
			});
	}

	for (const d of decisions) {
		if (!d.tool) continue;
		cards.push({
			id: `route-${d.node_id}`,
			title: d.tool,
			body: d.reason || d.node_kind,
			bullets: d.needs_confirm ? ['需要确认后再执行或写盘'] : undefined,
			emphasis: d.needs_confirm ? 'warning' : 'info',
		});
	}

	if (route?.warnings?.length) {
		cards.push({
			id: 'route-warnings',
			title: '路由告警',
			body: route.warnings.join('\n'),
			emphasis: 'warning',
		});
	}

/** 检测 spec YAML 是否声明了 MF 组件 */
function hasMFComponents(yamlContent: string): { has: boolean; component?: string; url?: string } {
	if (!yamlContent?.trim()) return { has: false };
	// 匹配 mf_components: [xxx] 或 mf_component: xxx 或 mf:
	const lines = yamlContent.split('\n');
	for (const line of lines) {
		const trimmed = line.trim();
		if (/^mf[_-]?component[s]?:/.test(trimmed)) {
			// 提取组件名
			const match = trimmed.match(/\[([^\]]+)\]/);
			if (match) {
				return { has: true, component: match[1].trim() };
			}
			const bare = trimmed.split(':')[1]?.trim().replace(/[\[\],]/g, '').trim();
			if (bare) return { has: true, component: bare };
		}
	}
	return { has: false };
}

function buildMFRemoteUrl(component: string): string {
	// MF dev server 默认端口 5177，/remote#/ComponentName
	return `http://localhost:5177/#/${component}`;
}

let prototype: A2UIPrototypePanel | undefined;
	let uiWorkflowGate: UiWorkflowGateModel | undefined;
	let showFireworks = true;
	let primaryStage: A2UIStageLayout = 'split';

	if (slotId === 'prototype') {
		const fromAssistant = params.assistantPrototypeHtml?.trim() || null;
		const shell = buildPrototypeShellHtml(params.specTitle, params.specPath, goal);

		let protoRel: string | null = null;
		if (typeof params.specYamlContent === 'string' && params.specYamlContent.trim()) {
			const rel = getPrototypeFileRel(params.specYamlContent);
			if (rel && isSafeWorkspaceRel(rel) && /\.html?$/i.test(rel)) {
				protoRel = rel;
			}
		}

		// 检测 MF 组件：spec prototype 槽或 YAML mf_component 声明
		const mf = hasMFComponents(params.specYamlContent ?? '');
		if (mf.has) {
			prototype = {
				mode: 'mf_remote',
				mfComponent: mf.component,
				mfRemoteUrl: buildMFRemoteUrl(mf.component!),
				caption: `MF 原型预览：${mf.component}（mfe_remote 模式；shared DataCenter @ localhost:7890）`,
			};
		} else if (protoRel) {
			prototype = {
				mode: 'workspace_file',
				fileRel: protoRel,
				assistantDraftHtml: fromAssistant ?? undefined,
				caption: fromAssistant
					? 'HTML 原型预览：主区为 prototype.file（工作区静态 HTML）；下方为助手 fenced HTML 草稿。'
					: 'HTML 原型预览（prototype.file；iframe 来自工作区 /api/workspace/file；sandbox：allow-scripts）。',
			};
		} else {
			prototype = {
				mode: 'html_snippet',
				html: fromAssistant || shell,
				caption: fromAssistant
					? 'HTML 原型预览（来自助手 fenced html；iframe sandbox：allow-scripts）。'
					: 'HTML prototype 预览（占位 shell；助手生成 fenced HTML 或将 prototype.file 指向 .html 后将自动替换）。',
			};
		}
		showFireworks = false;
		primaryStage = 'split';
		if (params.specYamlContent !== undefined) {
			uiWorkflowGate = evaluatePrototypeGate(params.specYamlContent);
			const failed = uiWorkflowGate.checks.filter(c => !c.passed);
			for (const c of failed.slice(0, 6)) {
				cards.push({
					id: `gate-${c.id}`,
					title: `Gate · ${c.label}`,
					body: c.detail || '请在本 spec 的 prototype.intent / prototype.ui_spec 中补齐 YAML。',
					emphasis: 'warning',
				});
			}
			if (!uiWorkflowGate.canCommitPrototype) {
				cards.push({
					id: 'gate-summary',
					title: 'Prototype Gate 未通过',
					body: `阶段：${uiWorkflowGate.stage}。提交物料写盘前请先补齐清单（仍可进行只读操作与对话预填）。`,
					emphasis: 'warning',
				});
			}
		}
	} else if (slotId === 'implementation') {
		showFireworks = true;
		primaryStage = 'fireworks';
	} else if (slotId === 'structure' || slotId === 'constraints') {
		primaryStage = 'cards';
		showFireworks = decisions.length > 0;
	} else {
		primaryStage = 'split';
		showFireworks = decisions.length > 0;
	}

	return {
		revision,
		slotId,
		headline: `${params.slotLabel} · ${params.specTitle}`,
		primaryStage,
		cards,
		prototype,
		uiWorkflowGate,
		showFireworks,
		routeSummaryLines,
		componentHints,
	};
}
