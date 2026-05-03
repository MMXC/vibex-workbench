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
}): Promise<PlanGraph> {
	const res = await fetch('/api/agent/plan-graph', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			goal: input.goal,
			spec_path: input.specPath,
			slot_id: input.slotId,
			mode: 'spec-slot-routing',
		}),
	});
	if (!res.ok) throw new Error(await res.text());
	const data = await res.json();
	return data.graph as PlanGraph;
}

export async function previewToolRoute(graph: PlanGraph): Promise<RoutePreview> {
	const res = await fetch('/api/agent/tool-route', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ graph }),
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
