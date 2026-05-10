<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { agentApiUrl } from '$lib/runtime/agent-transport';
	type GitFile = { path: string; index: string; worktree: string; status: string };
	let files = $state<GitFile[]>([]);
	let loading = $state(false);
	let error = $state('');
	let commitMsg = $state('');
	let committing = $state(false);
	let tip = $state('');

	async function refresh() {
		const ws = $specExplorerStore.workspaceRoot;
		if (!ws) return;
		loading = true;
		error = '';
		try {
			const res = await fetch(`${agentApiUrl('/api/workspace/git/status')}?workspaceRoot=${encodeURIComponent(ws)}`);
			const data = await res.json();
			if (!res.ok || !data.ok) throw new Error(data.error || `HTTP ${res.status}`);
			files = data.files || [];
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	async function commitBySpec() {
		const ws = $specExplorerStore.workspaceRoot;
		if (!ws) return;
		if (!commitMsg.trim()) {
			tip = '请先填写提交信息';
			return;
		}
		committing = true;
		tip = '';
		error = '';
		try {
			const res = await fetch(agentApiUrl('/api/workspace/git/commit-spec'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					workspace_root: ws,
					spec_path: $specExplorerStore.selectedSpecPath || '',
					message: commitMsg.trim(),
				}),
			});
			const data = await res.json();
			if (!res.ok || !data.ok) throw new Error(data.error || `HTTP ${res.status}`);
			tip = '提交成功';
			commitMsg = '';
			await refresh();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			committing = false;
		}
	}

	$effect(() => {
		void $specExplorerStore.workspaceRoot;
		void refresh();
	});
</script>

<div class="git-side">
	<div class="head">
		<strong>源代码管理</strong>
		<button type="button" onclick={refresh} disabled={loading}>↻</button>
	</div>
	{#if error}<p class="err">{error}</p>{/if}
	<div class="commit-box">
		<div class="spec">关联 spec：{$specExplorerStore.selectedSpecPath || '未选择（仅普通提交）'}</div>
		<input type="text" bind:value={commitMsg} placeholder="提交信息（必填）" />
		<button type="button" onclick={commitBySpec} disabled={committing || loading}>按 spec 提交代码</button>
		{#if tip}<small>{tip}</small>{/if}
	</div>
	<div class="list">
		{#if loading}
			<p class="muted">刷新中…</p>
		{:else if files.length === 0}
			<p class="muted">工作区干净，无待提交改动。</p>
		{:else}
			{#each files as f (f.path)}
				<div class="row">
					<span class="st">{f.status}</span>
					<span class="path" title={f.path}>{f.path}</span>
				</div>
			{/each}
		{/if}
	</div>
</div>

<style>
	.git-side { display: flex; flex-direction: column; height: 100%; background: #151820; color: #eef0f5; border-right: 1px solid #303746; }
	.head { display: flex; align-items: center; justify-content: space-between; padding: 12px; border-bottom: 1px solid #303746; }
	.head button { width: 28px; height: 28px; border-radius: 999px; border: 1px solid #465064; background: #1c202a; color: #a3abb9; cursor: pointer; }
	.commit-box { padding: 10px 12px; border-bottom: 1px solid #303746; display: flex; flex-direction: column; gap: 8px; }
	.commit-box .spec { font-size: 11px; color: #8aa4c7; }
	input { width: 100%; box-sizing: border-box; background: #0f131c; border: 1px solid #465064; border-radius: 6px; color: #eef0f5; padding: 7px 8px; font-size: 12px; }
	.commit-box button { border: 1px solid #7aa2ff; background: rgba(122,162,255,.2); color: #e8f0ff; border-radius: 6px; padding: 6px 8px; font-size: 12px; cursor: pointer; }
	.list { flex: 1; overflow: auto; padding: 8px; }
	.row { display: flex; gap: 8px; padding: 4px 6px; border-radius: 6px; }
	.row:hover { background: rgba(122,162,255,.08); }
	.st { width: 24px; font-family: monospace; color: #efc66b; }
	.path { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
	.err { color: #f87171; margin: 0; padding: 8px 12px; font-size: 12px; }
	.muted { color: #858fa1; font-size: 12px; padding: 8px 4px; margin: 0; }
	small { color: #8fd3c8; }
</style>

