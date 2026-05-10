<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { agentApiUrl } from '$lib/runtime/agent-transport';
	let content = $state('');
	let loading = $state(false);
	let error = $state('');
	let truncated = $state(false);

	async function load() {
		const state = $specExplorerStore;
		if (!state.workspaceRoot || !state.selectedFilePath) return;
		loading = true;
		error = '';
		content = '';
		try {
			const url = `${agentApiUrl('/api/workspace/read-file')}?workspaceRoot=${encodeURIComponent(state.workspaceRoot)}&path=${encodeURIComponent(state.selectedFilePath)}`;
			const res = await fetch(url);
			const data = await res.json();
			if (!res.ok || !data.ok) {
				throw new Error(data.error || `HTTP ${res.status}`);
			}
			content = data.content || '';
			truncated = !!data.truncated;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void $specExplorerStore.selectedFilePath;
		void $specExplorerStore.workspaceRoot;
		void load();
	});
</script>

<div class="file-viewer">
	<div class="head">
		<strong>{$specExplorerStore.selectedFilePath}</strong>
		<button type="button" onclick={load} disabled={loading}>刷新</button>
	</div>
	{#if loading}
		<p class="muted">加载中…</p>
	{:else if error}
		<p class="err">{error}</p>
	{:else}
		{#if truncated}
			<p class="warn">内容过大，已截断显示。</p>
		{/if}
		<pre>{content}</pre>
	{/if}
</div>

<style>
	.file-viewer { height: 100%; display: flex; flex-direction: column; background: #0e1016; color: #eef0f5; }
	.head { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 10px 12px; border-bottom: 1px solid #303746; }
	.head strong { font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.head button { padding: 4px 8px; border-radius: 6px; border: 1px solid #465064; background: #1c202a; color: #a3abb9; cursor: pointer; }
	pre { margin: 0; padding: 12px; overflow: auto; font-size: 12px; line-height: 1.5; flex: 1; }
	.muted { color: #858fa1; padding: 12px; }
	.err { color: #f87171; padding: 12px; }
	.warn { color: #efc66b; padding: 8px 12px; margin: 0; border-bottom: 1px dashed #465064; }
</style>

