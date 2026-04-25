<!-- R2 底部 Dock：问题 | 输出 | 终端 | 调试 | 更多 — prototypes/vibex-ide-chrome-r2.html #dock -->
<script lang="ts">
	import { outputText, outputVisible } from '$lib/stores/workspace-output-store';

	type PanelId = 'problems' | 'output' | 'terminal' | 'debug' | 'more';

	let panel = $state<PanelId>('problems');
</script>

<div class="dock">
	<div class="dock-tabs" role="tablist" aria-label="底部面板">
		<button
			type="button"
			class="dock-tab"
			class:active={panel === 'problems'}
			onclick={() => (panel = 'problems')}
		>
			问题
			<span class="badge">2</span>
		</button>
		<button
			type="button"
			class="dock-tab"
			class:active={panel === 'output'}
			onclick={() => (panel = 'output')}
		>
			输出
			{#if $outputVisible}
				<span class="badge"></span>
			{:else}
				<span class="badge none">0</span>
			{/if}
		</button>
		<button
			type="button"
			class="dock-tab"
			class:active={panel === 'terminal'}
			onclick={() => (panel = 'terminal')}
		>
			终端
			<span class="badge none">0</span>
		</button>
		<button
			type="button"
			class="dock-tab"
			class:active={panel === 'debug'}
			onclick={() => (panel = 'debug')}
		>
			调试
			<span class="badge none">0</span>
		</button>
		<button
			type="button"
			class="dock-tab dock-more"
			class:active={panel === 'more'}
			onclick={() => (panel = 'more')}
			aria-label="更多"
		>
			⋯
		</button>
	</div>

	<div class="dock-body">
		{#if panel === 'problems'}
			<div class="panel problems">
				<p class="muted">Problems — 占位（接入诊断 / validate 结果）</p>
			</div>
		{:else if panel === 'output'}
			<div class="panel output">
				{#if $outputText}
					<pre class="mono">{$outputText}</pre>
				{:else}
					<pre class="mono muted">[output] 构建与日志占位</pre>
				{/if}
			</div>
		{:else if panel === 'terminal'}
			<div class="panel terminal">
				<pre class="mono mono-term">~/vibex-workbench $<span class="cursor">▌</span></pre>
			</div>
		{:else if panel === 'debug'}
			<div class="panel">
				<p class="muted">暂无调试会话</p>
			</div>
		{:else}
			<div class="panel">
				<p class="muted">更多面板 — 占位</p>
			</div>
		{/if}
	</div>
</div>

<style>
	.dock {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-height: 0;
		background: var(--bg-panel, #131314);
	}

	.dock-tabs {
		display: flex;
		align-items: center;
		flex-shrink: 0;
		height: var(--dock-tab-h, 28px);
		padding: 0 4px;
		gap: 1px;
		background: var(--bg-surface, #1a1a1c);
		border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.07));
	}

	.dock-tab {
		display: flex;
		align-items: center;
		gap: 5px;
		padding: 0 10px;
		height: 100%;
		font-size: 11.5px;
		color: var(--text-muted, #555558);
		border: none;
		background: transparent;
		cursor: pointer;
		border-bottom: 1px solid transparent;
		transition:
			color 150ms ease,
			border-color 150ms ease;
		font-family: var(--font-ui, 'Inter', sans-serif);
	}

	.dock-tab:hover {
		color: var(--text-secondary, #8a8a8e);
	}

	.dock-tab.active {
		color: var(--text-primary, #e8e8ed);
		border-bottom-color: var(--brand, #5856d6);
	}

	.dock-more {
		padding: 0 12px;
		font-size: 14px;
		line-height: 1;
	}

	.badge {
		font-size: 9px;
		padding: 0 4px;
		border-radius: 9px;
		background: var(--error, #ef4444);
		color: #fff;
		font-weight: 700;
		line-height: 1.4;
	}

	.badge.none {
		display: none;
	}

	.dock-body {
		flex: 1;
		min-height: 0;
		overflow: auto;
	}

	.panel {
		padding: 10px 12px;
		font-size: 12px;
		min-height: 100%;
		box-sizing: border-box;
	}

	.muted {
		margin: 0;
		color: var(--text-muted, #555558);
	}

	.mono {
		margin: 0;
		font-family: var(--font-mono, 'JetBrains Mono', monospace);
		font-size: 11px;
		line-height: 1.6;
		color: var(--text-secondary, #8a8a8e);
		white-space: pre-wrap;
	}

	.mono-term {
		background: #0a0a0b;
		padding: 8px;
		border-radius: 4px;
	}

	.cursor {
		animation: blink 1s step-end infinite;
	}

	@keyframes blink {
		50% {
			opacity: 0;
		}
	}
</style>
