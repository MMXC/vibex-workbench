import { writable, get } from "svelte/store";

export type NodeType = "input" | "sequence" | "iteration" | "branch" | "subagent" | "gate" | "output";
export type NodeStatus = "pending" | "running" | "done" | "error";

export interface CanvasNode {
	id: string;
	eventId: string;
	threadId: string;
	nodeType: NodeType;
	label: string;
	x: number;
	y: number;
	status: NodeStatus;
	color: string;
	payload: Record<string, unknown>;
}

export interface CanvasConnection {
	id: string;
	fromNodeId: string;
	toNodeId: string;
	type: "sequence" | "iteration-loop";
}

const NODE_COLORS: Record<string, string> = {
	input: "#4D96FF",
	sequence: "#6BCB77",
	iteration: "#FFD93D",
	branch: "#FFA94D",
	subagent: "#B197FC",
	gate: "#FF6B6B",
	output: "#74C0FC",
};

function createCanvasStore() {
	let nodeCounter = 0;
	const { subscribe, set, update } = writable<{
		nodes: CanvasNode[];
		connections: CanvasConnection[];
		activeThreadId: string | null;
	}>({ nodes: [], connections: [], activeThreadId: null });

	const addNode = (threadId: string, eventId: string, label: string, nodeType: NodeType, payload: Record<string, unknown> = {}): string => {
		const id = `node-${++nodeCounter}`;
		update(state => ({
			...state,
			nodes: [...state.nodes, {
				id,
				eventId,
				threadId,
				nodeType,
				label,
				x: 100 + (nodeCounter % 5) * 180,
				y: 80 + Math.floor(nodeCounter / 5) * 110,
				status: "done",
				color: NODE_COLORS[nodeType] || "#888888",
				payload,
			}]
		}));
		return id;
	};

	const addConnection = (fromNodeId: string, toNodeId: string, type: CanvasConnection["type"] = "sequence") => {
		update(state => ({
			...state,
			connections: [...state.connections, {
				id: `conn-${fromNodeId}-${toNodeId}`,
				fromNodeId,
				toNodeId,
				type,
			}]
		}));
	};

	return {
		subscribe,
		addNode,
		addConnection,
		switchThread: (threadId: string) => {
			nodeCounter = 0;
			update(state => {
				const threadNodeIds = state.nodes
					.filter(n => n.threadId === threadId)
					.map(n => n.id);
				return {
					nodes: state.nodes.filter(n => n.threadId === threadId),
					connections: state.connections.filter(c =>
						threadNodeIds.includes(c.fromNodeId) && threadNodeIds.includes(c.toNodeId)
					),
					activeThreadId: threadId,
				};
			});
		},
		onSSEEvent: (event: { event: string; threadId: string; payload: Record<string, unknown> }) => {
			const { event: eventType, threadId, payload } = event;
			const label = (payload.label as string) || eventType;
			const nodeType = (payload.nodeType as NodeType) || "sequence";

			const active = get({ subscribe }).activeThreadId;
			if (active && active !== threadId) return;

			const nodes = get({ subscribe }).nodes;
			const lastNodeId = nodes.length > 0 ? nodes[nodes.length - 1].id : null;
			const newNodeId = addNode(threadId, eventType, label, nodeType, payload);

			if (lastNodeId && threadId === get({ subscribe }).activeThreadId) {
				addConnection(lastNodeId, newNodeId);
			}
		},
		clear: () => set({ nodes: [], connections: [], activeThreadId: null }),
	};
}

export const canvasStore = createCanvasStore();

// Wire up event listeners
if (typeof window !== "undefined") {
	window.addEventListener("thread:switch", (e: Event) => {
		const detail = (e as CustomEvent).detail;
		canvasStore.switchThread(detail.threadId);
	});

	window.addEventListener("sse:event", (e: Event) => {
		const detail = (e as CustomEvent).detail;
		canvasStore.onSSEEvent(detail);
	});
}
