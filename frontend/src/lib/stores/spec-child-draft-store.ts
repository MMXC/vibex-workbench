import { get, writable } from 'svelte/store';
import { specExplorerStore } from '$lib/stores/spec-explorer-store';
import {
	appendChildrenPaths,
	draftYamlForChildCandidate,
	type ChildSpawnCandidate,
	type SpawnLayer,
} from '$lib/workbench/spawn-child-specs';
import { wailsReadSpecFile, wailsWriteSpecFile } from '$lib/wails-filesystem';

export type SpecChildDraftSession = {
	parentSpecPath: string;
	parentYamlSnapshot: string;
	candidate: ChildSpawnCandidate;
	layer: SpawnLayer;
	draftYaml: string;
	relativePath: string;
	onDone?: () => void;
};

type State = {
	open: boolean;
	session: SpecChildDraftSession | null;
	busy: boolean;
	error: string | null;
	ok: string | null;
};

const initial: State = {
	open: false,
	session: null,
	busy: false,
	error: null,
	ok: null,
};

function toSpecsRelativePath(p: string): string {
	const norm = p.replace(/\\/g, '/');
	const i = norm.indexOf('specs/');
	return i >= 0 ? norm.slice(i) : norm;
}

function todayYmd(): string {
	const d = new Date();
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function createSpecChildDraftStore() {
	const state = writable<State>(initial);

	return {
		subscribe: state.subscribe,

		close() {
			state.set(initial);
		},

		setDraftYaml(next: string) {
			state.update(s => {
				if (!s.session) return s;
				return { ...s, session: { ...s.session, draftYaml: next }, error: null, ok: null };
			});
		},

		async openFromCandidate(input: {
			parentSpecPath: string;
			parentYaml: string;
			candidate: ChildSpawnCandidate;
			layer: SpawnLayer;
			onDone?: () => void;
		}) {
			const wsRoot = get(specExplorerStore).workspaceRoot;
			if (!wsRoot) throw new Error('未绑定 workspace root');

			const templateRel =
				input.layer === 'L2'
					? 'spec-templates/L3-module/L3-module-template.yaml'
					: 'spec-templates/L5-slice/L5-slice-template.yaml';
			const templateTxt = await wailsReadSpecFile(wsRoot, templateRel);
			const draftYaml = draftYamlForChildCandidate(
				input.layer,
				templateTxt,
				input.parentYaml,
				input.candidate,
				todayYmd()
			);

			state.set({
				open: true,
				session: {
					parentSpecPath: input.parentSpecPath,
					parentYamlSnapshot: input.parentYaml,
					candidate: input.candidate,
					layer: input.layer,
					draftYaml,
					relativePath: input.candidate.relativePath,
					onDone: input.onDone,
				},
				busy: false,
				error: null,
				ok: null,
			});
		},

		async confirmWrite() {
			const snap = get(state);
			const sess = snap.session;
			if (!sess) return;

			const wsRoot = get(specExplorerStore).workspaceRoot;
			if (!wsRoot) {
				state.update(s => ({ ...s, error: '未绑定 workspace root' }));
				return;
			}

			state.update(s => ({ ...s, busy: true, error: null, ok: null }));
			try {
				const relParent = toSpecsRelativePath(sess.parentSpecPath);
				const childPath = sess.relativePath.replace(/\\/g, '/');
				await wailsWriteSpecFile(wsRoot, childPath, sess.draftYaml);

				const parentOnDisk = await wailsReadSpecFile(wsRoot, relParent);
				const patched = appendChildrenPaths(parentOnDisk, [childPath]);
				await wailsWriteSpecFile(wsRoot, relParent, patched);

				await specExplorerStore.loadList(wsRoot);
				sess.onDone?.();

				state.update(s => ({
					...s,
					busy: false,
					ok: `已写入 ${childPath} 并已更新父 spec children`,
					open: false,
					session: null,
				}));
			} catch (e) {
				state.update(s => ({
					...s,
					busy: false,
					error: e instanceof Error ? e.message : String(e),
				}));
			}
		},
	};
}

export const specChildDraftStore = createSpecChildDraftStore();
