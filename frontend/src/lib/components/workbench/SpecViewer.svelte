<!-- 选中 spec 后主区：默认图谱，可切 Spec 文本（对齐 R2 center tabs） -->
<script lang="ts">
	import { parse as parseYaml } from 'yaml';
	import GoalSpecCanvas from '$lib/components/workbench/GoalSpecCanvas.svelte';
	import GenericSpecGraph from '$lib/components/workbench/GenericSpecGraph.svelte';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	let selectedPath = $state<string | null>(null);
	let centerView = $state<'graph' | 'text'>('graph');

	let raw = $state('');
	let loading = $state(false);
	let fetchErr = $state<string | null>(null);
	let isGoalFile = $state(false);

	$effect(() => {
		return specExplorerStore.subscribe(s => {
			selectedPath = s.selectedSpecPath;
			centerView = s.centerView;
		});
	});

	async function resolveIsGoal(goalPath: string): Promise<boolean> {
		try {
			const r = await fetch(
				`/api/workspace/specs/read?path=${encodeURIComponent('specs/meta/goal-aspect-bindings.yaml')}`
			);
			if (!r.ok) return false;
			const j = (await r.json()) as { content: string };
			const b = parseYaml(j.content) as { goal_spec_path?: string };
			const g = b.goal_spec_path?.replace(/\\/g, '/') ?? '';
			const t = goalPath.replace(/\\/g, '/');
			return g !== '' && t === g;
		} catch {
			return false;
		}
	}

	$effect(() => {
		const p = selectedPath;
		if (!p) {
			raw = '';
			fetchErr = null;
			isGoalFile = false;
			return;
		}
		loading = true;
		fetchErr = null;
		let cancelled = false;

		Promise.all([
			fetch(`/api/workspace/specs/read?path=${encodeURIComponent(p)}`).then(async r => {
				if (!r.ok) throw new Error(await r.text());
				return r.json() as Promise<{ content: string }>;
			}),
			resolveIsGoal(p),
		])
			.then(([data, goal]) => {
				if (cancelled) return;
				raw = data.content;
				isGoalFile = goal;
				fetchErr = null;
			})
			.catch(e => {
				if (cancelled) return;
				fetchErr = e instanceof Error ? e.message : String(e);
				raw = '';
				isGoalFile = false;
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});

		return () => {
			cancelled = true;
		};
	});
</script>

<div class="spec-viewer">
	{#if !selectedPath}
		<div class="empty">
			<p class="t">在左侧资源管理器（📂）中打开 specs 下的 YAML</p>
			<p class="s">
				默认<strong>图谱</strong>；切面/中心卡片详情在<strong>右侧抽屉</strong>（总目标）。
				「Spec 文本」查看完整源码。
			</p>
		</div>
	{:else}
		<div class="spec-tabbar" role="tablist" aria-label="规格主区视图">
			<button
				type="button"
				role="tab"
				class:active={centerView === 'graph'}
				onclick={() => specExplorerStore.setCenterView('graph')}
			>
				图谱
			</button>
			<button
				type="button"
				role="tab"
				class:active={centerView === 'text'}
				onclick={() => specExplorerStore.setCenterView('text')}
			>
				Spec 文本
			</button>
			<span class="spec-tab-spacer"></span>
			<button type="button" class="spec-close" onclick={() => specExplorerStore.selectSpec(null)}>
				返回画布
			</button>
		</div>

		<div class="spec-body">
			{#if loading}
				<p class="muted pad">加载 {selectedPath}…</p>
			{:else if fetchErr}
				<p class="err pad">{fetchErr}</p>
			{:else if centerView === 'text'}
				<pre class="spec-text">{raw}</pre>
			{:else if isGoalFile}
				<div class="goal-wrap">
					<GoalSpecCanvas />
				</div>
			{:else}
				<GenericSpecGraph specPath={selectedPath} content={raw} />
			{/if}
		</div>
	{/if}
</div>

<style>
	.spec-viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: var(--wb-base, #0d0d0e);
		color: var(--wb-text, #e8e8ed);
	}

	.empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		text-align: center;
		color: var(--wb-text-sec, #8a8a8e);
	}

	.empty .t {
		margin: 0 0 0.5rem;
		font-size: 14px;
		color: var(--wb-text, #e8e8ed);
	}

	.empty .s {
		margin: 0;
		font-size: 12px;
		max-width: 320px;
		line-height: 1.5;
	}

	.spec-tabbar {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		height: 36px;
		background: var(--wb-panel-bg, #131314);
		border-bottom: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
	}

	.spec-tabbar button {
		display: flex;
		align-items: center;
		padding: 0 14px;
		height: 100%;
		border: none;
		background: transparent;
		color: var(--wb-muted, #555558);
		font-family: inherit;
		font-size: 13px;
		cursor: pointer;
		border-right: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		position: relative;
	}

	.spec-tabbar button:hover {
		background: rgba(255, 255, 255, 0.05);
		color: var(--wb-text-sec, #8a8a8e);
	}

	.spec-tabbar button.active {
		color: var(--wb-text, #e8e8ed);
		background: var(--wb-base, #0d0d0e);
		margin-bottom: -1px;
		border-bottom: 1px solid var(--wb-base, #0d0d0e);
	}

	.spec-tabbar button.active::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 1px;
		background: var(--wb-brand, #5856d6);
	}

	.spec-tab-spacer {
		flex: 1;
	}

	.spec-close {
		margin-right: 8px;
		padding: 0 12px;
		height: calc(100% - 8px);
		align-self: center;
		font-size: 11px;
		border-radius: 5px;
		border: 1px solid var(--wb-border, rgba(255, 255, 255, 0.07));
		background: rgba(255, 255, 255, 0.04);
		color: var(--wb-text-sec, #8a8a8e);
		cursor: pointer;
	}

	.spec-close:hover {
		color: var(--wb-text, #e8e8ed);
		border-color: rgba(88, 86, 214, 0.35);
	}

	.spec-body {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.goal-wrap {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.spec-text {
		flex: 1;
		margin: 0;
		padding: 16px 24px;
		overflow: auto;
		font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
		font-size: 12px;
		line-height: 1.7;
		color: var(--wb-text-sec, #8a8a8e);
		white-space: pre-wrap;
		word-break: break-word;
	}

	.muted.pad,
	.err.pad {
		padding: 16px 20px;
		font-size: 13px;
	}

	.err {
		color: #f87171;
	}
</style>
