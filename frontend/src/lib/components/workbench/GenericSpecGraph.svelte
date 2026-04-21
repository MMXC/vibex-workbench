<!-- 非 L1 总目标时的轻量「图谱」：中心为 spec 标识，周围为顶层段落卡片 -->
<script lang="ts">
	import { parse as parseYaml } from 'yaml';

	let { specPath, content }: { specPath: string; content: string } = $props();

	const parsed = $derived.by(() => {
		try {
			return parseYaml(content) as Record<string, unknown>;
		} catch {
			return null;
		}
	});

	const title = $derived.by(() => {
		const p = parsed;
		if (!p) return specPath.split('/').pop() ?? specPath;
		const spec = p.spec as Record<string, unknown> | undefined;
		const name = spec?.name;
		return typeof name === 'string' ? name : (specPath.split('/').pop() ?? specPath);
	});

	const level = $derived.by(() => {
		const p = parsed;
		const spec = p?.spec as Record<string, unknown> | undefined;
		const l = spec?.level;
		return typeof l === 'string' ? l : '';
	});

	const sectionKeys = $derived.by(() => {
		const p = parsed;
		if (!p) return [] as string[];
		return Object.keys(p).filter(k => k !== 'spec');
	});
</script>

<div class="generic-graph">
	<div class="radial">
		{#each sectionKeys as key, i (key)}
			{@const n = Math.max(sectionKeys.length, 1)}
			{@const angle = (2 * Math.PI * i) / n - Math.PI / 2}
			{@const r = 38}
			{@const x = 50 + r * Math.cos(angle)}
			{@const y = 50 + r * Math.sin(angle)}
			<div class="orbit-card" style:left="{x}%" style:top="{y}%">
				<span class="k">{key}</span>
			</div>
		{/each}

		<div class="center-card">
			<span class="kicker">Spec 文件</span>
			<h2 class="title">{title}</h2>
			{#if level}
				<p class="meta">{level}</p>
			{/if}
			<p class="path">{specPath}</p>
		</div>
	</div>
	<p class="hint">顶层 YAML 键已分布在周围；详细契约请在「Spec 文本」中查看。</p>
</div>

<style>
	.generic-graph {
		height: 100%;
		min-height: 280px;
		display: flex;
		flex-direction: column;
		padding: 0.75rem;
		background: var(--wb-main-bg, #0d0d0d);
		color: #e5e5e5;
	}

	.radial {
		position: relative;
		flex: 1;
		min-height: 260px;
		max-height: min(48vh, 400px);
	}

	.center-card {
		position: absolute;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		width: min(88%, 320px);
		padding: 1rem;
		border-radius: 12px;
		border: 1px solid rgba(88, 86, 214, 0.35);
		background: linear-gradient(145deg, #18181b 0%, #0f0f12 100%);
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.45);
	}

	.kicker {
		display: block;
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #71717a;
		margin-bottom: 0.35rem;
	}

	.title {
		margin: 0 0 0.35rem;
		font-size: 1.05rem;
		font-weight: 600;
		color: #fafafa;
	}

	.meta {
		margin: 0;
		font-size: 11px;
		color: #a78bfa;
	}

	.path {
		margin: 0.5rem 0 0;
		font-size: 10px;
		font-family: ui-monospace, monospace;
		color: #71717a;
		word-break: break-all;
		line-height: 1.35;
	}

	.orbit-card {
		position: absolute;
		transform: translate(-50%, -50%);
		min-width: 72px;
		max-width: 120px;
		padding: 0.35rem 0.45rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 8px;
		border: 1px solid #3f3f46;
		background: #18181b;
		color: #d4d4d8;
		text-align: center;
		pointer-events: none;
	}

	.orbit-card .k {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.hint {
		flex-shrink: 0;
		margin: 0.5rem 0 0;
		font-size: 11px;
		color: #71717a;
		line-height: 1.45;
	}
</style>
