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
				<div class="problem-row ok">
					<span>✓</span>
					<p><b>make validate</b><small>63 specs indexed，parent chain valid</small></p>
				</div>
				<div class="problem-row warn">
					<span>!</span>
					<p><b>backend auto-spawn</b><small>Windows 产物路径仍需修正</small></p>
				</div>
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
				<pre class="mono mono-term">PS C:\project\vibex-workbench&gt; make wails-dev
[agent-build] Building Go agent...
[frontend-build] OK → frontend/build/
Using DevServer URL: http://localhost:34115
ERR | Auto-spawn backend failed: ./backend/vibex-backend
PS C:\project\vibex-workbench&gt;<span class="cursor">▌</span></pre>
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
		background: #1e1e1e;
	}

	.dock-tabs {
		display: flex;
		align-items: center;
		flex-shrink: 0;
		height: var(--dock-tab-h, 28px);
		padding: 0 4px;
		gap: 1px;
		background: #252526;
		border-bottom: 1px solid #2d2d2d;
	}

	.dock-tab {
		display: flex;
		align-items: center;
		gap: 5px;
		padding: 0 10px;
		height: 100%;
		font-size: 11.5px;
		color: #969696;
		border: none;
		background: transparent;
		cursor: pointer;
		border-bottom: 1px solid transparent;
		transition:
			color 150ms ease,
			border-color 150ms ease;
		font-family: var(--font-ui, 'Segoe UI', 'Microsoft YaHei', system-ui, sans-serif);
	}

	.dock-tab:hover {
		color: #cccccc;
	}

	.dock-tab.active {
		color: #ffffff;
		border-bottom-color: #007acc;
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
		background: #c42b1c;
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
		color: #858585;
	}

	.mono {
		margin: 0;
		font-family: var(--font-mono, 'Cascadia Mono', 'JetBrains Mono', Consolas, monospace);
		font-size: 11px;
		line-height: 1.6;
		color: #cccccc;
		white-space: pre-wrap;
	}

	.mono-term {
		background: #1e1e1e;
		padding: 0;
		border-radius: 0;
	}

	.problem-row {
		display: flex;
		align-items: flex-start;
		gap: 9px;
		padding: 6px 4px;
		color: #cccccc;
	}

	.problem-row > span {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		border-radius: 999px;
		font-size: 11px;
		font-weight: 800;
		margin-top: 1px;
	}

	.problem-row.ok > span {
		background: rgba(34, 197, 94, 0.16);
		color: #89d185;
	}

	.problem-row.warn > span {
		background: rgba(245, 158, 11, 0.16);
		color: #d7ba7d;
	}

	.problem-row p {
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.problem-row b {
		font-size: 12px;
		font-weight: 600;
	}

	.problem-row small {
		color: #858585;
		font-size: 11px;
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
