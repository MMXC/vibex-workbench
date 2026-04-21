import { writable } from 'svelte/store';

/** 与 Cursor/R2 对齐：左侧活动栏视图（资源管理器 / Git / 搜索 / 扩展） */
export type LeftActivity = 'explorer' | 'git' | 'search' | 'extensions';

/** 规格主区：图谱 | 文本 */
export type SpecCenterView = 'graph' | 'text';

export type SpecExplorerState = {
	leftActivity: LeftActivity;
	selectedSpecPath: string | null;
	centerView: SpecCenterView;
};

const initial: SpecExplorerState = {
	leftActivity: 'explorer',
	selectedSpecPath: null,
	centerView: 'graph',
};

function createSpecExplorerStore() {
	const { subscribe, set, update } = writable<SpecExplorerState>(initial);

	return {
		subscribe,
		reset() {
			set({ ...initial });
		},
		setLeftActivity(act: LeftActivity) {
			update(s => ({ ...s, leftActivity: act }));
		},
		selectSpec(path: string | null) {
			update(s => ({
				...s,
				selectedSpecPath: path,
				centerView: 'graph',
			}));
		},
		setCenterView(view: SpecCenterView) {
			update(s => ({ ...s, centerView: view }));
		},
	};
}

export const specExplorerStore = createSpecExplorerStore();
