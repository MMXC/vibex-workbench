import { writable, derived, get } from 'svelte/store';
import { wailsListSpecs, type WailsSpecFile } from '$lib/wails-filesystem';
import type { SpecDisplay } from '$lib/workbench/spec-display';

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
	/** 规格文件列表（从 store 加载） */
	specs: { path: string; level: number; name: string; status: string; display?: SpecDisplay }[];
	/** 列表加载中 */
	specsLoading: boolean;
	/** 列表加载错误 */
	specsError: string | null;
};

const initial: SpecExplorerState = {
	leftActivity: 'explorer',
	selectedSpecPath: null,
	centerView: 'graph',
	workspaceRoot: '',
	specs: [],
	specsLoading: false,
	specsError: null,
};

function createSpecExplorerStore() {
	const { subscribe, set, update } = writable<SpecExplorerState>(initial);

	/**
	 * 加载 spec 列表。
	 * 生产用 Wails binding（filesystem 直接读），开发用 HTTP fallback。
	 * 当 workspaceRoot 切换时由外部调用（见 SpecExplorer.svelte $effect）。
	 */
	async function loadList(workspaceRoot?: string): Promise<void> {
		const root = workspaceRoot ?? get({ subscribe }).workspaceRoot;
		if (!root) {
			update(s => ({ ...s, specs: [], specsError: null }));
			return;
		}
		update(s => ({ ...s, specsLoading: true, specsError: null }));
		try {
			const files: WailsSpecFile[] = await wailsListSpecs(root);
			update(s => ({
				...s,
				specs: files.map(f => ({
					path: f.path,
					level: f.level,
					name: f.name,
					status: f.status,
					display: f.display,
				})),
				specsLoading: false,
			}));
		} catch (e) {
			update(s => ({
				...s,
				specs: [],
				specsError: e instanceof Error ? e.message : String(e),
				specsLoading: false,
			}));
		}
	}

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
			// 切换后自动加载列表
			loadList(root);
		},

		/** 手动刷新 spec 列表 */
		loadList,
	};
}

export const specExplorerStore = createSpecExplorerStore();

/** 派生：workspace 根目录显示名（取路径最后一段） */
export const workspaceDisplayName = derived(
	specExplorerStore,
	$s => $s.workspaceRoot.split(/[\\/]/).pop() || $s.workspaceRoot || '—'
);
