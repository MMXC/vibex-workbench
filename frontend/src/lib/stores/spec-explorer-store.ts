import { writable, derived } from 'svelte/store';

/** 与 Cursor/R2 对齐：左侧活动栏视图（资源管理器 / Git / 搜索 / 扩展） */
export type LeftActivity = 'explorer' | 'git' | 'search' | 'extensions';

/** 规格主区：图谱 | 文本 */
export type SpecCenterView = 'graph' | 'text';

export type SpecExplorerState = {
	leftActivity: LeftActivity;
	selectedSpecPath: string | null;
	centerView: SpecCenterView;
	/** 当前 workspace 根路径，切换时驱动 SpecExplorer 刷新文件树 */
	workspaceRoot: string;
};

const initial: SpecExplorerState = {
	leftActivity: 'explorer',
	selectedSpecPath: null,
	centerView: 'graph',
	workspaceRoot: '',
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

		/** 切换 workspace 根路径，同时清空选中状态并触发文件树刷新 */
		setWorkspaceRoot(root: string) {
			update(s => ({
				...s,
				workspaceRoot: root,
				selectedSpecPath: null,
			}));
		},
	};
}

export const specExplorerStore = createSpecExplorerStore();

/** 派生：workspace 根目录显示名（取路径最后一段） */
export const workspaceDisplayName = derived(
	specExplorerStore,
	$s => $s.workspaceRoot.split('/').pop() || $s.workspaceRoot || '—'
);
