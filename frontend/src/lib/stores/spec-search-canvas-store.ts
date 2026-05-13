import { writable } from 'svelte/store';

/** 中央 Canvas 上展示的 agent 检索候选（与 run/tool 节点并存，id 前缀 speccand:） */
export type SpecSearchCanvasCandidate = {
	/** 在单次 resultSet 内唯一 */
	id: string;
	/** 工作区相对路径，如 .vibex/specs/L3/foo.yaml */
	path: string;
	title?: string;
	snippet?: string;
	/** 0–1 或 0–100，仅用于展示 */
	score?: number;
};

export type SpecSearchCanvasState = {
	query: string;
	resultSetId: string;
	candidates: SpecSearchCanvasCandidate[];
	updatedAt: string;
};

const empty: SpecSearchCanvasState = {
	query: '',
	resultSetId: '',
	candidates: [],
	updatedAt: '',
};

function createSpecSearchCanvasStore() {
	const { subscribe, set, update } = writable<SpecSearchCanvasState>(empty);

	return {
		subscribe,
		/**
		 * 右侧通用对话 / agent 工具在解析出检索结果后调用：
		 * 会触发 CanvasRenderer 将候选同步为 `type: specCandidate` 节点。
		 */
		setFromAgentSearch(query: string, candidates: SpecSearchCanvasCandidate[]) {
			set({
				query: query.trim(),
				resultSetId: crypto.randomUUID(),
				candidates,
				updatedAt: new Date().toISOString(),
			});
		},
		clear() {
			set({ ...empty, updatedAt: new Date().toISOString() });
		},
	};
}

export const specSearchCanvasStore = createSpecSearchCanvasStore();
