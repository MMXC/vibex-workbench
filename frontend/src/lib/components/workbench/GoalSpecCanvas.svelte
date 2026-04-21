<!-- GoalSpecCanvas — L1 总目标 + 切面钻取 MVP（Spec: specs/meta/goal-aspect-bindings.yaml） -->
<script lang="ts">
	import { parse as parseYaml } from 'yaml';

	type AspectBinding = {
		id: string;
		label: string;
		target_path: string;
		hint?: string;
	};

	type BindingsFile = {
		version: string;
		goal_spec_path: string;
		aspects: AspectBinding[];
	};

	let bindings = $state<BindingsFile | null>(null);
	let goalPayload = $state<{ path: string; content: string } | null>(null);
	let goalParsed = $state<Record<string, unknown> | null>(null);
	let loadError = $state<string | null>(null);

	/** 右侧抽屉：mission | aspect 文件详情 */
	type DrawerMode = 'off' | 'mission' | 'aspect';
	let drawerMode = $state<DrawerMode>('off');
	let drillPath = $state<string | null>(null);
	let drillContent = $state<string | null>(null);
	let drillLoading = $state(false);

	const projectTitle = $derived.by(() => {
		const m = goalParsed?.meta as Record<string, unknown> | undefined;
		const p = m?.project;
		return typeof p === 'string' ? p : 'VibeX Workbench';
	});

	const missionText = $derived.by(() => {
		const c = goalParsed?.content as Record<string, unknown> | undefined;
		const raw = c?.mission;
		return typeof raw === 'string' ? raw.trim() : '';
	});

	const missionPreview = $derived.by(() => {
		const t = missionText;
		if (!t) return '';
		const line = t.split('\n').find(l => l.trim().length > 0) ?? t;
		return line.length > 160 ? line.slice(0, 157) + '…' : line;
	});

	const milestoneCount = $derived.by(() => {
		const c = goalParsed?.content as Record<string, unknown> | undefined;
		const ms = c?.milestones;
		return Array.isArray(ms) ? ms.length : 0;
	});

	async function fetchSpec(path: string): Promise<{ path: string; content: string }> {
		const r = await fetch(`/api/workspace/specs/read?path=${encodeURIComponent(path)}`);
		if (!r.ok) {
			const t = await r.text();
			throw new Error(t || r.statusText);
		}
		return r.json();
	}

	async function loadAll() {
		loadError = null;
		try {
			const b = await fetchSpec('specs/meta/goal-aspect-bindings.yaml');
			bindings = parseYaml(b.content) as BindingsFile;
			const goalPath = bindings.goal_spec_path;
			goalPayload = await fetchSpec(goalPath);
			goalParsed = parseYaml(goalPayload.content) as Record<string, unknown>;
		} catch (e) {
			loadError = e instanceof Error ? e.message : String(e);
			bindings = null;
			goalPayload = null;
			goalParsed = null;
		}
	}

	async function openAspectDrawer(path: string) {
		drawerMode = 'aspect';
		drillPath = path;
		drillContent = null;
		drillLoading = true;
		try {
			const r = await fetchSpec(path);
			drillContent = r.content;
		} catch (e) {
			drillContent = `读取失败: ${e instanceof Error ? e.message : String(e)}`;
		} finally {
			drillLoading = false;
		}
	}

	function openMissionDrawer() {
		drawerMode = 'mission';
		drillPath = null;
		drillContent = null;
		drillLoading = false;
	}

	function closeDrawer() {
		drawerMode = 'off';
		drillPath = null;
		drillContent = null;
		drillLoading = false;
	}

	$effect(() => {
		loadAll();
	});
</script>

<div class="goal-canvas">
	{#if loadError}
		<div class="banner error">
			<p>无法加载总目标图谱（需本地 dev 且可读仓库 <code>specs/</code>）。</p>
			<code class="err-msg">{loadError}</code>
			<button type="button" class="retry" onclick={() => loadAll()}>重试</button>
		</div>
	{:else if !bindings || !goalParsed}
		<div class="banner loading">加载规格与绑定表…</div>
	{:else}
		<div class="graph-wrap">
			<div class="radial">
				{#each bindings.aspects as aspect, i (aspect.id)}
					{@const n = bindings.aspects.length}
					{@const angle = (2 * Math.PI * i) / n - Math.PI / 2}
					{@const r = 42}
					{@const x = 50 + r * Math.cos(angle)}
					{@const y = 50 + r * Math.sin(angle)}
					<button
						type="button"
						class="aspect-card"
						style:left="{x}%"
						style:top="{y}%"
						title={aspect.hint ?? aspect.target_path}
						onclick={() => openAspectDrawer(aspect.target_path)}
					>
						<span class="aspect-label">{aspect.label}</span>
					</button>
				{/each}

				<button type="button" class="center-card" onclick={openMissionDrawer}>
					<span class="center-kicker">L1 总目标</span>
					<h2 class="center-title">{projectTitle}</h2>
					<p class="center-desc">{missionPreview}</p>
					<span class="center-meta">里程碑条目：{milestoneCount} · 点击在右侧抽屉查看 mission 全文</span>
				</button>
			</div>
		</div>

		{#if drawerMode !== 'off'}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<div class="drawer-backdrop" onclick={closeDrawer} role="presentation" aria-hidden="true"></div>
			<aside class="detail-drawer" aria-label="规格详情">
				<div class="drawer-head">
					<span class="drawer-title">
						{#if drawerMode === 'mission'}
							Mission（总目标陈述）
						{:else}
							{drillPath ?? ''}
						{/if}
					</span>
					<button type="button" class="drawer-close" onclick={closeDrawer}>关闭</button>
				</div>
				<div class="drawer-scroll">
					{#if drawerMode === 'mission'}
						<pre class="drawer-pre">{missionText || '（无 mission 字段）'}</pre>
					{:else if drillLoading}
						<p class="drawer-muted">读取中…</p>
					{:else if drillContent}
						<pre class="drawer-pre">{drillContent}</pre>
					{/if}
				</div>
			</aside>
		{/if}
	{/if}
</div>

<style>
	.goal-canvas {
		position: relative;
		height: 100%;
		min-height: 0;
		display: flex;
		flex-direction: column;
		background: var(--wb-main-bg, #0d0d0d);
		color: #e5e5e5;
		font-size: 13px;
	}

	.banner {
		padding: 1rem 1.25rem;
		margin: 1rem;
		border-radius: 8px;
		background: #1a1a1a;
		border: 1px solid #333;
	}
	.banner.error {
		border-color: #7f1d1d;
	}
	.err-msg {
		display: block;
		margin: 0.5rem 0;
		font-size: 12px;
		color: #fca5a5;
		white-space: pre-wrap;
		word-break: break-all;
	}
	.retry {
		margin-top: 0.5rem;
		padding: 0.35rem 0.75rem;
		background: #27272a;
		border: 1px solid #3f3f46;
		color: #e4e4e7;
		border-radius: 6px;
		cursor: pointer;
	}

	.graph-wrap {
		flex: 1;
		min-height: 320px;
		display: flex;
		flex-direction: column;
		padding: 0.75rem;
		overflow: auto;
	}

	.radial {
		position: relative;
		width: 100%;
		flex: 1;
		min-height: 280px;
		max-height: min(52vh, 420px);
	}

	.center-card {
		position: absolute;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		width: min(92%, 340px);
		padding: 1rem 1.1rem;
		text-align: left;
		border-radius: 12px;
		border: 1px solid rgba(129, 140, 248, 0.35);
		background: linear-gradient(145deg, #18181b 0%, #0f0f12 100%);
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.45);
		cursor: pointer;
		color: inherit;
		font: inherit;
	}

	.center-kicker {
		display: block;
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: #a1a1aa;
		margin-bottom: 0.35rem;
	}

	.center-title {
		margin: 0 0 0.5rem;
		font-size: 1.15rem;
		font-weight: 600;
		color: #fafafa;
	}

	.center-desc {
		margin: 0;
		line-height: 1.45;
		color: #d4d4d8;
		font-size: 12px;
	}

	.center-meta {
		display: block;
		margin-top: 0.65rem;
		font-size: 11px;
		color: #71717a;
	}

	.aspect-card {
		position: absolute;
		transform: translate(-50%, -50%);
		min-width: 100px;
		max-width: 140px;
		padding: 0.45rem 0.55rem;
		font-size: 11px;
		line-height: 1.25;
		border-radius: 8px;
		border: 1px solid #3f3f46;
		background: #18181b;
		color: #e4e4e7;
		cursor: pointer;
		text-align: center;
		transition:
			border-color 0.15s,
			background 0.15s;
	}
	.aspect-card:hover {
		border-color: rgba(129, 140, 248, 0.65);
		background: #27272a;
	}

	.aspect-label {
		display: block;
	}

	.drawer-backdrop {
		position: absolute;
		inset: 0;
		z-index: 19;
		background: rgba(0, 0, 0, 0.42);
	}

	.detail-drawer {
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		z-index: 20;
		width: min(420px, 92vw);
		display: flex;
		flex-direction: column;
		background: #141416;
		border-left: 1px solid rgba(255, 255, 255, 0.1);
		box-shadow: -12px 0 40px rgba(0, 0, 0, 0.45);
	}

	.drawer-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 0.65rem 0.85rem;
		border-bottom: 1px solid #27272a;
		font-size: 11px;
	}

	.drawer-title {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: #c4b5fd;
		font-family: ui-monospace, monospace;
	}

	.drawer-close {
		flex-shrink: 0;
		padding: 0.3rem 0.55rem;
		font-size: 11px;
		background: #27272a;
		border: 1px solid #3f3f46;
		color: #e4e4e7;
		border-radius: 4px;
		cursor: pointer;
	}

	.drawer-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 0.75rem;
	}

	.drawer-pre {
		margin: 0;
		font-size: 11px;
		line-height: 1.45;
		white-space: pre-wrap;
		word-break: break-word;
		color: #d4d4d8;
	}

	.drawer-muted {
		margin: 0;
		padding: 0.75rem;
		color: #71717a;
		font-size: 12px;
	}
</style>
